package text

import (
	"fmt"
	"unsafe"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"github.com/yasufad/facet/geometry"
)

// Direction is the layout direction of a paragraph. Only horizontal directions
// are exposed; vertical text is not yet handled.
type Direction uint8

const (
	// LTR is left-to-right, the default for Latin and Cyrillic.
	LTR Direction = iota
	// RTL is right-to-left, the default for Arabic and Hebrew.
	RTL
)

func (d Direction) di() di.Direction {
	if d == RTL {
		return di.DirectionRTL
	}
	return di.DirectionLTR
}

// StyleRun styles a contiguous byte range of the text passed to ShapeLine or
// WrapText. The byte ranges of consecutive runs must exactly cover the text
// with no gaps or overlaps.
type StyleRun struct {
	// ByteLen is the number of UTF-8 bytes this run covers.
	ByteLen int
	// Font is the request to resolve a face for. The system resolves a primary
	// face from it and uses its family and aspect to pick fallback faces for
	// runes the primary face lacks.
	Font FontRequest
	// Size is the font size in logical pixels.
	Size geometry.Pixels
	// Direction is the paragraph direction of the run. It drives bidi
	// ordering; runs are still split by script automatically.
	Direction Direction
	// Language is a BCP-47 language tag such as "en" or "ar". It influences
	// shaping and font selection; the empty string defaults to English.
	Language string
	// Features activates or deactivates OpenType features for this run.
	Features []FontFeature
}

// Glyph is a single shaped glyph ready to place. Position is in line
// coordinates: X from the line's left edge, Y from the line's top edge with the
// baseline at the line's ascent. Cluster is the byte offset within the line's
// text of the first byte of the source cluster this glyph represents.
type Glyph struct {
	// ID identifies the glyph within its face. It is opaque above this package.
	ID GlyphID
	// Position is the glyph's pen position in line coordinates.
	Position geometry.Point[geometry.Pixels]
	// Cluster is the byte offset into the line's text of the rune cluster this
	// glyph belongs to.
	Cluster int
	// Face is the face this glyph was shaped in, which may differ from the
	// run's requested face when fallback selected another font for this rune.
	Face Face
}

// ShapedRun is a run of glyphs shaped in a single face and direction.
type ShapedRun struct {
	Face      Face
	Direction Direction
	Glyphs    []Glyph
}

// shapeInput assembles the typesetting Input for a homogeneous sub-run, shaped
// in isolation so the result depends only on the sub-run's own content and can
// be cached.
type shapeInput struct {
	text      string // the sub-run's runes, as UTF-8
	face      *font.Face
	size      float32
	direction di.Direction
	script    language.Script
	language  language.Language
	features  []shaping.FontFeature

	// languageKey and featuresKeyStr are string(language) and featuresKey(features),
	// computed once per style run by the caller rather than per sub-run: every
	// sub-run of a segmented style run shares the same language and features,
	// so segmentRun interns them once and every shapeInput built from that run
	// carries the same strings. key() then only copies them, instead of paying
	// for the conversion on every cache lookup, hit or miss.
	languageKey    string
	featuresKeyStr string
}

// cacheKey is the shapeCache key. It captures everything that changes a run's
// shaped output, so the same word in the same face at the same size hits. The
// face is a pointer, which is comparable and stable for the lifetime of the
// face, making it a sound map key.
type cacheKey struct {
	text      string
	face      *font.Face
	size      uint32
	direction uint8
	script    uint32
	language  string
	features  string
}

func (in shapeInput) key() cacheKey {
	return cacheKey{
		text:      in.text,
		face:      in.face,
		size:      bitsOfFloat32(in.size),
		direction: uint8(in.direction),
		script:    uint32(in.script),
		language:  in.languageKey,
		features:  in.featuresKeyStr,
	}
}

// featuresKey renders a feature list into a stable string for use as a cache
// key. The list is already in caller order, which shaping preserves.
func featuresKey(features []shaping.FontFeature) string {
	if len(features) == 0 {
		return ""
	}
	var b []byte
	for _, f := range features {
		tag := ot.Tag(f.Tag)
		b = append(b, byte(tag>>24), byte(tag>>16), byte(tag>>8), byte(tag))
		b = appendUint32(b, f.Value)
		b = append(b, ';')
	}
	return string(b)
}

func appendUint32(b []byte, v uint32) []byte {
	for i := 0; i < 4; i++ {
		b = append(b, byte(v>>24))
		v <<= 8
	}
	return b
}

// toTypesettingFeatures converts our FontFeature slice into the typesetting
// shaping.FontFeature slice.
func toTypesettingFeatures(features []FontFeature) []shaping.FontFeature {
	if len(features) == 0 {
		return nil
	}
	out := make([]shaping.FontFeature, len(features))
	for i, f := range features {
		out[i] = shaping.FontFeature{
			Tag:   ot.MustNewTag(f.Tag),
			Value: f.Value,
		}
	}
	return out
}

// defaultShapeCacheBytes bounds shapeCache. A shaping.Glyph is roughly 80
// bytes; a screen of code at 60 lines by 100 columns is around 6,000 glyphs,
// so one screen's worth of shaped output is under 500 KiB. 4 MiB covers
// several such screens across tabs, font sizes and scripts without the
// unbounded growth an editor session run across a working day would
// otherwise accumulate.
const defaultShapeCacheBytes = 4 << 20

// shapeCache memoises shaped Output by run, bounded by total byte size. It is
// not safe for concurrent use.
type shapeCache struct {
	lru *byteLRU[cacheKey, shaping.Output]
}

func newShapeCache() shapeCache {
	return shapeCache{lru: newByteLRU[cacheKey, shaping.Output](defaultShapeCacheBytes, sizeOfOutput)}
}

// sizeOfOutput estimates the byte weight of a shaped Output for the cache's
// eviction accounting: its glyph slice dominates the cost.
func sizeOfOutput(out shaping.Output) int64 {
	const glyphSize = int64(unsafe.Sizeof(shaping.Glyph{}))
	return int64(len(out.Glyphs))*glyphSize + 64 // +64 for the Output header itself
}

// SetMaxBytes sets the shape cache's byte ceiling, evicting immediately if
// the cache is already over it. The default is defaultShapeCacheBytes.
func (s *System) SetShapeCacheBytes(n int64) { s.shapeCache.lru.SetMaxBytes(n) }

// get returns the shaped Output for in, shaping on miss. The returned Output
// has ClusterIndex values 0-based within in.text and Runes.Offset zero.
func (c *shapeCache) get(shaper *shaping.HarfbuzzShaper, in shapeInput) shaping.Output {
	key := in.key()
	if out, ok := c.lru.Get(key); ok {
		return out
	}
	runes := []rune(in.text)
	input := shaping.Input{
		Text:         runes,
		RunStart:     0,
		RunEnd:       len(runes),
		Direction:    in.direction,
		Face:         in.face,
		Size:         toFixed(in.size),
		Script:       in.script,
		Language:     in.language,
		FontFeatures: in.features,
	}
	out := shaper.Shape(input)
	// Store a copy whose slices we own, so the cache does not alias the
	// shaper's internal buffers on future calls. Shape already allocates fresh
	// Glyphs, so this is belt and braces.
	stored := out
	stored.Glyphs = append([]shaping.Glyph(nil), out.Glyphs...)
	c.lru.Put(key, stored)
	return stored
}

// validateRuns checks that the runs cover exactly textLen bytes.
func validateRuns(textLen int, runs []StyleRun) error {
	total := 0
	for i, r := range runs {
		if r.ByteLen < 0 {
			return fmt.Errorf("text: run %d has negative length %d", i, r.ByteLen)
		}
		total += r.ByteLen
	}
	if total != textLen {
		return fmt.Errorf("text: runs cover %d bytes, text is %d bytes", total, textLen)
	}
	return nil
}
