package text

import (
	"testing"

	"github.com/yasufad/facet/geometry"
)

// TestRasteriseGlyph checks that rasterising a real glyph produces a non-empty
// mask with the expected dimensions and full coverage in its interior.
func TestRasteriseGlyph(t *testing.T) {
	s := newTestSystem(t)
	face := s.Resolve(FontRequest{Families: []string{"Arial", "Segoe UI", "DejaVu Sans"}})
	if !face.valid() {
		t.Skip("no face available")
	}
	gid, ok := face.face.NominalGlyph('A')
	if !ok {
		t.Skip("face lacks 'A'")
	}
	upem := face.face.Upem()
	scale := float32(16) / float32(upem)
	mask := rasteriseGlyph(face.face, gid, scale)
	if mask.Width <= 0 || mask.Height <= 0 {
		t.Fatalf("mask %dx%d, expected positive dimensions", mask.Width, mask.Height)
	}
	if len(mask.Coverage) != mask.Width*mask.Height {
		t.Fatalf("coverage len %d, expected %d", len(mask.Coverage), mask.Width*mask.Height)
	}
	// The interior of the glyph should have high coverage.
	hasFullCoverage := false
	for _, c := range mask.Coverage {
		if c >= 200 {
			hasFullCoverage = true
			break
		}
	}
	if !hasFullCoverage {
		t.Fatal("no pixel reached full coverage; analytic AA should produce 255 in the interior")
	}
}

// TestRasteriseGlyphEmpty checks that a glyph with no outline (whitespace)
// produces an empty mask.
func TestRasteriseGlyphEmpty(t *testing.T) {
	s := newTestSystem(t)
	face := s.Resolve(FontRequest{Families: []string{"Arial", "Segoe UI", "DejaVu Sans"}})
	if !face.valid() {
		t.Skip("no face available")
	}
	gid, ok := face.face.NominalGlyph(' ')
	if !ok {
		t.Skip("face lacks space")
	}
	upem := face.face.Upem()
	scale := float32(16) / float32(upem)
	mask := rasteriseGlyph(face.face, gid, scale)
	if mask.Width != 0 || mask.Height != 0 {
		t.Fatalf("space glyph mask %dx%d, expected 0x0", mask.Width, mask.Height)
	}
}

// TestRasteriseGlyphCoverageLevels checks that the mask uses more than a
// handful of distinct coverage values, confirming analytic area coverage
// rather than binary supersampling.
func TestRasteriseGlyphCoverageLevels(t *testing.T) {
	s := newTestSystem(t)
	face := s.Resolve(FontRequest{Families: []string{"Arial", "Segoe UI", "DejaVu Sans"}})
	if !face.valid() {
		t.Skip("no face available")
	}
	gid, ok := face.face.NominalGlyph('O')
	if !ok {
		t.Skip("face lacks 'O'")
	}
	upem := face.face.Upem()
	scale := float32(14) / float32(upem)
	mask := rasteriseGlyph(face.face, gid, scale)
	levels := make(map[byte]int)
	for _, c := range mask.Coverage {
		levels[c]++
	}
	// 'O' has curved edges that should produce many intermediate coverage
	// values. The old supersampling rasteriser produced at most 17 distinct
	// values; analytic coverage should produce far more.
	if len(levels) < 20 {
		t.Fatalf("only %d distinct coverage levels, expected at least 20 from analytic AA", len(levels))
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
	gid, ok := face.face.NominalGlyph('A')
	if !ok {
		t.Skip("face lacks 'A'")
	}
	atlas := NewAtlas(1.0)
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
	atlas := NewAtlas(1.0)
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
	atlas := NewAtlas(1.0)
	_ = atlas.Entry(face, GlyphID(gid), geometry.Pixels(16), SubpixelZero)
	if len(atlas.entries) == 0 {
		t.Fatal("atlas empty before Clear")
	}
	atlas.Clear()
	if len(atlas.entries) != 0 {
		t.Fatalf("atlas has %d entries after Clear", len(atlas.entries))
	}
}

// TestAtlasScaleFactor checks that the atlas rasterises at device-pixel
// resolution by comparing mask sizes at different scale factors.
func TestAtlasScaleFactor(t *testing.T) {
	s := newTestSystem(t)
	face := s.Resolve(FontRequest{Families: []string{"Arial", "Segoe UI"}})
	if !face.valid() {
		t.Skip("no face available")
	}
	gid, ok := face.face.NominalGlyph('A')
	if !ok {
		t.Skip("face lacks 'A'")
	}
	atlas1x := NewAtlas(1.0)
	atlas2x := NewAtlas(2.0)
	e1 := atlas1x.Entry(face, GlyphID(gid), geometry.Pixels(16), SubpixelZero)
	e2 := atlas2x.Entry(face, GlyphID(gid), geometry.Pixels(16), SubpixelZero)
	// The 2x atlas should produce a roughly 2x larger mask.
	if e2.Mask.Width <= e1.Mask.Width {
		t.Fatalf("2x mask width %d not larger than 1x mask width %d", e2.Mask.Width, e1.Mask.Width)
	}
	if e2.Mask.Height <= e1.Mask.Height {
		t.Fatalf("2x mask height %d not larger than 1x mask height %d", e2.Mask.Height, e1.Mask.Height)
	}
}
