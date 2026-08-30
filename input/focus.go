package input

import (
	"slices"
)

// FocusID uniquely identifies a focusable element across frames.
type FocusID uint64

var nextFocusID uint64

// NewFocusID generates a new unique FocusID.
func NewFocusID() FocusID {
	nextFocusID++
	return FocusID(nextFocusID)
}

// FocusTree manages keyboard focus state and the containment hierarchy of
// focusable nodes.
type FocusTree struct {
	focused  FocusID
	hasFocus bool
	parents  map[FocusID]FocusID // child -> parent
}

// NewFocusTree returns a new initialised FocusTree.
func NewFocusTree() *FocusTree {
	return &FocusTree{
		parents: make(map[FocusID]FocusID),
	}
}

// Focus moves keyboard focus to the node identified by id.
func (t *FocusTree) Focus(id FocusID) {
	t.focused = id
	t.hasFocus = true
}

// Blur clears keyboard focus so no node is focused.
func (t *FocusTree) Blur() {
	t.focused = 0
	t.hasFocus = false
}

// Focused returns the currently focused FocusID and true, or (0, false) if
// nothing is focused.
func (t *FocusTree) Focused() (FocusID, bool) {
	if !t.hasFocus {
		return 0, false
	}
	return t.focused, true
}

// SetParent records that child is contained within parent in the focus hierarchy.
func (t *FocusTree) SetParent(child, parent FocusID) {
	if child == 0 || child == parent {
		return
	}
	t.parents[child] = parent
}

// Parent returns the parent FocusID for child, or (0, false) if child has no
// recorded parent.
func (t *FocusTree) Parent(child FocusID) (FocusID, bool) {
	p, ok := t.parents[child]
	return p, ok
}

// Remove removes the node from the focus tree. If the removed node had focus,
// focus is cleared.
func (t *FocusTree) Remove(id FocusID) {
	delete(t.parents, id)
	if t.hasFocus && t.focused == id {
		t.Blur()
	}
}

// Contains reports whether parent is equal to or an ancestor of child in the
// focus hierarchy.
func (t *FocusTree) Contains(parent, child FocusID) bool {
	if parent == 0 || child == 0 {
		return false
	}
	if parent == child {
		return true
	}

	curr := child
	for {
		p, ok := t.parents[curr]
		if !ok || p == 0 {
			return false
		}
		if p == parent {
			return true
		}
		curr = p
	}
}

// FocusPath returns the sequence of FocusIDs from the root down to target
// (inclusive). If target has no ancestors in the tree, a slice containing just
// target is returned.
func (t *FocusTree) FocusPath(target FocusID) []FocusID {
	if target == 0 {
		return nil
	}

	var path []FocusID
	curr := target
	visited := make(map[FocusID]bool)

	for curr != 0 && !visited[curr] {
		visited[curr] = true
		path = append(path, curr)
		p, ok := t.parents[curr]
		if !ok {
			break
		}
		curr = p
	}

	slices.Reverse(path)
	return path
}

// Clear resets the focus tree, clearing all node relationships and focus state.
func (t *FocusTree) Clear() {
	t.Blur()
	clear(t.parents)
}
