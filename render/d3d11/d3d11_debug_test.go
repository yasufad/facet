//go:build windows && facet_debug

package d3d11_test

import (
	"math"
	"runtime"
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

// 5. PolychromeSprite: upload a two-colour image and assert both colours land in the right halves.
func TestRenderPolychromeSprite(t *testing.T) {
	width, height := 200, 200
	p, w, r, scale := setupTestWindow(t, "PolySprite", width, height)
	defer w.Close()
	defer r.Close()

	p.Dispatch(func() {
		defer p.Quit()

		// 16x16 polychrome texture: left 8 columns = red, right 8 columns = blue
		polyData := make([]byte, 16*16*4)
		for y := 0; y < 16; y++ {
			for x := 0; x < 16; x++ {
				idx := (y*16 + x) * 4
				if x < 8 {
					// Red (R=255, G=0, B=0, A=255)
					polyData[idx+0] = 0xff
					polyData[idx+1] = 0x00
					polyData[idx+2] = 0x00
					polyData[idx+3] = 0xff
				} else {
					// Blue (R=0, G=0, B=255, A=255)
					polyData[idx+0] = 0x00
					polyData[idx+1] = 0x00
					polyData[idx+2] = 0xff
					polyData[idx+3] = 0xff
				}
			}
		}

		tile, err := r.Upload(scene.TexturePolychrome, geometry.NewSize[geometry.DevicePixels](16, 16), polyData)
		if err != nil {
			t.Fatalf("r.Upload polychrome: %v", err)
		}

		sc := scene.New()
		sc.InsertPolychromeSprite(scene.PolychromeSprite{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(40*scale), geometry.ScaledPixels(40*scale)),
				geometry.NewSize(geometry.ScaledPixels(16*scale), geometry.ScaledPixels(16*scale)),
			),
			Tile:    tile,
			Opacity: 1.0,
		})
		sc.Finish()

		if err := r.Draw(sc); err != nil {
			t.Fatalf("r.Draw: %v", err)
		}

		pixels, err := d3d11.ReadBackbuffer(r)
		if err != nil {
			t.Fatalf("ReadBackbuffer: %v", err)
		}

		red := colour.Rgba{R: 1.0, G: 0.0, B: 0.0, A: 1.0}
		blue := colour.Rgba{R: 0.0, G: 0.0, B: 1.0, A: 1.0}

		// Left half: pixel at (44, 48) is red
		leftX, leftY := int(44*scale), int(48*scale)
		leftGot := pixels[leftY][leftX]
		if !coloursMatch(leftGot, red, 0.05) {
			t.Errorf("left half pixel (%d, %d): got %v, want %v (red)", leftX, leftY, leftGot, red)
		}

		// Right half: pixel at (52, 48) is blue
		rightX, rightY := int(52*scale), int(48*scale)
		rightGot := pixels[rightY][rightX]
		if !coloursMatch(rightGot, blue, 0.05) {
			t.Errorf("right half pixel (%d, %d): got %v, want %v (blue)", rightX, rightY, rightGot, blue)
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}

// 6. Shadow: a shadow under an opaque quad with smooth Gaussian blur falloff.
func TestRenderShadowBlurFalloff(t *testing.T) {
	width, height := 200, 200
	p, w, r, scale := setupTestWindow(t, "ShadowFalloff", width, height)
	defer w.Close()
	defer r.Close()

	p.Dispatch(func() {
		defer p.Quit()

		sc := scene.New()
		// Shadow bounds: (40, 40) to (160, 160) with blur radius 16
		sc.InsertShadow(scene.Shadow{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(40*scale), geometry.ScaledPixels(40*scale)),
				geometry.NewSize(geometry.ScaledPixels(120*scale), geometry.ScaledPixels(120*scale)),
			),
			CornerRadii: geometry.AllCorners(geometry.ScaledPixels(0)),
			Colour:      colour.Rgba{R: 0.0, G: 0.0, B: 0.0, A: 1.0},
			BlurRadius:  geometry.ScaledPixels(16 * scale),
			Inset:       false,
		})
		// Opaque white quad over shadow: (60, 60) to (140, 140)
		sc.InsertQuad(scene.Quad{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(60*scale), geometry.ScaledPixels(60*scale)),
				geometry.NewSize(geometry.ScaledPixels(80*scale), geometry.ScaledPixels(80*scale)),
			),
			Background: colour.Rgba{R: 1.0, G: 1.0, B: 1.0, A: 1.0},
		})
		sc.Finish()

		if err := r.Draw(sc); err != nil {
			t.Fatalf("r.Draw: %v", err)
		}

		pixels, err := d3d11.ReadBackbuffer(r)
		if err != nil {
			t.Fatalf("ReadBackbuffer: %v", err)
		}

		// 1. Inside quad (100, 100): white quad
		white := colour.Rgba{R: 1.0, G: 1.0, B: 1.0, A: 1.0}
		centerGot := pixels[int(100*scale)][int(100*scale)]
		if !coloursMatch(centerGot, white, 0.05) {
			t.Errorf("center quad pixel: got %v, want %v (white)", centerGot, white)
		}

		// 2. Just outside quad edge (55, 100): deep in shadow bounds -> high shadow alpha (> 0.8)
		nearShadow := pixels[int(100*scale)][int(55*scale)]
		if nearShadow.A < 0.8 {
			t.Errorf("near shadow pixel (55, 100): alpha %v too low, want >= 0.8", nearShadow.A)
		}

		// 3. At shadow outer boundary (40, 100): partial blur falloff (0.1 <= alpha <= 0.8)
		edgeShadow := pixels[int(100*scale)][int(40*scale)]
		if edgeShadow.A < 0.1 || edgeShadow.A > 0.85 {
			t.Errorf("edge shadow pixel (40, 100): alpha %v want partial coverage (0.1 to 0.85)", edgeShadow.A)
		}

		// 4. Well outside shadow (15, 100): background alpha near 0
		clearBg := colour.Rgba{R: 0.0, G: 0.0, B: 0.0, A: 0.0}
		outShadow := pixels[int(100*scale)][int(15*scale)]
		if !coloursMatch(outShadow, clearBg, 0.05) {
			t.Errorf("outside shadow pixel (15, 100): got %v, want %v (background)", outShadow, clearBg)
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}

// 7. Underline: a straight underline and a wavy underline showing wave offset.
func TestRenderUnderlineStraightAndWavy(t *testing.T) {
	width, height := 200, 200
	p, w, r, scale := setupTestWindow(t, "UnderlineStraightWavy", width, height)
	defer w.Close()
	defer r.Close()

	p.Dispatch(func() {
		defer p.Quit()

		white := colour.Rgba{R: 1.0, G: 1.0, B: 1.0, A: 1.0}
		clearBg := colour.Rgba{R: 0.0, G: 0.0, B: 0.0, A: 0.0}

		// Draw a straight underline from (40, 40) to (160, 60), center at y=50, thickness=4
		scStraight := scene.New()
		scStraight.InsertUnderline(scene.Underline{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(40*scale), geometry.ScaledPixels(40*scale)),
				geometry.NewSize(geometry.ScaledPixels(120*scale), geometry.ScaledPixels(20*scale)),
			),
			Colour:    white,
			Thickness: geometry.ScaledPixels(4 * scale),
			Wavy:      false,
		})
		scStraight.Finish()

		if err := r.Draw(scStraight); err != nil {
			t.Fatalf("r.Draw straight: %v", err)
		}

		pixelsStraight, err := d3d11.ReadBackbuffer(r)
		if err != nil {
			t.Fatalf("ReadBackbuffer straight: %v", err)
		}

		// On straight underline at (100, 50) -> white
		onStraight := pixelsStraight[int(50*scale)][int(100*scale)]
		if !coloursMatch(onStraight, white, 0.05) {
			t.Errorf("on straight underline (100, 50): got %v, want %v", onStraight, white)
		}

		// Above straight underline at (100, 43) -> background
		aboveStraight := pixelsStraight[int(43*scale)][int(100*scale)]
		if !coloursMatch(aboveStraight, clearBg, 0.05) {
			t.Errorf("above straight underline (100, 43): got %v, want %v", aboveStraight, clearBg)
		}

		// Now draw a wavy underline in a fresh scene
		scWavy := scene.New()
		scWavy.InsertUnderline(scene.Underline{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(40*scale), geometry.ScaledPixels(40*scale)),
				geometry.NewSize(geometry.ScaledPixels(120*scale), geometry.ScaledPixels(20*scale)),
			),
			Colour:    white,
			Thickness: geometry.ScaledPixels(4 * scale),
			Wavy:      true,
		})
		scWavy.Finish()

		if err := r.Draw(scWavy); err != nil {
			t.Fatalf("r.Draw wavy: %v", err)
		}

		pixelsWavy, err := d3d11.ReadBackbuffer(r)
		if err != nil {
			t.Fatalf("ReadBackbuffer wavy: %v", err)
		}

		// Period = 16. At x = 44 (offset 4, sin = +1), wave crests towards y=51.6.
		// At y = 52, wavy underline has full coverage (alpha > 0.8).
		wavyOn := pixelsWavy[int(52*scale)][int(44*scale)]
		if wavyOn.A < 0.8 {
			t.Errorf("on wavy underline crest (44, 52): alpha %v too low, want >= 0.8", wavyOn.A)
		}

		// At x = 44, y = 48 (above the wave crest): wavy underline is background.
		wavyOff := pixelsWavy[int(48*scale)][int(44*scale)]
		if !coloursMatch(wavyOff, clearBg, 0.1) {
			t.Errorf("off wavy underline (44, 48): got %v, want %v (background)", wavyOff, clearBg)
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}

// TestDrawAllocationsPerFrame reports the heap allocations a real Draw call
// makes over a text-heavy-sized scene, so docs/audit.md item 1 — the
// make([]quadInstance, ...) plus a second copy into the mapped buffer, once
// per batch — has a measured before/after rather than an estimate. Before
// this change, each of these batches allocated its own instance slice;
// after, drawQuadBatch writes straight into the mapped GPU region and the
// remaining allocations are the pre-existing COM call marshalling in
// comObject.call, unrelated to the batching this task scoped.
func TestDrawAllocationsPerFrame(t *testing.T) {
	width, height := 400, 400
	p, w, r, scale := setupTestWindow(t, "DrawAllocs", width, height)
	defer w.Close()
	defer r.Close()

	p.Dispatch(func() {
		defer p.Quit()

		// Chain quad and underline bounds so each one overlaps the last:
		// the scene's spatial tree then assigns strictly increasing draw
		// order to every primitive, so none of them can merge into a
		// neighbour's batch. batchPairs pairs of (quad, underline) forces
		// 2*batchPairs distinct batches in one frame — the shape a
		// text-heavy frame produces, where the old code allocated a fresh
		// instance slice per batch.
		const batchPairs = 250
		white := colour.Rgba{R: 1, G: 1, B: 1, A: 1}
		sc := scene.New()
		for i := 0; i < batchPairs; i++ {
			x := geometry.ScaledPixels(float32(i) * scale)
			bounds := geometry.NewBounds(
				geometry.NewPoint(x, geometry.ScaledPixels(0)),
				geometry.NewSize(geometry.ScaledPixels(2*scale), geometry.ScaledPixels(2*scale)),
			)
			sc.InsertQuad(scene.Quad{Bounds: bounds, Background: white})
			sc.InsertUnderline(scene.Underline{Bounds: bounds, Colour: white, Thickness: geometry.ScaledPixels(scale)})
		}
		sc.Finish()

		batchCount := 0
		for range sc.Batches() {
			batchCount++
		}
		if batchCount != 2*batchPairs {
			t.Fatalf("expected %d batches from non-mergeable chained bounds, got %d", 2*batchPairs, batchCount)
		}

		// Warm-up draw so one-time buffer creation doesn't pollute the count.
		if err := r.Draw(sc); err != nil {
			t.Fatalf("warm-up r.Draw: %v", err)
		}

		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		const iterations = 50
		for i := 0; i < iterations; i++ {
			if err := r.Draw(sc); err != nil {
				t.Fatalf("r.Draw: %v", err)
			}
		}
		runtime.ReadMemStats(&after)

		total := after.Mallocs - before.Mallocs
		t.Logf("%d Draw calls over a %d-batch scene: %d heap allocations total, %.2f per Draw, %.3f per batch", iterations, batchCount, total, float64(total)/iterations, float64(total)/float64(iterations*batchCount))
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}

// 8b. Two quad batches with a path batch drawn between them, all sharing a
// dynamic buffer written with a rolling NO_OVERWRITE offset (see
// docs/audit.md item 2). Each primitive's bounds overlap its neighbour's so
// the scene assigns them strictly increasing draw order and the batcher
// cannot merge the two quad runs into one, forcing three separate calls to
// dynamicBuffer.write in a single frame. A batch that reads the wrong
// start-instance offset draws the wrong primitive's data at its own
// location — a legal draw producing plausible-looking pixels rather than an
// error — so this checks a point exclusive to each of the three batches
// fails on its own if that batch's draw call used a stale or wrong offset.
func TestRenderMultiBatchSameKindInterleaved(t *testing.T) {
	width, height := 200, 200
	p, w, r, scale := setupTestWindow(t, "MultiBatchInterleaved", width, height)
	defer w.Close()
	defer r.Close()

	p.Dispatch(func() {
		defer p.Quit()

		red := colour.Rgba{R: 1.0, G: 0.0, B: 0.0, A: 1.0}
		green := colour.Rgba{R: 0.0, G: 1.0, B: 0.0, A: 1.0}
		blue := colour.Rgba{R: 0.0, G: 0.0, B: 1.0, A: 1.0}

		sc := scene.New()
		// Quad A: batch 1, x in [0, 70).
		sc.InsertQuad(scene.Quad{
			Bounds: geometry.NewBounds(
				geometry.NewPoint[geometry.ScaledPixels](0, 0),
				geometry.NewSize(geometry.ScaledPixels(70*scale), geometry.ScaledPixels(200*scale)),
			),
			Background: red,
		})
		// Triangle: batch 2, spans x in [50, 150), overlapping both quads so
		// the scene orders it strictly between them.
		sc.InsertPath(scene.Path[geometry.ScaledPixels]{
			Bounds: geometry.NewBounds(
				geometry.NewPoint[geometry.ScaledPixels](0, 0),
				geometry.NewSize(geometry.ScaledPixels(float32(width)*scale), geometry.ScaledPixels(float32(height)*scale)),
			),
			Colour: green,
			Vertices: []scene.PathVertex[geometry.ScaledPixels]{
				{XYPosition: geometry.NewPoint(geometry.ScaledPixels(50*scale), geometry.ScaledPixels(0*scale))},
				{XYPosition: geometry.NewPoint(geometry.ScaledPixels(150*scale), geometry.ScaledPixels(0*scale))},
				{XYPosition: geometry.NewPoint(geometry.ScaledPixels(100*scale), geometry.ScaledPixels(200*scale))},
			},
		})
		// Quad B: batch 3, x in [130, 200), overlapping the triangle so it
		// sorts after it and cannot merge with quad A's batch.
		sc.InsertQuad(scene.Quad{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(130*scale), geometry.ScaledPixels(0)),
				geometry.NewSize(geometry.ScaledPixels(70*scale), geometry.ScaledPixels(200*scale)),
			),
			Background: blue,
		})
		sc.Finish()

		batchKinds := make([]scene.BatchKind, 0, 3)
		for b := range sc.Batches() {
			batchKinds = append(batchKinds, b.Kind)
		}
		want := []scene.BatchKind{scene.BatchQuads, scene.BatchPaths, scene.BatchQuads}
		if len(batchKinds) != len(want) {
			t.Fatalf("expected %d batches (quad, path, quad), got %d: %v", len(want), len(batchKinds), batchKinds)
		}
		for i := range want {
			if batchKinds[i] != want[i] {
				t.Fatalf("batch %d: got kind %v, want %v (full sequence %v)", i, batchKinds[i], want[i], batchKinds)
			}
		}

		if err := r.Draw(sc); err != nil {
			t.Fatalf("r.Draw: %v", err)
		}

		pixels, err := d3d11.ReadBackbuffer(r)
		if err != nil {
			t.Fatalf("ReadBackbuffer: %v", err)
		}

		// Batch 1 (quad A), exclusive of the triangle: x = 20.
		gotA := pixels[int(100*scale)][int(20*scale)]
		if !coloursMatch(gotA, red, 0.05) {
			t.Errorf("quad A region (20, 100): got %v, want %v (batch 1 drew at the wrong offset)", gotA, red)
		}

		// Batch 2 (triangle), exclusive of both quads: x = 100, near the
		// base where the triangle is widest.
		gotPath := pixels[int(20*scale)][int(100*scale)]
		if !coloursMatch(gotPath, green, 0.05) {
			t.Errorf("triangle region (100, 20): got %v, want %v (batch 2 drew at the wrong offset)", gotPath, green)
		}

		// Batch 3 (quad B), exclusive of the triangle: x = 180.
		gotB := pixels[int(100*scale)][int(180*scale)]
		if !coloursMatch(gotB, blue, 0.05) {
			t.Errorf("quad B region (180, 100): got %v, want %v (batch 3 drew at the wrong offset)", gotB, blue)
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}

// 8. Path rasterisation: verifies geometry rendered through Draw.
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
