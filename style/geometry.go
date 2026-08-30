package style

// Point holds horizontal and vertical values.
type Point[T any] struct {
	X T
	Y T
}

// NewPoint constructs a Point with the given x and y values.
func NewPoint[T any](x, y T) Point[T] {
	return Point[T]{X: x, Y: y}
}

// Size holds width and height values.
type Size[T any] struct {
	Width  T
	Height T
}

// NewSize constructs a Size with the given width and height values.
func NewSize[T any](width, height T) Size[T] {
	return Size[T]{Width: width, Height: height}
}

// Edges holds values for the four sides of a rectangle.
type Edges[T any] struct {
	Top    T
	Right  T
	Bottom T
	Left   T
}

// NewEdges constructs an Edges struct with the given top, right, bottom, and left values.
func NewEdges[T any](top, right, bottom, left T) Edges[T] {
	return Edges[T]{Top: top, Right: right, Bottom: bottom, Left: left}
}

// Corners holds values for the four corners of a rectangle.
type Corners[T any] struct {
	TopLeft     T
	TopRight    T
	BottomRight T
	BottomLeft  T
}

// NewCorners constructs a Corners struct with the given values.
func NewCorners[T any](topLeft, topRight, bottomRight, bottomLeft T) Corners[T] {
	return Corners[T]{
		TopLeft:     topLeft,
		TopRight:    topRight,
		BottomRight: bottomRight,
		BottomLeft:  bottomLeft,
	}
}
