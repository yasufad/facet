package elementtest

import (
	"testing"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
)

func TestElementTestFrameLifecycle(t *testing.T) {
	frame := NewFrame()
	if frame == nil {
		t.Fatal("expected NewFrame to return non-nil Frame")
	}

	// 1. Layout phase
	st := layout.NewStyle()
	st.Size = layout.Size[layout.Dimension]{
		Width:  layout.DimLength(200),
		Height: layout.DimLength(100),
	}
	nodeID := frame.RequestLayout(st, nil)

	frame.Solve(nodeID, 200, 100)
	bounds := frame.LayoutBounds(nodeID)
	if bounds.Size.Width != 200 || bounds.Size.Height != 100 {
		t.Fatalf("expected solved bounds size (200, 100), got %v", bounds.Size)
	}

	// 2. Prepaint phase
	frame.SetPhase(PhasePrepaint)
	clicked := false
	var receivedEvt element.ClickEvent
	node := element.DispatchNode{
		ClickListeners: []func(element.ClickEvent) bool{
			func(e element.ClickEvent) bool {
				clicked = true
				receivedEvt = e
				return true
			},
		},
	}
	dispatchNodeID := frame.PushDispatchNode(node)
	hitID := frame.RegisterHitRegion(bounds, dispatchNodeID)
	frame.PopDispatchNode()

	if hitID == 0 {
		t.Fatal("expected non-zero hit region ID")
	}

	// 3. Paint phase
	frame.SetPhase(PhasePaint)
	frame.SetHovered(hitID, true)
	if !frame.IsHovered(hitID) {
		t.Fatal("expected hit region to be reported as hovered")
	}

	var r style.Refinement
	r.SetTextColour(colour.Rgba{R: 1, G: 0, B: 0, A: 1})
	frame.PushTextStyle(r)
	if frame.TextStyle().Colour.R != 1 {
		t.Fatalf("expected text style colour red, got %v", frame.TextStyle().Colour)
	}
	frame.PopTextStyle()

	// Simulate click
	clickPos := geometry.NewPoint[geometry.Pixels](50, 50)
	handled := frame.SimulateClick(dispatchNodeID, clickPos, element.MouseButtonLeft, 0)
	if !handled || !clicked {
		t.Fatal("expected click event to be simulated and handled")
	}
	if receivedEvt.Position != clickPos {
		t.Fatalf("expected click position %v, got %v", clickPos, receivedEvt.Position)
	}
	if receivedEvt.LocalPosition != clickPos { // origin is (0, 0)
		t.Fatalf("expected local click position %v, got %v", clickPos, receivedEvt.LocalPosition)
	}
}

func TestTabNavigationInTreeOrder(t *testing.T) {
	frame := NewFrame()
	focusA := input.NewFocusID()
	focusB := input.NewFocusID()
	focusC := input.NewFocusID()

	frame.SetPhase(PhasePrepaint)
	frame.PushDispatchNode(element.DispatchNode{FocusID: focusA, TabStop: true})
	frame.PopDispatchNode()
	frame.PushDispatchNode(element.DispatchNode{FocusID: focusB, TabStop: true})
	frame.PopDispatchNode()
	frame.PushDispatchNode(element.DispatchNode{FocusID: focusC, TabStop: true})
	frame.PopDispatchNode()

	frame.SetPhase(PhasePaint)

	tabOrder := frame.TabOrder()
	if len(tabOrder) != 3 {
		t.Fatalf("expected 3 tab stops, got %d", len(tabOrder))
	}
	if tabOrder[0] != focusA || tabOrder[1] != focusB || tabOrder[2] != focusC {
		t.Fatalf("expected tab order [A, B, C], got %v", tabOrder)
	}

	// 1. Initial Tab: moves to A
	frame.SimulateTab(false)
	if f, _ := frame.Focused(); f != focusA {
		t.Fatalf("expected focus on A, got %v", f)
	}

	// 2. Tab: moves to B
	frame.SimulateTab(false)
	if f, _ := frame.Focused(); f != focusB {
		t.Fatalf("expected focus on B, got %v", f)
	}

	// 3. Tab: moves to C
	frame.SimulateTab(false)
	if f, _ := frame.Focused(); f != focusC {
		t.Fatalf("expected focus on C, got %v", f)
	}

	// 4. Tab at end: wraps to A
	frame.SimulateTab(false)
	if f, _ := frame.Focused(); f != focusA {
		t.Fatalf("expected focus wrapped to A, got %v", f)
	}

	// 5. Shift+Tab at start: wraps to C
	frame.SimulateTab(true)
	if f, _ := frame.Focused(); f != focusC {
		t.Fatalf("expected focus wrapped to C on Shift-Tab, got %v", f)
	}

	// 6. Shift+Tab: moves back to B
	frame.SimulateTab(true)
	if f, _ := frame.Focused(); f != focusB {
		t.Fatalf("expected focus on B on Shift-Tab, got %v", f)
	}
}

func TestTabNavigationWithExplicitIndices(t *testing.T) {
	frame := NewFrame()
	focusA := input.NewFocusID()
	focusB := input.NewFocusID()
	focusC := input.NewFocusID()
	focusD := input.NewFocusID()

	frame.SetPhase(PhasePrepaint)
	frame.PushDispatchNode(element.DispatchNode{FocusID: focusA, TabIndex: 2})
	frame.PopDispatchNode()
	frame.PushDispatchNode(element.DispatchNode{FocusID: focusB, TabIndex: 1})
	frame.PopDispatchNode()
	frame.PushDispatchNode(element.DispatchNode{FocusID: focusC, TabIndex: 0, TabStop: true})
	frame.PopDispatchNode()
	frame.PushDispatchNode(element.DispatchNode{FocusID: focusD, TabStop: false})
	frame.PopDispatchNode()

	frame.SetPhase(PhasePaint)

	tabOrder := frame.TabOrder()
	if len(tabOrder) != 3 {
		t.Fatalf("expected 3 tab stops (D excluded), got %d: %v", len(tabOrder), tabOrder)
	}
	// Expected order: B (index 1), A (index 2), C (index 0 / tree order)
	if tabOrder[0] != focusB || tabOrder[1] != focusA || tabOrder[2] != focusC {
		t.Fatalf("expected tab order [B, A, C], got %v", tabOrder)
	}
}

func TestClippingMaskIntersection(t *testing.T) {
	frame := NewFrame()
	frame.SetScaleFactor(2.0)
	frame.SetPhase(PhasePaint)

	clipBounds := geometry.NewBounds(
		geometry.NewPoint[geometry.Pixels](10, 10),
		geometry.NewSize[geometry.Pixels](50, 50),
	)
	frame.PushClip(clipBounds)

	quadBounds := geometry.Bounds[geometry.ScaledPixels]{
		Origin: geometry.NewPoint[geometry.ScaledPixels](0, 0),
		Size:   geometry.NewSize[geometry.ScaledPixels](200, 200),
	}
	frame.InsertQuad(scene.Quad{
		Bounds: quadBounds,
	})
	frame.PopClip()

	quads := frame.Quads()
	if len(quads) != 1 {
		t.Fatalf("expected 1 quad, got %d", len(quads))
	}
	q := quads[0]
	expectedMask := geometry.Bounds[geometry.ScaledPixels]{
		Origin: geometry.NewPoint[geometry.ScaledPixels](20, 20),  // 10 * 2.0
		Size:   geometry.NewSize[geometry.ScaledPixels](100, 100), // 50 * 2.0
	}
	if q.ContentMask.Bounds != expectedMask {
		t.Fatalf("expected intersected content mask %v, got %v", expectedMask, q.ContentMask.Bounds)
	}
}
