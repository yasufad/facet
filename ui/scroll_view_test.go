package ui

import (
	"testing"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/element/elementtest"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/style"
)

func TestScrollViewClipping(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[ScrollState]) ScrollState {
		return NewScrollState()
	})

	childQuadBg := colour.Rgba{R: 0.1, G: 0.8, B: 0.3, A: 1.0}
	tallContent := element.NewDiv().
		Width(style.Px(200)).
		Height(style.Px(500)).
		Bg(childQuadBg)

	sv := NewScrollView(a, state).
		Child(tallContent)

	frame := elementtest.NewFrame()
	rootID := sv.RequestLayout(frame)
	frame.Solve(rootID, 200, 150)

	bounds := frame.LayoutBounds(rootID)
	frame.SetPhase(elementtest.PhasePrepaint)
	sv.Prepaint(frame, bounds)

	frame.SetPhase(elementtest.PhasePaint)
	sv.Paint(frame, bounds)

	quads := frame.Quads()
	if len(quads) == 0 {
		t.Fatalf("expected child quad emitted into scene")
	}

	scale := frame.ScaleFactor()
	expectedMaskBounds := geometry.Bounds[geometry.ScaledPixels]{
		Origin: geometry.Point[geometry.ScaledPixels]{
			X: bounds.Origin.X.Scale(scale),
			Y: bounds.Origin.Y.Scale(scale),
		},
		Size: geometry.Size[geometry.ScaledPixels]{
			Width:  bounds.Size.Width.Scale(scale),
			Height: bounds.Size.Height.Scale(scale),
		},
	}

	for _, q := range quads {
		if q.Background == childQuadBg {
			if q.ContentMask.Bounds != expectedMaskBounds {
				t.Errorf("expected child quad mask bounds %v, got %v", expectedMaskBounds, q.ContentMask.Bounds)
			}
		}
	}
}

func TestScrollViewWheelPixelDelta(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[ScrollState]) ScrollState {
		return NewScrollState()
	})

	sv := NewScrollView(a, state).
		Child(element.NewDiv().Height(style.Px(600)))

	frame := elementtest.NewFrame()
	rootID := sv.RequestLayout(frame)
	frame.Solve(rootID, 200, 150)

	bounds := frame.LayoutBounds(rootID)
	frame.SetPhase(elementtest.PhasePrepaint)
	sv.Prepaint(frame, bounds)

	dispatchNodes := frame.DispatchNodes()
	if len(dispatchNodes) == 0 {
		t.Fatalf("expected dispatch node registered for scroll view")
	}

	wheelEvt := platform.WheelEvent{
		Position: geometry.Point[geometry.DevicePixels]{X: 50, Y: 50},
		Delta: platform.ScrollDelta{
			Unit:   platform.ScrollPixels,
			DeltaY: 42.0,
		},
	}

	for _, n := range dispatchNodes {
		for _, wl := range n.WheelListeners {
			wl(wheelEvt, input.Bubble)
		}
	}

	offset := state.Read(a).Offset()
	if offset != 42.0 {
		t.Fatalf("expected scroll offset 42.0, got %v", offset)
	}
}

func TestScrollViewWheelLineDelta(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[ScrollState]) ScrollState {
		return NewScrollState()
	})

	sv := NewScrollView(a, state).
		LineHeight(geometry.Pixels(24)).
		Child(element.NewDiv().Height(style.Px(600)))

	frame := elementtest.NewFrame()
	rootID := sv.RequestLayout(frame)
	frame.Solve(rootID, 200, 150)

	bounds := frame.LayoutBounds(rootID)
	frame.SetPhase(elementtest.PhasePrepaint)
	sv.Prepaint(frame, bounds)

	wheelEvt := platform.WheelEvent{
		Position: geometry.Point[geometry.DevicePixels]{X: 50, Y: 50},
		Delta: platform.ScrollDelta{
			Unit:   platform.ScrollLines,
			DeltaY: 3.0,
		},
	}

	for _, n := range frame.DispatchNodes() {
		for _, wl := range n.WheelListeners {
			wl(wheelEvt, input.Bubble)
		}
	}

	expectedOffset := geometry.Pixels(3.0 * 24.0) // 72px
	offset := state.Read(a).Offset()
	if offset != expectedOffset {
		t.Fatalf("expected scroll offset %v, got %v", expectedOffset, offset)
	}
}

func TestScrollViewStateAcrossFrames(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[ScrollState]) ScrollState {
		return NewScrollState()
	})

	childBg := colour.Rgba{R: 0.9, G: 0.2, B: 0.2, A: 1.0}

	// Frame 1: Offset = 0
	frame1 := elementtest.NewFrame()
	sv1 := NewScrollView(a, state).
		Child(element.NewDiv().Width(style.Px(200)).Height(style.Px(500)).Bg(childBg))

	rootID1 := sv1.RequestLayout(frame1)
	frame1.Solve(rootID1, 200, 150)
	bounds1 := frame1.LayoutBounds(rootID1)

	frame1.SetPhase(elementtest.PhasePrepaint)
	sv1.Prepaint(frame1, bounds1)

	frame1.SetPhase(elementtest.PhasePaint)
	sv1.Paint(frame1, bounds1)

	quads1 := frame1.Quads()
	var originY1 geometry.ScaledPixels
	found1 := false
	for _, q := range quads1 {
		if q.Background == childBg {
			originY1 = q.Bounds.Origin.Y
			found1 = true
			break
		}
	}
	if !found1 {
		t.Fatalf("frame 1: child quad not found")
	}

	// Mutate state to offset 100
	state.Update(a, func(st *ScrollState, cx *app.Context[ScrollState]) {
		st.SetOffset(100)
	})

	// Frame 2: Rebuild ephemeral ScrollView reading updated state
	frame2 := elementtest.NewFrame()
	sv2 := NewScrollView(a, state).
		Child(element.NewDiv().Width(style.Px(200)).Height(style.Px(500)).Bg(childBg))

	rootID2 := sv2.RequestLayout(frame2)
	frame2.Solve(rootID2, 200, 150)
	bounds2 := frame2.LayoutBounds(rootID2)

	frame2.SetPhase(elementtest.PhasePrepaint)
	sv2.Prepaint(frame2, bounds2)

	frame2.SetPhase(elementtest.PhasePaint)
	sv2.Paint(frame2, bounds2)

	quads2 := frame2.Quads()
	var originY2 geometry.ScaledPixels
	found2 := false
	for _, q := range quads2 {
		if q.Background == childBg {
			originY2 = q.Bounds.Origin.Y
			found2 = true
			break
		}
	}
	if !found2 {
		t.Fatalf("frame 2: child quad not found")
	}

	scale := frame2.ScaleFactor()
	expectedDeltaY := geometry.ScaledPixels(100.0 * scale)
	actualDeltaY := originY1 - originY2
	if actualDeltaY != expectedDeltaY {
		t.Fatalf("expected vertical shift of %v, got %v (origin1: %v, origin2: %v)", expectedDeltaY, actualDeltaY, originY1, originY2)
	}
}

func TestScrollStateClamping(t *testing.T) {
	st := NewScrollState()
	st.SetOffset(-50)
	if st.Offset() != 0 {
		t.Errorf("expected negative offset to clamp to 0, got %v", st.Offset())
	}
	st.ScrollBy(100)
	if st.Offset() != 100 {
		t.Errorf("expected offset 100, got %v", st.Offset())
	}
	st.ScrollBy(-150)
	if st.Offset() != 0 {
		t.Errorf("expected offset to clamp to 0, got %v", st.Offset())
	}
}
