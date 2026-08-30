//go:build windows && facet_debug

package d3d11_test

import (
	"math"
	"testing"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/render"
	"github.com/yasufad/facet/render/d3d11"
	"github.com/yasufad/facet/scene"
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

func setupTestWindow(t *testing.T, title string, width, height int) (platform.Platform, platform.Window, render.Renderer, float32) {
	p, err := platform.New(platform.Options{Name: "facet-render-test"})
	if err != nil {
		t.Fatalf("platform.New: %v", err)
	}

	w, err := p.NewWindow(platform.WindowOptions{
		Title:     title,
		Size:      geometry.NewSize[geometry.Pixels](geometry.Pixels(width), geometry.Pixels(height)),
		Visible:   false,
		Resizable: false,
		Decorated: false,
	})
	if err != nil {
		t.Fatalf("NewWindow: %v", err)
	}

	scale := w.ScaleFactor()
	devSize := geometry.NewSize(
		geometry.Pixels(width).ToDevicePixels(scale),
		geometry.Pixels(height).ToDevicePixels(scale),
	)

	r, err := d3d11.New(w.NativeSurface(), devSize, render.Options{VSync: false})
	if err != nil {
		w.Close()
		t.Fatalf("d3d11.New: %v", err)
	}

	return p, w, r, scale
}

// 1. A full-window quad of a known colour makes the centre pixel that colour.
func TestRenderFullWindowQuad(t *testing.T) {
	width, height := 200, 200
	p, w, r, scale := setupTestWindow(t, "FullWindowQuad", width, height)
	defer w.Close()
	defer r.Close()

	p.Dispatch(func() {
		defer p.Quit()

		magenta := colour.Rgba{R: 1.0, G: 0.0, B: 1.0, A: 1.0}
		sc := scene.New()
		sc.InsertQuad(scene.Quad{
			Bounds: geometry.NewBounds(
				geometry.NewPoint[geometry.ScaledPixels](0, 0),
				geometry.NewSize(geometry.ScaledPixels(float32(width)*scale), geometry.ScaledPixels(float32(height)*scale)),
			),
			Background: magenta,
		})
		sc.Finish()

		if err := r.Draw(sc); err != nil {
			t.Fatalf("r.Draw: %v", err)
		}

		pixels, err := d3d11.ReadBackbuffer(r)
		if err != nil {
			t.Fatalf("ReadBackbuffer: %v", err)
		}

		centerX := int(float32(width) * scale / 2)
		centerY := int(float32(height) * scale / 2)
		got := pixels[centerY][centerX]
		if !coloursMatch(got, magenta, 0.05) {
			t.Errorf("center pixel (%d, %d): got %v, want %v", centerX, centerY, got, magenta)
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}

// 2. A quad with a corner radius leaves the corner pixel as the background and the centre as the fill.
func TestRenderQuadWithCornerRadius(t *testing.T) {
	width, height := 200, 200
	p, w, r, scale := setupTestWindow(t, "CornerRadius", width, height)
	defer w.Close()
	defer r.Close()

	p.Dispatch(func() {
		defer p.Quit()

		red := colour.Rgba{R: 1.0, G: 0.0, B: 0.0, A: 1.0}
		clearBg := colour.Rgba{R: 0.0, G: 0.0, B: 0.0, A: 0.0}

		sc := scene.New()
		sc.InsertQuad(scene.Quad{
			Bounds: geometry.NewBounds(
				geometry.NewPoint[geometry.ScaledPixels](0, 0),
				geometry.NewSize(geometry.ScaledPixels(float32(width)*scale), geometry.ScaledPixels(float32(height)*scale)),
			),
			Background:  red,
			CornerRadii: geometry.AllCorners(geometry.ScaledPixels(50 * scale)),
		})
		sc.Finish()

		if err := r.Draw(sc); err != nil {
			t.Fatalf("r.Draw: %v", err)
		}

		pixels, err := d3d11.ReadBackbuffer(r)
		if err != nil {
			t.Fatalf("ReadBackbuffer: %v", err)
		}

		// Corner at (2, 2) should be untouched transparent background
		cornerGot := pixels[2][2]
		if !coloursMatch(cornerGot, clearBg, 0.05) {
			t.Errorf("corner pixel (2, 2): got %v, want %v (background)", cornerGot, clearBg)
		}

		// Center at (100, 100) should be fill red
		centerX := int(float32(width) * scale / 2)
		centerY := int(float32(height) * scale / 2)
		centerGot := pixels[centerY][centerX]
		if !coloursMatch(centerGot, red, 0.05) {
			t.Errorf("center pixel (%d, %d): got %v, want %v (fill)", centerX, centerY, centerGot, red)
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}

// 3. A red quad drawn over a blue one shows red where they overlap and blue where they do not.
func TestRenderOverlapDrawOrder(t *testing.T) {
	width, height := 200, 200
	p, w, r, scale := setupTestWindow(t, "OverlapDrawOrder", width, height)
	defer w.Close()
	defer r.Close()

	p.Dispatch(func() {
		defer p.Quit()

		blue := colour.Rgba{R: 0.0, G: 0.0, B: 1.0, A: 1.0}
		red := colour.Rgba{R: 1.0, G: 0.0, B: 0.0, A: 1.0}

		sc := scene.New()
		// Blue quad drawn first: (0, 0) to (140, 140)
		sc.InsertQuad(scene.Quad{
			Bounds: geometry.NewBounds(
				geometry.NewPoint[geometry.ScaledPixels](0, 0),
				geometry.NewSize(geometry.ScaledPixels(140*scale), geometry.ScaledPixels(140*scale)),
			),
			Background: blue,
		})
		// Red quad drawn second: (60, 60) to (200, 200)
		sc.InsertQuad(scene.Quad{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(60*scale), geometry.ScaledPixels(60*scale)),
				geometry.NewSize(geometry.ScaledPixels(140*scale), geometry.ScaledPixels(140*scale)),
			),
			Background: red,
		})
		sc.Finish()

		if err := r.Draw(sc); err != nil {
			t.Fatalf("r.Draw: %v", err)
		}

		pixels, err := d3d11.ReadBackbuffer(r)
		if err != nil {
			t.Fatalf("ReadBackbuffer: %v", err)
		}

		// Non-overlapping blue region: (20, 20)
		blueX, blueY := int(20*scale), int(20*scale)
		blueGot := pixels[blueY][blueX]
		if !coloursMatch(blueGot, blue, 0.05) {
			t.Errorf("blue region (%d, %d): got %v, want %v", blueX, blueY, blueGot, blue)
		}

		// Overlapping region: (100, 100) should be top (red)
		overlapX, overlapY := int(100*scale), int(100*scale)
		overlapGot := pixels[overlapY][overlapX]
		if !coloursMatch(overlapGot, red, 0.05) {
			t.Errorf("overlap region (%d, %d): got %v, want %v (top quad)", overlapX, overlapY, overlapGot, red)
		}

		// Non-overlapping red region: (180, 180)
		redX, redY := int(180*scale), int(180*scale)
		redGot := pixels[redY][redX]
		if !coloursMatch(redGot, red, 0.05) {
			t.Errorf("red region (%d, %d): got %v, want %v", redX, redY, redGot, red)
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}

// 4. A monochrome sprite uploaded with known coverage samples to the tint colour at full coverage and the background at zero.
func TestRenderMonochromeSpriteCoverage(t *testing.T) {
	width, height := 200, 200
	p, w, r, scale := setupTestWindow(t, "MonoSpriteCoverage", width, height)
	defer w.Close()
	defer r.Close()

	p.Dispatch(func() {
		defer p.Quit()

		// 16x16 monochrome texture: left 8 columns = 255 (full coverage), right 8 columns = 0 (zero coverage)
		monoData := make([]byte, 16*16)
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				if x < 8 {
					monoData[y*16+x] = 0xff
				} else {
					monoData[y*16+x] = 0x00
				}
			}
		}

		tile, err := r.Upload(scene.TextureMonochrome, geometry.NewSize[geometry.DevicePixels](16, 16), monoData)
		if err != nil {
			t.Fatalf("r.Upload: %v", err)
		}

		green := colour.Rgba{R: 0.0, G: 1.0, B: 0.0, A: 1.0}
		clearBg := colour.Rgba{R: 0.0, G: 0.0, B: 0.0, A: 0.0}

		sc := scene.New()
		sc.InsertMonochromeSprite(scene.MonochromeSprite{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(40*scale), geometry.ScaledPixels(40*scale)),
				geometry.NewSize(geometry.ScaledPixels(16*scale), geometry.ScaledPixels(16*scale)),
			),
			Tile:           tile,
			Colour:         green,
			Transformation: scene.IdentityMatrix,
		})
		sc.Finish()

		if err := r.Draw(sc); err != nil {
			t.Fatalf("r.Draw: %v", err)
		}

		pixels, err := d3d11.ReadBackbuffer(r)
		if err != nil {
			t.Fatalf("ReadBackbuffer: %v", err)
		}

		// Left half: pixel at (44, 48) is full coverage -> green
		leftX, leftY := int(44*scale), int(48*scale)
		leftGot := pixels[leftY][leftX]
		if !coloursMatch(leftGot, green, 0.05) {
			t.Errorf("full coverage pixel (%d, %d): got %v, want %v (tint)", leftX, leftY, leftGot, green)
		}

		// Right half: pixel at (52, 48) is zero coverage -> background
		rightX, rightY := int(52*scale), int(48*scale)
		rightGot := pixels[rightY][rightX]
		if !coloursMatch(rightGot, clearBg, 0.05) {
			t.Errorf("zero coverage pixel (%d, %d): got %v, want %v (background)", rightX, rightY, rightGot, clearBg)
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}

// 5. Test quad with borders: verifies fill colour vs border colour vs background.
func TestRenderQuadWithBorder(t *testing.T) {
	width, height := 200, 200
	p, w, r, scale := setupTestWindow(t, "QuadBorder", width, height)
	defer w.Close()
	defer r.Close()

	p.Dispatch(func() {
		defer p.Quit()

		blue := colour.Rgba{R: 0.0, G: 0.0, B: 1.0, A: 1.0}
		yellow := colour.Rgba{R: 1.0, G: 1.0, B: 0.0, A: 1.0}
		clearBg := colour.Rgba{R: 0.0, G: 0.0, B: 0.0, A: 0.0}

		sc := scene.New()
		sc.InsertQuad(scene.Quad{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(40*scale), geometry.ScaledPixels(40*scale)),
				geometry.NewSize(geometry.ScaledPixels(120*scale), geometry.ScaledPixels(120*scale)),
			),
			Background:   blue,
			BorderColour: yellow,
			BorderWidths: geometry.AllEdges(geometry.ScaledPixels(10 * scale)),
			BorderStyle:  scene.BorderSolid,
		})
		sc.Finish()

		if err := r.Draw(sc); err != nil {
			t.Fatalf("r.Draw: %v", err)
		}

		pixels, err := d3d11.ReadBackbuffer(r)
		if err != nil {
			t.Fatalf("ReadBackbuffer: %v", err)
		}

		// Centre (100, 100) -> blue fill
		centerX, centerY := int(100*scale), int(100*scale)
		centerGot := pixels[centerY][centerX]
		if !coloursMatch(centerGot, blue, 0.05) {
			t.Errorf("center fill (%d, %d): got %v, want %v", centerX, centerY, centerGot, blue)
		}

		// Border edge (45, 100) -> yellow border
		borderX, borderY := int(45*scale), int(100*scale)
		borderGot := pixels[borderY][borderX]
		if !coloursMatch(borderGot, yellow, 0.05) {
			t.Errorf("border edge (%d, %d): got %v, want %v", borderX, borderY, borderGot, yellow)
		}

		// Outside (20, 20) -> background
		outX, outY := int(20*scale), int(20*scale)
		outGot := pixels[outY][outX]
		if !coloursMatch(outGot, clearBg, 0.05) {
			t.Errorf("outside (%d, %d): got %v, want %v", outX, outY, outGot, clearBg)
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}

// 6. Test path rasterisation: verifies geometry rendered through Draw.
func TestRenderPathTriangle(t *testing.T) {
	width, height := 200, 200
	p, w, r, scale := setupTestWindow(t, "PathTriangle", width, height)
	defer w.Close()
	defer r.Close()

	p.Dispatch(func() {
		defer p.Quit()

		green := colour.Rgba{R: 0.0, G: 0.8, B: 0.0, A: 1.0}
		clearBg := colour.Rgba{R: 0.0, G: 0.0, B: 0.0, A: 0.0}

		sc := scene.New()
		sc.InsertPath(scene.Path[geometry.ScaledPixels]{
			Bounds: geometry.NewBounds(
				geometry.NewPoint[geometry.ScaledPixels](0, 0),
				geometry.NewSize(geometry.ScaledPixels(float32(width)*scale), geometry.ScaledPixels(float32(height)*scale)),
			),
			Colour: green,
			Vertices: []scene.PathVertex[geometry.ScaledPixels]{
				{XYPosition: geometry.NewPoint(geometry.ScaledPixels(20*scale), geometry.ScaledPixels(20*scale))},
				{XYPosition: geometry.NewPoint(geometry.ScaledPixels(180*scale), geometry.ScaledPixels(20*scale))},
				{XYPosition: geometry.NewPoint(geometry.ScaledPixels(100*scale), geometry.ScaledPixels(180*scale))},
			},
		})
		sc.Finish()

		if err := r.Draw(sc); err != nil {
			t.Fatalf("r.Draw: %v", err)
		}

		pixels, err := d3d11.ReadBackbuffer(r)
		if err != nil {
			t.Fatalf("ReadBackbuffer: %v", err)
		}

		// Centroid inside triangle (100, 60) -> green
		inX, inY := int(100*scale), int(60*scale)
		inGot := pixels[inY][inX]
		if !coloursMatch(inGot, green, 0.05) {
			t.Errorf("inside triangle (%d, %d): got %v, want %v", inX, inY, inGot, green)
		}

		// Point outside triangle (20, 180) -> background
		outX, outY := int(20*scale), int(180*scale)
		outGot := pixels[outY][outX]
		if !coloursMatch(outGot, clearBg, 0.05) {
			t.Errorf("outside triangle (%d, %d): got %v, want %v", outX, outY, outGot, clearBg)
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}
