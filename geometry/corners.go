package geometry

// Corners holds the four per-corner values of a box, such as border radii. It
// is generic over the length unit.
type Corners[T Number] struct {
	TopLeft     T
	TopRight    T
	BottomRight T
	BottomLeft  T
}

// NewCorners constructs Corners in clockwise order from the top-left corner,
// the same order CSS uses for border-radius.
func NewCorners[T Number](topLeft, topRight, bottomRight, bottomLeft T) Corners[T] {
	return Corners[T]{
		TopLeft:     topLeft,
		TopRight:    topRight,
		BottomRight: bottomRight,
		BottomLeft:  bottomLeft,
	}
}

// AllCorners constructs Corners with every corner set to v.
func AllCorners[T Number](v T) Corners[T] {
	return Corners[T]{TopLeft: v, TopRight: v, BottomRight: v, BottomLeft: v}
}

// SymmetricCorners constructs Corners mirrored across the horizontal midline:
// the two top corners are set to top and the two bottom corners to bottom.
func SymmetricCorners[T Number](top, bottom T) Corners[T] {
	return Corners[T]{TopLeft: top, TopRight: top, BottomRight: bottom, BottomLeft: bottom}
}

// MapCorners applies f to each corner of c, producing Corners in a new unit.
// It is a free function because Go methods cannot introduce the second type
// parameter a cross-unit mapping requires.
func MapCorners[T, U Number](c Corners[T], f func(T) U) Corners[U] {
	return Corners[U]{
		TopLeft:     f(c.TopLeft),
		TopRight:    f(c.TopRight),
		BottomRight: f(c.BottomRight),
		BottomLeft:  f(c.BottomLeft),
	}
}

// Max returns the greatest of the four corners.
func (c Corners[T]) Max() T {
	return max(max(c.TopLeft, c.TopRight), max(c.BottomRight, c.BottomLeft))
}

// Corner returns the value at a. For the four corner anchors it returns the
// matching field; for the four centre anchors it returns the average of the
// two adjacent corners.
func (c Corners[T]) Corner(a Anchor) T {
	switch a {
	case TopLeft:
		return c.TopLeft
	case TopRight:
		return c.TopRight
	case BottomLeft:
		return c.BottomLeft
	case BottomRight:
		return c.BottomRight
	case TopCenter:
		return (c.TopLeft + c.TopRight) / 2
	case BottomCenter:
		return (c.BottomLeft + c.BottomRight) / 2
	case LeftCenter:
		return (c.TopLeft + c.BottomLeft) / 2
	case RightCenter:
		return (c.TopRight + c.BottomRight) / 2
	}
	return c.TopLeft
}

// ScaleCorners multiplies each corner by the display scale factor, producing
// Corners in ScaledPixels. It is a free function because a method on the
// instantiated generic Corners[Pixels] cannot reach the methods on its Pixels
// components.
func ScaleCorners(c Corners[Pixels], factor float32) Corners[ScaledPixels] {
	return Corners[ScaledPixels]{
		TopLeft:     c.TopLeft.Scale(factor),
		TopRight:    c.TopRight.Scale(factor),
		BottomRight: c.BottomRight.Scale(factor),
		BottomLeft:  c.BottomLeft.Scale(factor),
	}
}
