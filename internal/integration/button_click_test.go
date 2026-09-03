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
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/ui"
	"github.com/yasufad/facet/window"
)

type stubRenderer struct {
	size     geometry.Size[geometry.DevicePixels]
	quads    []scene.Quad
	presents int
}

func newStubRenderer(size geometry.Size[geometry.DevicePixels]) *stubRenderer {
	return &stubRenderer{size: size}
}

func (s *stubRenderer) Resize(size geometry.Size[geometry.DevicePixels]) error {
	s.size = size
	return nil
}

func (s *stubRenderer) Draw(sc *scene.Scene) error {
	s.quads = append([]scene.Quad(nil), sc.Quads()...)
	return nil
}

func (s *stubRenderer) Present() error {
	s.presents++
	return nil
}

func (s *stubRenderer) Upload(kind scene.AtlasTextureKind, size geometry.Size[geometry.DevicePixels], data []byte) (scene.AtlasTile, error) {
	return scene.AtlasTile{}, nil
}

func (s *stubRenderer) ClearAtlas(kind scene.AtlasTextureKind) {}

func (s *stubRenderer) Size() geometry.Size[geometry.DevicePixels] {
	return s.size
}

func (s *stubRenderer) Close() error {
	return nil
}

type stubPlatformWindow struct {
	size         geometry.Size[geometry.Pixels]
	pos          geometry.Point[geometry.Pixels]
	scale        float32
	eventHandler func(platform.Event)
	cursors      []platform.Cursor
	state        platform.WindowState
}

func newStubPlatformWindow(size geometry.Size[geometry.Pixels], scale float32) *stubPlatformWindow {
	return &stubPlatformWindow{
		size:  size,
		scale: scale,
	}
}

func (w *stubPlatformWindow) Show()                                           {}
func (w *stubPlatformWindow) Hide()                                           {}
func (w *stubPlatformWindow) Close()                                          {}
func (w *stubPlatformWindow) SetTitle(title string)                           {}
func (w *stubPlatformWindow) SetSize(size geometry.Size[geometry.Pixels])     { w.size = size }
func (w *stubPlatformWindow) Size() geometry.Size[geometry.Pixels]            { return w.size }
func (w *stubPlatformWindow) SetPosition(pos geometry.Point[geometry.Pixels]) { w.pos = pos }
func (w *stubPlatformWindow) Position() geometry.Point[geometry.Pixels]       { return w.pos }
func (w *stubPlatformWindow) SetMinSize(size geometry.Size[geometry.Pixels])  {}
func (w *stubPlatformWindow) SetMaxSize(size geometry.Size[geometry.Pixels])  {}
func (w *stubPlatformWindow) SetResizable(resizable bool)                     {}
func (w *stubPlatformWindow) SetAlwaysOnTop(onTop bool)                       {}
func (w *stubPlatformWindow) State() platform.WindowState                     { return w.state }
func (w *stubPlatformWindow) SetState(state platform.WindowState)             { w.state = state }
func (w *stubPlatformWindow) SetBackground(c colour.Rgba)                     {}
func (w *stubPlatformWindow) ScaleFactor() float32                            { return w.scale }
func (w *stubPlatformWindow) NativeHandle() uintptr                           { return 0 }
func (w *stubPlatformWindow) NativeSurface() uintptr                          { return 0 }
func (w *stubPlatformWindow) Focus()                                          {}
func (w *stubPlatformWindow) IsFocused() bool                                 { return true }
func (w *stubPlatformWindow) IsVisible() bool                                 { return true }
func (w *stubPlatformWindow) SetCursor(shape platform.Cursor) {
	w.cursors = append(w.cursors, shape)
}
func (w *stubPlatformWindow) SetEventHandler(h func(platform.Event)) {
	w.eventHandler = h
}
func (w *stubPlatformWindow) SetCloseHandler(h func() bool) { return }

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
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
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
	if len(r.quads) < 2 {
		t.Fatalf("expected at least 2 quads in frame 1, got %d", len(r.quads))
	}
	if r.quads[0].Background.R != 0 {
		t.Fatalf("expected initial container quad red component 0, got %v", r.quads[0].Background.R)
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
	if len(r.quads) < 2 {
		t.Fatalf("expected at least 2 quads in frame 2, got %d", len(r.quads))
	}
	if r.quads[0].Background.R != 1.0 {
		t.Fatalf("expected frame 2 container quad red component 1.0, got %v", r.quads[0].Background.R)
	}
}
