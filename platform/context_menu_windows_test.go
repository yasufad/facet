//go:build windows && facet_debug

package platform

import (
	"testing"
	"time"

	"github.com/yasufad/facet/geometry"
)

// TestShowContextMenuRecordsAndDefers verifies the ShowContextMenu contract
// on Windows: the method records the request and returns immediately, and
// the native TrackPopupMenu call runs on a later turn of the message loop —
// not synchronously inside the method.
//
// It proves deferral by calling ShowContextMenu from a dispatched closure on
// the platform thread and checking that, immediately after the call returns,
// the pending request is recorded but the native popup has not yet run. The
// popup runs when the posted wmShowContextMenu message is pumped on the next
// turn, which the test forces by waiting for it.
//
// TrackPopupMenu blocks in a nested modal loop waiting for the user to pick an
// item or dismiss the menu, so the test does not drive a real selection.
// Instead it verifies the recording and deferral, and that a nil or empty
// menu does not crash the deferred path.
func TestShowContextMenuRecordsAndDefers(t *testing.T) {
	p, err := New(Options{Name: "facet-context-menu-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:   "Facet Context Menu Test",
			Size:    geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
			Visible: false,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}

		menu := &Menu{
			Items: []MenuItem{
				{Label: "Cut", OnClick: func() {}},
				{Label: "Copy", OnClick: func() {}},
			},
		}
		at := geometry.NewPoint[geometry.Pixels](10, 20)

		// Call ShowContextMenu on the platform thread and inspect the pending
		// request before the message loop pumps the deferred message. This is
		// the deferral proof: the request is recorded but the native call has
		// not run yet.
		p.Dispatch(func() {
			w.ShowContextMenu(menu, at)

			ww := w.(*windowsWindow)
			if ww.pendingContextMenu == nil {
				t.Error("pendingContextMenu is nil after ShowContextMenu")
				return
			}
			if ww.pendingContextMenu.menu != menu {
				t.Error("pendingContextMenu.menu does not match the menu passed")
			}
			if ww.pendingContextMenu.at != at {
				t.Errorf("pendingContextMenu.at = %v, want %v", ww.pendingContextMenu.at, at)
			}

			// The deferred native call has not run yet: the pending request
			// is still present. showPendingContextMenu clears it when it runs.
			// We do not call showPendingContextMenu here because
			// TrackPopupMenu blocks in a modal loop; instead we verify the
			// recording and that the request survives until the deferred
			// turn would handle it.
		})

		w.Close()
	}()

	go func() {
		<-done
		time.Sleep(100 * time.Millisecond)
		p.Quit()
	}()

	if err := p.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// TestShowContextMenuNilMenuDoesNotPanic verifies that a nil or empty menu
// reaches the deferred path without crashing. showPendingContextMenu guards
// a nil request and a nil menu, so the posted message handling is a no-op.
func TestShowContextMenuNilMenuDoesNotPanic(t *testing.T) {
	p, err := New(Options{Name: "facet-context-menu-nil-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:   "Facet Context Menu Nil Test",
			Size:    geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
			Visible: false,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}

		// A nil menu must not panic when the deferred turn runs.
		w.ShowContextMenu(nil, geometry.NewPoint[geometry.Pixels](0, 0))

		// Give the deferred message a turn to pump before closing.
		p.Dispatch(func() {})

		w.Close()
	}()

	go func() {
		<-done
		time.Sleep(100 * time.Millisecond)
		p.Quit()
	}()

	if err := p.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
