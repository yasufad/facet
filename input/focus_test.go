package input

import (
	"testing"
)

func TestFocusTree(t *testing.T) {
	tree := NewFocusTree()

	if _, ok := tree.Focused(); ok {
		t.Fatalf("expected initial focused to be false")
	}

	root := NewFocusID()
	panel := NewFocusID()
	editor := NewFocusID()
	unrelated := NewFocusID()

	tree.SetParent(panel, root)
	tree.SetParent(editor, panel)

	if !tree.Contains(root, editor) {
		t.Fatalf("expected root to contain editor")
	}
	if !tree.Contains(panel, editor) {
		t.Fatalf("expected panel to contain editor")
	}
	if !tree.Contains(editor, editor) {
		t.Fatalf("expected editor to contain editor")
	}
	if tree.Contains(editor, root) {
		t.Fatalf("expected editor to not contain root")
	}
	if tree.Contains(unrelated, editor) {
		t.Fatalf("expected unrelated to not contain editor")
	}

	path := tree.FocusPath(editor)
	expectedPath := []FocusID{root, panel, editor}
	if len(path) != len(expectedPath) {
		t.Fatalf("expected path len %d, got %d", len(expectedPath), len(path))
	}
	for i := range path {
		if path[i] != expectedPath[i] {
			t.Fatalf("path[%d] = %v, want %v", i, path[i], expectedPath[i])
		}
	}

	tree.Focus(editor)
	focused, ok := tree.Focused()
	if !ok || focused != editor {
		t.Fatalf("expected focused editor, got %v (ok=%v)", focused, ok)
	}

	tree.Blur()
	if _, ok := tree.Focused(); ok {
		t.Fatalf("expected blur to clear focus")
	}

	tree.Focus(editor)
	tree.Remove(editor)
	if _, ok := tree.Focused(); ok {
		t.Fatalf("expected removing focused node to clear focus")
	}
}

func TestEmptyFocusTree(t *testing.T) {
	tree := NewFocusTree()
	if len(tree.FocusPath(0)) != 0 {
		t.Fatalf("expected empty path for 0 ID")
	}
	if tree.Contains(0, 0) {
		t.Fatalf("expected Contains(0,0) to be false")
	}
}
