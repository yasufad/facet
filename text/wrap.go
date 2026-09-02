package text

import (
	"fmt"
	"unicode/utf8"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/language"
	"github.com/go-text/typesetting/shaping"
	"github.com/yasufad/facet/geometry"
	"golang.org/x/image/math/fixed"
)

// noWrapWidth is a width large enough that the line wrapper never breaks a
// line of realistic length. It is used by ShapeLine to lay out a single line.
var noWrapWidth = fixed.Int26_6(1<<23) * 64

// ShapeLine shapes a single line of text (no newlines) and returns its layout.
// It is the entry point for painting one line and for hit testing within it.
// The text must not contain '\n'; use WrapText for multi-line text.
func (s *System) ShapeLine(text string, runs []StyleRun) (ShapedLine, error) {
	if stringsContainsNewline(text) {
		return ShapedLine{}, fmt.Errorf("text: ShapeLine text must not contain newlines")
	}
	if err := validateRuns(len(text), runs); err != nil {
		return ShapedLine{}, err
	}
	lines, err := s.wrap(text, runs, noWrapWidth)
	if err != nil {
		return ShapedLine{}, err
	}
	if len(lines) == 0 {
		return ShapedLine{}, nil
	}
	return lines[0], nil
}

// WrapText shapes text and wraps it to maxWidth, returning one ShapedLine per
// visual line. Newlines in the text force line breaks.
func (s *System) WrapText(text string, runs []StyleRun, maxWidth geometry.Pixels) ([]ShapedLine, error) {
	if err := validateRuns(len(text), runs); err != nil {
		return nil, err
	}
	return s.wrap(text, runs, toFixed(float32(maxWidth)))
}

// wrap is the shared pipeline: check the line cache, and on miss segment and
// shape each style run, feed the shaped runs to the line wrapper, and build a
// ShapedLine per wrapped line.
//
// Both returns go through cloneShapedLines, not just the cache hit: the
// slice built on a miss is also what gets stored in the cache, so returning
// it unmodified would let this call's caller mutate the very entry the next
// call reads.
func (s *System) wrap(text string, runs []StyleRun, maxWidth fixed.Int26_6) ([]ShapedLine, error) {
	key := lineCacheKey{text: text, runsHash: s.lineCache.hashRuns(runs), maxWidth: maxWidth}
	if lines, ok := s.lineCache.lru.Get(key); ok {
		return cloneShapedLines(lines), nil
	}
	lines, err := s.wrapUncached(text, runs, maxWidth)
	if err != nil {
		return nil, err
	}
	s.lineCache.lru.Put(key, lines)
	return cloneShapedLines(lines), nil
}

// wrapUncached does the actual segmentation, shaping and line wrapping that
// wrap memoises.
func (s *System) wrapUncached(text string, runs []StyleRun, maxWidth fixed.Int26_6) ([]ShapedLine, error) {
	paragraph := []rune(text)
	if len(paragraph) == 0 {
		return []ShapedLine{{}}, nil
	}

	runeToByte := make([]int, len(paragraph)+1)
	b := 0
	for i, r := range paragraph {
		runeToByte[i] = b
		b += utf8RuneLen(r)
	}
	runeToByte[len(paragraph)] = b

	// Shape every style run into paragraph-relative Outputs in logical order.
	outs := make([]shaping.Output, 0, len(runs))
	byteOff := 0
	runeStart := 0
	for _, sr := range runs {
		runeCount := utf8.RuneCountInString(text[byteOff : byteOff+sr.ByteLen])
		runeEnd := runeStart + runeCount

		lang := language.NewLanguage(sr.Language)
		features := toTypesettingFeatures(sr.Features)
		subs := s.segmentRun(
			paragraph, runeStart, runeEnd, byteOff,
			sr.Font.withDefaults(),
			float32(sr.Size),
			sr.Direction.di(),
			lang,
			features,
		)
		for _, sub := range subs {
			outs = append(outs, paragraphOutput(sub))
		}

		byteOff += sr.ByteLen
		runeStart = runeEnd
	}

	config := shaping.WrapConfig{
		Direction: di.DirectionLTR,
	}
	if len(runs) > 0 && runs[0].Direction == RTL {
		config.Direction = di.DirectionRTL
	}

	iter := shaping.NewSliceIterator(outs)
	var wrapper shaping.LineWrapper
	wrapper.Prepare(config, paragraph, iter)

	var lines []ShapedLine
	lineStartRune := 0
	for {
		wl, done := wrapper.WrapNextLineF(maxWidth)
		if wl.Line != nil {
			lines = append(lines, buildLine(wl.Line, paragraph, runeToByte, lineStartRune))
		} else if done {
			// An empty paragraph or a trailing empty line: emit an empty line
			// only when there is content still to place.
			if lineStartRune < len(paragraph) {
				lines = append(lines, buildLine(nil, paragraph, runeToByte, lineStartRune))
			}
		}
		lineStartRune = wl.NextLine
		if done {
			break
		}
	}
	if len(lines) == 0 {
		lines = append(lines, buildLine(nil, paragraph, runeToByte, 0))
	}
	return lines, nil
}

// paragraphOutput returns a copy of the sub-run's isolated shaped Output,
// shifted into paragraph-rune coordinates so the line wrapper can map glyph
// clusters back to the paragraph. The copy owns its glyph slice, so the
// wrapper may set VisualIndex without touching the cached original.
func paragraphOutput(sub subRun) shaping.Output {
	out := sub.out
	out.Glyphs = append([]shaping.Glyph(nil), sub.out.Glyphs...)
	out.Runes.Offset = sub.baseRune
	out.Face = sub.face
	if sub.baseRune != 0 {
		for i := range out.Glyphs {
			out.Glyphs[i].ClusterIndex += sub.baseRune
		}
	}
	return out
}

// stringsContainsNewline reports whether s contains a line feed.
func stringsContainsNewline(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			return true
		}
	}
	return false
}
