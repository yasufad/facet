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

// TestHashRunsDistinguishesFields checks that hashRuns does not collapse
// distinct runs onto the same digest for the fields it is supposed to be
// sensitive to. This is not a guarantee against collision — a 64-bit hash
// can always collide — but it pins that a naive implementation (one that,
// say, forgot to feed Size into the hash) does not pass by accident.
func TestHashRunsDistinguishesFields(t *testing.T) {
	base := StyleRun{
		ByteLen:   5,
		Font:      FontRequest{Family: "Arial"},
		Size:      geometry.Pixels(16),
		Direction: LTR,
		Language:  "en",
	}
	variants := []StyleRun{
		base,
		{ByteLen: 6, Font: base.Font, Size: base.Size, Direction: base.Direction, Language: base.Language},
		{ByteLen: base.ByteLen, Font: FontRequest{Family: "Georgia"}, Size: base.Size, Direction: base.Direction, Language: base.Language},
		{ByteLen: base.ByteLen, Font: base.Font, Size: geometry.Pixels(20), Direction: base.Direction, Language: base.Language},
		{ByteLen: base.ByteLen, Font: base.Font, Size: base.Size, Direction: RTL, Language: base.Language},
		{ByteLen: base.ByteLen, Font: base.Font, Size: base.Size, Direction: base.Direction, Language: "fr"},
	}

	c := newLineCache()
	seen := map[uint64]int{}
	for i, v := range variants {
		h := c.hashRuns([]StyleRun{v})
		if prior, ok := seen[h]; ok {
			t.Fatalf("variant %d hashed the same as variant %d: both produced %d", i, prior, h)
		}
		seen[h] = i
	}
}
