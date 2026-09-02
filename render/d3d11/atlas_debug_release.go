//go:build windows && !facet_debug

package d3d11

import "github.com/yasufad/facet/scene"

// debugCheckTile is a no-op in release builds; see the facet_debug variant
// in atlas_debug_facet_debug.go.
func (r *d3d11Renderer) debugCheckTile(scene.AtlasTile) {}
