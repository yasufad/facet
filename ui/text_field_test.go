package ui

import (
	"testing"
	"time"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/element/elementtest"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/style"
)

func TestTextFieldLifecycleAndQuads(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[TextFieldState]) TextFieldState {
		return NewTextFieldState("Hello")
	})
	defer state.Release()

	tf := NewTextField(a, state)
	frame := elementtest.NewFrame()

	rootID := tf.RequestLayout(frame)
	frame.Solve(rootID, 200, 40)

	bounds := frame.LayoutBounds(rootID)
	frame.SetPhase(elementtest.PhasePrepaint)
	tf.Prepaint(frame, bounds)

	nodes := frame.DispatchNodes()
	if len(nodes) == 0 {
		t.Fatalf("expected at least one dispatch node from TextField")
	}
	node := nodes[0]
	if node.Cursor != style.CursorText {
		t.Errorf("expected CursorText, got %v", node.Cursor)
	}
	if !node.TabStop {
		t.Errorf("expected TabStop to be true")
	}
	if len(node.TextListeners) == 0 {
		t.Fatalf("expected text listener registered on dispatch node")
	}
	if len(node.KeyListeners) == 0 {
		t.Fatalf("expected key listener registered on dispatch node")
	}

	frame.SetPhase(elementtest.PhasePaint)
	tf.Paint(frame, bounds)

	quads := frame.Quads()
	// Should have container background quad + caret quad
	if len(quads) < 2 {
		t.Fatalf("expected at least 2 quads (container + caret), got %d", len(quads))
	}
	if quads[0].Background != defaultTextFieldBg {
		t.Errorf("expected container background %v, got %v", defaultTextFieldBg, quads[0].Background)
	}
	// Last quad is the caret quad
	caretQuad := quads[len(quads)-1]
	if caretQuad.Background != defaultTextFieldCaret {
		t.Errorf("expected caret background %v, got %v", defaultTextFieldCaret, caretQuad.Background)
	}

	sprites := frame.MonochromeSprites()
	if len(sprites) == 0 {
		t.Fatalf("expected text sprites for label 'Hello'")
	}
}

func TestTextFieldTypingAndCaret(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[TextFieldState]) TextFieldState {
		return NewTextFieldState("")
	})
	defer state.Release()

	tf := NewTextField(a, state)
	frame := elementtest.NewFrame()

	rootID := tf.RequestLayout(frame)
	frame.Solve(rootID, 200, 40)
	bounds := frame.LayoutBounds(rootID)

	frame.SetPhase(elementtest.PhasePrepaint)
	tf.Prepaint(frame, bounds)

	frame.SetPhase(elementtest.PhasePaint)
	tf.Paint(frame, bounds)

	// Simulate typing "Facet"
	node := frame.DispatchNodes()[0]
	handled := node.TextListeners[0](input.TextEvent{Text: "Facet"})
	if !handled {
		t.Fatalf("expected text listener to handle text event")
	}

	a.Flush()

	st := state.Read(a)
	if st.Text() != "Facet" {
		t.Fatalf("expected text 'Facet', got %q", st.Text())
	}
	if st.Cursor() != 5 {
		t.Fatalf("expected cursor 5, got %d", st.Cursor())
	}
}

func TestTextFieldArrowKeysAndSelection(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[TextFieldState]) TextFieldState {
		return NewTextFieldState("World")
	})
	defer state.Release()

	tf := NewTextField(a, state)
	frame := elementtest.NewFrame()

	rootID := tf.RequestLayout(frame)
	frame.Solve(rootID, 200, 40)
	bounds := frame.LayoutBounds(rootID)

	frame.SetPhase(elementtest.PhasePrepaint)
	tf.Prepaint(frame, bounds)

	frame.SetPhase(elementtest.PhasePaint)
	tf.Paint(frame, bounds)

	node := frame.DispatchNodes()[0]
	kl := node.KeyListeners[0]

	// Move Left without shift
	leftKey := input.KeyEvent{Code: platform.KeyArrowLeft, Phase: platform.KeyDown}
	kl(leftKey, input.Bubble)
	a.Flush()
	if st := state.Read(a); st.Cursor() != 4 || st.HasSelection() {
		t.Fatalf("after left arrow: cursor=%d, hasSelection=%v, want 4, false", st.Cursor(), st.HasSelection())
	}

	// Move Left with Shift (selects one character)
	shiftLeftKey := input.KeyEvent{Code: platform.KeyArrowLeft, Modifiers: platform.Shift, Phase: platform.KeyDown}
	kl(shiftLeftKey, input.Bubble)
	a.Flush()
	if st := state.Read(a); st.Cursor() != 3 || !st.HasSelection() {
		t.Fatalf("after shift-left arrow: cursor=%d, hasSelection=%v, want 3, true", st.Cursor(), st.HasSelection())
	}
	if st := state.Read(a); true {
		start, end := st.Selection()
		if start != 3 || end != 4 {
			t.Fatalf("expected selection [3, 4], got [%d, %d]", start, end)
		}
	}

	// Home key with Shift (selects all preceding text)
	shiftHomeKey := input.KeyEvent{Code: platform.KeyHome, Modifiers: platform.Shift, Phase: platform.KeyDown}
	kl(shiftHomeKey, input.Bubble)
	a.Flush()
	if st := state.Read(a); st.Cursor() != 0 {
		t.Fatalf("after shift-home: cursor=%d, want 0", st.Cursor())
	}
	if st := state.Read(a); true {
		start, end := st.Selection()
		if start != 0 || end != 4 {
			t.Fatalf("expected selection [0, 4], got [%d, %d]", start, end)
		}
	}

	// Select All (Ctrl-A)
	ctrlAKey := input.KeyEvent{Code: platform.KeyA, Modifiers: platform.Control, Phase: platform.KeyDown}
	kl(ctrlAKey, input.Bubble)
	a.Flush()
	if st := state.Read(a); true {
		start, end := st.Selection()
		if start != 0 || end != 5 {
			t.Fatalf("expected select all [0, 5], got [%d, %d]", start, end)
		}
	}

	// When painted with selection, selection quad should be rendered
	frame2 := elementtest.NewFrame()
	tf2 := NewTextField(a, state)
	root2 := tf2.RequestLayout(frame2)
	frame2.Solve(root2, 200, 40)
	bounds2 := frame2.LayoutBounds(root2)
	frame2.SetPhase(elementtest.PhasePrepaint)
	tf2.Prepaint(frame2, bounds2)
	frame2.SetPhase(elementtest.PhasePaint)
	tf2.Paint(frame2, bounds2)

	quads := frame2.Quads()
	// Should have container background, selection quad, caret quad
	hasSelectionQuad := false
	for _, q := range quads {
		if q.Background == defaultTextFieldSelection {
			hasSelectionQuad = true
			if q.Bounds.Size.Width <= 0 {
				t.Fatalf("expected positive selection quad width, got %v", q.Bounds.Size.Width)
			}
			break
		}
	}
	if !hasSelectionQuad {
		t.Fatalf("expected selection quad rendered into scene")
	}
}

func TestTextFieldBackspaceAndDelete(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[TextFieldState]) TextFieldState {
		return NewTextFieldState("ABC")
	})
	defer state.Release()

	tf := NewTextField(a, state)
	frame := elementtest.NewFrame()
	rootID := tf.RequestLayout(frame)
	frame.Solve(rootID, 200, 40)
	bounds := frame.LayoutBounds(rootID)
	frame.SetPhase(elementtest.PhasePrepaint)
	tf.Prepaint(frame, bounds)
	frame.SetPhase(elementtest.PhasePaint)
	tf.Paint(frame, bounds)

	node := frame.DispatchNodes()[0]
	kl := node.KeyListeners[0]

	// Backspace at end removes 'C'
	backspace := input.KeyEvent{Code: platform.KeyBackspace, Phase: platform.KeyDown}
	kl(backspace, input.Bubble)
	a.Flush()
	if st := state.Read(a); st.Text() != "AB" || st.Cursor() != 2 {
		t.Fatalf("expected 'AB' cursor 2, got %q cursor %d", st.Text(), st.Cursor())
	}

	// Move Left to between 'A' and 'B'
	left := input.KeyEvent{Code: platform.KeyArrowLeft, Phase: platform.KeyDown}
	kl(left, input.Bubble)
	a.Flush()

	// Delete removes 'B'
	del := input.KeyEvent{Code: platform.KeyDelete, Phase: platform.KeyDown}
	kl(del, input.Bubble)
	a.Flush()
	if st := state.Read(a); st.Text() != "A" || st.Cursor() != 1 {
		t.Fatalf("expected 'A' cursor 1, got %q cursor %d", st.Text(), st.Cursor())
	}

	// Type replaces with text
	node.TextListeners[0](input.TextEvent{Text: "X"})
	a.Flush()
	if st := state.Read(a); st.Text() != "AX" || st.Cursor() != 2 {
		t.Fatalf("expected 'AX' cursor 2, got %q cursor %d", st.Text(), st.Cursor())
	}
}

func TestTextFieldClickToPlaceCaret(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[TextFieldState]) TextFieldState {
		return NewTextFieldState("Testing Caret Placement")
	})
	defer state.Release()

	tf := NewTextField(a, state)
	frame := elementtest.NewFrame()
	rootID := tf.RequestLayout(frame)
	frame.Solve(rootID, 300, 40)
	bounds := frame.LayoutBounds(rootID)

	frame.SetPhase(elementtest.PhasePrepaint)
	tf.Prepaint(frame, bounds)

	frame.SetPhase(elementtest.PhasePaint)
	tf.Paint(frame, bounds)

	// Click near the start (local X = 10, paddingX = 8 -> localTextX = 2)
	clickPt := geometry.NewPoint[geometry.Pixels](bounds.Origin.X+10, bounds.Origin.Y+15)
	frame.DispatchPointer(elementtest.PointerDown, clickPt, element.MouseButtonLeft, 0)
	frame.DispatchPointer(elementtest.PointerUp, clickPt, element.MouseButtonLeft, 0)
	a.Flush()

	st := state.Read(a)
	if st.Cursor() > 2 {
		t.Fatalf("expected cursor near 0, got %d", st.Cursor())
	}
}

func TestTextFieldPointerDragSelection(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[TextFieldState]) TextFieldState {
		return NewTextFieldState("Hello World")
	})
	defer state.Release()

	tf := NewTextField(a, state)
	frame := elementtest.NewFrame()
	rootID := tf.RequestLayout(frame)
	frame.Solve(rootID, 300, 40)
	bounds := frame.LayoutBounds(rootID)

	frame.SetPhase(elementtest.PhasePrepaint)
	tf.Prepaint(frame, bounds)

	frame.SetPhase(elementtest.PhasePaint)
	tf.Paint(frame, bounds)

	node := frame.DispatchNodes()[0]
	pl := node.PointerListeners[0]

	// Pointer down at start
	downEvt := input.PointerEvent{
		Phase:    platform.PointerDown,
		Position: geometry.Point[geometry.DevicePixels]{X: 10, Y: 15},
		Button:   platform.PointerLeft,
		Buttons:  platform.ButtonLeft,
		Time:     time.Now(),
	}
	pl(downEvt, input.Bubble)
	a.Flush()

	// Pointer move dragging toward the right (X = 80)
	moveEvt := input.PointerEvent{
		Phase:    platform.PointerMove,
		Position: geometry.Point[geometry.DevicePixels]{X: 80, Y: 15},
		Buttons:  platform.ButtonLeft,
		Time:     time.Now(),
	}
	pl(moveEvt, input.Bubble)
	a.Flush()

	st := state.Read(a)
	if !st.HasSelection() {
		t.Fatalf("expected drag to produce selection, got cursor=%d anchor=%d", st.Cursor(), st.selectionAnchor)
	}

	// Pointer up finishes drag
	upEvt := input.PointerEvent{
		Phase:    platform.PointerUp,
		Position: geometry.Point[geometry.DevicePixels]{X: 80, Y: 15},
		Button:   platform.PointerLeft,
		Buttons:  0,
		Time:     time.Now(),
	}
	pl(upEvt, input.Bubble)
	a.Flush()

	st = state.Read(a)
	if st.isDragging {
		t.Fatalf("expected isDragging to be false after PointerUp")
	}
}

func TestTextFieldDisabled(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[TextFieldState]) TextFieldState {
		return NewTextFieldState("Disabled")
	})
	defer state.Release()

	tf := NewTextField(a, state).Disabled(true)
	frame := elementtest.NewFrame()
	rootID := tf.RequestLayout(frame)
	frame.Solve(rootID, 200, 40)
	bounds := frame.LayoutBounds(rootID)

	frame.SetPhase(elementtest.PhasePrepaint)
	tf.Prepaint(frame, bounds)

	node := frame.DispatchNodes()[0]
	if node.Cursor != style.CursorNotAllowed {
		t.Fatalf("expected CursorNotAllowed for disabled text field, got %v", node.Cursor)
	}
	if len(node.TextListeners) != 0 || len(node.KeyListeners) != 0 {
		t.Fatalf("expected no text or key listeners on disabled text field")
	}

	frame.SetPhase(elementtest.PhasePaint)
	tf.Paint(frame, bounds)

	// Caret quad should NOT be painted
	quads := frame.Quads()
	for _, q := range quads {
		if q.Background == defaultTextFieldCaret {
			t.Fatalf("disabled text field must not render a caret quad")
		}
	}
}

func TestTextFieldCustomRefinement(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	state := app.New(a, func(cx *app.Context[TextFieldState]) TextFieldState {
		return NewTextFieldState("Custom")
	})
	defer state.Release()

	customBg := colour.Rgba{R: 0.1, G: 0.2, B: 0.3, A: 1.0}
	var ref style.Refinement
	ref.SetBackground(customBg)

	tf := NewTextField(a, state).Refine(ref)
	frame := elementtest.NewFrame()
	rootID := tf.RequestLayout(frame)
	frame.Solve(rootID, 200, 40)
	bounds := frame.LayoutBounds(rootID)

	frame.SetPhase(elementtest.PhasePrepaint)
	tf.Prepaint(frame, bounds)

	frame.SetPhase(elementtest.PhasePaint)
	tf.Paint(frame, bounds)

	quads := frame.Quads()
	if len(quads) == 0 || quads[0].Background != customBg {
		t.Fatalf("expected custom background %v, got %v", customBg, quads[0].Background)
	}
}
