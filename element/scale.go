package element

import "github.com/yasufad/facet/geometry"

// scaleBounds converts bounds from logical to scaled pixels. Every scene
// primitive element inserts is in scaled pixels; this is the conversion
// every one of them needs at its own bounds.
func scaleBounds(b geometry.Bounds[geometry.Pixels], scale float32) geometry.Bounds[geometry.ScaledPixels] {
	return geometry.Bounds[geometry.ScaledPixels]{
		Origin: geometry.Point[geometry.ScaledPixels]{
			X: b.Origin.X.Scale(scale),
			Y: b.Origin.Y.Scale(scale),
		},
		Size: geometry.Size[geometry.ScaledPixels]{
			Width:  b.Size.Width.Scale(scale),
			Height: b.Size.Height.Scale(scale),
		},
	}
}

// scaleCorners converts corner radii from logical to scaled pixels.
func scaleCorners(c geometry.Corners[geometry.Pixels], scale float32) geometry.Corners[geometry.ScaledPixels] {
	return geometry.Corners[geometry.ScaledPixels]{
		TopLeft:     c.TopLeft.Scale(scale),
		TopRight:    c.TopRight.Scale(scale),
		BottomRight: c.BottomRight.Scale(scale),
		BottomLeft:  c.BottomLeft.Scale(scale),
	}
}
