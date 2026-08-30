package scene

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
)

// Shadow is a blurred, rounded rectangle drawn behind or inside an element.
// The caller bakes offset and spread into Bounds: a drop shadow's bounds are
// the element bounds translated by the offset and dilated by the blur radius
// and spread; an inset shadow's bounds are the element bounds inset likewise.
//
// ElementBounds and ElementCornerRadii describe the element casting the
// shadow, so the renderer can mask the blur to the element's shape. Inset
// distinguishes a drop shadow (false, drawn outside the element) from an inset
// shadow (true, drawn inside).
//
// Colour carries straight alpha; the renderer premultiplies.
type Shadow struct {
	Order              DrawOrder
	BlurRadius         geometry.ScaledPixels
	Bounds             geometry.Bounds[geometry.ScaledPixels]
	CornerRadii        geometry.Corners[geometry.ScaledPixels]
	ContentMask        ContentMask[geometry.ScaledPixels]
	Colour             colour.Rgba
	ElementBounds      geometry.Bounds[geometry.ScaledPixels]
	ElementCornerRadii geometry.Corners[geometry.ScaledPixels]
	Inset              bool
}
