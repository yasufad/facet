package text

import (
	"testing"

	"github.com/yasufad/facet/geometry"
)

// TestShapeNonLatin shapes CJK text and checks the line has non-zero width
// and the runs contain glyphs. This exercises the script segmentation and
// fallback paths for non-Latin scripts.
func TestShapeNonLatin(t *testing.T) {
	s := newTestSystem(t)
	text := "こんにちは世界"
	run := []StyleRun{{
		ByteLen:   len(text),
		Font:      FontRequest{Families: []string{"Noto Sans CJK JP", "Microsoft YaHei", "PingFang SC", "Arial"}},
		Size:      geometry.Pixels(16),
		Direction: LTR,
		Language:  "ja",
	}}
	line, err := s.ShapeLine(text, run)
	if err != nil {
		t.Fatalf("ShapeLine: %v", err)
	}
	if line.Width() <= 0 {
		t.Fatalf("CJK line width %v, expected positive", line.Width())
	}
	totalGlyphs := 0
	for _, r := range line.Runs() {
		totalGlyphs += len(r.Glyphs)
	}
	if totalGlyphs == 0 {
		t.Fatal("CJK line produced no glyphs")
	}
}

// TestShapeRTL shapes Arabic text right-to-left and checks the line has
// non-zero width and glyphs. This exercises the bidi segmentation and RTL
// shaping paths.
func TestShapeRTL(t *testing.T) {
	s := newTestSystem(t)
	text := "مرحبا بالعالم"
	run := []StyleRun{{
		ByteLen:   len(text),
		Font:      FontRequest{Families: []string{"Arial", "Noto Sans Arabic", "Segoe UI"}},
		Size:      geometry.Pixels(16),
		Direction: RTL,
		Language:  "ar",
	}}
	line, err := s.ShapeLine(text, run)
	if err != nil {
		t.Fatalf("ShapeLine: %v", err)
	}
	if line.Width() <= 0 {
		t.Fatalf("Arabic line width %v, expected positive", line.Width())
	}
	totalGlyphs := 0
	for _, r := range line.Runs() {
		totalGlyphs += len(r.Glyphs)
	}
	if totalGlyphs == 0 {
		t.Fatal("Arabic line produced no glyphs")
	}
}

// TestShapeMixedBidi shapes a line with mixed LTR and RTL text, checking it
// does not panic and produces glyphs. This exercises the bidi ordering within
// a single line.
func TestShapeMixedBidi(t *testing.T) {
	s := newTestSystem(t)
	text := "Hello مرحبا World"
	run := []StyleRun{{
		ByteLen:   len(text),
		Font:      FontRequest{Families: []string{"Arial", "Segoe UI"}},
		Size:      geometry.Pixels(16),
		Direction: LTR,
		Language:  "en",
	}}
	line, err := s.ShapeLine(text, run)
	if err != nil {
		t.Fatalf("ShapeLine: %v", err)
	}
	if line.Width() <= 0 {
		t.Fatalf("mixed bidi line width %v, expected positive", line.Width())
	}
}

// TestFallbackForMissingGlyph loads a font from bytes that only covers Latin
// (if one is available), then shapes a rune it does not cover and checks the
// system fallback supplies a glyph. Since we cannot guarantee a Latin-only
// font is available, this test uses a high-codepoint rune that most Latin
// fonts lack and checks shaping still produces output.
func TestFallbackForMissingGlyph(t *testing.T) {
	s := newTestSystem(t)
	// U+1F600 GRINNING FACE is unlikely to be in a basic Latin font but is
	// widely available in system emoji fonts. The fallback should find it.
	text := "😀"
	run := []StyleRun{{
		ByteLen:   len(text),
		Font:      FontRequest{Families: []string{"Arial", "Segoe UI"}},
		Size:      geometry.Pixels(16),
		Direction: LTR,
		Language:  "en",
	}}
	line, err := s.ShapeLine(text, run)
	if err != nil {
		t.Fatalf("ShapeLine: %v", err)
	}
	totalGlyphs := 0
	for _, r := range line.Runs() {
		totalGlyphs += len(r.Glyphs)
	}
	if totalGlyphs == 0 {
		t.Fatal("emoji rune produced no glyphs; fallback did not engage")
	}
}

// TestShapeCacheHits shapes the same text twice and checks the cache serves
// the second call. We cannot inspect the cache directly, but we can check
// that repeated shaping produces identical output.
func TestShapeCacheHits(t *testing.T) {
	s := newTestSystem(t)
	text := "cached"
	run := defaultRun(text)
	line1, err := s.ShapeLine(text, run)
	if err != nil {
		t.Fatalf("first ShapeLine: %v", err)
	}
	line2, err := s.ShapeLine(text, run)
	if err != nil {
		t.Fatalf("second ShapeLine: %v", err)
	}
	if line1.Width() != line2.Width() {
		t.Fatalf("cache miss: widths differ %v vs %v", line1.Width(), line2.Width())
	}
	if len(line1.Runs()) != len(line2.Runs()) {
		t.Fatalf("cache miss: run counts differ %d vs %d", len(line1.Runs()), len(line2.Runs()))
	}
}

// BenchmarkShapeLineCold measures ShapeLine on a fresh System for each
// iteration, so the line cache and shape cache never hit. It is the cost the
// pipeline actually has before either cache helps.
func BenchmarkShapeLineCold(b *testing.B) {
	s, err := NewSystem()
	if err != nil {
		b.Skipf("system fonts unavailable: %v", err)
	}
	text := "The quick brown fox jumps over the lazy dog"
	run := defaultRun(text)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.lineCache.lru.Clear()
		s.shapeCache.lru.Clear()
		if _, err := s.ShapeLine(text, run); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkShapeLineLineCacheHit measures a repeated ShapeLine call for the
// same text and runs, which after the first call is served entirely from
// lineCache: no segmentation, no line wrapping, no shapeCache lookup.
func BenchmarkShapeLineLineCacheHit(b *testing.B) {
	s, err := NewSystem()
	if err != nil {
		b.Skipf("system fonts unavailable: %v", err)
	}
	text := "The quick brown fox jumps over the lazy dog"
	run := defaultRun(text)
	if _, err := s.ShapeLine(text, run); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := s.ShapeLine(text, run); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkShapeLineShapeCacheHit measures a repeated ShapeLine call with the
// line cache disabled (evicted every iteration) but the shape cache warm, so
// segmentation and line wrapping still run but the HarfBuzz call is skipped.
// The gap between this and BenchmarkShapeLineLineCacheHit is what caching the
// finished ShapedLine saves beyond caching shaped output alone.
func BenchmarkShapeLineShapeCacheHit(b *testing.B) {
	s, err := NewSystem()
	if err != nil {
		b.Skipf("system fonts unavailable: %v", err)
	}
	text := "The quick brown fox jumps over the lazy dog"
	run := defaultRun(text)
	if _, err := s.ShapeLine(text, run); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		s.lineCache.lru.Clear()
		if _, err := s.ShapeLine(text, run); err != nil {
			b.Fatal(err)
		}
	}
}
