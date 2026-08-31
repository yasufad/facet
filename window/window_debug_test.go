//go:build windows && facet_debug

package window_test

import (
	"math"
	"testing"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/render/d3d11"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/window"
)

func coloursMatch(a, b colour.Rgba, tol float32) bool {
	diff := func(x, y float32) float32 {
		return float32(math.Abs(float64(x - y)))
	}
	return diff(a.R, b.R) <= tol &&
		diff(a.G, b.G) <= tol &&
		diff(a.B, b.B) <= tol &&
		diff(a.A, b.A) <= tol
}

func TestRenderWindowPixelAssertion(t *testing.T) {
	p, err := platform.New(platform.Options{Name: "facet-window-pixel-test"})
	if err != nil {
		t.Fatalf("platform.New: %v", err)
	}

	a := app.NewApp()
	defer a.Close()

	w, err := window.New(p, a, window.WindowOptions{
		Title:       "PixelAssertion",
		Size:        geometry.NewSize[geometry.Pixels](200, 200),
		Visible:     false,
		Resizable:   false,
		Decorated:   false,
		Transparent: false,
		VSync:       false,
	})
	if err != nil {
		t.Fatalf("window.New: %v", err)
	}
	defer w.Close()

	blue := colour.Rgba{R: 0.0, G: 0.0, B: 1.0, A: 1.0}
	magenta := colour.Rgba{R: 1.0, G: 0.0, B: 1.0, A: 1.0}

	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Width(style.Px(200)).
			Height(style.Px(200)).
			Bg(blue).
			Child(
				element.NewDiv().
					Width(style.Px(100)).
					Height(style.Px(100)).
					Bg(magenta),
			)
	})

	p.Dispatch(func() {
		defer p.Quit()

		w.Draw()

		pixels, err := d3d11.ReadBackbuffer(w.Renderer())
		if err != nil {
			t.Fatalf("ReadBackbuffer: %v", err)
		}

		scale := w.ScaleFactor()

		// 1. Pixel inside child div (50, 50 in logical pixels) must be magenta.
		insideX := int(50.0 * scale)
		insideY := int(50.0 * scale)
		gotInside := pixels[insideY][insideX]
		if !coloursMatch(gotInside, magenta, 0.05) {
			t.Errorf("inside pixel (%d, %d): got %v, want %v (magenta)", insideX, insideY, gotInside, magenta)
		}

		// 2. Pixel outside child div (180, 180 in logical pixels) must be blue.
		outsideX := int(180.0 * scale)
		outsideY := int(180.0 * scale)
		gotOutside := pixels[outsideY][outsideX]
		if !coloursMatch(gotOutside, blue, 0.05) {
			t.Errorf("outside pixel (%d, %d): got %v, want %v (blue)", outsideX, outsideY, gotOutside, blue)
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}
