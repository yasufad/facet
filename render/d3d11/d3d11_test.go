//go:build windows

package d3d11

import (
	"testing"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/render"
	"github.com/yasufad/facet/scene"
)

func TestShelfPacker(t *testing.T) {
	packer := newShelfPacker(100, 100)

	// First tile fits at (0, 0)
	x, y, ok := packer.allocate(30, 20)
	if !ok || x != 0 || y != 0 {
		t.Fatalf("expected (0, 0), got (%d, %d), ok=%v", x, y, ok)
	}

	// Second tile fits on same shelf at (30, 0)
	x, y, ok = packer.allocate(40, 25)
	if !ok || x != 30 || y != 0 {
		t.Fatalf("expected (30, 0), got (%d, %d), ok=%v", x, y, ok)
	}

	// Third tile doesn't fit on first shelf (30 + 40 + 50 > 100), starts new shelf at y=25
	x, y, ok = packer.allocate(50, 30)
	if !ok || x != 0 || y != 25 {
		t.Fatalf("expected (0, 25), got (%d, %d), ok=%v", x, y, ok)
	}

	// Oversized tile fails
	_, _, ok = packer.allocate(150, 10)
	if ok {
		t.Fatalf("expected oversized tile to fail")
	}

	// Zero or negative size fails
	_, _, ok = packer.allocate(0, 10)
	if ok {
		t.Fatalf("expected zero width to fail")
	}
	_, _, ok = packer.allocate(10, 0)
	if ok {
		t.Fatalf("expected zero height to fail")
	}

	// Reset resets cursor
	packer.reset()
	x, y, ok = packer.allocate(10, 10)
	if !ok || x != 0 || y != 0 {
		t.Fatalf("expected (0, 0) after reset, got (%d, %d)", x, y)
	}
}

// TestShelfPackerSessionOccupancy simulates a realistic glyph session — a
// mix of tile sizes in the range a proportional font produces at UI sizes —
// packed until several pages fill, so the page-growth decision in
// docs/audit.md item 4 has a number behind it instead of a guess. It drives
// the packer directly rather than atlasManager.upload, which needs a real
// D3D11 device and belongs with the facet_debug GPU tests.
func TestShelfPackerSessionOccupancy(t *testing.T) {
	sizes := []struct{ w, h int }{
		{8, 12}, {10, 14}, {12, 16}, {9, 13}, {14, 18},
		{16, 20}, {11, 15}, {20, 24}, {13, 17}, {24, 28},
	}
	const tileCount = 20000

	pages := []*shelfPacker{newShelfPacker(atlasPageWidth, atlasPageHeight)}
	current := pages[len(pages)-1]
	var handoffOccupancy []float64

	for i := 0; i < tileCount; i++ {
		sz := sizes[i%len(sizes)]
		if _, _, ok := current.allocate(sz.w, sz.h); ok {
			continue
		}

		handoffOccupancy = append(handoffOccupancy, current.occupancy())
		current = newShelfPacker(atlasPageWidth, atlasPageHeight)
		pages = append(pages, current)
		if _, _, ok := current.allocate(sz.w, sz.h); !ok {
			t.Fatalf("tile %dx%d does not fit an empty %dx%d page", sz.w, sz.h, atlasPageWidth, atlasPageHeight)
		}
	}

	var sum float64
	for _, o := range handoffOccupancy {
		sum += o
	}
	avg := 0.0
	if len(handoffOccupancy) > 0 {
		avg = sum / float64(len(handoffOccupancy))
	}

	t.Logf("session of %d tiles packed into %d pages; average occupancy of a page when the next one was started: %.1f%%", tileCount, len(pages), avg*100)
	t.Logf("final page occupancy: %.1f%%", current.occupancy()*100)
}

// TestAtlasTileGeneration exercises the encoding docs/audit.md item 3 asks
// for without a GPU device: a tile is valid against the page it names until
// that page's generation advances (what ClearAtlas does), then invalid.
func TestAtlasTileGeneration(t *testing.T) {
	page := &atlasPage{kind: scene.TextureMonochrome, packer: newShelfPacker(atlasPageWidth, atlasPageHeight)}
	m := &atlasManager{monoPages: []*atlasPage{page}, nextTileID: 1}

	tile := scene.AtlasTile{
		TextureID: scene.AtlasTextureID{Index: 0, Kind: scene.TextureMonochrome},
		TileID:    makeTileID(page.generation, m.nextTileID),
		Bounds:    geometry.NewBounds(geometry.NewPoint[geometry.DevicePixels](0, 0), geometry.NewSize[geometry.DevicePixels](4, 4)),
	}

	if !m.tileValid(tile) {
		t.Fatalf("tile should be valid against the page it was just minted from")
	}

	page.generation++ // what atlasManager.clear does to every page of a kind

	if m.tileValid(tile) {
		t.Fatalf("tile handed out before the page's generation advanced should be invalid after")
	}

	// A tile naming a page index that no longer (or never did) exist is
	// invalid rather than a panic or an out-of-range read.
	stray := tile
	stray.TextureID.Index = 7
	if m.tileValid(stray) {
		t.Fatalf("tile naming a nonexistent page should be invalid")
	}
}

func TestNewInvalidSurface(t *testing.T) {
	_, err := New(0, geometry.NewSize[geometry.DevicePixels](100, 100), render.Options{})
	if err == nil {
		t.Fatalf("expected error for surface handle 0, got nil")
	}
}
