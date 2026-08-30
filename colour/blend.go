package colour

// Blend composites other onto c with source-over alpha blending.
//
// other is the source, c is the backdrop. A fully opaque source replaces the
// backdrop; a fully transparent source leaves it unchanged. Otherwise the
// result is the standard source-over composite, with the output alpha
// accounting for both alphas:
//
//	outA = c.A + other.A * (1 - c.A)
//
// and the colour channels weighted by each layer's alpha and divided by the
// result alpha. The returned colour is straight (non-premultiplied).
func (c Rgba) Blend(other Rgba) Rgba {
	switch {
	case other.A >= 1:
		return other
	case other.A <= 0:
		return c
	}

	outA := c.A + other.A*(1-c.A)
	if outA <= 0 {
		return Rgba{}
	}
	w := 1 / outA
	return Rgba{
		R: (c.R*c.A*(1-other.A) + other.R*other.A) * w,
		G: (c.G*c.A*(1-other.A) + other.G*other.A) * w,
		B: (c.B*c.A*(1-other.A) + other.B*other.A) * w,
		A: outA,
	}
}

// Blend composites other onto c with source-over alpha blending, converting
// through RGB so the alpha weighting is applied in a linear space.
func (c Hsla) Blend(other Hsla) Hsla {
	return c.Rgba().Blend(other.Rgba()).Hsla()
}
