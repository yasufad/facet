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

func TestRenderTextWindowPixelAssertion(t *testing.T) {
	p, err := platform.New(platform.Options{Name: "facet-text-pixel-test"})
	if err != nil {
		t.Fatalf("platform.New: %v", err)
	}

	a := app.NewApp()
	defer a.Close()

	w, err := window.New(p, a, window.WindowOptions{
		Title:       "TextPixelAssertion",
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

	black := colour.Rgba{R: 0.0, G: 0.0, B: 0.0, A: 1.0}
	white := colour.Rgba{R: 1.0, G: 1.0, B: 1.0, A: 1.0}

	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Width(style.Px(200)).
			Height(style.Px(200)).
			Bg(black).
			Child(
				element.NewText("A").
					FontSize(64).
					TextColour(white),
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
		h := len(pixels)
		wPx := len(pixels[0])

		// Search within the 0..100 region where glyph "A" is placed for non-black pixel coverage.
		var maxCoverage float32
		searchMaxX := int(100.0 * scale)
		if searchMaxX > wPx {
			searchMaxX = wPx
		}
		searchMaxY := int(100.0 * scale)
		if searchMaxY > h {
			searchMaxY = h
		}

		for y := 0; y < searchMaxY; y++ {
			for x := 0; x < searchMaxX; x++ {
				px := pixels[y][x]
				if px.R > maxCoverage {
					maxCoverage = px.R
				}
			}
		}

		if maxCoverage < 0.5 {
			t.Fatalf("expected glyph raster coverage inside text bounding box, max coverage was %v", maxCoverage)
		}

		// Check outside text region: bottom-right corner must be background black.
		outsideX := int(180.0 * scale)
		outsideY := int(180.0 * scale)
		gotOutside := pixels[outsideY][outsideX]
		black := colour.Rgba{R: 0.0, G: 0.0, B: 0.0, A: 1.0}
		if !coloursMatch(gotOutside, black, 0.05) {
			t.Errorf("outside text pixel (%d, %d): got %v, want %v (black)", outsideX, outsideY, gotOutside, black)
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}

type unbalancingClipElement struct{}

func (u *unbalancingClipElement) RequestLayout(f element.Frame) element.NodeID {
	return f.RequestLayout(style.Default().ToLayout(f.RemSize()), nil)
}

func (u *unbalancingClipElement) Prepaint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {}

func (u *unbalancingClipElement) Paint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {
	f.PushClip(bounds)
	// deliberately omit f.PopClip()
}

func TestUnbalancedClipStackPanicsUnderDebug(t *testing.T) {
	p, err := platform.New(platform.Options{Name: "facet-clip-debug-test"})
	if err != nil {
		t.Fatalf("platform.New: %v", err)
	}

	a := app.NewApp()
	defer a.Close()

	w, err := window.New(p, a, window.WindowOptions{
		Title:   "UnbalancedClip",
		Size:    geometry.NewSize[geometry.Pixels](200, 200),
		Visible: false,
	})
	if err != nil {
		t.Fatalf("window.New: %v", err)
	}
	defer w.Close()

	panicked := false
	p.Dispatch(func() {
		defer p.Quit()
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()

		w.SetRootFn(func() element.Element {
			return &unbalancingClipElement{}
		})
		w.Draw()
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}

	if !panicked {
		t.Fatalf("expected panic on unbalanced clip stack at end of paint under facet_debug")
	}
}
