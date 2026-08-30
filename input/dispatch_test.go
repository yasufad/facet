package input

import (
	"testing"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/platform"
)

func TestDispatchKeyActionResolution(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("ctrl-s", actionA{}, "Workspace"),
		MustKeyBinding("ctrl-s", actionB{}, "Editor"),
	)
	ft := NewFocusTree()
	dt := NewDispatchTree(km, ft)

	workspaceFocus := NewFocusID()
	editorFocus := NewFocusID()
	ft.SetParent(editorFocus, workspaceFocus)
	ft.Focus(editorFocus)

	workspaceCtx, _ := ParseKeyContext("Workspace")
	editorCtx, _ := ParseKeyContext("Editor")

	// Build dispatch tree: root (Workspace) -> child (Editor)
	dt.PushNode() // node 0: Workspace
	dt.SetFocusID(workspaceFocus)
	dt.SetContext(workspaceCtx)

	var workspaceHandled bool
	dt.OnAction("test::ActionA", func(action Action, phase DispatchPhase) bool {
		workspaceHandled = true
		return true
	})

	dt.PushNode() // node 1: Editor
	dt.SetFocusID(editorFocus)
	dt.SetContext(editorCtx)

	var editorHandled bool
	var editorPhase DispatchPhase
	dt.OnAction("test::ActionB", func(action Action, phase DispatchPhase) bool {
		editorHandled = true
		editorPhase = phase
		return true
	})

	dt.PopNode()
	dt.PopNode()

	event := platform.KeyEvent{
		Code:      platform.KeyS,
		Modifiers: platform.Control,
		Phase:     platform.KeyDown,
	}

	res := dt.DispatchKey(event)
	if !res.Handled {
		t.Fatalf("expected event to be handled")
	}
	if !ActionsEqual(res.ActionDispatched, actionB{}) {
		t.Fatalf("expected actionB to be dispatched, got %v", res.ActionDispatched)
	}
	if !editorHandled {
		t.Fatalf("expected editor handler to be invoked")
	}
	if editorPhase != Capture && editorPhase != Bubble {
		t.Fatalf("unexpected phase: %v", editorPhase)
	}
	if workspaceHandled {
		t.Fatalf("expected workspace handler to not be invoked because editor handled it")
	}
}

func TestDispatchCaptureAndBubblePhases(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("ctrl-s", actionA{}, ""),
	)
	ft := NewFocusTree()
	dt := NewDispatchTree(km, ft)

	rootFocus := NewFocusID()
	childFocus := NewFocusID()
	ft.SetParent(childFocus, rootFocus)
	ft.Focus(childFocus)

	var phaseOrder []string

	dt.PushNode() // root
	dt.SetFocusID(rootFocus)
	dt.OnAction("test::ActionA", func(action Action, phase DispatchPhase) bool {
		if phase == Capture {
			phaseOrder = append(phaseOrder, "root-capture")
		} else {
			phaseOrder = append(phaseOrder, "root-bubble")
		}
		return false // do not stop propagation
	})

	dt.PushNode() // child
	dt.SetFocusID(childFocus)
	dt.OnAction("test::ActionA", func(action Action, phase DispatchPhase) bool {
		if phase == Capture {
			phaseOrder = append(phaseOrder, "child-capture")
			return false // do not stop propagation in capture
		}
		phaseOrder = append(phaseOrder, "child-bubble")
		return true // stop propagation in bubble
	})
	dt.PopNode()
	dt.PopNode()

	event := platform.KeyEvent{
		Code:      platform.KeyS,
		Modifiers: platform.Control,
		Phase:     platform.KeyDown,
	}

	res := dt.DispatchKey(event)
	if !res.Handled {
		t.Fatalf("expected event to be handled")
	}

	expectedOrder := []string{"root-capture", "child-capture", "child-bubble"}
	if len(phaseOrder) != len(expectedOrder) {
		t.Fatalf("expected phase order %v, got %v", expectedOrder, phaseOrder)
	}
	for i := range phaseOrder {
		if phaseOrder[i] != expectedOrder[i] {
			t.Fatalf("step %d: got %s, want %s", i, phaseOrder[i], expectedOrder[i])
		}
	}
}

func TestDispatchMultiKeyChordWithReplay(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("ctrl-w left", actionA{}, ""),
		MustKeyBinding("ctrl-x", actionB{}, ""),
	)
	ft := NewFocusTree()
	dt := NewDispatchTree(km, ft)

	var actionDispatched Action
	dt.PushNode()
	dt.OnAction("", func(action Action, phase DispatchPhase) bool {
		actionDispatched = action
		return true
	})
	dt.PopNode()

	ctrlW := platform.KeyEvent{Code: platform.KeyW, Modifiers: platform.Control, Phase: platform.KeyDown}
	ctrlX := platform.KeyEvent{Code: platform.KeyX, Modifiers: platform.Control, Phase: platform.KeyDown}
	left := platform.KeyEvent{Code: platform.KeyArrowLeft, Modifiers: 0, Phase: platform.KeyDown}

	// 1. Send ctrl-w -> pending
	res := dt.DispatchKey(ctrlW)
	if !res.Pending {
		t.Fatalf("expected pending after ctrl-w")
	}

	// 2. Send left -> resolves actionA
	res = dt.DispatchKey(left)
	if res.Pending || !res.Handled || !ActionsEqual(res.ActionDispatched, actionA{}) || !ActionsEqual(actionDispatched, actionA{}) {
		t.Fatalf("expected actionA after ctrl-w left, got %+v (actionDispatched=%v)", res, actionDispatched)
	}

	// 3. Send ctrl-w (pending), then non-matching ctrl-x -> prefix fails, ctrl-x replayed and dispatches actionB
	dt.DispatchKey(ctrlW)
	res = dt.DispatchKey(ctrlX)
	if res.Pending || !res.Handled || !ActionsEqual(res.ActionDispatched, actionB{}) || !ActionsEqual(actionDispatched, actionB{}) {
		t.Fatalf("expected actionB after replaying ctrl-x, got %+v (actionDispatched=%v)", res, actionDispatched)
	}
	if len(res.ReplayedKeys) != 1 || res.ReplayedKeys[0].Code != platform.KeyW {
		t.Fatalf("expected ctrl-w in replayed keys, got %+v", res.ReplayedKeys)
	}
}

func TestDispatchPointerAndWheel(t *testing.T) {
	km := NewKeymap()
	ft := NewFocusTree()
	dt := NewDispatchTree(km, ft)

	var pointerHandledBy DispatchNodeID = -1
	var wheelHandledBy DispatchNodeID = -1

	dt.PushNode() // node 0: root
	dt.OnPointerEvent(func(event platform.PointerEvent, phase DispatchPhase) bool {
		if phase == Bubble {
			pointerHandledBy = 0
			return true
		}
		return false
	})

	child := dt.PushNode() // node 1: child
	dt.OnWheelEvent(func(event platform.WheelEvent, phase DispatchPhase) bool {
		if phase == Bubble {
			wheelHandledBy = 1
			return true
		}
		return false
	})
	dt.PopNode()
	dt.PopNode()

	pEvent := platform.PointerEvent{
		Position: geometry.NewPoint[geometry.DevicePixels](100, 100),
		Phase:    platform.PointerDown,
		Button:   platform.PointerLeft,
	}
	handled := dt.DispatchPointer(pEvent, child)
	if !handled || pointerHandledBy != 0 {
		t.Fatalf("expected pointer event to bubble to node 0, got %v (handled=%v)", pointerHandledBy, handled)
	}

	wEvent := platform.WheelEvent{
		Delta: platform.ScrollDelta{Unit: platform.ScrollLines, DeltaY: 3},
		Phase: platform.ScrollMoved,
	}
	handled = dt.DispatchWheel(wEvent, child)
	if !handled || wheelHandledBy != 1 {
		t.Fatalf("expected wheel event to be handled by node 1, got %v (handled=%v)", wheelHandledBy, handled)
	}
}

func TestDispatchTextEvent(t *testing.T) {
	km := NewKeymap()
	ft := NewFocusTree()
	dt := NewDispatchTree(km, ft)

	focusID := NewFocusID()
	ft.Focus(focusID)

	var textReceived string
	dt.PushNode()
	dt.SetFocusID(focusID)
	dt.OnTextEvent(func(event platform.TextEvent) bool {
		textReceived = event.Text
		return true
	})
	dt.PopNode()

	tEvent := platform.TextEvent{Text: "hello"}
	handled := dt.DispatchText(tEvent)
	if !handled || textReceived != "hello" {
		t.Fatalf("expected text 'hello' to be received, got %q (handled=%v)", textReceived, handled)
	}
}

func TestFlushPending(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("ctrl-w left", actionA{}, ""),
	)
	ft := NewFocusTree()
	dt := NewDispatchTree(km, ft)

	dt.DispatchKey(platform.KeyEvent{Code: platform.KeyW, Modifiers: platform.Control, Phase: platform.KeyDown})
	if len(dt.PendingKeystrokes()) != 1 {
		t.Fatalf("expected 1 pending keystroke")
	}

	flushed := dt.FlushPending()
	if len(flushed) != 1 || flushed[0].Code != platform.KeyW {
		t.Fatalf("expected flushed ctrl-w, got %v", flushed)
	}
	if len(dt.PendingKeystrokes()) != 0 {
		t.Fatalf("expected 0 pending after flush")
	}
}

func TestEmptyDispatchTree(t *testing.T) {
	dt := NewDispatchTree(nil, nil)
	var event platform.KeyEvent
	res := dt.DispatchKey(event)
	if res.Handled || res.Pending {
		t.Fatalf("expected empty dispatch to not be handled or pending")
	}

	exp := dt.ExplainKey(event)
	if exp.WinningBinding != nil {
		t.Fatalf("expected no winning binding on empty tree")
	}
}
