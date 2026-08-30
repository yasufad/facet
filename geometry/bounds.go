package geometry

// Bounds is an axis-aligned rectangle: an origin and a size. It is generic
// over the coordinate unit so the same type serves logical pixels, device
// pixels and layout units.
type Bounds[T Number] struct {
	Origin Point[T]
	Size   Size[T]
}

// NewBounds constructs a Bounds from its origin and size.
func NewBounds[T Number](origin Point[T], size Size[T]) Bounds[T] {
	return Bounds[T]{Origin: origin, Size: size}
}

// FromCorners constructs a Bounds from its top-left and bottom-right corners.
// The origin is the top-left corner and the size is the difference between
// the corners.
func FromCorners[T Number](topLeft, bottomRight Point[T]) Bounds[T] {
	return Bounds[T]{
		Origin: topLeft,
		Size:   Size[T]{Width: bottomRight.X - topLeft.X, Height: bottomRight.Y - topLeft.Y},
	}
}

// FromAnchorAndSize places a Bounds of the given size so that the named anchor
// sits at origin. For corner anchors the corner is pinned; for centre anchors
// the edge midpoint is pinned.
func FromAnchorAndSize[T Number](a Anchor, origin Point[T], size Size[T]) Bounds[T] {
	switch a {
	case TopLeft:
	case TopRight:
		origin = Point[T]{X: origin.X - size.Width, Y: origin.Y}
	case BottomLeft:
		origin = Point[T]{X: origin.X, Y: origin.Y - size.Height}
	case BottomRight:
		origin = Point[T]{X: origin.X - size.Width, Y: origin.Y - size.Height}
	case TopCenter:
		origin = Point[T]{X: origin.X - size.Width/2, Y: origin.Y}
	case BottomCenter:
		origin = Point[T]{X: origin.X - size.Width/2, Y: origin.Y - size.Height}
	case LeftCenter:
		origin = Point[T]{X: origin.X, Y: origin.Y - size.Height/2}
	case RightCenter:
		origin = Point[T]{X: origin.X - size.Width, Y: origin.Y - size.Height/2}
	}
	return Bounds[T]{Origin: origin, Size: size}
}

// CenteredAt constructs a Bounds of the given size centred on the given point.
func CenteredAt[T Number](center Point[T], size Size[T]) Bounds[T] {
	return Bounds[T]{
		Origin: Point[T]{X: center.X - size.Width/2, Y: center.Y - size.Height/2},
		Size:   size,
	}
}

// Center returns the point at the centre of b.
func (b Bounds[T]) Center() Point[T] {
	return Point[T]{
		X: b.Origin.X + b.Size.Width/2,
		Y: b.Origin.Y + b.Size.Height/2,
	}
}

// Top returns the y coordinate of b's top edge.
func (b Bounds[T]) Top() T { return b.Origin.Y }

// Bottom returns the y coordinate of b's bottom edge.
func (b Bounds[T]) Bottom() T { return b.Origin.Y + b.Size.Height }

// Left returns the x coordinate of b's left edge.
func (b Bounds[T]) Left() T { return b.Origin.X }

// Right returns the x coordinate of b's right edge.
func (b Bounds[T]) Right() T { return b.Origin.X + b.Size.Width }

// TopLeft returns the origin corner of b.
func (b Bounds[T]) TopLeft() Point[T] { return b.Origin }

// TopRight returns the top-right corner of b.
func (b Bounds[T]) TopRight() Point[T] {
	return Point[T]{X: b.Origin.X + b.Size.Width, Y: b.Origin.Y}
}

// BottomRight returns the bottom-right corner of b.
func (b Bounds[T]) BottomRight() Point[T] {
	return Point[T]{X: b.Origin.X + b.Size.Width, Y: b.Origin.Y + b.Size.Height}
}

// BottomLeft returns the bottom-left corner of b.
func (b Bounds[T]) BottomLeft() Point[T] {
	return Point[T]{X: b.Origin.X, Y: b.Origin.Y + b.Size.Height}
}

// TopCenter returns the midpoint of b's top edge.
func (b Bounds[T]) TopCenter() Point[T] {
	return Point[T]{X: b.Origin.X + b.Size.Width/2, Y: b.Origin.Y}
}

// BottomCenter returns the midpoint of b's bottom edge.
func (b Bounds[T]) BottomCenter() Point[T] {
	return Point[T]{X: b.Origin.X + b.Size.Width/2, Y: b.Origin.Y + b.Size.Height}
}

// LeftCenter returns the midpoint of b's left edge.
func (b Bounds[T]) LeftCenter() Point[T] {
	return Point[T]{X: b.Origin.X, Y: b.Origin.Y + b.Size.Height/2}
}

// RightCenter returns the midpoint of b's right edge.
func (b Bounds[T]) RightCenter() Point[T] {
	return Point[T]{X: b.Origin.X + b.Size.Width, Y: b.Origin.Y + b.Size.Height/2}
}

// Corner returns the point named by a.
func (b Bounds[T]) Corner(a Anchor) Point[T] {
	switch a {
	case TopLeft:
		return b.TopLeft()
	case TopRight:
		return b.TopRight()
	case BottomLeft:
		return b.BottomLeft()
	case BottomRight:
		return b.BottomRight()
	case TopCenter:
		return b.TopCenter()
	case BottomCenter:
		return b.BottomCenter()
	case LeftCenter:
		return b.LeftCenter()
	case RightCenter:
		return b.RightCenter()
	}
	return b.Origin
}

// Contains reports whether p lies within b, including the top and left edges
// but excluding the bottom and right edges.
func (b Bounds[T]) Contains(p Point[T]) bool {
	return p.X >= b.Origin.X &&
		p.X < b.Origin.X+b.Size.Width &&
		p.Y >= b.Origin.Y &&
		p.Y < b.Origin.Y+b.Size.Height
}

// IsContainedWithin reports whether b lies entirely within outer.
func (b Bounds[T]) IsContainedWithin(outer Bounds[T]) bool {
	return outer.Contains(b.Origin) && outer.Contains(b.BottomRight())
}

// Intersects reports whether b and other overlap.
func (b Bounds[T]) Intersects(other Bounds[T]) bool {
	myBR := b.BottomRight()
	theirBR := other.BottomRight()
	return b.Origin.X < theirBR.X &&
		myBR.X > other.Origin.X &&
		b.Origin.Y < theirBR.Y &&
		myBR.Y > other.Origin.Y
}

// Intersect returns the overlap of b and other. If they do not intersect the
// result has zero width and height.
func (b Bounds[T]) Intersect(other Bounds[T]) Bounds[T] {
	upperLeft := b.Origin.Max(other.Origin)
	bottomRight := b.BottomRight().Min(other.BottomRight()).Max(upperLeft)
	return FromCorners(upperLeft, bottomRight)
}

// Union returns the smallest Bounds that contains both b and other.
func (b Bounds[T]) Union(other Bounds[T]) Bounds[T] {
	topLeft := b.Origin.Min(other.Origin)
	bottomRight := b.BottomRight().Max(other.BottomRight())
	return FromCorners(topLeft, bottomRight)
}

// Dilate expands b by amount in every direction: the origin moves out by
// amount and the size grows by twice amount.
func (b Bounds[T]) Dilate(amount T) Bounds[T] {
	double := amount + amount
	return Bounds[T]{
		Origin: Point[T]{X: b.Origin.X - amount, Y: b.Origin.Y - amount},
		Size:   Size[T]{Width: b.Size.Width + double, Height: b.Size.Height + double},
	}
}

// Inset shrinks b by amount in every direction. It is Dilate with the amount
// negated.
func (b Bounds[T]) Inset(amount T) Bounds[T] { return b.Dilate(-amount) }

// Extend grows b by the given per-edge amounts.
func (b Bounds[T]) Extend(amount Edges[T]) Bounds[T] {
	return Bounds[T]{
		Origin: Point[T]{X: b.Origin.X - amount.Left, Y: b.Origin.Y - amount.Top},
		Size: Size[T]{
			Width:  b.Size.Width + amount.Left + amount.Right,
			Height: b.Size.Height + amount.Top + amount.Bottom,
		},
	}
}

// SpaceWithin returns the gaps between b and an enclosing Bounds, one Edges
// value per side.
func (b Bounds[T]) SpaceWithin(outer Bounds[T]) Edges[T] {
	return Edges[T]{
		Top:    b.Top() - outer.Top(),
		Right:  outer.Right() - b.Right(),
		Bottom: outer.Bottom() - b.Bottom(),
		Left:   b.Left() - outer.Left(),
	}
}

// HalfPerimeter returns the sum of b's width and height.
func (b Bounds[T]) HalfPerimeter() T { return b.Size.Width + b.Size.Height }

// IsEmpty reports whether b has zero or negative area.
func (b Bounds[T]) IsEmpty() bool { return b.Size.Width <= 0 || b.Size.Height <= 0 }

// Localize expresses p in the coordinate space whose origin is b's top-left.
// The bool is false when p lies outside b.
func (b Bounds[T]) Localize(p Point[T]) (Point[T], bool) {
	if !b.Contains(p) {
		return Point[T]{}, false
	}
	return p.RelativeTo(b.Origin), true
}

// MapBounds applies f to every coordinate and dimension of b, producing a
// Bounds in a new unit. It is a free function because Go methods cannot
// introduce the second type parameter a cross-unit mapping requires.
func MapBounds[T, U Number](b Bounds[T], f func(T) U) Bounds[U] {
	return Bounds[U]{Origin: MapPoint(b.Origin, f), Size: MapSize(b.Size, f)}
}

// MapOrigin applies f to b's origin, leaving the size unchanged.
func (b Bounds[T]) MapOrigin(f func(T) T) Bounds[T] {
	return Bounds[T]{Origin: MapPoint(b.Origin, f), Size: b.Size}
}

// MapSize applies f to b's size, leaving the origin unchanged.
func (b Bounds[T]) MapSize(f func(T) T) Bounds[T] {
	return Bounds[T]{Origin: b.Origin, Size: MapSize(b.Size, f)}
}

// AddPoint returns b with its origin translated by p.
func (b Bounds[T]) AddPoint(p Point[T]) Bounds[T] {
	return Bounds[T]{Origin: b.Origin.Add(p), Size: b.Size}
}

// SubPoint returns b with its origin translated back by p.
func (b Bounds[T]) SubPoint(p Point[T]) Bounds[T] {
	return Bounds[T]{Origin: b.Origin.Sub(p), Size: b.Size}
}

// ScaleBounds multiplies b's origin and size by the display scale factor,
// producing a Bounds in ScaledPixels. It is a free function because a method
// on the instantiated generic Bounds[Pixels] cannot reach the methods on its
// Pixels components.
func ScaleBounds(b Bounds[Pixels], factor float32) Bounds[ScaledPixels] {
	return Bounds[ScaledPixels]{
		Origin: ScalePoint(b.Origin, factor),
		Size:   ScaleSize(b.Size, factor),
	}
}

// BoundsToDevicePixels converts a logical-pixel Bounds to device pixels by
// snapping both edges to the nearest device pixel and deriving the size from
// them. Two rectangles that share an edge in logical pixels share the same
// device edge, because a neighbour's near edge is the same expression as this
// rectangle's far edge. Rounding origin and size independently would not give
// that: a one-pixel gap or overlap appears along the seam at fractional scales.
func BoundsToDevicePixels(b Bounds[Pixels], factor float32) Bounds[DevicePixels] {
	x0 := b.Origin.X.ToDevicePixels(factor)
	x1 := (b.Origin.X + b.Size.Width).ToDevicePixels(factor)
	y0 := b.Origin.Y.ToDevicePixels(factor)
	y1 := (b.Origin.Y + b.Size.Height).ToDevicePixels(factor)
	return Bounds[DevicePixels]{
		Origin: Point[DevicePixels]{X: x0, Y: y0},
		Size:   Size[DevicePixels]{Width: x1 - x0, Height: y1 - y0},
	}
}

// DeviceBoundsToPixels converts a device-pixel Bounds to logical pixels by
// dividing both edges by the display scale factor and deriving the size from
// them, the inverse of BoundsToDevicePixels. Deriving the size from the edges
// rather than converting it independently keeps a rectangle's far edge equal
// to a neighbour's near edge in logical space, so adjacency survives a
// logical-device-logical round trip.
func DeviceBoundsToPixels(b Bounds[DevicePixels], factor float32) Bounds[Pixels] {
	x0 := b.Origin.X.ToPixels(factor)
	x1 := (b.Origin.X + b.Size.Width).ToPixels(factor)
	y0 := b.Origin.Y.ToPixels(factor)
	y1 := (b.Origin.Y + b.Size.Height).ToPixels(factor)
	return Bounds[Pixels]{
		Origin: Point[Pixels]{X: x0, Y: y0},
		Size:   Size[Pixels]{Width: x1 - x0, Height: y1 - y0},
	}
}
