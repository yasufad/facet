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

// TestListDeclaresExtentAndVirtualises verifies the two load-bearing
// properties of the virtual list:
//
//  1. The scroll range matches the entity's contentHeight even though only a
//     subset of items is built — because the container declares its extent as
//     a definite height rather than deriving it from built children.
//  2. Only the visible items are built.
//
// Two frames are run because the first frame has no recorded viewport height
// and builds nothing; Paint records it, and the second frame virtualises.
func TestListDeclaresExtentAndVirtualises(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[ListState]) ListState {
		return NewListState()
	})

	const count = 100
	itemHeight := geometry.Pixels(30)
	declaredExtent := geometry.Pixels(count) * itemHeight // 3000

	var built []int
	list := NewList(a, state).
		Count(count).
		ItemHeight(itemHeight).
		Render(func(i int) element.Element {
			built = append(built, i)
			return element.NewDiv().
				Height(style.Px(itemHeight)).
				Bg(colour.Rgba{R: 0.5, G: 0.5, B: 0.5, A: 1.0})
		})

	// Frame 1: bootstrap. Viewport height is unknown, so no items are
	// built. The container's definite height still gives the correct
	// extent, and Paint records the viewport height for frame 2.
	frame1 := elementtest.NewFrame()
	root1 := list.RequestLayout(frame1)
	frame1.Solve(root1, 200, 150)
	bounds1 := frame1.LayoutBounds(root1)
	frame1.SetPhase(elementtest.PhasePrepaint)
	list.Prepaint(frame1, bounds1)
	frame1.SetPhase(elementtest.PhasePaint)
	list.Paint(frame1, bounds1)

	st := state.Read(a)
	if st.viewportHeight != 150 {
		t.Fatalf("frame 1: expected viewport height 150, got %v", st.viewportHeight)
	}
	if st.contentHeight != declaredExtent {
		t.Fatalf("frame 1: expected content height %v, got %v", declaredExtent, st.contentHeight)
	}

	// Frame 2: virtualise. The list reads the recorded viewport height and
	// builds only the visible items.
	built = nil
	frame2 := elementtest.NewFrame()
	root2 := list.RequestLayout(frame2)
	frame2.Solve(root2, 200, 150)
	bounds2 := frame2.LayoutBounds(root2)
	frame2.SetPhase(elementtest.PhasePrepaint)
	list.Prepaint(frame2, bounds2)
	frame2.SetPhase(elementtest.PhasePaint)
	list.Paint(frame2, bounds2)

	st = state.Read(a)

	// The scroll range matches the entity's contentHeight.
	expectedMaxOffset := declaredExtent - 150 // 2850
	if st.MaxOffset() != expectedMaxOffset {
		t.Fatalf("expected max offset %v, got %v (contentHeight=%v, viewportHeight=%v)",
			expectedMaxOffset, st.MaxOffset(), st.contentHeight, st.viewportHeight)
	}

	// Only a subset of items was built — not all 100.
	if len(built) == 0 {
		t.Fatalf("frame 2: expected some items built, got 0")
	}
	if len(built) >= count {
		t.Fatalf("expected virtualised subset, built all %d items", count)
	}
	// 150px / 30px = 5 items visible (indices 0–4).
	if len(built) != 5 {
		t.Fatalf("expected 5 items built, got %d (%v)", len(built), built)
	}
	for i, idx := range built {
		if idx != i {
			t.Fatalf("expected built indices 0..4, got %v", built)
		}
	}
}

// TestListScrollRangeMatchesExtentAfterScroll verifies that scrolling past
// the first viewport still produces the correct scroll range and builds only
// the visible subset at the new offset.
func TestListScrollRangeMatchesExtentAfterScroll(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[ListState]) ListState {
		return NewListState()
	})

	const count = 100
	itemHeight := geometry.Pixels(30)
	declaredExtent := geometry.Pixels(count) * itemHeight

	var built []int
	list := NewList(a, state).
		Count(count).
		ItemHeight(itemHeight).
		Render(func(i int) element.Element {
			built = append(built, i)
			return element.NewDiv().Height(style.Px(itemHeight))
		})

	// Bootstrap frame.
	frame1 := elementtest.NewFrame()
	root1 := list.RequestLayout(frame1)
	frame1.Solve(root1, 200, 150)
	bounds1 := frame1.LayoutBounds(root1)
	frame1.SetPhase(elementtest.PhasePrepaint)
	list.Prepaint(frame1, bounds1)
	frame1.SetPhase(elementtest.PhasePaint)
	list.Paint(frame1, bounds1)

	// Scroll to offset 600 (item 20).
	state.Update(a, func(st *ListState, cx *app.Context[ListState]) {
		st.SetOffset(600)
	})

	// Frame 2 at offset 600: items 20–24 visible.
	built = nil
	frame2 := elementtest.NewFrame()
	root2 := list.RequestLayout(frame2)
	frame2.Solve(root2, 200, 150)
	bounds2 := frame2.LayoutBounds(root2)
	frame2.SetPhase(elementtest.PhasePrepaint)
	list.Prepaint(frame2, bounds2)
	frame2.SetPhase(elementtest.PhasePaint)
	list.Paint(frame2, bounds2)

	st := state.Read(a)
	expectedMaxOffset := declaredExtent - 150
	if st.MaxOffset() != expectedMaxOffset {
		t.Fatalf("expected max offset %v, got %v", expectedMaxOffset, st.MaxOffset())
	}
	if len(built) != 5 {
		t.Fatalf("expected 5 items built at offset 600, got %d (%v)", len(built), built)
	}
	if built[0] != 20 {
		t.Fatalf("expected first built index 20, got %d", built[0])
	}
}

// TestListWheelScrollUpdatesOffset verifies that wheel events update the
// list's scroll offset through the entity.
func TestListWheelScrollUpdatesOffset(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[ListState]) ListState {
		return NewListState()
	})

	list := NewList(a, state).
		Count(100).
		ItemHeight(geometry.Pixels(30)).
		Render(func(i int) element.Element {
			return element.NewDiv().Height(style.Px(30))
		})

	// Bootstrap to record viewport height.
	frame1 := elementtest.NewFrame()
	root1 := list.RequestLayout(frame1)
	frame1.Solve(root1, 200, 150)
	bounds1 := frame1.LayoutBounds(root1)
	frame1.SetPhase(elementtest.PhasePrepaint)
	list.Prepaint(frame1, bounds1)
	frame1.SetPhase(elementtest.PhasePaint)
	list.Paint(frame1, bounds1)

	wheelEvt := platform.WheelEvent{
		Position: geometry.Point[geometry.DevicePixels]{X: 50, Y: 50},
		Delta: platform.ScrollDelta{
			Unit:   platform.ScrollPixels,
			DeltaY: 42.0,
		},
	}
	for _, n := range frame1.DispatchNodes() {
		for _, wl := range n.WheelListeners {
			wl(wheelEvt, input.Bubble)
		}
	}

	offset := state.Read(a).Offset()
	if offset != 42.0 {
		t.Fatalf("expected scroll offset 42, got %v", offset)
	}
}

// TestListPaintDoesNotNotify verifies that recording metrics during paint
// does not notify, matching the paint-phase rule in ui/doc.go.
func TestListPaintDoesNotNotify(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[ListState]) ListState {
		return NewListState()
	})
	defer state.Release()

	notifications := 0
	sub := a.Observe(state.AnyEntity(), func(app *app.App) bool {
		notifications++
		return true
	})
	defer sub.Close()

	list := NewList(a, state).
		Count(100).
		ItemHeight(geometry.Pixels(30)).
		Render(func(i int) element.Element {
			return element.NewDiv().Height(style.Px(30))
		})

	frame := elementtest.NewFrame()
	root := list.RequestLayout(frame)
	frame.Solve(root, 200, 150)
	bounds := frame.LayoutBounds(root)
	frame.SetPhase(elementtest.PhasePrepaint)
	list.Prepaint(frame, bounds)
	frame.SetPhase(elementtest.PhasePaint)
	list.Paint(frame, bounds)

	a.Flush()

	st := state.Read(a)
	if st.viewportHeight != 150 || st.contentHeight != 3000 {
		t.Fatalf("expected metrics recorded (viewport=150, content=3000), got viewport=%v, content=%v",
			st.viewportHeight, st.contentHeight)
	}
	if notifications != 0 {
		t.Fatalf("List.Paint notified entity state (%d notifications); paint-phase notify triggers infinite frame loops",
			notifications)
	}
}
