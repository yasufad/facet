package window

import (
	"testing"
	"time"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
)

type stubRenderer struct {
	size           geometry.Size[geometry.DevicePixels]
	quads          []scene.Quad
	presents       int
	resizes        int
	clearedAtlases []scene.AtlasTextureKind
}

func newStubRenderer(size geometry.Size[geometry.DevicePixels]) *stubRenderer {
	return &stubRenderer{size: size}
}

func (s *stubRenderer) Resize(size geometry.Size[geometry.DevicePixels]) error {
	s.size = size
	s.resizes++
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

func (s *stubRenderer) ClearAtlas(kind scene.AtlasTextureKind) {
	s.clearedAtlases = append(s.clearedAtlases, kind)
}

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
func (w *stubPlatformWindow) SetBackground(c colour.Rgba)                     {}
func (w *stubPlatformWindow) ScaleFactor() float32                            { return w.scale }
func (w *stubPlatformWindow) NativeHandle() uintptr                           { return 0 }
func (w *stubPlatformWindow) NativeSurface() uintptr                          { return 0 }
func (w *stubPlatformWindow) Focus()                                          {}
func (w *stubPlatformWindow) IsFocused() bool                                 { return true }
func (w *stubPlatformWindow) IsVisible() bool                                 { return true }
func (w *stubPlatformWindow) SetEventHandler(h func(platform.Event)) {
	w.eventHandler = h
}
func (w *stubPlatformWindow) SetCloseHandler(h func() bool) { return }

func TestEmptyWindowOptions(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	w := NewWithRenderer(nil, nil, a, WindowOptions{})
	if w.Size().Width <= 0 || w.Size().Height <= 0 {
		t.Fatalf("expected non-zero default size, got %v", w.Size())
	}
	if w.RemSize() <= 0 {
		t.Fatalf("expected non-zero default rem size, got %v", w.RemSize())
	}
	if w.ScaleFactor() <= 0 {
		t.Fatalf("expected non-zero default scale factor, got %v", w.ScaleFactor())
	}
}

func TestFrameLoopSceneAssertion(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	scale := float32(2.0)
	pw := newStubPlatformWindow(size, scale)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, scale))

	w := NewWithRenderer(pw, r, a, WindowOptions{
		Size:    size,
		RemSize: 16,
	})

	bgParent := colour.Rgba{R: 0.1, G: 0.2, B: 0.3, A: 1.0}
	borderParent := colour.Rgba{R: 0.9, G: 0.8, B: 0.7, A: 1.0}
	bgChild := colour.Rgba{R: 0.4, G: 0.5, B: 0.6, A: 1.0}

	root := element.NewDiv().
		Width(style.Px(200)).
		Height(style.Px(100)).
		Bg(bgParent).
		BorderColour(borderParent).
		Border(geometry.Pixels(4)).
		Rounded(geometry.Pixels(8)).
		Child(
			element.NewDiv().
				Width(style.Px(50)).
				Height(style.Px(50)).
				Bg(bgChild),
		)

	w.SetRoot(root)
	w.Draw()

	if r.presents != 1 {
		t.Fatalf("expected 1 present call, got %d", r.presents)
	}

	if len(r.quads) != 2 {
		t.Fatalf("expected 2 scene quads, got %d", len(r.quads))
	}

	parentQuad := r.quads[0]
	if parentQuad.Background != bgParent {
		t.Errorf("parent quad background = %v, want %v", parentQuad.Background, bgParent)
	}
	if parentQuad.BorderColour != borderParent {
		t.Errorf("parent quad border colour = %v, want %v", parentQuad.BorderColour, borderParent)
	}
	wantParentBounds := geometry.Bounds[geometry.ScaledPixels]{
		Origin: geometry.Point[geometry.ScaledPixels]{X: 0, Y: 0},
		Size:   geometry.Size[geometry.ScaledPixels]{Width: 400, Height: 200}, // 200*2, 100*2
	}
	if parentQuad.Bounds != wantParentBounds {
		t.Errorf("parent quad bounds = %v, want %v", parentQuad.Bounds, wantParentBounds)
	}
	wantParentBorder := geometry.Edges[geometry.ScaledPixels]{
		Top: 8, Right: 8, Bottom: 8, Left: 8, // 4*2
	}
	if parentQuad.BorderWidths != wantParentBorder {
		t.Errorf("parent quad border widths = %v, want %v", parentQuad.BorderWidths, wantParentBorder)
	}
	wantParentRadii := geometry.Corners[geometry.ScaledPixels]{
		TopLeft: 16, TopRight: 16, BottomRight: 16, BottomLeft: 16, // 8*2
	}
	if parentQuad.CornerRadii != wantParentRadii {
		t.Errorf("parent quad corner radii = %v, want %v", parentQuad.CornerRadii, wantParentRadii)
	}

	childQuad := r.quads[1]
	if childQuad.Background != bgChild {
		t.Errorf("child quad background = %v, want %v", childQuad.Background, bgChild)
	}
	// Child is offset by parent's 4px border (8 scaled pixels in both X and Y)
	wantChildBounds := geometry.Bounds[geometry.ScaledPixels]{
		Origin: geometry.Point[geometry.ScaledPixels]{X: 8, Y: 8},
		Size:   geometry.Size[geometry.ScaledPixels]{Width: 100, Height: 100}, // 50*2, 50*2
	}
	if childQuad.Bounds != wantChildBounds {
		t.Errorf("child quad bounds = %v, want %v", childQuad.Bounds, wantChildBounds)
	}
}

func TestIntraFrameHoverStyle(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))

	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	normalBg := colour.Rgba{R: 0.0, G: 0.0, B: 1.0, A: 1.0}
	hoverBg := colour.Rgba{R: 1.0, G: 0.0, B: 0.0, A: 1.0}

	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Width(style.Px(100)).
			Height(style.Px(50)).
			Bg(normalBg).
			Hover(func(s *style.Refinement) {
				s.SetBackground(hoverBg)
			})
	})

	// Step 1: Pointer outside element bounds (e.g. at 200, 200).
	w.pointerPos = geometry.Point[geometry.Pixels]{X: 200, Y: 200}
	w.Draw()

	if len(r.quads) != 1 {
		t.Fatalf("expected 1 quad, got %d", len(r.quads))
	}
	if r.quads[0].Background != normalBg {
		t.Errorf("expected normal background %v, got %v", normalBg, r.quads[0].Background)
	}

	// Step 2: Pointer placed over element (at 50, 25).
	// Intra-frame hit test in step 5 must resolve hover before paint in step 6.
	w.pointerPos = geometry.Point[geometry.Pixels]{X: 50, Y: 25}
	w.Draw()

	if len(r.quads) != 1 {
		t.Fatalf("expected 1 quad, got %d", len(r.quads))
	}
	if r.quads[0].Background != hoverBg {
		t.Errorf("expected hover background %v in the same frame, got %v", hoverBg, r.quads[0].Background)
	}

	// Step 3: Pointer moves away again.
	w.pointerPos = geometry.Point[geometry.Pixels]{X: 300, Y: 300}
	w.Draw()

	if r.quads[0].Background != normalBg {
		t.Errorf("expected reverted normal background %v, got %v", normalBg, r.quads[0].Background)
	}
}

func TestTwoFramesInputIsolation(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))

	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	clicked := false
	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Width(style.Px(100)).
			Height(style.Px(100)).
			OnClick(func(e element.ClickEvent) bool {
				clicked = true
				return true
			})
	})
	w.Draw() // Frame 1 renders button at (0, 0, 100, 100)

	// User clicks at (50, 50). Event resolves against rendered frame.
	downEvt := platform.PointerEvent{
		Phase:    platform.PointerDown,
		Position: geometry.Point[geometry.DevicePixels]{X: 50, Y: 50},
		Button:   platform.PointerLeft,
		Time:     time.Now(),
	}
	upEvt := platform.PointerEvent{
		Phase:    platform.PointerUp,
		Position: geometry.Point[geometry.DevicePixels]{X: 50, Y: 50},
		Button:   platform.PointerLeft,
		Time:     time.Now(),
	}

	w.DispatchEvent(downEvt)
	w.DispatchEvent(upEvt)

	if !clicked {
		t.Fatal("expected click event to be dispatched to rendered frame handler")
	}
}

type counterView struct {
	count int
}

func (c *counterView) Render(cx *app.Context[counterView]) element.Element {
	c.count++
	return element.NewDiv().Width(style.Px(100)).Height(style.Px(50))
}

func TestFlushNotificationDeduplication(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))

	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	ent := app.New(a, func(cx *app.Context[counterView]) counterView {
		return counterView{}
	})

	v := element.NewView(ent)
	w.SetRootView(v)

	// Trigger multiple notify calls inside a single update.
	ent.Update(a, func(val *counterView, cx *app.Context[counterView]) {
		cx.Notify()
		cx.Notify()
		cx.Notify()
	})

	w.Draw()

	read := ent.Read(a)
	if read.count != 1 {
		t.Fatalf("expected view to render exactly once after multiple notifications, got %d", read.count)
	}
}

func TestPhaseOrderingInvariants(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	w := NewWithRenderer(nil, nil, a, WindowOptions{})

	// Calling RequestLayout outside phaseLayout must panic.
	assertPanic(t, "RequestLayout outside phaseLayout", func() {
		w.RequestLayout(layout.Style{}, nil)
	})

	// Calling RegisterHitRegion outside phasePrepaint must panic.
	assertPanic(t, "RegisterHitRegion outside phasePrepaint", func() {
		w.RegisterHitRegion(geometry.Bounds[geometry.Pixels]{}, 0)
	})

	// Calling InsertQuad outside phasePaint must panic.
	assertPanic(t, "InsertQuad outside phasePaint", func() {
		w.InsertQuad(scene.Quad{})
	})

	// Calling IsHovered outside phasePaint must panic.
	assertPanic(t, "IsHovered outside phasePaint", func() {
		w.IsHovered(1)
	})

	// Calling LayoutBounds outside prepaint/paint must panic.
	assertPanic(t, "LayoutBounds outside prepaint/paint", func() {
		w.LayoutBounds(layout.NodeID{})
	})
}

func assertPanic(t *testing.T, name string, f func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected %s to panic, but it completed normally", name)
		}
	}()
	f()
}

func TestScaleFactorChangeAndResize(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))

	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})
	w.SetRootFn(func() element.Element {
		return element.NewDiv().Width(style.Px(100)).Height(style.Px(100))
	})

	w.Draw()

	// Scale changed event.
	scaleEvt := platform.ScaleChangedEvent{
		ScaleFactor: 2.0,
		Time:        time.Now(),
	}
	w.DispatchEvent(scaleEvt)

	if w.ScaleFactor() != 2.0 {
		t.Fatalf("expected scale factor 2.0, got %f", w.ScaleFactor())
	}
	if len(r.clearedAtlases) == 0 {
		t.Fatal("expected GPU atlas to be cleared on scale factor change")
	}

	// Resize event.
	newSize := geometry.NewSize[geometry.Pixels](800, 600)
	resizeEvt := platform.ResizeEvent{
		Size: newSize,
		Time: time.Now(),
	}
	w.DispatchEvent(resizeEvt)

	if w.Size() != newSize {
		t.Fatalf("expected window logical size %v, got %v", newSize, w.Size())
	}

	w.Draw()

	wantDevSize := geometry.SizeToDevicePixels(newSize, 2.0) // 1600x1200
	if r.Size() != wantDevSize {
		t.Fatalf("expected renderer device size %v after resize draw, got %v", wantDevSize, r.Size())
	}
}
