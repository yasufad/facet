package main

import (
	"fmt"
	"log"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/ui"
	"github.com/yasufad/facet/window"
)

// CounterView is a reactive view entity that holds click count state.
type CounterView struct {
	count int
}

// Render produces the element tree displaying the current count and button.
func (c *CounterView) Render(cx *app.Context[CounterView]) element.Element {
	return element.NewDiv().
		Flex().
		FlexCol().
		WFull().
		HFull().
		AlignItems(style.AlignItemsCentre).
		JustifyContent(style.AlignContentCentre).
		GapRow(style.Px(16)).
		Bg(colour.Rgba{R: 0.12, G: 0.14, B: 0.18, A: 1.0}).
		Children(
			element.NewText(fmt.Sprintf("Clicks: %d", c.count)).
				FontSize(geometry.Pixels(22)).
				TextColour(colour.Rgba{R: 0.9, G: 0.9, B: 0.95, A: 1.0}),
			ui.NewButton("Click Me").
				OnClick(element.Listener(cx, func(c *CounterView, event element.ClickEvent, cx *app.Context[CounterView]) bool {
					c.count++
					fmt.Printf("Button clicked! Count: %d\n", c.count)
					cx.Notify()
					return true
				})),
		)
}

func main() {
	p, err := platform.New(platform.Options{Name: "Facet Button Example"})
	if err != nil {
		log.Fatal(err)
	}

	a := app.NewApp()
	defer a.Close()

	w, err := window.New(p, a, window.WindowOptions{
		Title:     "Facet Button Example",
		Size:      geometry.NewSize[geometry.Pixels](480, 320),
		Resizable: true,
		Decorated: true,
		Visible:   true,
		VSync:     true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()

	counterEnt := app.New(a, func(cx *app.Context[CounterView]) CounterView {
		return CounterView{}
	})

	w.SetRootView(element.NewView(counterEnt))

	p.Dispatch(w.Draw)
	if err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
