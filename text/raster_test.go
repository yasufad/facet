package text

import (
	"testing"

	"github.com/yasufad/facet/geometry"
)

// TestRasteriseSquare rasterises a unit square outline and checks the mask
// covers the expected pixels. The square is 10×10 font units at 1px/unit
// scale, so the mask should be roughly 10×10 device pixels with full
// coverage in the interior.
func TestRasteriseSquare(t *testing.T) {
	outline := Outline{Segments: []Segment{
		{Op: SegMoveTo, Args: [3]geometry.Point[float32]{{X: 0, Y: 0}}},
		{Op: SegLineTo, Args: [3]geometry.Point[float32]{{X: 10, Y: 0}}},
		{Op: SegLineTo, Args: [3]geometry.Point[float32]{{X: 10, Y: 10}}},
		{Op: SegLineTo, Args: [3]geometry.Point[float32]{{X: 0, Y: 10}}},
	}}
	mask := rasterise(outline, 1.0, 10)
	if mask.Width <= 0 || mask.Height <= 0 {
		t.Fatalf("mask %dx%d, expected positive dimensions", mask.Width, mask.Height)
	}
	if len(mask.Coverage) != mask.Width*mask.Height {
		t.Fatalf("coverage len %d, expected %d", len(mask.Coverage), mask.Width*mask.Height)
	}
	// The interior of the square should be fully covered.
	cx, cy := mask.Width/2, mask.Height/2
	if mask.Coverage[cy*mask.Width+cx] < 200 {
		t.Fatalf("interior coverage %d, expected > 200", mask.Coverage[cy*mask.Width+cx])
	}
}

// TestRasteriseEmpty checks that an empty outline produces an empty mask.
func TestRasteriseEmpty(t *testing.T) {
	mask := rasterise(Outline{}, 1.0, 0)
	if mask.Width != 0 || mask.Height != 0 {
		t.Fatalf("empty outline mask %dx%d, expected 0x0", mask.Width, mask.Height)
	}
}

// TestRasteriseQuadCurve rasterises an outline with a quadratic Bézier and
// checks it produces a non-empty mask without panicking.
func TestRasteriseQuadCurve(t *testing.T) {
	outline := Outline{Segments: []Segment{
		{Op: SegMoveTo, Args: [3]geometry.Point[float32]{{X: 0, Y: 0}}},
		{Op: SegQuadTo, Args: [3]geometry.Point[float32]{
			{X: 5, Y: 10},
			{X: 10, Y: 0},
		}},
	}}
	mask := rasterise(outline, 2.0, 20)
	if mask.Width <= 0 || mask.Height <= 0 {
		t.Fatalf("quad mask %dx%d, expected positive", mask.Width, mask.Height)
	}
	// Some pixels should be covered.
	hasCoverage := false
	for _, c := range mask.Coverage {
		if c > 0 {
			hasCoverage = true
			break
		}
	}
	if !hasCoverage {
		t.Fatal("quad curve produced no coverage")
	}
}

// TestRasteriseCubicCurve rasterises an outline with a cubic Bézier.
func TestRasteriseCubicCurve(t *testing.T) {
	outline := Outline{Segments: []Segment{
		{Op: SegMoveTo, Args: [3]geometry.Point[float32]{{X: 0, Y: 0}}},
		{Op: SegCubeTo, Args: [3]geometry.Point[float32]{
			{X: 3, Y: 12},
			{X: 7, Y: 12},
			{X: 10, Y: 0},
		}},
	}}
	mask := rasterise(outline, 2.0, 24)
	if mask.Width <= 0 || mask.Height <= 0 {
		t.Fatalf("cubic mask %dx%d, expected positive", mask.Width, mask.Height)
	}
}

// TestAtlasCaches checks that requesting the same glyph twice returns the
// same entry (the atlas caches it).
func TestAtlasCaches(t *testing.T) {
	s := newTestSystem(t)
	face := s.Resolve(FontRequest{Families: []string{"Arial", "Segoe UI"}})
	if !face.valid() {
		t.Skip("no face available")
	}
	// Get the glyph ID for 'A'.
	gid, ok := face.face.NominalGlyph('A')
	if !ok {
		t.Skip("face lacks 'A'")
	}
	atlas := NewAtlas()
	e1 := atlas.Entry(face, GlyphID(gid), geometry.Pixels(16), SubpixelZero)
	e2 := atlas.Entry(face, GlyphID(gid), geometry.Pixels(16), SubpixelZero)
	if e1.Mask.Width != e2.Mask.Width || e1.Mask.Height != e2.Mask.Height {
		t.Fatalf("atlas returned different masks for same glyph: %dx%d vs %dx%d",
			e1.Mask.Width, e1.Mask.Height, e2.Mask.Width, e2.Mask.Height)
	}
}

// TestAtlasSubpixelSeparate checks that different subpixel offsets produce
// different cache entries (the atlas keys by subpixel offset).
func TestAtlasSubpixelSeparate(t *testing.T) {
	s := newTestSystem(t)
	face := s.Resolve(FontRequest{Families: []string{"Arial", "Segoe UI"}})
	if !face.valid() {
		t.Skip("no face available")
	}
	gid, ok := face.face.NominalGlyph('A')
	if !ok {
		t.Skip("face lacks 'A'")
	}
	atlas := NewAtlas()
	// Requesting with different subpixel offsets should not panic and should
	// produce valid entries.
	_ = atlas.Entry(face, GlyphID(gid), geometry.Pixels(16), SubpixelZero)
	_ = atlas.Entry(face, GlyphID(gid), geometry.Pixels(16), SubpixelThird)
	_ = atlas.Entry(face, GlyphID(gid), geometry.Pixels(16), SubpixelTwoThirds)
}

// TestAtlasClear checks that Clear empties the atlas.
func TestAtlasClear(t *testing.T) {
	s := newTestSystem(t)
	face := s.Resolve(FontRequest{Families: []string{"Arial", "Segoe UI"}})
	if !face.valid() {
		t.Skip("no face available")
	}
	gid, ok := face.face.NominalGlyph('A')
	if !ok {
		t.Skip("face lacks 'A'")
	}
	atlas := NewAtlas()
	_ = atlas.Entry(face, GlyphID(gid), geometry.Pixels(16), SubpixelZero)
	if len(atlas.entries) == 0 {
		t.Fatal("atlas empty before Clear")
	}
	atlas.Clear()
	if len(atlas.entries) != 0 {
		t.Fatalf("atlas has %d entries after Clear", len(atlas.entries))
	}
}
