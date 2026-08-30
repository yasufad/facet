package colour

// Hsla is a colour in the HSL space with alpha.
//
// Hue, saturation, lightness and alpha are each float32 in [0, 1]; hue is a
// fraction of a full turn, so 0 and 1 are both red. The zero value is
// transparent black.
type Hsla struct {
	H float32 // Hue, in [0, 1].
	S float32 // Saturation, in [0, 1].
	L float32 // Lightness, in [0, 1].
	A float32 // Alpha, in [0, 1]; 1 is fully opaque.
}

// IsOpaque reports whether the colour is fully opaque. The renderer uses this
// to skip blending.
func (c Hsla) IsOpaque() bool {
	return c.A >= 1
}

// Opacity returns a copy with the alpha scaled by factor, clamped to [0, 1].
func (c Hsla) Opacity(factor float32) Hsla {
	return Hsla{H: c.H, S: c.S, L: c.L, A: c.A * clamp01(factor)}
}

// Lighten returns the colour with its lightness increased by amount, clamped
// to [0, 1]. The hue, saturation and alpha are unchanged.
func (c Hsla) Lighten(amount float32) Hsla {
	return Hsla{H: c.H, S: c.S, L: clamp01(c.L + amount), A: c.A}
}

// Darken returns the colour with its lightness decreased by amount, clamped
// to [0, 1]. The hue, saturation and alpha are unchanged.
func (c Hsla) Darken(amount float32) Hsla {
	return Hsla{H: c.H, S: c.S, L: clamp01(c.L - amount), A: c.A}
}
