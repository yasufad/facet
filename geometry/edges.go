package geometry

// Edges holds the four per-side lengths of a box, such as padding, margin or
// a border. It is generic over the length unit.
type Edges[T Number] struct {
	Top    T
	Right  T
	Bottom T
	Left   T
}

// NewEdges constructs Edges from its four sides, in top, right, bottom, left
// order — the same clockwise order CSS uses for padding and margin.
func NewEdges[T Number](top, right, bottom, left T) Edges[T] {
	return Edges[T]{Top: top, Right: right, Bottom: bottom, Left: left}
}

// AllEdges constructs Edges with every side set to v.
func AllEdges[T Number](v T) Edges[T] {
	return Edges[T]{Top: v, Right: v, Bottom: v, Left: v}
}

// SymmetricEdges constructs Edges with the vertical sides (top and bottom) set
// to vertical and the horizontal sides (left and right) set to horizontal,
// matching the two-value CSS shorthand.
func SymmetricEdges[T Number](vertical, horizontal T) Edges[T] {
	return Edges[T]{Top: vertical, Right: horizontal, Bottom: vertical, Left: horizontal}
}

// MapEdges applies f to each side of e, producing Edges in a new unit. It is
// a free function because Go methods cannot introduce the second type
// parameter a cross-unit mapping requires.
func MapEdges[T, U Number](e Edges[T], f func(T) U) Edges[U] {
	return Edges[U]{
		Top:    f(e.Top),
		Right:  f(e.Right),
		Bottom: f(e.Bottom),
		Left:   f(e.Left),
	}
}

// Max returns the greatest of the four sides.
func (e Edges[T]) Max() T {
	return max(max(e.Top, e.Right), max(e.Bottom, e.Left))
}

// Add returns the componentwise sum of e and r.
func (e Edges[T]) Add(r Edges[T]) Edges[T] {
	return Edges[T]{
		Top:    e.Top + r.Top,
		Right:  e.Right + r.Right,
		Bottom: e.Bottom + r.Bottom,
		Left:   e.Left + r.Left,
	}
}

// ScaleEdges multiplies each side by the display scale factor, producing
// Edges in ScaledPixels. It is a free function because a method on the
// instantiated generic Edges[Pixels] cannot reach the methods on its Pixels
// components.
func ScaleEdges(e Edges[Pixels], factor float32) Edges[ScaledPixels] {
	return Edges[ScaledPixels]{
		Top:    e.Top.Scale(factor),
		Right:  e.Right.Scale(factor),
		Bottom: e.Bottom.Scale(factor),
		Left:   e.Left.Scale(factor),
	}
}
