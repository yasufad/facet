//go:build windows && facet_debug

package d3d11

import (
	"fmt"

	"github.com/yasufad/facet/scene"
)

// debugCheckTile panics if tile was handed out by a page that has since
// been cleared. See the doc comment on [d3d11Renderer.ClearAtlas] for the
// contract this enforces. Empty tiles (the zero value Upload returns for a
// zero-sized request) are never stale.
func (r *d3d11Renderer) debugCheckTile(tile scene.AtlasTile) {
	if tile.Bounds.Size.Width == 0 && tile.Bounds.Size.Height == 0 {
		return
	}
	if !r.atlas.tileValid(tile) {
		panic(fmt.Sprintf("d3d11: stale atlas tile %+v: its page was cleared by ClearAtlas after this tile was handed out", tile))
	}
}
