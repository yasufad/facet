package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/platform/platformtest"
	"github.com/yasufad/facet/render/rendertest"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/ui"
	"github.com/yasufad/facet/window"
)

type buttonIntegrationView struct {
	clicks int
}

func (v *buttonIntegrationView) Render(cx *app.Context[buttonIntegrationView]) element.Element {
	bg := colour.Rgba{R: float32(v.clicks), G: 0, B: 0, A: 1.0}
	return element.NewDiv().
		Width(style.Px(400)).
		Height(style.Px(300)).
		Bg(bg).
		Child(
			ui.NewButton(fmt.Sprintf("Clicks: %d", v.clicks)).
				OnClick(element.Listener(cx, func(v *buttonIntegrationView, e element.ClickEvent, cx *app.Context[buttonIntegrationView]) bool {
					v.clicks++
					cx.Notify()
					return true
				})),
		)
}

func TestButtonClickInWindowMutatesEntityAndRendersNextFrame(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := platformtest.NewWindow(size, 1.0)
	r := rendertest.NewRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := window.NewWithRenderer(pw, r, a, window.WindowOptions{Size: size})

	ent := app.New(a, func(cx *app.Context[buttonIntegrationView]) buttonIntegrationView {
		return buttonIntegrationView{}
	})
	defer ent.Release()

	w.SetRootView(element.NewView(ent))

	// Frame 1: initial render
	w.Draw()

	if read := ent.Read(a); read.clicks != 0 {
		t.Fatalf("expected initial clicks 0, got %d", read.clicks)
	}
	if len(r.Quads) < 2 {
		t.Fatalf("expected at least 2 quads in frame 1, got %d", len(r.Quads))
	}
	if r.Quads[0].Background.R != 0 {
		t.Fatalf("expected initial container quad red component 0, got %v", r.Quads[0].Background.R)
	}

	// Dispatch synthetic pointer down and up inside the button at (30, 15).
	downEvt := platform.PointerEvent{
		Phase:    platform.PointerDown,
		Position: geometry.Point[geometry.DevicePixels]{X: 30, Y: 15},
		Button:   platform.PointerLeft,
		Time:     time.Now(),
	}
	upEvt := platform.PointerEvent{
		Phase:    platform.PointerUp,
		Position: geometry.Point[geometry.DevicePixels]{X: 30, Y: 15},
		Button:   platform.PointerLeft,
		Time:     time.Now(),
	}

	w.DispatchEvent(downEvt)
	w.DispatchEvent(upEvt)

	// Assert 1: The entity changed via element.Listener.
	read := ent.Read(a)
	if read.clicks != 1 {
		t.Fatalf("expected entity clicks to be 1 after click, got %d", read.clicks)
	}

	// Frame 2: Draw the next frame.
	w.Draw()

	// Assert 2: Next frame reflects changed entity state.
	if len(r.Quads) < 2 {
		t.Fatalf("expected at least 2 quads in frame 2, got %d", len(r.Quads))
	}
	if r.Quads[0].Background.R != 1.0 {
		t.Fatalf("expected frame 2 container quad red component 1.0, got %v", r.Quads[0].Background.R)
	}
}
