package platformtest

import (
	"testing"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/platform"
)

// TestWindowImplementsInterface verifies that Window satisfies the
// platform.Window interface at compile time through the assertion in
// window.go, and exercises the round-trip methods a test would use: SetSize
// followed by Size, SetCursor recording the shape, and SetEventHandler
// storing the closure so a test can call it.
func TestWindowRoundTrips(t *testing.T) {
	w := NewWindow(geometry.Size[geometry.Pixels]{Width: 100, Height: 80}, 2.0)

	if w.Size().Width != 100 || w.Size().Height != 80 {
		t.Errorf("Size = %v, want 100x80", w.Size())
	}
	if w.ScaleFactor() != 2.0 {
		t.Errorf("ScaleFactor = %v, want 2.0", w.ScaleFactor())
	}

	w.SetSize(geometry.Size[geometry.Pixels]{Width: 200, Height: 160})
	if w.Size().Width != 200 || w.Size().Height != 160 {
		t.Errorf("Size after SetSize = %v, want 200x160", w.Size())
	}

	w.SetCursor(platform.CursorDefault)
	w.SetCursor(platform.CursorText)
	if len(w.Cursors) != 2 {
		t.Errorf("Cursors has %d entries, want 2", len(w.Cursors))
	}
	if w.Cursors[0] != platform.CursorDefault || w.Cursors[1] != platform.CursorText {
		t.Errorf("Cursors = %v, want [Default, Text]", w.Cursors)
	}

	var fired bool
	w.SetEventHandler(func(platform.Event) { fired = true })
	if w.EventHandler == nil {
		t.Fatal("EventHandler not stored")
	}
	w.EventHandler(nil)
	if !fired {
		t.Error("EventHandler closure did not fire")
	}

	w.SetCloseHandler(func() bool { return false })
	if w.CloseHandler == nil {
		t.Fatal("CloseHandler not stored")
	}
}
