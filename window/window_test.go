package window

import (
	"testing"
	"time"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/text"
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
func (w *stubPlatformWindow) SetCursor(shape platform.Cursor)                 {}
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
	var receivedEvt element.ClickEvent
	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Child(
				element.NewDiv().
					MarginLeft(style.Px(20)).
					MarginTop(style.Px(30)).
					Width(style.Px(100)).
					Height(style.Px(100)).
					OnClick(func(e element.ClickEvent) bool {
						clicked = true
						receivedEvt = e
						return true
					}),
			)
	})
	w.Draw() // Frame 1 renders child button at (20, 30, 100, 100)

	// User clicks at (50, 50) in window coordinates.
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
	if receivedEvt.Position.X != 50 || receivedEvt.Position.Y != 50 {
		t.Errorf("expected Position (50, 50), got %v", receivedEvt.Position)
	}
	if receivedEvt.LocalPosition.X != 30 || receivedEvt.LocalPosition.Y != 20 { // 50 - 20, 50 - 30
		t.Errorf("expected LocalPosition (30, 20), got %v", receivedEvt.LocalPosition)
	}
}

type counterView struct {
	count int
}

func (c *counterView) Render(cx *app.Context[counterView]) element.Element {
	c.count++
	return element.NewDiv().Width(style.Px(100)).Height(style.Px(50))
}

type stubPlatform struct {
	platform.Platform
	dispatched []func()
	cursors    []platform.Cursor
}

func (p *stubPlatform) Run() error { return nil }
func (p *stubPlatform) Quit()      {}
func (p *stubPlatform) NewWindow(opts platform.WindowOptions) (platform.Window, error) {
	return newStubPlatformWindow(opts.Size, 1.0), nil
}
func (p *stubPlatform) SetCursor(shape platform.Cursor) {
	p.cursors = append(p.cursors, shape)
}
func (p *stubPlatform) Dispatch(f func()) {
	p.dispatched = append(p.dispatched, f)
}
func (p *stubPlatform) Drain() int {
	count := 0
	for len(p.dispatched) > 0 {
		callbacks := p.dispatched
		p.dispatched = nil
		for _, cb := range callbacks {
			count++
			cb()
		}
	}
	return count
}

func TestFlushNotificationDeduplication(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	plat := &stubPlatform{}
	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))

	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})
	w.platform = plat

	ent := app.New(a, func(cx *app.Context[counterView]) counterView {
		return counterView{}
	})

	v := element.NewView(ent)
	w.SetRootView(v)

	// Drain initial frame scheduled by SetRootView.
	plat.Drain()
	if read := ent.Read(a); read.count != 1 {
		t.Fatalf("expected initial render count 1, got %d", read.count)
	}

	// Trigger multiple notify calls inside a single update.
	// Notification must travel through app.Observe -> w.ScheduleFrame -> plat.Dispatch.
	ent.Update(a, func(val *counterView, cx *app.Context[counterView]) {
		cx.Notify()
		cx.Notify()
		cx.Notify()
	})

	if len(plat.dispatched) != 1 {
		t.Fatalf("expected exactly 1 scheduled frame dispatch after burst of 3 notifies, got %d", len(plat.dispatched))
	}

	// Process the scheduled frame turn without calling w.Draw() manually.
	plat.Drain()

	read := ent.Read(a)
	if read.count != 2 {
		t.Fatalf("expected view to render exactly once more (count=2) after notifications, got %d", read.count)
	}

	// Verify no redundant draws are pending.
	if plat.Drain() != 0 {
		t.Fatal("expected no further frames scheduled when window is clean")
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

	// Calling RequestMeasuredLayout outside phaseLayout must panic.
	assertPanic(t, "RequestMeasuredLayout outside phaseLayout", func() {
		w.RequestMeasuredLayout(layout.Style{}, nil)
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

	// Calling RasteriseGlyph outside phasePaint must panic.
	assertPanic(t, "RasteriseGlyph outside phasePaint", func() {
		w.RasteriseGlyph(text.Face{}, 0, 16, text.SubpixelZero)
	})
}

type testMeasureElement struct {
	onMeasure func(f element.Frame)
}

func (m *testMeasureElement) RequestLayout(f element.Frame) layout.NodeID {
	return f.RequestMeasuredLayout(layout.Style{}, func(known layout.Size[layout.OptF32], avail layout.Size[layout.AvailableSpace]) geometry.Size[geometry.Pixels] {
		if m.onMeasure != nil {
			m.onMeasure(f)
		}
		return geometry.NewSize[geometry.Pixels](100, 20)
	})
}

func (m *testMeasureElement) Prepaint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {}
func (m *testMeasureElement) Paint(f element.Frame, bounds geometry.Bounds[geometry.Pixels])    {}

func TestMeasureCallbackPhaseEnforcement(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	// 1. ShapeLine inside measure callback (phaseLayoutSolve) must succeed without panic.
	var shapeSucceeded bool
	w.SetRootFn(func() element.Element {
		return &testMeasureElement{
			onMeasure: func(f element.Frame) {
				_, err := f.ShapeLine("measure test", []text.StyleRun{{ByteLen: len("measure test"), Size: 16}})
				if err == nil {
					shapeSucceeded = true
				}
			},
		}
	})
	w.Draw()
	if !shapeSucceeded {
		t.Fatal("expected ShapeLine to succeed inside measure callback (phaseLayoutSolve)")
	}

	// 2. RequestLayout called inside measure callback must panic.
	assertPanic(t, "RequestLayout inside measure callback", func() {
		w.SetRootFn(func() element.Element {
			return &testMeasureElement{
				onMeasure: func(f element.Frame) {
					f.RequestLayout(layout.Style{}, nil)
				},
			}
		})
		w.Draw()
	})

	// 3. RegisterHitRegion called inside measure callback must panic.
	assertPanic(t, "RegisterHitRegion inside measure callback", func() {
		w.SetRootFn(func() element.Element {
			return &testMeasureElement{
				onMeasure: func(f element.Frame) {
					f.RegisterHitRegion(geometry.Bounds[geometry.Pixels]{}, 0)
				},
			}
		})
		w.Draw()
	})

	// 4. InsertQuad called inside measure callback must panic.
	assertPanic(t, "InsertQuad inside measure callback", func() {
		w.SetRootFn(func() element.Element {
			return &testMeasureElement{
				onMeasure: func(f element.Frame) {
					f.InsertQuad(scene.Quad{})
				},
			}
		})
		w.Draw()
	})

	// 5. RasteriseGlyph called inside measure callback must panic.
	assertPanic(t, "RasteriseGlyph inside measure callback", func() {
		w.SetRootFn(func() element.Element {
			return &testMeasureElement{
				onMeasure: func(f element.Frame) {
					f.RasteriseGlyph(text.Face{}, 0, 16, text.SubpixelZero)
				},
			}
		})
		w.Draw()
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

func TestTextElementMeasuresFromShaping(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	txt := element.NewText("Hello Facet").FontSize(16).LineHeight(24)
	root := element.NewDiv().Child(txt)

	w.SetRoot(root)
	w.Draw()

	if len(w.nodeBounds) < 2 {
		t.Fatalf("expected at least 2 node bounds, got %d", len(w.nodeBounds))
	}

	var foundChild bool
	for _, b := range w.nodeBounds {
		if b.Size.Width != 400 || b.Size.Height != 300 {
			if b.Size.Width <= 0 {
				t.Errorf("expected text node width > 0, got %v", b.Size.Width)
			}
			if b.Size.Height != 24 {
				t.Errorf("expected text node height 24, got %v", b.Size.Height)
			}
			foundChild = true
		}
	}
	if !foundChild {
		t.Fatal("child text node not found in nodeBounds")
	}
}

func TestPointerDownFocusAndStyling(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	focusID := input.NewFocusID()
	defaultBg := colour.Rgba{R: 0.1, G: 0.1, B: 0.1, A: 1.0}
	focusBg := colour.Rgba{R: 0.0, G: 1.0, B: 0.0, A: 1.0}

	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Width(style.Px(400)).
			Height(style.Px(300)).
			Child(
				element.NewDiv().
					Width(style.Px(100)).
					Height(style.Px(50)).
					Bg(defaultBg).
					TrackFocus(focusID).
					Focus(func(r *style.Refinement) {
						r.SetBackground(focusBg)
					}),
			)
	})

	// Initial frame: nothing focused
	w.Draw()
	if len(r.quads) != 1 {
		t.Fatalf("expected exactly 1 quad, got %d", len(r.quads))
	}
	if r.quads[0].Background != defaultBg {
		t.Fatalf("expected initial quad background %v, got %v", defaultBg, r.quads[0].Background)
	}
	if _, ok := w.focusTree.Focused(); ok {
		t.Fatalf("element unexpectedly focused initially")
	}

	// Pointer down inside the child button (50, 25)
	w.DispatchEvent(platform.PointerEvent{
		Position: geometry.NewPoint[geometry.DevicePixels](50, 25),
		Phase:    platform.PointerDown,
		Button:   platform.PointerLeft,
	})

	// Redraw: focus style should now be active
	w.Draw()
	if focused, ok := w.focusTree.Focused(); !ok || focused != focusID {
		t.Fatalf("expected element %v to be focused after pointer down, got (%v, %v)", focusID, focused, ok)
	}
	if len(r.quads) != 1 || r.quads[0].Background != focusBg {
		t.Fatalf("expected focused quad background %v, got %v", focusBg, r.quads[0].Background)
	}

	// Pointer down outside the child button (200, 200)
	w.DispatchEvent(platform.PointerEvent{
		Position: geometry.NewPoint[geometry.DevicePixels](200, 200),
		Phase:    platform.PointerDown,
		Button:   platform.PointerLeft,
	})

	// Redraw: focus should be cleared
	w.Draw()
	if focused, ok := w.focusTree.Focused(); ok {
		t.Fatalf("expected focus to be cleared after clicking outside, got %v", focused)
	}
	if len(r.quads) != 1 || r.quads[0].Background != defaultBg {
		t.Fatalf("expected unfocused quad background %v, got %v", defaultBg, r.quads[0].Background)
	}
}

func TestFocusDroppedWhenElementLeavesTree(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	focusID := input.NewFocusID()
	renderChild := true

	w.SetRootFn(func() element.Element {
		div := element.NewDiv().Width(style.Px(400)).Height(style.Px(300))
		if renderChild {
			div.Child(
				element.NewDiv().
					Width(style.Px(100)).
					Height(style.Px(50)).
					TrackFocus(focusID),
			)
		}
		return div
	})

	// Frame 1: child is present; focus it
	w.Draw()
	w.DispatchEvent(platform.PointerEvent{
		Position: geometry.NewPoint[geometry.DevicePixels](50, 25),
		Phase:    platform.PointerDown,
		Button:   platform.PointerLeft,
	})
	w.Draw()

	if focused, ok := w.focusTree.Focused(); !ok || focused != focusID {
		t.Fatalf("expected focusTree focused on %v, got (%v, %v)", focusID, focused, ok)
	}

	// Frame 2: child is removed from the tree
	renderChild = false
	w.dirty = true
	w.Draw()

	// Focus must drop to nothing
	if focused, ok := w.focusTree.Focused(); ok || focused != 0 {
		t.Fatalf("expected focus to drop when element left tree, got (%v, %v)", focused, ok)
	}
}

func TestCursorTransitionsAndDeduplication(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	plat := &stubPlatform{}
	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})
	w.platform = plat

	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Width(style.Px(400)).
			Height(style.Px(300)).
			Children(
				// Div A: 0..100 -> CursorPointer
				element.NewDiv().
					Width(style.Px(100)).
					Height(style.Px(100)).
					Cursor(style.CursorPointer),
				// Div B: 100..200 -> CursorNotAllowed
				element.NewDiv().
					Width(style.Px(100)).
					Height(style.Px(100)).
					Cursor(style.CursorNotAllowed),
			)
	})

	// 1. Initial draw with pointer at (350, 250) (background area)
	w.DispatchEvent(platform.PointerEvent{
		Position: geometry.NewPoint[geometry.DevicePixels](350, 250),
		Phase:    platform.PointerMove,
	})
	w.Draw()
	if len(plat.cursors) != 0 {
		t.Fatalf("expected no SetCursor calls for default background cursor, got %v", plat.cursors)
	}

	// 2. Move pointer onto Div A (50, 50)
	w.DispatchEvent(platform.PointerEvent{
		Position: geometry.NewPoint[geometry.DevicePixels](50, 50),
		Phase:    platform.PointerMove,
	})
	w.Draw()
	if len(plat.cursors) != 1 || plat.cursors[0] != platform.CursorPointer {
		t.Fatalf("expected [CursorPointer], got %v", plat.cursors)
	}

	// 3. Move pointer slightly within Div A (60, 60): must NOT call SetCursor again
	w.DispatchEvent(platform.PointerEvent{
		Position: geometry.NewPoint[geometry.DevicePixels](60, 60),
		Phase:    platform.PointerMove,
	})
	w.Draw()
	if len(plat.cursors) != 1 {
		t.Fatalf("expected no redundant SetCursor call when moving within same cursor region, got %v", plat.cursors)
	}

	// 4. Move pointer onto Div B (150, 50)
	w.DispatchEvent(platform.PointerEvent{
		Position: geometry.NewPoint[geometry.DevicePixels](150, 50),
		Phase:    platform.PointerMove,
	})
	w.Draw()
	if len(plat.cursors) != 2 || plat.cursors[1] != platform.CursorNotAllowed {
		t.Fatalf("expected [CursorPointer, CursorNotAllowed], got %v", plat.cursors)
	}

	// 5. Move pointer back to background area (350, 250)
	w.DispatchEvent(platform.PointerEvent{
		Position: geometry.NewPoint[geometry.DevicePixels](350, 250),
		Phase:    platform.PointerMove,
	})
	w.Draw()
	if len(plat.cursors) != 3 || plat.cursors[2] != platform.CursorDefault {
		t.Fatalf("expected [CursorPointer, CursorNotAllowed, CursorDefault], got %v", plat.cursors)
	}
}

func TestRequestFocusPhaseEnforcement(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	focusID := input.NewFocusID()

	// Calling RequestFocus inside a measure callback (phaseLayoutSolve) must panic.
	assertPanic(t, "RequestFocus inside measure callback", func() {
		w.SetRootFn(func() element.Element {
			return &testMeasureElement{
				onMeasure: func(f element.Frame) {
					f.RequestFocus(focusID)
				},
			}
		})
		w.Draw()
	})
}

func TestTabNavigationThroughWindow(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	focus1 := input.NewFocusID()
	focus2 := input.NewFocusID()

	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Width(style.Px(400)).
			Height(style.Px(300)).
			Children(
				element.NewDiv().
					Width(style.Px(100)).
					Height(style.Px(50)).
					TrackFocus(focus1),
				element.NewDiv().
					Width(style.Px(100)).
					Height(style.Px(50)).
					TrackFocus(focus2),
			)
	})

	w.Draw()

	// Initial Tab -> focuses focus1
	w.DispatchEvent(platform.KeyEvent{
		Phase: platform.KeyDown,
		Code:  platform.KeyTab,
	})
	if f, _ := w.focusTree.Focused(); f != focus1 {
		t.Fatalf("expected focus on %v, got %v", focus1, f)
	}

	// Tab -> focuses focus2
	w.DispatchEvent(platform.KeyEvent{
		Phase: platform.KeyDown,
		Code:  platform.KeyTab,
	})
	if f, _ := w.focusTree.Focused(); f != focus2 {
		t.Fatalf("expected focus on %v, got %v", focus2, f)
	}

	// Tab -> wraps to focus1
	w.DispatchEvent(platform.KeyEvent{
		Phase: platform.KeyDown,
		Code:  platform.KeyTab,
	})
	if f, _ := w.focusTree.Focused(); f != focus1 {
		t.Fatalf("expected focus wrapped to %v, got %v", focus1, f)
	}

	// Shift-Tab -> wraps to focus2
	w.DispatchEvent(platform.KeyEvent{
		Phase:     platform.KeyDown,
		Code:      platform.KeyTab,
		Modifiers: platform.Shift,
	})
	if f, _ := w.focusTree.Focused(); f != focus2 {
		t.Fatalf("expected focus wrapped to %v on Shift-Tab, got %v", focus2, f)
	}
}

func TestWindowPushClipPrimitiveMask(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 2.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 2.0))
	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Width(style.Px(100)).
			Height(style.Px(100)).
			OverflowHidden().
			Child(
				element.NewDiv().
					Width(style.Px(300)).
					Height(style.Px(300)).
					Bg(colour.Rgba{R: 0, G: 0, B: 1, A: 1}),
			)
	})

	w.Draw()

	if len(r.quads) == 0 {
		t.Fatalf("expected at least 1 quad, got %d", len(r.quads))
	}
	childQuad := r.quads[len(r.quads)-1]
	expectedMask := geometry.Bounds[geometry.ScaledPixels]{
		Origin: geometry.NewPoint[geometry.ScaledPixels](0, 0),
		Size:   geometry.NewSize[geometry.ScaledPixels](200, 200), // 100 * 2.0
	}
	if childQuad.ContentMask.Bounds != expectedMask {
		t.Fatalf("expected child quad content mask %v, got %v", expectedMask, childQuad.ContentMask.Bounds)
	}
}

// clipTestElement registers one hit region under a prepaint clip narrower than
// the region itself, so the region's effective (post-clip) bounds are smaller
// than its nominal bounds — a shape no Div can produce yet, since Div does not
// call PushClip during Prepaint. It exercises window's side of the relaxation
// directly.
type clipTestElement struct {
	regionBounds geometry.Bounds[geometry.Pixels]
	clipBounds   geometry.Bounds[geometry.Pixels]
	clicked      bool
}

func (c *clipTestElement) RequestLayout(f element.Frame) element.NodeID {
	return f.RequestMeasuredLayout(layout.Style{}, func(known layout.Size[layout.OptF32], avail layout.Size[layout.AvailableSpace]) geometry.Size[geometry.Pixels] {
		return c.regionBounds.Size
	})
}

func (c *clipTestElement) Prepaint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {
	nodeID := f.PushDispatchNode(element.DispatchNode{
		ClickListeners: []func(element.ClickEvent) bool{
			func(element.ClickEvent) bool {
				c.clicked = true
				return true
			},
		},
	})
	f.PushClip(c.clipBounds)
	f.RegisterHitRegion(c.regionBounds, nodeID)
	f.PopClip()
	f.PopDispatchNode()
}

func (c *clipTestElement) Paint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {}

func TestHitRegionClippedDuringPrepaintMisses(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	el := &clipTestElement{
		regionBounds: geometry.Bounds[geometry.Pixels]{
			Origin: geometry.NewPoint[geometry.Pixels](0, 0),
			Size:   geometry.NewSize[geometry.Pixels](300, 300),
		},
		clipBounds: geometry.Bounds[geometry.Pixels]{
			Origin: geometry.NewPoint[geometry.Pixels](0, 0),
			Size:   geometry.NewSize[geometry.Pixels](100, 100),
		},
	}
	w.SetRoot(el)
	w.Draw()

	// (200, 200) lies within the region's nominal 300x300 bounds but outside
	// the 100x100 clip it was registered under.
	w.DispatchEvent(platform.PointerEvent{
		Phase:    platform.PointerDown,
		Position: geometry.NewPoint[geometry.DevicePixels](200, 200),
		Button:   platform.PointerLeft,
	})
	w.DispatchEvent(platform.PointerEvent{
		Phase:    platform.PointerUp,
		Position: geometry.NewPoint[geometry.DevicePixels](200, 200),
		Button:   platform.PointerLeft,
	})
	if el.clicked {
		t.Fatal("expected click outside the prepaint clip mask to miss the hit region")
	}

	el.clicked = false

	// (50, 50) lies within both the region and the clip.
	w.DispatchEvent(platform.PointerEvent{
		Phase:    platform.PointerDown,
		Position: geometry.NewPoint[geometry.DevicePixels](50, 50),
		Button:   platform.PointerLeft,
	})
	w.DispatchEvent(platform.PointerEvent{
		Phase:    platform.PointerUp,
		Position: geometry.NewPoint[geometry.DevicePixels](50, 50),
		Button:   platform.PointerLeft,
	})
	if !el.clicked {
		t.Fatal("expected click inside the prepaint clip mask to hit the region")
	}
}

func TestPointerCaptureRoutesMoveOutsideRegionAndClearsActive(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	normalBg := colour.Rgba{R: 0.0, G: 0.0, B: 1.0, A: 1.0}
	activeBg := colour.Rgba{R: 1.0, G: 0.0, B: 0.0, A: 1.0}

	var moveEvents []geometry.Point[geometry.DevicePixels]

	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Width(style.Px(100)).
			Height(style.Px(100)).
			Bg(normalBg).
			Active(func(s *style.Refinement) {
				s.SetBackground(activeBg)
			}).
			OnMouseMove(func(e input.PointerEvent, phase input.DispatchPhase) bool {
				if e.Phase == platform.PointerMove {
					moveEvents = append(moveEvents, e.Position)
				}
				return false
			})
	})
	w.Draw()

	// Press inside the region: begins capture, and the frame after paints active.
	w.DispatchEvent(platform.PointerEvent{
		Phase:    platform.PointerDown,
		Position: geometry.NewPoint[geometry.DevicePixels](50, 50),
		Button:   platform.PointerLeft,
	})
	w.Draw()
	if len(r.quads) != 1 || r.quads[0].Background != activeBg {
		t.Fatalf("expected active background %v while pressed inside region, got %v", activeBg, r.quads[0].Background)
	}

	// Move far outside the region (and the window). The capture must still
	// route the move to the pressed node, and IsActive must go false because
	// the pointer has left the captured region's bounds.
	outside := geometry.NewPoint[geometry.DevicePixels](5000, 5000)
	w.DispatchEvent(platform.PointerEvent{
		Phase:    platform.PointerMove,
		Position: outside,
	})

	// The listener fires once per dispatch phase (capture, then bubble).
	if len(moveEvents) != 2 || moveEvents[0] != outside || moveEvents[1] != outside {
		t.Fatalf("expected the captured node to receive the move event at %v, got %v", outside, moveEvents)
	}

	w.Draw()
	if len(r.quads) != 1 || r.quads[0].Background != normalBg {
		t.Fatalf("expected reverted background %v (IsActive false) once the pointer left the captured bounds, got %v", normalBg, r.quads[0].Background)
	}
}

func TestPointerDownOnUnfocusableElementLeavesFocusAlone(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	focusID := input.NewFocusID()
	buttonClicked := false

	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Width(style.Px(400)).
			Height(style.Px(300)).
			Children(
				element.NewDiv().
					Width(style.Px(100)).
					Height(style.Px(50)).
					TrackFocus(focusID),
				element.NewDiv().
					Width(style.Px(100)).
					Height(style.Px(50)).
					MarginLeft(style.Px(50)).
					OnClick(func(element.ClickEvent) bool {
						buttonClicked = true
						return true
					}),
			)
	})
	w.Draw()

	// Focus the field.
	w.DispatchEvent(platform.PointerEvent{
		Phase:    platform.PointerDown,
		Position: geometry.NewPoint[geometry.DevicePixels](50, 25),
		Button:   platform.PointerLeft,
	})
	w.Draw()
	if focused, ok := w.focusTree.Focused(); !ok || focused != focusID {
		t.Fatalf("expected field %v to be focused, got (%v, %v)", focusID, focused, ok)
	}

	// Click the button, which registers no focus id (default row layout
	// places it to the right of the field, at x >= 150).
	w.DispatchEvent(platform.PointerEvent{
		Phase:    platform.PointerDown,
		Position: geometry.NewPoint[geometry.DevicePixels](200, 25),
		Button:   platform.PointerLeft,
	})
	w.DispatchEvent(platform.PointerEvent{
		Phase:    platform.PointerUp,
		Position: geometry.NewPoint[geometry.DevicePixels](200, 25),
		Button:   platform.PointerLeft,
	})
	w.Draw()

	if !buttonClicked {
		t.Fatal("expected the button's click handler to fire")
	}
	if focused, ok := w.focusTree.Focused(); !ok || focused != focusID {
		t.Fatalf("expected focus to remain on field %v after clicking an unfocusable button, got (%v, %v)", focusID, focused, ok)
	}
}

func TestReentrantDrawPanics(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	assertPanic(t, "Draw called re-entrantly from within measure", func() {
		w.SetRootFn(func() element.Element {
			return &testMeasureElement{
				onMeasure: func(f element.Frame) {
					w.Draw()
				},
			}
		})
		w.Draw()
	})
}

func TestNilRootDrawLeavesFrameConsistent(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	size := geometry.NewSize[geometry.Pixels](400, 300)
	pw := newStubPlatformWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := NewWithRenderer(pw, r, a, WindowOptions{Size: size})

	// Frame 1: root renders content.
	w.SetRootFn(func() element.Element {
		return element.NewDiv().Width(style.Px(100)).Height(style.Px(100)).
			Bg(colour.Rgba{R: 1, G: 0, B: 0, A: 1})
	})
	w.Draw()
	if len(r.quads) != 1 {
		t.Fatalf("expected 1 quad after first frame, got %d", len(r.quads))
	}
	if r.presents != 1 {
		t.Fatalf("expected 1 present after first frame, got %d", r.presents)
	}

	// Frame 2: root renders nil. The window must still complete a full frame
	// cycle — present an empty scene rather than leaving frame 1's stale
	// content or half-updated state behind.
	w.SetRootFn(func() element.Element { return nil })
	w.dirty = true
	w.Draw()

	if w.phase != phaseNone {
		t.Fatalf("expected phaseNone after a nil-root Draw, got %v", w.phase)
	}
	if w.dirty {
		t.Fatal("expected dirty to be cleared after a nil-root Draw")
	}
	if r.presents != 2 {
		t.Fatalf("expected a second present for the empty frame, got %d", r.presents)
	}
	if len(r.quads) != 0 {
		t.Fatalf("expected the empty frame to present 0 quads, got %d", len(r.quads))
	}

	// A further Draw() must not panic or misbehave: the window is left in a
	// state a completed frame would leave it in, not a half-drawn one.
	w.SetRootFn(func() element.Element {
		return element.NewDiv().Width(style.Px(50)).Height(style.Px(50)).
			Bg(colour.Rgba{R: 0, G: 1, B: 0, A: 1})
	})
	w.dirty = true
	w.Draw()
	if len(r.quads) != 1 {
		t.Fatalf("expected 1 quad after recovering from a nil-root frame, got %d", len(r.quads))
	}
}
