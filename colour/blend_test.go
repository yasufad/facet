package colour

import "testing"

func TestRgbaBlendOpaqueSource(t *testing.T) {
	// A fully opaque source replaces the backdrop entirely.
	backdrop := Rgba{0.2, 0.4, 0.6, 1}
	source := Rgba{1, 0, 0, 1}
	if got := backdrop.Blend(source); !rgbaEq(got, source) {
		t.Errorf("Blend opaque source = %v, want %v", got, source)
	}
}

func TestRgbaBlendTransparentSource(t *testing.T) {
	// A fully transparent source leaves the backdrop unchanged.
	backdrop := Rgba{0.2, 0.4, 0.6, 0.7}
	source := Rgba{1, 0, 0, 0}
	if got := backdrop.Blend(source); !rgbaEq(got, backdrop) {
		t.Errorf("Blend transparent source = %v, want %v", got, backdrop)
	}
}

func TestRgbaBlendOpaqueBackdrop(t *testing.T) {
	// Red backdrop, 50% blue source over it: the result is purple with full
	// alpha. outA = 1, channels are (1*1*0.5 + 0*0.5) = 0.5 for red and
	// (0*1*0.5 + 1*0.5) = 0.5 for blue.
	backdrop := Rgba{1, 0, 0, 1}
	source := Rgba{0, 0, 1, 0.5}
	want := Rgba{0.5, 0, 0.5, 1}
	if got := backdrop.Blend(source); !rgbaEq(got, want) {
		t.Errorf("Blend = %v, want %v", got, want)
	}
}

func TestRgbaBlendTwoTransparents(t *testing.T) {
	// Half-transparent black over half-transparent white.
	// outA = 0.5 + 0.5*0.5 = 0.75
	// outR = (0*0.5*0.5 + 1*0.5) / 0.75 = 0.5/0.75
	backdrop := Rgba{0, 0, 0, 0.5}
	source := Rgba{1, 1, 1, 0.5}
	want := Rgba{0.5 / 0.75, 0.5 / 0.75, 0.5 / 0.75, 0.75}
	if got := backdrop.Blend(source); !rgbaEq(got, want) {
		t.Errorf("Blend = %v, want %v", got, want)
	}
}

func TestHslaBlendMatchesRgba(t *testing.T) {
	// Hsla.Blend converts through RGB, so it must agree with Rgba.Blend on the
	// same colours.
	backdrop := Hsla{0, 1, 0.5, 1}         // red
	source := Hsla{2.0 / 3.0, 1, 0.5, 0.5} // blue, 50% alpha
	got := backdrop.Blend(source)
	want := backdrop.Rgba().Blend(source.Rgba()).Hsla()
	if !hslaEq(got, want) {
		t.Errorf("Hsla.Blend = %v, want %v (via Rgba)", got, want)
	}
}

func TestRgbaMix(t *testing.T) {
	a := Rgba{1, 0, 0, 1}
	b := Rgba{0, 0, 1, 1}
	if got := a.Mix(b, 0); !rgbaEq(got, a) {
		t.Errorf("Mix(t=0) = %v, want %v", got, a)
	}
	if got := a.Mix(b, 1); !rgbaEq(got, b) {
		t.Errorf("Mix(t=1) = %v, want %v", got, b)
	}
	if got := a.Mix(b, 0.5); !rgbaEq(got, Rgba{0.5, 0, 0.5, 1}) {
		t.Errorf("Mix(t=0.5) = %v, want {0.5, 0, 0.5, 1}", got)
	}
}

func TestHslaMix(t *testing.T) {
	a := Hsla{0, 1, 0.5, 1}
	b := Hsla{2.0 / 3.0, 1, 0.5, 1}
	if got := a.Mix(b, 0.5); !hslaEq(got, Hsla{1.0 / 3.0, 1, 0.5, 1}) {
		t.Errorf("Mix(t=0.5) = %v, want {1/3, 1, 0.5, 1}", got)
	}
}
