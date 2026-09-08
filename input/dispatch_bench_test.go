package input

import "testing"

// BenchmarkDispatchPointerMove measures the path a user writes: a pointer
// move dispatched through a realistically deep tree, with no listener
// consuming the event, so the full capture-then-bubble walk runs on every
// iteration. Mouse motion fires this on every move, not every click, which
// is what makes per-dispatch allocations on this path felt.
//
// The tree is linear (each node the sole child of the previous one) at a
// depth that is realistic for a nested UI: a window, a workspace, a pane,
// a panel, a scroll view, a list, an item, a row, a label. Ten levels.
func BenchmarkDispatchPointerMove(b *testing.B) {
	km := NewKeymap()
	ft := NewFocusTree()
	dt := NewDispatchTree(km, ft)

	const depth = 10
	var leaf DispatchNodeID
	for i := 0; i < depth; i++ {
		leaf = dt.PushNode()
		dt.OnPointerEvent(func(event PointerEvent, phase DispatchPhase) bool {
			return false // not handled: full capture + bubble walk
		})
	}
	for i := 0; i < depth; i++ {
		dt.PopNode()
	}

	event := PointerEvent{Phase: PointerMove}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		dt.DispatchPointer(event, leaf)
	}
}
