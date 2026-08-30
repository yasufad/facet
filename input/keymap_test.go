package input

import (
	"testing"
)

type actionA struct{}

func (actionA) ActionName() string { return "test::ActionA" }

type actionB struct{}

func (actionB) ActionName() string { return "test::ActionB" }

type actionC struct{}

func (actionC) ActionName() string { return "test::ActionC" }

// Hard Case 1: A chord bound in two contexts resolving to the inner one.
func TestChordBoundInTwoContextsResolvesToInner(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("ctrl-a", actionA{}, "Workspace"),
		MustKeyBinding("ctrl-a", actionB{}, "Editor"),
	)

	workspaceCtx, _ := ParseKeyContext("Workspace")
	editorCtx, _ := ParseKeyContext("Editor")
	stack := []KeyContext{workspaceCtx, editorCtx}

	ctrlA, _ := ParseKeySequence("ctrl-a")
	matches, pending := km.BindingsForInput(ctrlA, stack)

	if pending {
		t.Fatalf("expected pending to be false")
	}
	if len(matches) < 2 {
		t.Fatalf("expected matches, got %d", len(matches))
	}
	// The highest precedence match must be actionB (inner context Editor)
	if !ActionsEqual(matches[0].Action, actionB{}) {
		t.Fatalf("expected inner context actionB, got %v", matches[0].Action)
	}
	// Second match is actionA (outer context Workspace)
	if !ActionsEqual(matches[1].Action, actionA{}) {
		t.Fatalf("expected outer context actionA, got %v", matches[1].Action)
	}
}

// Hard Case 2: A chord bound only in an outer context still firing from within an inner one.
func TestChordBoundOnlyInOuterContextFiresFromInner(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("ctrl-s", actionA{}, "Workspace"),
	)

	workspaceCtx, _ := ParseKeyContext("Workspace")
	paneCtx, _ := ParseKeyContext("Pane")
	editorCtx, _ := ParseKeyContext("Editor")
	stack := []KeyContext{workspaceCtx, paneCtx, editorCtx}

	ctrlS, _ := ParseKeySequence("ctrl-s")
	matches, pending := km.BindingsForInput(ctrlS, stack)

	if pending {
		t.Fatalf("expected pending to be false")
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}
	if !ActionsEqual(matches[0].Action, actionA{}) {
		t.Fatalf("expected outer context actionA to fire, got %v", matches[0].Action)
	}
}

// Hard Case 3: An unbound chord falling through.
func TestUnboundChordFallsThrough(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("ctrl-s", actionA{}, "Workspace"),
		MustKeyBinding("ctrl-a", actionB{}, "Editor"),
	)

	workspaceCtx, _ := ParseKeyContext("Workspace")
	editorCtx, _ := ParseKeyContext("Editor")
	stack := []KeyContext{workspaceCtx, editorCtx}

	ctrlZ, _ := ParseKeySequence("ctrl-z")
	matches, pending := km.BindingsForInput(ctrlZ, stack)

	if pending {
		t.Fatalf("expected pending to be false")
	}
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches for unbound chord, got %d", len(matches))
	}
}

// Hard Case 4: A multi-key chord that is a prefix of another.
func TestMultiKeyChordPrefix(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("space", actionA{}, "Editor"),
		MustKeyBinding("space w w", actionB{}, "Workspace"),
	)

	workspaceCtx, _ := ParseKeyContext("Workspace")
	editorCtx, _ := ParseKeyContext("Editor")
	stack := []KeyContext{workspaceCtx, editorCtx}

	space, _ := ParseKeySequence("space")
	matches, pending := km.BindingsForInput(space, stack)

	if !pending {
		t.Fatalf("expected pending to be true because 'space w w' exists")
	}
	if len(matches) != 1 || !ActionsEqual(matches[0].Action, actionA{}) {
		t.Fatalf("expected exact match actionA, got %v", matches)
	}

	spaceW, _ := ParseKeySequence("space w")
	matches, pending = km.BindingsForInput(spaceW, stack)
	if !pending {
		t.Fatalf("expected pending to be true for 'space w'")
	}
	if len(matches) != 0 {
		t.Fatalf("expected 0 exact matches for 'space w', got %d", len(matches))
	}

	spaceWW, _ := ParseKeySequence("space w w")
	matches, pending = km.BindingsForInput(spaceWW, stack)
	if pending {
		t.Fatalf("expected pending to be false for 'space w w'")
	}
	if len(matches) != 1 || !ActionsEqual(matches[0].Action, actionB{}) {
		t.Fatalf("expected exact match actionB for 'space w w', got %v", matches)
	}
}

func TestNoActionSuppression(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("ctrl-x", actionA{}, "Editor"),
		MustKeyBinding("ctrl-x", NoAction{}, "Editor && mode==full"),
	)

	editorNormal, _ := ParseKeyContext("Editor mode=normal")
	editorFull, _ := ParseKeyContext("Editor mode=full")

	ctrlX, _ := ParseKeySequence("ctrl-x")

	// In mode=normal, actionA should match
	matches, _ := km.BindingsForInput(ctrlX, []KeyContext{editorNormal})
	if len(matches) != 1 || !ActionsEqual(matches[0].Action, actionA{}) {
		t.Fatalf("expected actionA in normal mode, got %v", matches)
	}

	// In mode=full, NoAction suppresses actionA
	matches, _ = km.BindingsForInput(ctrlX, []KeyContext{editorFull})
	if len(matches) != 0 {
		t.Fatalf("expected NoAction to suppress matches in full mode, got %d", len(matches))
	}
}

func TestTargetedUnbind(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("tab", actionA{}, "Editor"),
		MustKeyBinding("tab", Unbind{TargetAction: "test::ActionA"}, "Editor && edit_prediction"),
		MustKeyBinding("tab", actionB{}, "Editor && showing_completions"),
	)

	editorCtx, _ := ParseKeyContext("Editor edit_prediction showing_completions")
	tab, _ := ParseKeySequence("tab")

	matches, _ := km.BindingsForInput(tab, []KeyContext{editorCtx})
	// actionA is unbound, but actionB should still match
	if len(matches) != 1 || !ActionsEqual(matches[0].Action, actionB{}) {
		t.Fatalf("expected actionB to match while actionA is unbound, got %v", matches)
	}
}

func TestEqualDepthTieBreakingByLoadOrder(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("cmd-r", actionA{}, "Workspace"),
		MustKeyBinding("cmd-r", actionB{}, "Workspace"), // added later, should win tie
	)

	workspaceCtx, _ := ParseKeyContext("Workspace")
	cmdR, _ := ParseKeySequence("cmd-r")

	matches, _ := km.BindingsForInput(cmdR, []KeyContext{workspaceCtx})
	if len(matches) < 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}
	if !ActionsEqual(matches[0].Action, actionB{}) {
		t.Fatalf("expected later-added actionB to win equal-depth tie, got %v", matches[0].Action)
	}
}

func TestBindingsForAction(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("ctrl-s", actionA{}, "Workspace"),
		MustKeyBinding("cmd-s", actionA{}, "Workspace"),
		MustKeyBinding("ctrl-s", actionB{}, "Editor"), // shadows ctrl-s for actionA in Editor
	)

	workspaceCtx, _ := ParseKeyContext("Workspace")
	editorCtx, _ := ParseKeyContext("Editor")

	// In Workspace, both ctrl-s and cmd-s are valid for actionA
	bindings := km.BindingsForAction(actionA{}, []KeyContext{workspaceCtx})
	if len(bindings) != 2 {
		t.Fatalf("expected 2 bindings in Workspace, got %d", len(bindings))
	}

	// In Editor, ctrl-s is shadowed by actionB, so only cmd-s remains for actionA
	bindings = km.BindingsForAction(actionA{}, []KeyContext{workspaceCtx, editorCtx})
	if len(bindings) != 1 || bindings[0].KeystrokesString() != "super-s" {
		t.Fatalf("expected only cmd-s in Editor for actionA, got %v", bindings)
	}

	highest, ok := km.HighestPrecedenceBindingForAction(actionA{}, []KeyContext{workspaceCtx, editorCtx})
	if !ok || highest.KeystrokesString() != "super-s" {
		t.Fatalf("expected highest precedence binding cmd-s, got %v (ok=%v)", highest, ok)
	}
}

func TestPossibleNextBindings(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("ctrl-w left", actionA{}, "Editor"),
		MustKeyBinding("ctrl-w right", actionB{}, "Editor"),
		MustKeyBinding("ctrl-x 0", actionC{}, "Workspace"),
	)

	editorCtx, _ := ParseKeyContext("Editor")
	ctrlW, _ := ParseKeySequence("ctrl-w")

	next := km.PossibleNextBindingsForInput(ctrlW, []KeyContext{editorCtx})
	if len(next) != 2 {
		t.Fatalf("expected 2 possible next bindings for ctrl-w, got %d", len(next))
	}
}

func TestEmptyKeymap(t *testing.T) {
	km := NewKeymap()
	if km.Version() != 0 {
		t.Fatalf("expected initial version 0, got %d", km.Version())
	}
	if len(km.Bindings()) != 0 {
		t.Fatalf("expected empty bindings")
	}

	ctrlA, _ := ParseKeySequence("ctrl-a")
	matches, pending := km.BindingsForInput(ctrlA, nil)
	if len(matches) != 0 || pending {
		t.Fatalf("expected no matches and no pending on empty keymap")
	}
}
