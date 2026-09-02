package text

import (
	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
)

// subRun is a homogeneous slice of a style run after segmentation: one
// direction, one script, one face. baseRune is its start rune index within the
// paragraph; baseByte is its start byte offset within the paragraph text.
type subRun struct {
	baseRune int
	baseByte int
	dir      di.Direction
	script   language.Script
	face     *font.Face
	out      shaping.Output // shaped in isolation, ClusterIndex 0-based within the run
}

// segmentRun splits a single style run (described by its rune range within the
// paragraph) into homogeneous sub-runs by bidi level, script and face, then
// shapes each. The paragraph runes and rune-to-byte map provide context and
// coordinate translation.
//
// Segmentation uses the full paragraph as context so bidi ordering and script
// inheritance are correct at run boundaries; shaping is done in isolation per
// sub-run so the result is cacheable by content.
func (s *System) segmentRun(
	paragraph []rune,
	runStart, runEnd int,
	baseByte int,
	req FontRequest,
	size float32,
	paragraphDir di.Direction,
	lang language.Language,
	features []shaping.FontFeature,
) []subRun {
	primary := s.Resolve(req).face

	input := shaping.Input{
		Text:         paragraph,
		RunStart:     runStart,
		RunEnd:       runEnd,
		Direction:    paragraphDir,
		Face:         primary,
		Script:       language.Unknown,
		Language:     lang,
		FontFeatures: features,
	}

	fmap := runFontmap{
		primary: primary,
		system:  s.fm,
		query:   fontscan.Query{Families: req.families(), Aspect: req.aspect()},
	}

	runs := s.seg.Split(input, fmap)

	// Split carries the input's Language and FontFeatures through to every
	// sub-run unchanged, so their cache-key string forms are interned once
	// here rather than once per sub-run.
	langKey := string(lang)
	featsKey := featuresKey(features)

	out := make([]subRun, 0, len(runs))
	for _, r := range runs {
		// The sub-run's runes, isolated for cacheable shaping.
		subText := string(paragraph[r.RunStart:r.RunEnd])
		in := shapeInput{
			text:           subText,
			face:           r.Face,
			size:           size,
			direction:      r.Direction,
			script:         r.Script,
			language:       r.Language,
			languageKey:    langKey,
			features:       features,
			featuresKeyStr: featsKey,
		}
		shaped := s.shapeCache.get(&s.shaper, in)

		// Byte offset of the sub-run's start: count bytes up to its first rune.
		subBaseByte := baseByte + runeByteOffset(paragraph, runStart, r.RunStart)

		out = append(out, subRun{
			baseRune: r.RunStart,
			baseByte: subBaseByte,
			dir:      r.Direction,
			script:   r.Script,
			face:     r.Face,
			out:      shaped,
		})
	}
	return out
}

// runeByteOffset returns the byte offset of rune at index toRune, measured from
// the byte position of rune at index fromRune. It walks forward over the
// paragraph runes, encoding each to count its bytes.
func runeByteOffset(paragraph []rune, fromRune, toRune int) int {
	off := 0
	for i := fromRune; i < toRune; i++ {
		off += utf8RuneLen(paragraph[i])
	}
	return off
}

// utf8RuneLen returns the number of bytes r encodes to in UTF-8.
func utf8RuneLen(r rune) int {
	switch {
	case r < 0:
		return 1
	case r < 0x80:
		return 1
	case r < 0x800:
		return 2
	case r < 0x10000:
		return 3
	default:
		return 4
	}
}
