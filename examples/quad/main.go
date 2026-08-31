package main

import (
	"log"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/window"
)

func main() {
	p, err := platform.New(platform.Options{Name: "Facet Quad"})
	if err != nil {
		log.Fatal(err)
	}

	a := app.NewApp()
	defer a.Close()

	w, err := window.New(p, a, window.WindowOptions{
		Title:     "Facet Quad",
		Size:      geometry.NewSize[geometry.Pixels](640, 480),
		Resizable: true,
		Decorated: true,
		Visible:   true,
		VSync:     true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()

	bgColour := colour.Rgba{R: 0.15, G: 0.5, B: 0.85, A: 1.0}
	hoverBgColour := colour.Rgba{R: 0.25, G: 0.65, B: 0.95, A: 1.0}
	borderColour := colour.Rgba{R: 0.95, G: 0.75, B: 0.2, A: 1.0}

	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Flex().
			WFull().
			HFull().
			AlignItems(style.AlignItemsCentre).
			JustifyContent(style.JustifyContentCentre).
			Child(
				element.NewDiv().
					Width(style.Px(440)).
					Height(style.Px(320)).
					Bg(bgColour).
					BorderColour(borderColour).
					Border(geometry.Pixels(4)).
					Rounded(geometry.Pixels(24)).
					Hover(func(s *style.Refinement) {
						s.SetBackground(hoverBgColour)
					}),
			)
	})

	p.Dispatch(w.Draw)
	if err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
