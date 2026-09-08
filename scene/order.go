package scene

import "github.com/yasufad/facet/geometry"

// DrawOrder is the depth of a primitive within the scene. Lower values are
// drawn first; higher values are drawn on top. The Scene assigns draw orders
// from a spatial tree so that overlapping primitives receive strictly
// increasing orders, while primitives that share no screen space may reuse an
// order and be batched together.
type DrawOrder uint32

// ContentMask is the rectangle a primitive is clipped to. The Scene maintains a
// clip stack: PushClip intersects the new mask with the one already on top, and
// every inserted primitive records the stack's current mask.
//
// The meaning of an empty mask depends on whether the Scene has a viewport
// (see Scene.SetViewport). With a viewport, the stack is never empty and an
// empty mask means nothing is visible — a container that resolves to zero
// height clips its children. Without a viewport, an empty mask means "no
// clipping", the encoding the render package's readback tests rely on for
// synthetic scenes built by hand. The Scene never inserts a primitive whose
// bounds are fully clipped away: the intersection with the mask is empty and
// the primitive is skipped before it reaches a per-type slice.
type ContentMask[T geometry.Number] struct {
	Bounds geometry.Bounds[T]
}

// Intersect returns the overlap of m and other. The intersection of an empty
// mask with any other is empty: with a viewport in force, an empty mask means
// nothing is visible. Without a viewport, PushClip does not intersect against
// the absent base, so an empty mask pushed first is preserved unchanged.
func (m ContentMask[T]) Intersect(other ContentMask[T]) ContentMask[T] {
	return ContentMask[T]{Bounds: m.Bounds.Intersect(other.Bounds)}
}
