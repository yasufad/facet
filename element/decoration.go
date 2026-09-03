package element

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/scene"
)

// decorationColour returns c, or fallback if c is the zero value —
// style.UnderlineStyle and style.StrikethroughStyle both document that an
// unset decoration colour defaults to the text colour it decorates.
func decorationColour(c, fallback colour.Rgba) colour.Rgba {
	if c == (colour.Rgba{}) {
		return fallback
	}
	return c
}

// decorationLine builds the scene.Underline a text decoration paints as, from
// its logical-pixel line geometry. Underline and strikethrough are the same
// primitive at a different vertical position; wavy is only ever true for an
// underline.
func decorationLine(x, y, width, thickness geometry.Pixels, c colour.Rgba, wavy bool, scale float32) scene.Underline {
	bounds := geometry.NewBounds(
		geometry.NewPoint(x, y),
		geometry.NewSize(width, thickness),
	)
	return scene.Underline{
		Bounds:    scaleBounds(bounds, scale),
		Colour:    c,
		Thickness: thickness.Scale(scale),
		Wavy:      wavy,
	}
}
