package style

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
)

// BoxShadow defines a drop shadow or inset shadow cast by an element.
type BoxShadow struct {
	// Offset is the shadow offset from the element origin.
	Offset geometry.Point[geometry.Pixels]

	// Blur is the Gaussian blur radius.
	Blur geometry.Pixels

	// Spread is the positive or negative enlargement of the shadow shape.
	Spread geometry.Pixels

	// Colour is the shadow fill colour.
	Colour colour.Rgba

	// Inset indicates whether the shadow is drawn inside the border box.
	Inset bool
}

// Shadow constructs a BoxShadow with the given offset, blur, spread and colour.
func Shadow(offsetX, offsetY, blur, spread geometry.Pixels, c colour.Rgba) BoxShadow {
	return BoxShadow{
		Offset: geometry.NewPoint(offsetX, offsetY),
		Blur:   blur,
		Spread: spread,
		Colour: c,
		Inset:  false,
	}
}
