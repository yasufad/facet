package scene

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
)

// PathID identifies a path within its Scene. The Scene assigns the ID at
// insertion time from the current length of its paths slice; the ID is a
// stable identifier, not an index into the sorted slice produced by Finish.
type PathID int

// PathVertex is one vertex of a Path. XYPosition is the vertex location in the
// path's coordinate unit. STPosition is the texture coordinate the renderer
// uses to interpolate coverage across the triangle. ContentMask clips this
// vertex.
type PathVertex[T geometry.Number] struct {
	XYPosition  geometry.Point[T]
	STPosition  geometry.Point[float32]
	ContentMask ContentMask[T]
}

// Path is a filled shape built from triangles, for what the other primitives
// cannot express: arbitrary beziers, icons, custom shapes. The builder methods
// — MoveTo, LineTo, CurveTo, PushTriangle — accumulate vertices; the caller
// then passes the path to Scene.InsertPath.
//
// Path is generic over the coordinate unit. The Scene stores Path[ScaledPixels];
// the caller typically builds in pixels and converts with ScalePath.
//
// Colour carries straight alpha; the renderer premultiplies.
type Path[T geometry.Number] struct {
	ID          PathID
	Order       DrawOrder
	Bounds      geometry.Bounds[T]
	ContentMask ContentMask[T]
	Vertices    []PathVertex[T]
	Colour      colour.Rgba

	// Builder state. Start is the first point of the current contour; Current
	// is the pen position; ContourCount tracks how many points have been added
	// to the current contour, so the first line or curve after a MoveTo is
	// skipped (a single point draws nothing).
	start        geometry.Point[T]
	current      geometry.Point[T]
	contourCount int
}

// NewPath creates a path whose pen starts at start. The bounds are initialised
// to the start point with zero size.
func NewPath(start geometry.Point[geometry.Pixels]) Path[geometry.Pixels] {
	return Path[geometry.Pixels]{
		Bounds: geometry.Bounds[geometry.Pixels]{
			Origin: start,
		},
		start:   start,
		current: start,
	}
}

// MoveTo lifts the pen and starts a new contour at to.
func (p *Path[T]) MoveTo(to geometry.Point[T]) {
	p.contourCount = 1
	p.start = to
	p.current = to
}

// LineTo draws a straight line from the current point to to. The first LineTo
// after a MoveTo only moves the pen; a line needs two points.
func (p *Path[T]) LineTo(to geometry.Point[T]) {
	p.contourCount++
	if p.contourCount > 1 {
		p.PushTriangle(
			p.start, p.current, to,
			geometry.Point[float32]{X: 0, Y: 1},
			geometry.Point[float32]{X: 0, Y: 1},
			geometry.Point[float32]{X: 0, Y: 1},
		)
	}
	p.current = to
}

// CurveTo draws a quadratic bezier from the current point to to with control
// point ctrl. The first CurveTo after a MoveTo only moves the pen.
func (p *Path[T]) CurveTo(to, ctrl geometry.Point[T]) {
	p.contourCount++
	if p.contourCount > 1 {
		p.PushTriangle(
			p.start, p.current, to,
			geometry.Point[float32]{X: 0, Y: 1},
			geometry.Point[float32]{X: 0, Y: 1},
			geometry.Point[float32]{X: 0, Y: 1},
		)
	}
	p.PushTriangle(
		p.current, ctrl, to,
		geometry.Point[float32]{X: 0, Y: 0},
		geometry.Point[float32]{X: 0.5, Y: 0},
		geometry.Point[float32]{X: 1, Y: 1},
	)
	p.current = to
}

// PushTriangle appends one triangle to the path, given its three XY positions
// and three ST texture coordinates. It also grows the path's bounds to cover
// the triangle.
func (p *Path[T]) PushTriangle(
	xy0, xy1, xy2 geometry.Point[T],
	st0, st1, st2 geometry.Point[float32],
) {
	p.Bounds = p.Bounds.Union(geometry.Bounds[T]{Origin: xy0}).
		Union(geometry.Bounds[T]{Origin: xy1}).
		Union(geometry.Bounds[T]{Origin: xy2})

	p.Vertices = append(p.Vertices, PathVertex[T]{
		XYPosition:  xy0,
		STPosition:  st0,
		ContentMask: ContentMask[T]{},
	})
	p.Vertices = append(p.Vertices, PathVertex[T]{
		XYPosition:  xy1,
		STPosition:  st1,
		ContentMask: ContentMask[T]{},
	})
	p.Vertices = append(p.Vertices, PathVertex[T]{
		XYPosition:  xy2,
		STPosition:  st2,
		ContentMask: ContentMask[T]{},
	})
}

// ClippedBounds returns the path's bounds intersected with its content mask.
func (p Path[T]) ClippedBounds() geometry.Bounds[T] {
	return p.Bounds.Intersect(p.ContentMask.Bounds)
}

// ScalePath converts a path in pixels to scaled pixels by multiplying every
// coordinate by factor. It is a free function because a method on
// Path[Pixels] cannot introduce the Path[ScaledPixels] return type.
func ScalePath(p Path[geometry.Pixels], factor float32) Path[geometry.ScaledPixels] {
	verts := make([]PathVertex[geometry.ScaledPixels], len(p.Vertices))
	for i, v := range p.Vertices {
		verts[i] = PathVertex[geometry.ScaledPixels]{
			XYPosition:  geometry.ScalePoint(v.XYPosition, factor),
			STPosition:  v.STPosition,
			ContentMask: ContentMask[geometry.ScaledPixels]{Bounds: geometry.ScaleBounds(v.ContentMask.Bounds, factor)},
		}
	}
	return Path[geometry.ScaledPixels]{
		ID:           p.ID,
		Order:        p.Order,
		Bounds:       geometry.ScaleBounds(p.Bounds, factor),
		ContentMask:  ContentMask[geometry.ScaledPixels]{Bounds: geometry.ScaleBounds(p.ContentMask.Bounds, factor)},
		Vertices:     verts,
		Colour:       p.Colour,
		start:        geometry.ScalePoint(p.start, factor),
		current:      geometry.ScalePoint(p.current, factor),
		contourCount: p.contourCount,
	}
}
