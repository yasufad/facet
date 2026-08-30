package geometry

// Axis names one of the two cartesian axes. Layout algorithms take it as a
// parameter so they can be written once rather than once per axis.
type Axis int

const (
	// Horizontal is the x axis, left and right.
	Horizontal Axis = iota
	// Vertical is the y axis, up and down.
	Vertical
)

// Invert returns the opposite axis.
func (a Axis) Invert() Axis {
	if a == Horizontal {
		return Vertical
	}
	return Horizontal
}

// along returns the component of a two-value pair selected by a. It is the
// shared read behind Point.Along and Size.Along.
func along[T Number](a Axis, first, second T) T {
	if a == Horizontal {
		return first
	}
	return second
}

// applyAlong returns a pair with the component selected by a replaced by the
// result of f. It is the shared write behind Point.ApplyAlong and
// Size.ApplyAlong.
func applyAlong[T Number](a Axis, first, second T, f func(T) T) (T, T) {
	if a == Horizontal {
		return f(first), second
	}
	return first, f(second)
}
