package element

import (
	"testing"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/style"
)

func TestClickEventSynthesisation(t *testing.T) {
	frame := newFakeFrame()
	var clicked bool
	var clickedPos geometry.Point[geometry.Pixels]
	var clickedMod Modifiers

	btn := NewDiv().
		Width(style.Px(100)).
		Height(style.Px(40)).
		OnClick(func(event ClickEvent) bool {
			clicked = true
			clickedPos = event.Position
			clickedMod = event.Modifiers
			return true
		})

	// Lifecycle: RequestLayout -> Prepaint
	frame.phase = phaseLayoutRequested
	rootID := btn.RequestLayout(frame)
	frame.solve(rootID, 100, 40)
	rootBounds := frame.LayoutBounds(rootID)

	frame.phase = phasePrepainted
	btn.Prepaint(frame, rootBounds)

	if len(frame.hitRegions) != 1 {
		t.Fatalf("expected 1 hit region, got %d", len(frame.hitRegions))
	}
	if len(frame.dispatchNodes) != 1 {
		t.Fatalf("expected 1 dispatch node, got %d", len(frame.dispatchNodes))
	}

	dn := frame.dispatchNodes[0]
	if len(dn.ClickListeners) != 1 {
		t.Fatalf("expected 1 click listener on dispatch node, got %d", len(dn.ClickListeners))
	}

	// Simulate delivery of a synthesised ClickEvent
	ce := ClickEvent{
		Position:   geometry.Point[geometry.Pixels]{X: 50, Y: 20},
		Button:     MouseButtonLeft,
		Modifiers:  ModShift | ModControl,
		ClickCount: 1,
	}
	handled := dn.ClickListeners[0](ce)
	if !handled {
		t.Errorf("expected click listener to report handled")
	}
	if !clicked {
		t.Errorf("expected click handler to be invoked")
	}
	if clickedPos.X != 50 || clickedPos.Y != 20 {
		t.Errorf("clicked pos = %v, want (50, 20)", clickedPos)
	}
	if !clickedMod.Has(ModShift) || !clickedMod.Has(ModControl) {
		t.Errorf("clicked modifiers = %v, want Shift|Control", clickedMod)
	}
}

func TestDispatchNodeAtomicHandoff(t *testing.T) {
	frame := newFakeFrame()
	keyCtx := input.NewKeyContext()
	keyCtx.Set("Editor", "true")
	focusID := input.FocusID(42)

	child := NewDiv().
		TrackFocus(focusID).
		KeyContext(keyCtx).
		OnAction("test-action", func(action input.Action, phase input.DispatchPhase) bool {
			return true
		}).
		OnKeyDown(func(event platform.KeyEvent, phase input.DispatchPhase) bool {
			return true
		})

	parent := NewDiv().Children(child)

	frame.phase = phaseLayoutRequested
	rootID := parent.RequestLayout(frame)
	frame.solve(rootID, 200, 100)
	rootBounds := frame.LayoutBounds(rootID)

	frame.phase = phasePrepainted
	parent.Prepaint(frame, rootBounds)

	// Parent has no interactivity -> 0 dispatch nodes for parent.
	// Child has interactivity -> 1 dispatch node for child.
	if len(frame.dispatchNodes) != 1 {
		t.Fatalf("expected 1 dispatch node created atomically, got %d", len(frame.dispatchNodes))
	}
	dn := frame.dispatchNodes[0]
	if dn.FocusID != focusID {
		t.Errorf("dispatch node FocusID = %v, want %v", dn.FocusID, focusID)
	}
	if dn.KeyContext == nil {
		t.Fatalf("expected non-nil KeyContext")
	}
	val, ok := dn.KeyContext.Get("Editor")
	if !ok || val != "true" {
		t.Errorf("dispatch node KeyContext mismatch: %v, %v", val, ok)
	}
	if len(dn.ActionBindings) != 1 || dn.ActionBindings[0].ActionName != "test-action" {
		t.Errorf("dispatch node ActionBindings mismatch: %v", dn.ActionBindings)
	}

	// Verify dispatch node stack was fully popped at the end of prepaint
	if len(frame.dispatchNodeStack) != 0 {
		t.Errorf("expected dispatchNodeStack to be empty after prepaint, got %d", len(frame.dispatchNodeStack))
	}
}

func TestOccludeRegistersHitRegion(t *testing.T) {
	frame := newFakeFrame()
	box := NewDiv().Occlude()

	frame.phase = phaseLayoutRequested
	id := box.RequestLayout(frame)
	frame.solve(id, 100, 100)
	bounds := frame.LayoutBounds(id)

	frame.phase = phasePrepainted
	box.Prepaint(frame, bounds)

	if len(frame.hitRegions) != 1 {
		t.Fatalf("expected Occlude to register 1 hit region, got %d", len(frame.hitRegions))
	}
}

func TestHoverPseudoStyleTwoFrameLag(t *testing.T) {
	frame := newFakeFrame()
	red := colour.Rgba{R: 1, G: 0, B: 0, A: 1}
	blue := colour.Rgba{R: 0, G: 0, B: 1, A: 1}

	btn := NewDiv().
		Bg(red).
		Hover(func(r *style.Refinement) {
			r.SetBackground(blue)
		})

	// Frame 1: Initial layout and prepaint (not hovered in rendered frame)
	frame.phase = phaseLayoutRequested
	id1 := btn.RequestLayout(frame)
	frame.solve(id1, 100, 50)
	bounds1 := frame.LayoutBounds(id1)

	frame.phase = phasePrepainted
	btn.Prepaint(frame, bounds1)

	frame.phase = phasePainted
	btn.Paint(frame, bounds1)

	if len(frame.quads) != 1 || frame.quads[0].Background != red {
		t.Fatalf("frame 1: expected red background, got %v", frame.quads[0].Background)
	}

	hitRegionID := frame.hitRegions[0].id

	// Frame 2: Window reports hitRegionID is hovered in the rendered frame
	frame.hoveredHitRegions[hitRegionID] = true
	frame.quads = frame.quads[:0]

	btn2 := NewDiv().
		Bg(red).
		Hover(func(r *style.Refinement) {
			r.SetBackground(blue)
		})
	// Carry previous frame hit region ID into the element for simulation
	btn2.interactivity.hitRegionID = hitRegionID

	frame.phase = phaseLayoutRequested
	id2 := btn2.RequestLayout(frame)
	frame.solve(id2, 100, 50)
	bounds2 := frame.LayoutBounds(id2)

	frame.phase = phasePrepainted
	btn2.Prepaint(frame, bounds2)

	frame.phase = phasePainted
	btn2.Paint(frame, bounds2)

	if len(frame.quads) != 1 || frame.quads[0].Background != blue {
		t.Fatalf("frame 2: expected blue background from hover style, got %v", frame.quads[0].Background)
	}
}

func TestFocusPseudoStyle(t *testing.T) {
	frame := newFakeFrame()
	borderDefault := colour.Rgba{R: 0.5, G: 0.5, B: 0.5, A: 1}
	borderFocused := colour.Rgba{R: 0, G: 0, B: 1, A: 1}
	focusID := input.FocusID(10)

	inputField := NewDiv().
		Border(1).
		BorderColour(borderDefault).
		TrackFocus(focusID).
		Focus(func(r *style.Refinement) {
			r.SetBorderColour(borderFocused)
		})

	// Case 1: Not focused
	frame.phase = phaseLayoutRequested
	id := inputField.RequestLayout(frame)
	frame.solve(id, 200, 30)
	bounds := frame.LayoutBounds(id)

	frame.phase = phasePrepainted
	inputField.Prepaint(frame, bounds)

	frame.phase = phasePainted
	inputField.Paint(frame, bounds)

	if len(frame.quads) != 1 || frame.quads[0].BorderColour != borderDefault {
		t.Fatalf("unfocused: expected default border colour, got %v", frame.quads[0].BorderColour)
	}

	// Case 2: Focused in rendered frame
	frame.focusedIDs[focusID] = true
	frame.quads = frame.quads[:0]

	inputField2 := NewDiv().
		Border(1).
		BorderColour(borderDefault).
		TrackFocus(focusID).
		Focus(func(r *style.Refinement) {
			r.SetBorderColour(borderFocused)
		})

	frame.phase = phaseLayoutRequested
	id2 := inputField2.RequestLayout(frame)
	frame.solve(id2, 200, 30)
	bounds2 := frame.LayoutBounds(id2)

	frame.phase = phasePrepainted
	inputField2.Prepaint(frame, bounds2)

	frame.phase = phasePainted
	inputField2.Paint(frame, bounds2)

	if len(frame.quads) != 1 || frame.quads[0].BorderColour != borderFocused {
		t.Fatalf("focused: expected focused border colour, got %v", frame.quads[0].BorderColour)
	}
}
