package main

import (
	"log"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/render"
	"github.com/yasufad/facet/render/d3d11"
	"github.com/yasufad/facet/scene"
)

func main() {
	p, err := platform.New(platform.Options{Name: "Facet Quad"})
	if err != nil {
		log.Fatal(err)
	}
	w, err := p.NewWindow(platform.WindowOptions{Title: "Facet Quad", Size: geometry.NewSize[geometry.Pixels](640, 480), Resizable: true, Decorated: true, Visible: true})
	if err != nil {
		log.Fatal(err)
	}
	sf := w.ScaleFactor()
	r, err := d3d11.New(w.NativeSurface(), geometry.NewSize(geometry.Pixels(640).ToDevicePixels(sf), geometry.Pixels(480).ToDevicePixels(sf)), render.Options{VSync: true})
	if err != nil {
		log.Fatal(err)
	}
	defer r.Close()
	draw := func() {
		sc, s := w.ScaleFactor(), scene.New()
		s.InsertQuad(scene.Quad{
			Bounds:       geometry.NewBounds(geometry.NewPoint(geometry.ScaledPixels(100*sc), geometry.ScaledPixels(80*sc)), geometry.NewSize(geometry.ScaledPixels(440*sc), geometry.ScaledPixels(320*sc))),
			Background:   colour.Rgba{R: 0.15, G: 0.5, B: 0.85, A: 1.0},
			BorderColour: colour.Rgba{R: 0.95, G: 0.75, B: 0.2, A: 1.0},
			CornerRadii:  geometry.AllCorners(geometry.ScaledPixels(24 * sc)),
			BorderWidths: geometry.AllEdges(geometry.ScaledPixels(4 * sc)),
		})
		s.Finish()
		_ = r.Draw(s)
		_ = r.Present()
	}
	w.SetEventHandler(func(e platform.Event) {
		if _, ok := e.(platform.ResizeEvent); ok {
			_ = r.Resize(geometry.NewSize(w.Size().Width.ToDevicePixels(w.ScaleFactor()), w.Size().Height.ToDevicePixels(w.ScaleFactor())))
			draw()
		}
	})
	p.Dispatch(draw)
	_ = p.Run()
}
