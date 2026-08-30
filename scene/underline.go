package scene

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
)

// Underline is a straight or wavy line drawn beneath text. Bounds gives the
// line's rectangle; Thickness is the line weight; Wavy selects a wavy path
// instead of a straight one.
//
// Colour carries straight alpha; the renderer premultiplies.
type Underline struct {
	Order       DrawOrder
	Bounds      geometry.Bounds[geometry.ScaledPixels]
	ContentMask ContentMask[geometry.ScaledPixels]
	Colour      colour.Rgba
	Thickness   geometry.ScaledPixels
	Wavy        bool
}
