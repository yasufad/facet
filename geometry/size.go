package geometry

// Size is a 2D extent: a width and a height. It is generic over the dimension
// unit so the same type serves logical pixels, device pixels and layout units.
type Size[T Number] struct {
	Width  T
	Height T
}

// NewSize constructs a Size from its width and height.
func NewSize[T Number](width, height T) Size[T] { return Size[T]{Width: width, Height: height} }

// MapSize applies f to both dimensions of s, producing a Size in a new unit.
// It is a free function because Go methods cannot introduce the second type
// parameter a cross-unit mapping requires.
func MapSize[T, U Number](s Size[T], f func(T) U) Size[U] {
	return Size[U]{Width: f(s.Width), Height: f(s.Height)}
}

// Along returns the dimension of s on the given axis: Width for Horizontal,
// Height for Vertical.
func (s Size[T]) Along(a Axis) T { return along(a, s.Width, s.Height) }

// ApplyAlong returns a copy of s with the dimension on the given axis
// replaced by the result of f. The other dimension is left untouched.
func (s Size[T]) ApplyAlong(a Axis, f func(T) T) Size[T] {
	w, h := applyAlong(a, s.Width, s.Height, f)
	return Size[T]{Width: w, Height: h}
}

// Add returns the componentwise sum of s and r.
func (s Size[T]) Add(r Size[T]) Size[T] {
	return Size[T]{Width: s.Width + r.Width, Height: s.Height + r.Height}
}

// Sub returns the componentwise difference of s and r.
func (s Size[T]) Sub(r Size[T]) Size[T] {
	return Size[T]{Width: s.Width - r.Width, Height: s.Height - r.Height}
}

// Mul returns s scaled by the scalar v.
func (s Size[T]) Mul(v T) Size[T] { return Size[T]{Width: s.Width * v, Height: s.Height * v} }

// Div returns s divided by the scalar v.
func (s Size[T]) Div(v T) Size[T] { return Size[T]{Width: s.Width / v, Height: s.Height / v} }

// Max returns a Size whose dimensions are the greater of s's and r's.
func (s Size[T]) Max(r Size[T]) Size[T] {
	return Size[T]{Width: max(s.Width, r.Width), Height: max(s.Height, r.Height)}
}

// Min returns a Size whose dimensions are the lesser of s's and r's.
func (s Size[T]) Min(r Size[T]) Size[T] {
	return Size[T]{Width: min(s.Width, r.Width), Height: min(s.Height, r.Height)}
}

// Center returns the point at the centre of a Size of the given dimensions,
// treating the top-left as the origin.
func (s Size[T]) Center() Point[T] {
	return Point[T]{X: s.Width / 2, Y: s.Height / 2}
}

// IsZero reports whether both dimensions are zero.
func (s Size[T]) IsZero() bool { return s.Width == 0 && s.Height == 0 }

// ScaleSize multiplies both dimensions by the display scale factor, producing
// a Size in ScaledPixels. It is a free function because a method on the
// instantiated generic Size[Pixels] cannot reach the methods on its Pixels
// components.
func ScaleSize(s Size[Pixels], factor float32) Size[ScaledPixels] {
	return Size[ScaledPixels]{
		Width:  s.Width.Scale(factor),
		Height: s.Height.Scale(factor),
	}
}

// SizeToDevicePixels converts a logical-pixel size to device pixels, rounding
// each dimension to the nearest device pixel.
func SizeToDevicePixels(s Size[Pixels], factor float32) Size[DevicePixels] {
	return Size[DevicePixels]{
		Width:  s.Width.ToDevicePixels(factor),
		Height: s.Height.ToDevicePixels(factor),
	}
}

// DeviceSizeToPixels converts a device-pixel size to logical pixels by
// dividing by the display scale factor.
func DeviceSizeToPixels(s Size[DevicePixels], factor float32) Size[Pixels] {
	return Size[Pixels]{
		Width:  s.Width.ToPixels(factor),
		Height: s.Height.ToPixels(factor),
	}
}
