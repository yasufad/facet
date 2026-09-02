//go:build windows && facet_debug

package d3d11

import (
	"testing"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/scene"
)

// TestDebugCheckTilePanicsAfterClear proves debugCheckTile actually
// detects the stale-tile case docs/audit.md item 3 asks for, rather than
// being a check that can never fire. Break the generation bump in
// atlasManager.clear (or the comparison in tileValid) and this test is the
// one that catches it — everything else in the package draws fresh tiles
// and would stay green.
func TestDebugCheckTilePanicsAfterClear(t *testing.T) {
	page := &atlasPage{kind: scene.TextureMonochrome, packer: newShelfPacker(atlasPageWidth, atlasPageHeight)}
	atlas := &atlasManager{monoPages: []*atlasPage{page}, nextTileID: 1}
	r := &d3d11Renderer{atlas: atlas}

	tile := scene.AtlasTile{
		TextureID: scene.AtlasTextureID{Index: 0, Kind: scene.TextureMonochrome},
		TileID:    makeTileID(page.generation, atlas.nextTileID),
		Bounds:    geometry.NewBounds(geometry.NewPoint[geometry.DevicePixels](0, 0), geometry.NewSize[geometry.DevicePixels](4, 4)),
	}

	func() {
		defer func() {
			if rec := recover(); rec != nil {
				t.Fatalf("debugCheckTile panicked on a fresh tile: %v", rec)
			}
		}()
		r.debugCheckTile(tile)
	}()

	atlas.clear(scene.TextureMonochrome)

	func() {
		defer func() {
			if recover() == nil {
				t.Fatalf("expected debugCheckTile to panic on a tile from a page ClearAtlas invalidated")
			}
		}()
		r.debugCheckTile(tile)
	}()
}
