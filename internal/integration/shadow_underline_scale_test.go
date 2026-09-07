//go:build windows && facet_debug

package integration

import (
	"errors"
	"testing"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/render"
	"github.com/yasufad/facet/render/d3d11"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/window"
)

// readbackRenderer wraps a real D3D11 renderer so the integration test can
// drive the full window frame loop (which calls Draw then Present) and still
// read the backbuffer afterwards. Present swaps the backbuffer out, leaving
// undefined content behind; the wrapper captures the last scene Draw saw so
// the test can re-render it into the fresh backbuffer and read those pixels.
type readbackRenderer struct {
	inner     render.Renderer
	lastScene *scene.Scene
}

func (r *readbackRenderer) Resize(size geometry.Size[geometry.DevicePixels]) error {
	return r.inner.Resize(size)
}

func (r *readbackRenderer) Draw(s *scene.Scene) error {
	r.lastScene = s
	return r.inner.Draw(s)
}

func (r *readbackRenderer) Present() error {
	return r.inner.Present()
}

func (r *readbackRenderer) Upload(kind scene.AtlasTextureKind, size geometry.Size[geometry.DevicePixels], data []byte) (scene.AtlasTile, error) {
	return r.inner.Upload(kind, size, data)
}

func (r *readbackRenderer) ClearAtlas(kind scene.AtlasTextureKind) {
	r.inner.ClearAtlas(kind)
}

func (r *readbackRenderer) Size() geometry.Size[geometry.DevicePixels] {
	return r.inner.Size()
}

func (r *readbackRenderer) Close() error {
	return r.inner.Close()
}

// TestShadowAndUnderlineScaleInterpretation is a joint integration test
// verifying that element's BoxShadow and Underline primitives are
// interpreted at the correct device-pixel coordinates when the scale factor
// is not 1.
//
// The insert-level tests in element prove the primitives are emitted with
// the right logical-pixel values. render's readback tests prove the shaders
// draw a synthetic scene.Shadow and scene.Underline correctly. Neither
// covers the join between them: whether the blur radius, offsets, bounds
// and thickness element puts in the struct — after multiplying by the scale
// factor — mean the same thing to the shader that they mean to element. A
// logical-versus-device-pixel mistake in the scale conversion is invisible
// at scale 1, which is exactly why it would survive undetected.
//
// The test renders a Div with a drop BoxShadow and a Text with an Underline
// through the real window frame loop at the system's scale factor, reads
// the backbuffer, and checks two things:
//
//  1. Shadow: a device pixel that is inside the unscaled shadow bounds but
//     outside the scaled shadow bounds is transparent. If the scale
//     conversion is missing, that pixel has shadow alpha instead.
//  2. Underline: no red pixels appear in the unscaled y range of the text
//     Div, and red pixels do appear in the scaled y range. If the scale
//     conversion is missing, the underline lands in the unscaled range.
//
// It skips on render.ErrNoAdapter and nothing broader: a hosted runner with
// no GPU cannot run the test, but a genuine device failure must surface as
// a failure, not a silent skip. It also skips when the scale factor is
// ≤ 1.2, because below that the scaled and unscaled underline positions
// are too close to distinguish reliably — at scale 1 the bug is invisible
// by construction.
func TestShadowAndUnderlineScaleInterpretation(t *testing.T) {
	const winW, winH = 200, 200

	p, err := platform.New(platform.Options{Name: "facet-scale-test"})
	if err != nil {
		t.Fatalf("platform.New: %v", err)
	}

	pw, err := p.NewWindow(platform.WindowOptions{
		Title:     "ScaleTest",
		Size:      geometry.NewSize[geometry.Pixels](winW, winH),
		Visible:   false,
		Resizable: false,
		Decorated: false,
	})
	if err != nil {
		t.Fatalf("NewWindow: %v", err)
	}

	scale := pw.ScaleFactor()
	if scale <= 1.2 {
		pw.Close()
		t.Skipf("scale factor %.2f; test requires > 1.2 to distinguish scaled from unscaled positions", scale)
	}

	devSize := geometry.NewSize(
		geometry.Pixels(winW).ToDevicePixels(scale),
		geometry.Pixels(winH).ToDevicePixels(scale),
	)

	inner, err := d3d11.New(pw.NativeSurface(), devSize, render.Options{VSync: false})
	if err != nil {
		pw.Close()
		if errors.Is(err, render.ErrNoAdapter) {
			t.Skipf("d3d11.New: %v", err)
		}
		t.Fatalf("d3d11.New: %v", err)
	}

	r := &readbackRenderer{inner: inner}

	p.Dispatch(func() {
		defer p.Quit()
		defer r.Close()
		defer pw.Close()

		a := app.NewApp()
		defer a.Close()

		w := window.NewWithRenderer(pw, r, a, window.WindowOptions{
			Size: geometry.NewSize[geometry.Pixels](winW, winH),
		})

		// Layout (logical pixels):
		//   Root Div: fills 200×200, flex column, padding 40, row gap 70
		//     Shadow Div: 120×60 at (40, 40) — white bg, drop shadow blur 16
		//     Text Div:   120×20 at (40, 170) — underlined text, transparent glyphs
		//
		// Shadow Div bounds:      (40, 40) to (160, 100)
		// Shadow primitive bounds (dilated by blur 16): (24, 24) to (176, 116)
		//
		// Text Div bounds: (40, 170) to (160, 190)
		// The underline sits within the text Div's logical y range [170, 190].
		// At scale s the device y range is [170*s, 190*s].
		// Without scale conversion the device y would be [170, 190].
		// For s > 190/170 ≈ 1.12 the two ranges are disjoint, so the
		// underline's presence in one but not the other distinguishes
		// correct from broken scaling. The skip threshold of 1.2 gives
		// margin above that boundary.

		black := colour.Rgba{A: 1}
		white := colour.Rgba{R: 1, G: 1, B: 1, A: 1}
		red := colour.Rgba{R: 1, A: 1}
		// Transparent text colour makes glyphs invisible so the only red
		// pixels in the frame come from the underline.
		transparent := colour.Rgba{}

		root := element.NewDiv().
			FlexCol().
			Padding(style.Px(40)).
			GapRow(style.Px(70)).
			Child(
				element.NewDiv().
					Width(style.Px(120)).
					Height(style.Px(60)).
					Bg(white).
					BoxShadow([]style.BoxShadow{
						style.Shadow(0, 0, 16, 0, black),
					}),
			).
			Child(
				element.NewDiv().
					Width(style.Px(120)).
					Height(style.Px(20)).
					Child(
						element.NewText("AAAAAAAAAA").
							FontSize(16).
							TextColour(transparent).
							Underline(style.UnderlineStyle{
								Thickness: 4,
								Colour:    red,
							}),
					),
			)

		w.SetRoot(root)
		w.Draw()

		// w.Draw() called Present(), which swapped the backbuffer out.
		// Re-render the captured scene into the fresh backbuffer so
		// ReadBackbuffer sees the pixels the frame produced.
		if r.lastScene == nil {
			t.Fatal("no scene captured during Draw")
		}
		if err := inner.Draw(r.lastScene); err != nil {
			t.Fatalf("inner.Draw: %v", err)
		}

		pixels, err := d3d11.ReadBackbuffer(inner)
		if err != nil {
			t.Fatalf("ReadBackbuffer: %v", err)
		}

		devH := len(pixels)
		devW := 0
		if devH > 0 {
			devW = len(pixels[0])
		}

		// --- Sanity: the white div is rendered (frame is not empty) ---
		whiteX := int(100 * scale)
		whiteY := int(70 * scale)
		if whiteY >= devH || whiteX >= devW {
			t.Fatalf("white div pixel (%d, %d) out of bounds %dx%d", whiteX, whiteY, devW, devH)
		}
		if !coloursMatch(pixels[whiteY][whiteX], white, 0.1) {
			t.Fatalf("white div pixel (%d, %d): got %v, want white — the frame did not render the background quad", whiteX, whiteY, pixels[whiteY][whiteX])
		}

		// --- Shadow scale interpretation ---
		//
		// Shadow primitive bounds: (24, 24) to (176, 116) logical.
		// Scaled device bounds:     (24*s, 24*s) to (176*s, 116*s).
		// Unscaled device bounds:   (24, 24) to (176, 116).
		//
		// A pixel at device x = int(24*s)−2, y = 60 is:
		//   outside the scaled shadow (x < 24*s)         → transparent
		//   inside the unscaled shadow (24 < x < 176,
		//     24 < 60 < 116)                             → shadow alpha
		// If the scale conversion is missing on the shadow primitive, this
		// pixel has shadow alpha instead of being transparent.
		shadowEdgeX := int(24*scale) - 2
		shadowEdgeY := 60
		if shadowEdgeY >= devH || shadowEdgeX >= devW || shadowEdgeX < 0 {
			t.Fatalf("shadow edge pixel (%d, %d) out of bounds %dx%d", shadowEdgeX, shadowEdgeY, devW, devH)
		}
		edgePixel := pixels[shadowEdgeY][shadowEdgeX]
		if edgePixel.A > 0.15 {
			t.Errorf("shadow edge pixel (%d, %d): got alpha %.3f, want ~0 (outside scaled shadow bounds); non-zero alpha means the shadow bounds were not scaled to device pixels", shadowEdgeX, shadowEdgeY, edgePixel.A)
		}

		// Verify the shadow IS present at the scaled position: a pixel at
		// (30*s, 30*s) is inside the scaled shadow but outside the white
		// div (30 < 40 in logical x, so 30*s < 40*s in device x).
		shadowInnerX := int(30 * scale)
		shadowInnerY := int(30 * scale)
		if shadowInnerY >= devH || shadowInnerX >= devW {
			t.Fatalf("shadow inner pixel (%d, %d) out of bounds %dx%d", shadowInnerX, shadowInnerY, devW, devH)
		}
		innerPixel := pixels[shadowInnerY][shadowInnerX]
		if innerPixel.A < 0.1 {
			t.Errorf("shadow inner pixel (%d, %d): got alpha %.3f, want > 0.1 (inside scaled shadow, outside white div)", shadowInnerX, shadowInnerY, innerPixel.A)
		}

		// --- Underline scale interpretation ---
		//
		// Text Div at logical (40, 170) size (120, 20).
		// Device bounds at scale s: (40*s, 170*s) to (160*s, 190*s).
		// The underline is within the text Div's logical y range [170, 190].
		// At the correct scale the device y is in [170*s, 190*s].
		// Without scale conversion it would be in [170, 190].
		// For s > 1.2, 170*s > 204 > 190, so the ranges are disjoint.

		// Check 1: no red pixels in the unscaled y range [170, 190].
		// If the scale conversion is missing on the underline primitive,
		// the underline lands here and this check fails.
		unscaledTop := 170
		unscaledBottom := 190
		if unscaledBottom >= devH {
			unscaledBottom = devH - 1
		}
		unscaledRangeHasRed := false
		for y := unscaledTop; y <= unscaledBottom && !unscaledRangeHasRed; y++ {
			for x := 0; x < devW; x++ {
				if isRedPixel(pixels[y][x]) {
					unscaledRangeHasRed = true
					t.Errorf("red pixel at (%d, %d) in unscaled underline y range [170, 190]; the underline should be at scaled y ≥ %d, not at unscaled y", x, y, int(170*scale))
					break
				}
			}
		}

		// Check 2: red pixels exist in the scaled y range.
		// This verifies the underline is rendered at the scaled position.
		scaledTop := int(170 * scale)
		scaledBottom := int(190 * scale)
		if scaledBottom >= devH {
			scaledBottom = devH - 1
		}
		foundUnderline := false
		for y := scaledTop; y <= scaledBottom && y < devH; y++ {
			for x := 0; x < devW; x++ {
				if isRedPixel(pixels[y][x]) {
					foundUnderline = true
					break
				}
			}
			if foundUnderline {
				break
			}
		}
		if !foundUnderline {
			t.Errorf("no red underline pixels found in scaled y range [%d, %d]; the underline should be at the scaled position", scaledTop, scaledBottom)
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}

// isRedPixel returns true for a pixel that is predominantly red and
// non-transparent. The underline is the only red source in the frame
// (text colour is transparent, shadow is black, div background is white).
func isRedPixel(c colour.Rgba) bool {
	return c.R > 0.5 && c.G < 0.3 && c.B < 0.3 && c.A > 0.3
}

// coloursMatch returns true when each component of a and b is within tol.
func coloursMatch(a, b colour.Rgba, tol float32) bool {
	diff := func(x, y float32) float32 {
		if x > y {
			return x - y
		}
		return y - x
	}
	return diff(a.R, b.R) <= tol &&
		diff(a.G, b.G) <= tol &&
		diff(a.B, b.B) <= tol &&
		diff(a.A, b.A) <= tol
}
