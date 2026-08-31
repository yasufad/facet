package scene

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
)

// MonochromeSprite draws a single-channel atlas tile tinted with a colour: a
// glyph. The tile's coverage modulates Colour, so a white colour reproduces the
// glyph as rasterised and any other colour tints it.
//
// Transformation is applied to the sprite before sampling the tile; glyphs
// usually carry the identity matrix. Colour carries straight alpha; the
// renderer premultiplies.
type MonochromeSprite struct {
	Order          DrawOrder
	Bounds         geometry.Bounds[geometry.ScaledPixels]
	ContentMask    ContentMask[geometry.ScaledPixels]
	Colour         colour.Rgba
	Tile           AtlasTile
	Transformation TransformationMatrix
}

// PolychromeSprite draws a full-colour atlas tile: an image or emoji. Opacity
// scales the tile's alpha; Greyscale desaturates it; CornerRadii rounds its
// corners.
//
// The tile carries its own colour, so there is no Colour field. Opacity is
// separate from straight-alpha premultiplication: it multiplies the tile's
// alpha at sample time.
type PolychromeSprite struct {
	Order       DrawOrder
	Greyscale   bool
	Opacity     float32
	Bounds      geometry.Bounds[geometry.ScaledPixels]
	ContentMask ContentMask[geometry.ScaledPixels]
	CornerRadii geometry.Corners[geometry.ScaledPixels]
	Tile        AtlasTile
}
