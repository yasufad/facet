//go:build windows && facet_debug

package d3d11_test

import (
	"testing"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/render"
	"github.com/yasufad/facet/render/d3d11"
	"github.com/yasufad/facet/scene"
)

func TestRendererSmoke(t *testing.T) {
	p, err := platform.New(platform.Options{Name: "facet-render-test"})
	if err != nil {
		t.Fatalf("platform.New: %v", err)
	}

	w, err := p.NewWindow(platform.WindowOptions{
		Title:     "Facet Render Test",
		Size:      geometry.NewSize[geometry.Pixels](400, 300),
		Visible:   false,
		Resizable: true,
		Decorated: true,
	})
	if err != nil {
		t.Fatalf("NewWindow: %v", err)
	}
	defer w.Close()

	scale := w.ScaleFactor()
	devSize := geometry.NewSize(
		geometry.Pixels(400).ToDevicePixels(scale),
		geometry.Pixels(300).ToDevicePixels(scale),
	)

	r, err := d3d11.New(w.NativeSurface(), devSize, render.Options{VSync: true})
	if err != nil {
		t.Fatalf("d3d11.New: %v", err)
	}
	defer r.Close()

	p.Dispatch(func() {
		defer p.Quit()

		// Test atlas uploads
		monoData := make([]byte, 16*16)
		for i := range monoData {
			monoData[i] = 0xff
		}
		monoTile, err := r.Upload(scene.TextureMonochrome, geometry.NewSize[geometry.DevicePixels](16, 16), monoData)
		if err != nil {
			t.Errorf("r.Upload monochrome: %v", err)
			return
		}

		polyData := make([]byte, 16*16*4)
		for i := 0; i < len(polyData); i += 4 {
			polyData[i] = 0xff   // B
			polyData[i+1] = 0x00 // G
			polyData[i+2] = 0x00 // R
			polyData[i+3] = 0xff // A
		}
		polyTile, err := r.Upload(scene.TexturePolychrome, geometry.NewSize[geometry.DevicePixels](16, 16), polyData)
		if err != nil {
			t.Errorf("r.Upload polychrome: %v", err)
			return
		}

		sc := scene.New()

		// 1. Quad
		sc.InsertQuad(scene.Quad{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(50*scale), geometry.ScaledPixels(50*scale)),
				geometry.NewSize(geometry.ScaledPixels(200*scale), geometry.ScaledPixels(150*scale)),
			),
			Background:   colour.Rgba{R: 1.0, G: 0.0, B: 0.0, A: 1.0},
			BorderColour: colour.Rgba{R: 0.0, G: 1.0, B: 0.0, A: 1.0},
			CornerRadii:  geometry.AllCorners(geometry.ScaledPixels(12 * scale)),
			BorderWidths: geometry.AllEdges(geometry.ScaledPixels(2 * scale)),
			BorderStyle:  scene.BorderSolid,
		})

		// 2. Shadow
		sc.InsertShadow(scene.Shadow{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(40*scale), geometry.ScaledPixels(40*scale)),
				geometry.NewSize(geometry.ScaledPixels(220*scale), geometry.ScaledPixels(170*scale)),
			),
			CornerRadii:        geometry.AllCorners(geometry.ScaledPixels(16 * scale)),
			Colour:             colour.Rgba{R: 0.0, G: 0.0, B: 0.0, A: 0.5},
			ElementBounds:      geometry.NewBounds(geometry.NewPoint(geometry.ScaledPixels(50*scale), geometry.ScaledPixels(50*scale)), geometry.NewSize(geometry.ScaledPixels(200*scale), geometry.ScaledPixels(150*scale))),
			ElementCornerRadii: geometry.AllCorners(geometry.ScaledPixels(12 * scale)),
			BlurRadius:         geometry.ScaledPixels(8 * scale),
			Inset:              false,
		})

		// 3. Monochrome sprite (glyph)
		sc.InsertMonochromeSprite(scene.MonochromeSprite{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(60*scale), geometry.ScaledPixels(60*scale)),
				geometry.NewSize(geometry.ScaledPixels(16*scale), geometry.ScaledPixels(16*scale)),
			),
			Tile:           monoTile,
			Colour:         colour.Rgba{R: 1.0, G: 1.0, B: 1.0, A: 1.0},
			Transformation: scene.IdentityMatrix,
		})

		// 4. Polychrome sprite
		sc.InsertPolychromeSprite(scene.PolychromeSprite{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(80*scale), geometry.ScaledPixels(60*scale)),
				geometry.NewSize(geometry.ScaledPixels(16*scale), geometry.ScaledPixels(16*scale)),
			),
			Tile:    polyTile,
			Opacity: 1.0,
		})

		// 5. Underline
		sc.InsertUnderline(scene.Underline{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(60*scale), geometry.ScaledPixels(85*scale)),
				geometry.NewSize(geometry.ScaledPixels(50*scale), geometry.ScaledPixels(2*scale)),
			),
			Colour:    colour.Rgba{R: 1.0, G: 1.0, B: 0.0, A: 1.0},
			Thickness: geometry.ScaledPixels(2 * scale),
			Wavy:      false,
		})

		// 6. Path
		sc.InsertPath(scene.Path[geometry.ScaledPixels]{
			Bounds: geometry.NewBounds(
				geometry.NewPoint(geometry.ScaledPixels(60*scale), geometry.ScaledPixels(100*scale)),
				geometry.NewSize(geometry.ScaledPixels(40*scale), geometry.ScaledPixels(40*scale)),
			),
			Colour: colour.Rgba{R: 0.0, G: 0.8, B: 0.2, A: 1.0},
			Vertices: []scene.PathVertex[geometry.ScaledPixels]{
				{XYPosition: geometry.NewPoint(geometry.ScaledPixels(60*scale), geometry.ScaledPixels(100*scale))},
				{XYPosition: geometry.NewPoint(geometry.ScaledPixels(100*scale), geometry.ScaledPixels(100*scale))},
				{XYPosition: geometry.NewPoint(geometry.ScaledPixels(80*scale), geometry.ScaledPixels(140*scale))},
			},
		})

		sc.Finish()

		if err := r.Draw(sc); err != nil {
			t.Errorf("r.Draw: %v", err)
			return
		}

		if err := r.Present(); err != nil {
			t.Errorf("r.Present: %v", err)
			return
		}

		// Also test resize
		newSize := geometry.NewSize(
			geometry.Pixels(500).ToDevicePixels(scale),
			geometry.Pixels(400).ToDevicePixels(scale),
		)
		if err := r.Resize(newSize); err != nil {
			t.Errorf("r.Resize: %v", err)
			return
		}

		if err := r.Draw(sc); err != nil {
			t.Errorf("r.Draw after resize: %v", err)
			return
		}

		if err := r.Present(); err != nil {
			t.Errorf("r.Present after resize: %v", err)
			return
		}
	})

	if err := p.Run(); err != nil {
		t.Fatalf("p.Run: %v", err)
	}
}
