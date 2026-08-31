package elementtest

import (
	"testing"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/layout"
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
