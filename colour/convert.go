package colour

import "math"

// Rgba returns the colour converted to the sRGB space.
//
// The conversion follows the standard HSL to RGB map. Components are clamped
// to [0, 1], which matters for HSL values that fall outside the sRGB gamut.
func (c Hsla) Rgba() Rgba {
	h, s, l := c.H, c.S, c.L
	chroma := (1 - abs32(2*l-1)) * s
	x := chroma * (1 - abs32(mod32(h*6, 2)-1))
	m := l - chroma/2
	cm, xm := chroma+m, x+m

	var r, g, b float32
	switch int(floor32(h * 6)) {
	case 0, 6:
		r, g, b = cm, xm, m
	case 1:
		r, g, b = xm, cm, m
	case 2:
		r, g, b = m, cm, xm
	case 3:
		r, g, b = m, xm, cm
	case 4:
		r, g, b = xm, m, cm
	default:
		r, g, b = cm, m, xm
	}
	return Rgba{R: clamp01(r), G: clamp01(g), B: clamp01(b), A: c.A}
}

// Hsla returns the colour converted to the HSL space.
func (c Rgba) Hsla() Hsla {
	r, g, b := c.R, c.G, c.B
	max := max(r, max(g, b))
	min := min(r, min(g, b))
	delta := max - min

	l := (max + min) / 2
	var s float32
	switch {
	case l == 0 || l == 1:
		s = 0
	case l < 0.5:
		s = delta / (2 * l)
	default:
		s = delta / (2 - 2*l)
	}

	var h float32
	switch {
	case delta == 0:
		h = 0
	case max == r:
		h = euclidMod((g-b)/delta, 6) / 6
	case max == g:
		h = ((b - r) / delta) / 6
		h += 2.0 / 6.0
	default:
		h = ((r - g) / delta) / 6
		h += 4.0 / 6.0
	}

	return Hsla{H: h, S: s, L: l, A: c.A}
}

// clamp01 clamps a float32 to the range [0, 1].
func clamp01(x float32) float32 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// abs32, floor32 and mod32 are float32 wrappers over the math package. The
// conversions through float64 are not on a per-frame hot path.
func abs32(x float32) float32    { return float32(math.Abs(float64(x))) }
func floor32(x float32) float32  { return float32(math.Floor(float64(x))) }
func mod32(x, y float32) float32 { return float32(math.Mod(float64(x), float64(y))) }

// euclidMod returns the Euclidean remainder of x divided by y, always in
// [0, y). math.Mod is truncated, so a negative result is shifted by y.
func euclidMod(x, y float32) float32 {
	m := mod32(x, y)
	if m < 0 {
		m += y
	}
	return m
}
