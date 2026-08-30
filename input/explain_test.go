package input

import (
	"strings"
	"testing"

	"github.com/yasufad/facet/platform"
)

func TestExplainKeymap(t *testing.T) {
	km := NewKeymap(
		MustKeyBinding("ctrl-s", actionA{}, "Workspace"),
		MustKeyBinding("ctrl-s", actionB{}, "Editor"),
		MustKeyBinding("ctrl-s", actionC{}, "Picker"),
	)

	workspaceCtx, _ := ParseKeyContext("Workspace")
	editorCtx, _ := ParseKeyContext("Editor")
	stack := []KeyContext{workspaceCtx, editorCtx}

	ctrlS, _ := ParseKeySequence("ctrl-s")
	cands := km.Explain(ctrlS, stack)

	if len(cands) != 3 {
		t.Fatalf("expected 3 candidates explained, got %d", len(cands))
	}

	// actionB (Editor) should be winner
	if !cands[1].Winner || !cands[1].Matched {
		t.Fatalf("expected actionB to be winner, got %+v", cands[1])
	}
	// actionA (Workspace) should be matched but shadowed
	if cands[0].Winner || !cands[0].Matched {
		t.Fatalf("expected actionA to be matched but shadowed, got %+v", cands[0])
	}
	// actionC (Picker) should not be matched
	if cands[2].Matched {
		t.Fatalf("expected actionC to not be matched, got %+v", cands[2])
	}

	exp := DispatchExplanation{
		Event: platform.KeyEvent{
			Code:      platform.KeyS,
			Modifiers: platform.Control,
			Phase:     platform.KeyDown,
		},
		Keystroke:      ctrlS[0],
		ContextStack:   stack,
		Candidates:     cands,
		WinningBinding: &cands[1].Binding,
		TargetNode:     2,
		HandlerFound:   true,
		Handled:        true,
	}

	report := exp.String()
	if !strings.Contains(report, "ctrl-s") || !strings.Contains(report, "WINNER") || !strings.Contains(report, "SHADOWED") || !strings.Contains(report, "MISMATCH") {
		t.Fatalf("expected diagnostic keywords in report, got:\n%s", report)
	}
}

func TestEmptyDispatchExplanation(t *testing.T) {
	var empty DispatchExplanation
	report := empty.String()
	if !strings.Contains(report, "no matching keybindings") || !strings.Contains(report, "fell through") {
		t.Fatalf("expected fallthrough report for empty explanation, got:\n%s", report)
	}
}
