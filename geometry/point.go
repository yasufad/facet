package geometry

// Point is a location in 2D space. It is generic over the coordinate unit so
// the same type serves logical pixels, device pixels and layout units.
type Point[T Number] struct {
	X T
	Y T
}

// NewPoint constructs a Point from its x and y coordinates.
func NewPoint[T Number](x, y T) Point[T] { return Point[T]{X: x, Y: y} }

// MapPoint applies f to both coordinates of p, producing a Point in a new
// unit. It is a free function because Go methods cannot introduce the second
// type parameter a cross-unit mapping requires.
func MapPoint[T, U Number](p Point[T], f func(T) U) Point[U] {
	return Point[U]{X: f(p.X), Y: f(p.Y)}
}

// Along returns the coordinate of p on the given axis: X for Horizontal, Y
// for Vertical.
func (p Point[T]) Along(a Axis) T { return along(a, p.X, p.Y) }

// ApplyAlong returns a copy of p with the coordinate on the given axis
// replaced by the result of f. The other coordinate is left untouched.
func (p Point[T]) ApplyAlong(a Axis, f func(T) T) Point[T] {
	x, y := applyAlong(a, p.X, p.Y, f)
	return Point[T]{X: x, Y: y}
}

// Add returns the componentwise sum of p and q.
func (p Point[T]) Add(q Point[T]) Point[T] { return Point[T]{X: p.X + q.X, Y: p.Y + q.Y} }

// Sub returns the componentwise difference of p and q.
func (p Point[T]) Sub(q Point[T]) Point[T] { return Point[T]{X: p.X - q.X, Y: p.Y - q.Y} }

// Mul returns p scaled by the scalar s.
func (p Point[T]) Mul(s T) Point[T] { return Point[T]{X: p.X * s, Y: p.Y * s} }

// Div returns p divided by the scalar s.
func (p Point[T]) Div(s T) Point[T] { return Point[T]{X: p.X / s, Y: p.Y / s} }

// RelativeTo returns p expressed relative to origin, subtracting origin from
// each coordinate.
func (p Point[T]) RelativeTo(origin Point[T]) Point[T] { return p.Sub(origin) }

// Max returns a Point whose coordinates are the greater of p's and q's.
func (p Point[T]) Max(q Point[T]) Point[T] {
	return Point[T]{
		X: max(p.X, q.X),
		Y: max(p.Y, q.Y),
	}
}

// Min returns a Point whose coordinates are the lesser of p's and q's.
func (p Point[T]) Min(q Point[T]) Point[T] {
	return Point[T]{
		X: min(p.X, q.X),
		Y: min(p.Y, q.Y),
	}
}

// Clamp constrains each coordinate of p to the range [lo, hi].
func (p Point[T]) Clamp(lo, hi Point[T]) Point[T] { return p.Max(lo).Min(hi) }

// ScalePoint multiplies both coordinates by the display scale factor,
// producing a Point in ScaledPixels. It is a free function because a method
// on the instantiated generic Point[Pixels] cannot reach the methods on its
// Pixels components.
func ScalePoint(p Point[Pixels], factor float32) Point[ScaledPixels] {
	return Point[ScaledPixels]{
		X: p.X.Scale(factor),
		Y: p.Y.Scale(factor),
	}
}
