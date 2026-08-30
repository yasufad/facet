package colour

import (
	"math"
	"testing"
)

func hslaEq(a, b Hsla) bool {
	return approx(a.H, b.H) && approx(a.S, b.S) && approx(a.L, b.L) && approx(a.A, b.A)
}

func TestHslaToRgba(t *testing.T) {
	cases := []struct {
		name string
		in   Hsla
		want Rgba
	}{
		{"red", Hsla{0, 1, 0.5, 1}, Rgba{1, 0, 0, 1}},
		{"green", Hsla{1.0 / 3.0, 1, 0.25, 1}, Rgba{0, 0.5, 0, 1}},
		{"blue", Hsla{2.0 / 3.0, 1, 0.5, 1}, Rgba{0, 0, 1, 1}},
		{"grey", Hsla{0, 0, 0.5, 1}, Rgba{0.5, 0.5, 0.5, 1}},
		{"white", Hsla{0, 0, 1, 1}, Rgba{1, 1, 1, 1}},
		{"black", Hsla{0, 1, 0, 1}, Rgba{0, 0, 0, 1}},
	}
	for _, c := range cases {
		got := c.in.Rgba()
		if !rgbaEq(got, c.want) {
			t.Errorf("%s: %v.Rgba() = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}

func TestRgbaToHsla(t *testing.T) {
	cases := []struct {
		name string
		in   Rgba
		want Hsla
	}{
		{"red", Rgba{1, 0, 0, 1}, Hsla{0, 1, 0.5, 1}},
		{"green", Rgba{0, 1, 0, 1}, Hsla{1.0 / 3.0, 1, 0.5, 1}},
		{"blue", Rgba{0, 0, 1, 1}, Hsla{2.0 / 3.0, 1, 0.5, 1}},
		{"grey", Rgba{0.5, 0.5, 0.5, 1}, Hsla{0, 0, 0.5, 1}},
		{"white", Rgba{1, 1, 1, 1}, Hsla{0, 0, 1, 1}},
		{"black", Rgba{0, 0, 0, 1}, Hsla{0, 0, 0, 1}},
	}
	for _, c := range cases {
		got := c.in.Hsla()
		if !hslaEq(got, c.want) {
			t.Errorf("%s: %v.Hsla() = %v, want %v", c.name, c.in, got, c.want)
		}
	}
}

// TestRgbaHslaRoundTrip checks that converting an Rgba to Hsla and back
// recovers the original colour. The RGB to HSL map is a bijection on the sRGB
// cube, so this round trip is lossless up to floating-point error.
func TestRgbaHslaRoundTrip(t *testing.T) {
	colours := []Rgba{
		{1, 0, 0, 1},
		{0, 1, 0, 1},
		{0, 0, 1, 1},
		{0.2, 0.4, 0.6, 1},
		{0.75, 0.55, 0.25, 1},
		{0.1, 0.2, 0.3, 0.5},
		{0.5, 0.5, 0.5, 0.8},
		{0.9, 0.1, 0.4, 0.0},
	}
	for _, c := range colours {
		round := c.Hsla().Rgba()
		if !rgbaEq(c, round) {
			t.Errorf("round trip %v -> %v -> %v", c, c.Hsla(), round)
		}
	}
}

// TestHslaRgbaRoundTrip checks the reverse round trip for HSL colours that lie
// inside the sRGB gamut, where the map is invertible.
func TestHslaRgbaRoundTrip(t *testing.T) {
	colours := []Hsla{
		{0, 1, 0.5, 1},
		{1.0 / 3.0, 1, 0.5, 1},
		{2.0 / 3.0, 1, 0.5, 1},
		{0.1, 0.5, 0.5, 1},
		{0.55, 0.3, 0.6, 0.7},
		{0, 0, 0.5, 1}, // grey: saturation zero
	}
	for _, c := range colours {
		round := c.Rgba().Hsla()
		if math.Abs(float64(c.H-round.H)) > 1e-4 ||
			math.Abs(float64(c.S-round.S)) > 1e-4 ||
			math.Abs(float64(c.L-round.L)) > 1e-4 ||
			math.Abs(float64(c.A-round.A)) > 1e-5 {
			t.Errorf("round trip %v -> %v -> %v", c, c.Rgba(), round)
		}
	}
}

func TestHslaLightenDarken(t *testing.T) {
	c := Hsla{0.5, 0.5, 0.5, 1}
	if got := c.Lighten(0.2); !hslaEq(got, Hsla{0.5, 0.5, 0.7, 1}) {
		t.Errorf("Lighten = %v, want {0.5, 0.5, 0.7, 1}", got)
	}
	if got := c.Darken(0.2); !hslaEq(got, Hsla{0.5, 0.5, 0.3, 1}) {
		t.Errorf("Darken = %v, want {0.5, 0.5, 0.3, 1}", got)
	}
	// Hue, saturation and alpha are preserved.
	if got := c.Lighten(0.1); got.H != c.H || got.S != c.S || got.A != c.A {
		t.Errorf("Lighten altered H/S/A: %v", got)
	}
}

func TestHslaOpacity(t *testing.T) {
	c := Hsla{0.5, 0.5, 0.5, 0.8}
	got := c.Opacity(0.5)
	if !hslaEq(got, Hsla{0.5, 0.5, 0.5, 0.4}) {
		t.Errorf("Opacity(0.5) = %v, want alpha 0.4", got)
	}
	if got := c.Opacity(2.0); !approx(got.A, 0.8) {
		t.Errorf("Opacity(2.0).A = %v, want 0.8", got.A)
	}
}
