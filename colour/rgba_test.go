package colour

import (
	"math"
	"testing"
)

// approx reports whether two float32 values are equal within a small tolerance.
// Conversions between Rgba and Hsla go through float64 math helpers, so a tiny
// epsilon accounts for rounding without hiding real regressions.
func approx(a, b float32) bool {
	return math.Abs(float64(a-b)) < 1e-5
}

func rgbaEq(a, b Rgba) bool {
	return approx(a.R, b.R) && approx(a.G, b.G) && approx(a.B, b.B) && approx(a.A, b.A)
}

func TestRgb(t *testing.T) {
	cases := []struct {
		hex  uint32
		want Rgba
	}{
		{0xff0000, Rgba{1, 0, 0, 1}},
		{0x00ff00, Rgba{0, 1, 0, 1}},
		{0x0000ff, Rgba{0, 0, 1, 1}},
		{0x000000, Rgba{0, 0, 0, 1}},
		{0xffffff, Rgba{1, 1, 1, 1}},
	}
	for _, c := range cases {
		got := Rgb(c.hex)
		if !rgbaEq(got, c.want) {
			t.Errorf("Rgb(%#x) = %v, want %v", c.hex, got, c.want)
		}
		if !got.IsOpaque() {
			t.Errorf("Rgb(%#x): not opaque", c.hex)
		}
	}
}

func TestRgba32(t *testing.T) {
	cases := []struct {
		hex  uint32
		want Rgba
	}{
		{0xff0000ff, Rgba{1, 0, 0, 1}},
		{0xff000000, Rgba{1, 0, 0, 0}},
		{0x3399ffcc, Rgba{0x33 / 255.0, 0x99 / 255.0, 0xff / 255.0, 0xcc / 255.0}},
	}
	for _, c := range cases {
		got := Rgba32(c.hex)
		if !rgbaEq(got, c.want) {
			t.Errorf("Rgba32(%#x) = %v, want %v", c.hex, got, c.want)
		}
	}
}

func TestRgbaIsOpaque(t *testing.T) {
	if !(Rgba{1, 1, 1, 1}.IsOpaque()) {
		t.Error("alpha 1 should be opaque")
	}
	if (Rgba{1, 1, 1, 0.5}.IsOpaque()) {
		t.Error("alpha 0.5 should not be opaque")
	}
	if (Rgba{1, 1, 1, 0}.IsOpaque()) {
		t.Error("alpha 0 should not be opaque")
	}
}

func TestRgbaOpacity(t *testing.T) {
	c := Rgba{0.2, 0.6, 1.0, 0.8}
	got := c.Opacity(0.5)
	want := Rgba{0.2, 0.6, 1.0, 0.4}
	if !rgbaEq(got, want) {
		t.Errorf("Opacity(0.5) = %v, want %v", got, want)
	}

	// A factor above 1 is clamped, so alpha cannot grow beyond its current value.
	if got := c.Opacity(2.0); !approx(got.A, 0.8) {
		t.Errorf("Opacity(2.0).A = %v, want 0.8", got.A)
	}
	// A factor below 0 is clamped to fully transparent.
	if got := c.Opacity(-1.0); !approx(got.A, 0) {
		t.Errorf("Opacity(-1).A = %v, want 0", got.A)
	}
	// RGB channels are untouched.
	if got.R != c.R || got.G != c.G || got.B != c.B {
		t.Errorf("Opacity altered RGB channels: %v", got)
	}
}

func TestPremultiply(t *testing.T) {
	c := Rgba{1, 0.5, 0.25, 0.5}
	got := c.Premultiply()
	want := Rgba{0.5, 0.25, 0.125, 0.5}
	if !rgbaEq(got, want) {
		t.Errorf("Premultiply = %v, want %v", got, want)
	}
	// An opaque colour is unchanged by premultiplication.
	opaque := Rgba{0.2, 0.4, 0.6, 1}
	if g := opaque.Premultiply(); !rgbaEq(g, opaque) {
		t.Errorf("Premultiply of opaque = %v, want %v", g, opaque)
	}
	// The original is left straight.
	if c.A != 0.5 {
		t.Error("Premultiply mutated the receiver")
	}
}

func TestRgbaLightenDarken(t *testing.T) {
	// Lighten and darken move through HSL, so a mid-grey lightens and darkens
	// symmetrically about its lightness.
	grey := Rgba{0.5, 0.5, 0.5, 1}
	light := grey.Lighten(0.2)
	dark := grey.Darken(0.2)
	if !approx(light.Hsla().L, 0.5+0.2) {
		t.Errorf("Lighten: L = %v, want %v", light.Hsla().L, 0.7)
	}
	if !approx(dark.Hsla().L, 0.5-0.2) {
		t.Errorf("Darken: L = %v, want %v", dark.Hsla().L, 0.3)
	}
	// Lightness clamps at the bounds.
	if g := grey.Lighten(1).Hsla().L; !approx(g, 1) {
		t.Errorf("Lighten past white: L = %v, want 1", g)
	}
	if g := grey.Darken(1).Hsla().L; !approx(g, 0) {
		t.Errorf("Darken past black: L = %v, want 0", g)
	}
}
