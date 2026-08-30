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
// A ContentMask whose Bounds is empty means the primitive is not masked — the
// renderer treats an empty mask as "no clipping". The Scene never inserts a
// primitive whose bounds are fully clipped away, because the intersection of
// the primitive's bounds with the mask is empty and the primitive is skipped
// before it reaches a per-type slice.
type ContentMask[T geometry.Number] struct {
	Bounds geometry.Bounds[T]
}

// Intersect returns the overlap of m and other. If either mask is empty (no
// clipping) the other is returned unchanged; if both are non-empty the result
// is their intersection.
func (m ContentMask[T]) Intersect(other ContentMask[T]) ContentMask[T] {
	if m.Bounds.IsEmpty() {
		return other
	}
	if other.Bounds.IsEmpty() {
		return m
	}
	return ContentMask[T]{Bounds: m.Bounds.Intersect(other.Bounds)}
}
