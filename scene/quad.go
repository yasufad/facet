package scene

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
)

// Quad is a filled, optionally rounded rectangle with a per-edge border. It is
// the most common primitive: backgrounds, surfaces and borders all paint as
// quads.
//
// Bounds, CornerRadii and BorderWidths are in scaled pixels. Background and
// BorderColour carry straight alpha; the renderer premultiplies when uploading
// instance data.
type Quad struct {
	Order        DrawOrder
	Bounds       geometry.Bounds[geometry.ScaledPixels]
	ContentMask  ContentMask[geometry.ScaledPixels]
	Background   colour.Rgba
	BorderColour colour.Rgba
	CornerRadii  geometry.Corners[geometry.ScaledPixels]
	BorderWidths geometry.Edges[geometry.ScaledPixels]
	BorderStyle  BorderStyle
}
