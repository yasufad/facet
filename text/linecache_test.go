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

// TestLineCacheHitRejectsForcedDigestCollision puts two entries into the line
// cache with identical text, maxWidth and runsHash but different StyleRun
// styling, asserting that a cache lookup confirms the runs field-by-field and
// does not return the first entry's lines on a collision. It must fail if the
// confirmation in lineCache.get is removed.
func TestLineCacheHitRejectsForcedDigestCollision(t *testing.T) {
	c := newLineCache()
	key := lineCacheKey{text: "hello", runsHash: 0xdeadbeef, maxWidth: noWrapWidth}

	run1 := []StyleRun{{
		ByteLen: 5,
		Font:    FontRequest{Family: "Arial"},
		Size:    16,
	}}
	lines1 := []ShapedLine{{runs: []ShapedRun{{Glyphs: []Glyph{{ID: 1}}}}}}

	run2 := []StyleRun{{
		ByteLen: 5,
		Font:    FontRequest{Family: "Arial"},
		Size:    24,
	}}
	lines2 := []ShapedLine{{runs: []ShapedRun{{Glyphs: []Glyph{{ID: 2}}}}}}

	c.put(key, run1, lines1)

	// Digest matches key, but runs differ: confirmation must reject the hit.
	if got, ok := c.get(key, run2); ok {
		t.Fatalf("forced digest collision returned first entry's lines: %v", got)
	}

	// Putting the second entry replaces the first under the colliding key.
	c.put(key, run2, lines2)
	got, ok := c.get(key, run2)
	if !ok {
		t.Fatal("expected hit for second entry after put")
	}
	if len(got) == 0 || len(got[0].runs) == 0 || len(got[0].runs[0].Glyphs) == 0 || got[0].runs[0].Glyphs[0].ID != 2 {
		t.Fatalf("expected second entry's lines, got %v", got)
	}
	if _, ok := c.get(key, run1); ok {
		t.Fatal("first entry's runs still matched after replacement")
	}
}

// TestSystemShapeLineRejectsHashCollision tests the confirmation through the
// high-level System API: planting an entry under the colliding digest of a
// second run set must not cause ShapeLine to return the planted lines. It must
// fail if the confirmation in lineCache.get is removed.
func TestSystemShapeLineRejectsHashCollision(t *testing.T) {
	s := newTestSystem(t)
	text := "collision"
	run1 := defaultRun(text)
	lines1, err := s.ShapeLine(text, run1)
	if err != nil {
		t.Fatalf("first ShapeLine: %v", err)
	}

	run2 := []StyleRun{{
		ByteLen: len(text),
		Font:    FontRequest{Family: "Arial"},
		Size:    geometry.Pixels(32),
	}}
	// Force a collision: plant run1's lines under run2's digest with run1's runs.
	collidingKey := lineCacheKey{
		text:     text,
		runsHash: s.lineCache.hashRuns(run2),
		maxWidth: noWrapWidth,
	}
	s.lineCache.put(collidingKey, run1, []ShapedLine{lines1})

	// ShapeLine for run2 must reject the planted lines and shape run2 cleanly.
	lines2, err := s.ShapeLine(text, run2)
	if err != nil {
		t.Fatalf("second ShapeLine: %v", err)
	}
	if len(lines2.Runs()) == 0 || len(lines1.Runs()) == 0 {
		t.Fatal("expected runs in shaped lines")
	}
	if lines2.Runs()[0].Glyphs[0].ID == lines1.Runs()[0].Glyphs[0].ID &&
		lines2.Width() == lines1.Width() {
		t.Fatal("ShapeLine returned first's lines on forced digest collision")
	}
}

// TestLineCacheHitDoesNotAliasCallerRunsMutation checks that a caller
// mutating slice fields (Families and Features) on a StyleRun passed to
// ShapeLine does not corrupt the cached entry's confirmation data.
func TestLineCacheHitDoesNotAliasCallerRunsMutation(t *testing.T) {
	s := newTestSystem(t)
	text := "mutation"
	families := []string{"Arial"}
	features := []FontFeature{{Tag: "kern", Value: 1}}
	run := []StyleRun{{
		ByteLen: len(text),
		Font: FontRequest{
			Family:   "Arial",
			Families: families,
		},
		Size:     16,
		Features: features,
	}}

	first, err := s.ShapeLine(text, run)
	if err != nil {
		t.Fatalf("first ShapeLine: %v", err)
	}

	// Mutate caller's slice backing arrays.
	families[0] = "Comic Sans"
	features[0].Value = 0

	// Cache entry should have its own cloned copy of Families and Features,
	// so a query with the original run definition still confirms and hits.
	origRun := []StyleRun{{
		ByteLen: len(text),
		Font: FontRequest{
			Family:   "Arial",
			Families: []string{"Arial"},
		},
		Size:     16,
		Features: []FontFeature{{Tag: "kern", Value: 1}},
	}}
	second, err := s.ShapeLine(text, origRun)
	if err != nil {
		t.Fatalf("second ShapeLine: %v", err)
	}
	if second.Width() != first.Width() {
		t.Fatalf("caller mutation of run slices corrupted cached entry: got width %v, want %v", second.Width(), first.Width())
	}
}
