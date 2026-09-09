//go:build windows && facet_debug

package platform

import (
	"testing"
	"time"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/third_party/w32"
)

// TestHideAndShowChangeWindowVisibility verifies that Hide and Show act on
// every window the backend owns, and that a window's visibility actually
// changes — asked of the OS through IsWindowVisible, the way
// TestSetApplicationMenuAttachesToWindow asks GetMenu rather than reading
// our own record.
//
// Before the fix, Hide and Show were no-ops that returned without doing
// anything: a promise that compiled. The test fails by checking the OS's
// own visibility state after each call, so a no-op stub leaves the window
// visible after Hide and the test catches it.
func TestHideAndShowChangeWindowVisibility(t *testing.T) {
	p, err := New(Options{Name: "facet-visibility-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:     "Facet Visibility Test",
			Size:      geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
			Visible:   true,
			Decorated: true,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}
		hwnd := w32.HWND(w.NativeHandle())

		// Confirm the window is visible before Hide. Visible: true sets
		// WS_VISIBLE at creation, so IsWindowVisible must report true.
		p.Dispatch(func() {
			if !w32.IsWindowVisible(hwnd) {
				t.Error("window is not visible before Hide")
				return
			}
		})

		// Hide all windows and verify the OS reports the window invisible.
		// p.Hide dispatches onto the platform thread; the Dispatch below
		// runs after it because the dispatcher is FIFO.
		p.Hide()
		p.Dispatch(func() {
			if w32.IsWindowVisible(hwnd) {
				t.Error("window is still visible after Hide")
				return
			}
		})

		// Show all windows and verify the OS reports it visible again.
		p.Show()
		p.Dispatch(func() {
			if !w32.IsWindowVisible(hwnd) {
				t.Error("window is not visible after Show")
				return
			}
		})

		// Close by posting WM_CLOSE rather than calling w.Close, which
		// uses SendMessage. SendMessage is a sent message, which Win32
		// processes before posted messages — so it would destroy the
		// window before the dispatch closures above run. PostMessage
		// queues WM_CLOSE after the closures, so they all run first.
		p.Dispatch(func() {
			postClose(hwnd)
		})
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
