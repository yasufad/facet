package text

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"sort"

	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
)

// System is the text system: a font map over the system's installed fonts (and
// any fonts added explicitly), a reusable shaper and segmenter, and the caches
// that make repeated shaping cheap.
//
// A System is not safe for concurrent use. The UI runs on one goroutine and a
// System is shared across the frames that goroutine drives; it must not cross
// goroutines.
type System struct {
	// fm resolves faces by family, aspect and rune, applying CSS-style
	// fallback when a face lacks a glyph.
	fm *fontscan.FontMap
	// shaper shapes segmented runs. It is reused to amortise its internal
	// HarfBuzz buffer and font cache.
	shaper shaping.HarfbuzzShaper
	// seg splits text into bidi, script and face runs. Reused to amortise
	// its scratch buffers.
	seg shaping.Segmenter

	// shapeCache holds shaped output keyed by run, so the same word in the
	// same face at the same size is shaped once.
	shapeCache shapeCache

	// lineCache holds fully wrapped ShapedLines keyed by text, style runs and
	// wrap width, so a repeated ShapeLine or WrapText call for the same
	// arguments skips segmentation and wrapping entirely, not just the
	// HarfBuzz call shapeCache saves.
	lineCache lineCache

	// faceCache memoises resolved primary faces by request, so repeated
	// Resolve calls do not re-run the matcher.
	faceCache map[fontRequestKey]Face

	// families are the distinct family names found on the system, sorted.
	families []string
}

// fontRequestKey is the cache key for a resolved Face. Stretch and Weight are
// bit-stable as float32 via bits.Float32ToBits; Style is a small int.
type fontRequestKey struct {
	family  string
	weight  uint32
	style   uint8
	stretch uint32
}

func (r FontRequest) key() fontRequestKey {
	return fontRequestKey{
		family:  r.Family,
		weight:  bitsOfFloat32(float32(r.Weight)),
		style:   uint8(r.Style),
		stretch: bitsOfFloat32(float32(r.Stretch)),
	}
}

// NewSystem loads the system fonts and returns a text system ready to resolve
// faces and shape text.
//
// The first call scans the system font directories, which is slow (a few
// tenths of a second); typesetting caches the resulting index on disk so
// subsequent calls are fast.
func NewSystem() (*System, error) {
	logger := log.New(io.Discard, "facet/text", 0)
	fm := fontscan.NewFontMap(logger)
	if err := fm.UseSystemFonts(""); err != nil {
		return nil, fmt.Errorf("load system fonts: %w", err)
	}

	footprints, err := fontscan.SystemFonts(logger, "")
	if err != nil {
		return nil, fmt.Errorf("enumerate system fonts: %w", err)
	}

	s := &System{
		fm:         fm,
		faceCache:  make(map[fontRequestKey]Face),
		shapeCache: newShapeCache(),
		lineCache:  newLineCache(),
	}
	s.families = distinctFamilies(footprints)
	return s, nil
}

// distinctFamilies returns the sorted, de-duplicated family names from the
// system font footprints.
func distinctFamilies(footprints []fontscan.Footprint) []string {
	seen := make(map[string]struct{}, len(footprints))
	for _, fp := range footprints {
		if fp.Family != "" {
			seen[fp.Family] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for family := range seen {
		out = append(out, family)
	}
	sort.Strings(out)
	return out
}

// Families returns the distinct family names available on the system, sorted.
// The slice is not mutated by the caller and is shared; do not modify it.
func (s *System) Families() []string { return s.families }

// AddFont loads a font from its bytes and adds it to the system under the
// given family name. If family is empty the name embedded in the font is used.
// The returned Face refers to the first face in the file.
func (s *System) AddFont(data []byte, family string) (Face, error) {
	resource := bytes.NewReader(data)
	if err := s.fm.AddFont(resource, family, family); err != nil {
		return Face{}, fmt.Errorf("add font %q: %w", family, err)
	}
	// AddFont does not return the face; resolve it by querying for the family
	// and probing a rune every font covers.
	req := FontRequest{Family: family}
	s.fm.SetQuery(fontscan.Query{Families: req.families(), Aspect: req.aspect()})
	face := s.fm.ResolveFace(' ')
	if face == nil {
		return Face{}, fmt.Errorf("add font %q: loaded face is nil", family)
	}
	s.faceCache[req.withDefaults().key()] = Face{face: face}
	return Face{face: face}, nil
}

// Resolve returns the face that best matches the request, applying CSS-style
// font matching. When no exact match exists the system fallback stack is
// consulted, so Resolve only fails when the system has no usable font at all.
// The result is cached by request.
func (s *System) Resolve(req FontRequest) Face {
	req = req.withDefaults()
	if face, ok := s.faceCache[req.key()]; ok {
		return face
	}
	s.fm.SetQuery(fontscan.Query{Families: req.families(), Aspect: req.aspect()})
	face := s.fm.ResolveFace(' ')
	if face == nil {
		return Face{}
	}
	f := Face{face: face}
	s.faceCache[req.key()] = f
	return f
}

// resolveFallback returns a face covering r for the given request, or the
// request's primary face when nothing better is available. It is the per-rune
// fallback used during segmentation.
func (s *System) resolveFallback(req FontRequest, script language.Script, r rune) *font.Face {
	s.fm.SetQuery(fontscan.Query{Families: req.families(), Aspect: req.aspect()})
	s.fm.SetScript(script)
	return s.fm.ResolveFace(r)
}

// runFontmap is a shaping.Fontmap that uses a run's primary face for every
// rune it covers, and falls back to the system font map for runes it does not.
// It is the mechanism behind "the request is for text, not for a font": a
// missing glyph does not stop shaping, it switches face for that rune.
type runFontmap struct {
	primary *font.Face
	system  *fontscan.FontMap
	query   fontscan.Query
}

// ResolveFace implements shaping.Fontmap. It must never return nil.
func (m runFontmap) ResolveFace(r rune) *font.Face {
	if m.primary != nil {
		if _, ok := m.primary.NominalGlyph(r); ok {
			return m.primary
		}
	}
	m.system.SetQuery(m.query)
	if face := m.system.ResolveFace(r); face != nil {
		return face
	}
	// ResolveFace is contractually non-nil; fall back to the primary so that
	// shaping still produces a glyph (the notdef box) rather than panicking.
	return m.primary
}

// SetScript implements shaping.FontmapScript, letting the system font map pick
// script-aware fallbacks for Common and Inherited runes.
func (m runFontmap) SetScript(script language.Script) {
	m.system.SetQuery(m.query)
	m.system.SetScript(script)
}
