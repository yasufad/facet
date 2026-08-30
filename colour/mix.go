package colour

// Mix returns the linear interpolation between c and other at factor t.
//
// t of 0 returns c, t of 1 returns other; values in between blend each
// component linearly. t is not clamped, so values outside [0, 1] extrapolate.
func (c Rgba) Mix(other Rgba, t float32) Rgba {
	return Rgba{
		R: c.R + (other.R-c.R)*t,
		G: c.G + (other.G-c.G)*t,
		B: c.B + (other.B-c.B)*t,
		A: c.A + (other.A-c.A)*t,
	}
}

// Mix returns the linear interpolation between c and other at factor t,
// component-wise in HSL space.
//
// Hue is interpolated linearly, not along the shortest path around the wheel,
// so mixing two hues on opposite sides of the wheel passes through the
// midpoint rather than wrapping. t is not clamped.
func (c Hsla) Mix(other Hsla, t float32) Hsla {
	return Hsla{
		H: c.H + (other.H-c.H)*t,
		S: c.S + (other.S-c.S)*t,
		L: c.L + (other.L-c.L)*t,
		A: c.A + (other.A-c.A)*t,
	}
}
