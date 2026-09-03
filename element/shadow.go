package element

import (
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
)

// boxShadowPrimitive converts one style.BoxShadow into the scene.Shadow it
// paints as, following scene.Shadow's documented contract: a drop shadow's
// bounds are the element bounds translated by the offset and dilated by the
// blur radius and spread; an inset shadow's bounds are the element bounds
// inset likewise. elementBounds and elementCornerRadii describe the element
// casting the shadow, in logical pixels, before this shadow's own offset.
func boxShadowPrimitive(elementBounds geometry.Bounds[geometry.Pixels], elementCornerRadii geometry.Corners[geometry.Pixels], sh style.BoxShadow, scale float32) scene.Shadow {
	shapeBounds := elementBounds
	shapeBounds.Origin = shapeBounds.Origin.Add(sh.Offset)
	shapeRadii := elementCornerRadii
	if sh.Inset {
		shapeBounds = shapeBounds.Dilate(-(sh.Blur + sh.Spread))
		// A spread wider than a corner's own radius would drive it negative,
		// which breaks the rounded-box SDF the shader evaluates against it.
		shapeRadii = shrinkCorners(shapeRadii, sh.Spread)
	} else {
		shapeBounds = shapeBounds.Dilate(sh.Blur + sh.Spread)
	}

	return scene.Shadow{
		BlurRadius:         sh.Blur.Scale(scale),
		Bounds:             scaleBounds(shapeBounds, scale),
		CornerRadii:        scaleCorners(shapeRadii, scale),
		Colour:             sh.Colour,
		ElementBounds:      scaleBounds(elementBounds, scale),
		ElementCornerRadii: scaleCorners(elementCornerRadii, scale),
		Inset:              sh.Inset,
	}
}

// shrinkCorners reduces each corner radius by amount, clamped at zero.
func shrinkCorners(c geometry.Corners[geometry.Pixels], amount geometry.Pixels) geometry.Corners[geometry.Pixels] {
	shrink := func(r geometry.Pixels) geometry.Pixels {
		r -= amount
		if r < 0 {
			return 0
		}
		return r
	}
	return geometry.Corners[geometry.Pixels]{
		TopLeft:     shrink(c.TopLeft),
		TopRight:    shrink(c.TopRight),
		BottomRight: shrink(c.BottomRight),
		BottomLeft:  shrink(c.BottomLeft),
	}
}
