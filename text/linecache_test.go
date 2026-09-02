package text

import (
	"testing"

	"github.com/yasufad/facet/geometry"
)

// TestLineCacheHitDoesNotAliasCallerMutation reproduces the aliasing hazard
// through the exported API only: ShapedLine.Runs() exposes a mutable Glyphs
// slice, so if wrap ever hands back the cache's own backing array, one
// caller mutating a glyph corrupts every other caller's result for the same
// text — and the cache's own future hits — even though nothing in this
// package's own code ever mutates a glyph. It must fail if cloneShapedLines
// is removed from wrap's return path.
func TestLineCacheHitDoesNotAliasCallerMutation(t *testing.T) {
	s := newTestSystem(t)
	text := "cached"
	run := defaultRun(text)

	first, err := s.ShapeLine(text, run)
	if err != nil {
		t.Fatalf("first ShapeLine: %v", err)
	}
	if len(first.Runs()) == 0 || len(first.Runs()[0].Glyphs) == 0 {
		t.Fatal("expected at least one glyph to mutate")
	}
	want := first.Runs()[0].Glyphs[0].Position
	first.Runs()[0].Glyphs[0].Position = geometry.NewPoint(geometry.Pixels(999), geometry.Pixels(999))

	second, err := s.ShapeLine(text, run)
	if err != nil {
		t.Fatalf("second ShapeLine: %v", err)
	}
	got := second.Runs()[0].Glyphs[0].Position
	if got != want {
		t.Fatalf("cache hit aliased a caller's mutation: got %v, want %v", got, want)
	}
}
