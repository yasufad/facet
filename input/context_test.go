package input

import (
	"testing"
)

func TestKeyContext(t *testing.T) {
	ctx, err := ParseKeyContext("Editor mode=full vim=normal")
	if err != nil {
		t.Fatalf("ParseKeyContext error: %v", err)
	}

	if !ctx.Contains("Editor") {
		t.Fatalf("expected context to contain Editor")
	}
	if ctx.Contains("Terminal") {
		t.Fatalf("expected context to not contain Terminal")
	}

	mode, ok := ctx.Get("mode")
	if !ok || mode != "full" {
		t.Fatalf("expected mode=full, got %q (ok=%v)", mode, ok)
	}

	vim, ok := ctx.Get("vim")
	if !ok || vim != "normal" {
		t.Fatalf("expected vim=normal, got %q (ok=%v)", vim, ok)
	}

	ctx.Set("mode", "compact")
	mode, _ = ctx.Get("mode")
	if mode != "compact" {
		t.Fatalf("expected updated mode=compact, got %q", mode)
	}

	ctx.Add("Custom")
	if !ctx.Contains("Custom") {
		t.Fatalf("expected context to contain Custom")
	}

	entries := ctx.Entries()
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	var empty KeyContext
	if !empty.IsEmpty() {
		t.Fatalf("expected empty.IsEmpty() to be true")
	}
	if empty.String() != "" {
		t.Fatalf("expected empty.String() == %q, got %q", "", empty.String())
	}
}

func TestContextPredicateParsingAndMatching(t *testing.T) {
	workspaceCtx, _ := ParseKeyContext("Workspace")
	paneCtx, _ := ParseKeyContext("Pane")
	editorCtx, _ := ParseKeyContext("Editor mode=full")

	stack := []KeyContext{workspaceCtx, paneCtx, editorCtx}

	// 1. Simple Identifier
	pred, err := ParseContextPredicate("Editor")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	depth, matched := pred.Match(stack)
	if !matched || depth != 3 {
		t.Fatalf("expected Editor match at depth 3, got depth=%d, matched=%v", depth, matched)
	}

	// 2. Equality
	pred, err = ParseContextPredicate("mode == full")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	depth, matched = pred.Match(stack)
	if !matched || depth != 3 {
		t.Fatalf("expected mode == full match at depth 3, got depth=%d, matched=%v", depth, matched)
	}

	// 3. Inequality
	pred, err = ParseContextPredicate("mode != compact")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	depth, matched = pred.Match(stack)
	if !matched || depth != 3 {
		t.Fatalf("expected mode != compact match at depth 3, got depth=%d, matched=%v", depth, matched)
	}

	// 4. And combination
	pred, err = ParseContextPredicate("Editor && mode == full")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	depth, matched = pred.Match(stack)
	if !matched || depth != 3 {
		t.Fatalf("expected And match at depth 3, got depth=%d, matched=%v", depth, matched)
	}

	// 5. Descendant: Workspace > Editor
	pred, err = ParseContextPredicate("Workspace > Editor")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	depth, matched = pred.Match(stack)
	if !matched || depth != 3 {
		t.Fatalf("expected Workspace > Editor match at depth 3, got depth=%d, matched=%v", depth, matched)
	}

	// 6. Descendant: Workspace > Pane > Editor
	pred, err = ParseContextPredicate("Workspace > Pane > Editor")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	depth, matched = pred.Match(stack)
	if !matched || depth != 3 {
		t.Fatalf("expected Workspace > Pane > Editor match at depth 3, got depth=%d, matched=%v", depth, matched)
	}

	// 7. Non-matching descendant: Pane > Workspace
	pred, err = ParseContextPredicate("Pane > Workspace")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, matched = pred.Match(stack)
	if matched {
		t.Fatalf("expected Pane > Workspace to NOT match")
	}

	// 8. Negation: !Terminal (should match because Terminal is nowhere in stack)
	pred, err = ParseContextPredicate("!Terminal")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	depth, matched = pred.Match(stack)
	if !matched || depth != 3 {
		t.Fatalf("expected !Terminal to match, got depth=%d, matched=%v", depth, matched)
	}

	// 9. Negation: !Editor (should NOT match because Editor is in stack)
	pred, err = ParseContextPredicate("!Editor")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	_, matched = pred.Match(stack)
	if matched {
		t.Fatalf("expected !Editor to NOT match")
	}

	// 10. Complex parenthesized: (Workspace || Modal) > Editor && mode == full
	pred, err = ParseContextPredicate("(Workspace || Modal) > Editor && mode == full")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	depth, matched = pred.Match(stack)
	if !matched || depth != 3 {
		t.Fatalf("expected complex match at depth 3, got depth=%d, matched=%v", depth, matched)
	}
}

func TestContextPredicateIsSuperset(t *testing.T) {
	editor, _ := ParseContextPredicate("Editor")
	editorFull, _ := ParseContextPredicate("Editor && mode == full")
	terminal, _ := ParseContextPredicate("Terminal")
	orPred, _ := ParseContextPredicate("Editor || Terminal")

	if !editor.IsSuperset(editor) {
		t.Fatalf("expected Editor to be superset of Editor")
	}
	if !editor.IsSuperset(editorFull) {
		t.Fatalf("expected Editor to be superset of Editor && mode == full")
	}
	if editor.IsSuperset(terminal) {
		t.Fatalf("expected Editor to NOT be superset of Terminal")
	}
	if !orPred.IsSuperset(editor) {
		t.Fatalf("expected (Editor || Terminal) to be superset of Editor")
	}
}
