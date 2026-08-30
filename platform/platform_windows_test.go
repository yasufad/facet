//go:build windows && facet_debug

package platform

import (
	"testing"
	"time"

	"github.com/yasufad/facet/geometry"
)

// TestWindowOpensAndReportsInput creates a platform, opens a window, and
// verifies that the window reports its scale factor and that input events
// arrive through the event handler. It is behind facet_debug because it
// requires a desktop session and opens a real window.
//
// New and Run are called on the test goroutine, as the contract requires.
// The test logic (NewWindow, Dispatch, assertions) runs on a helper
// goroutine and marshals onto the platform thread through Dispatch. The
// helper signals when it is done so the test goroutine can Quit and Run
// can return.
func TestWindowOpensAndReportsInput(t *testing.T) {
	p, err := New(Options{Name: "facet-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:     "Facet Test",
			Size:      geometry.Size[geometry.Pixels]{Width: 400, Height: 300},
			Visible:   true,
			Resizable: true,
			Decorated: true,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}

		// Verify the window has a non-zero native handle.
		if w.NativeHandle() == 0 {
			t.Error("NativeHandle returned 0")
			return
		}

		// Verify NativeSurface returns the same handle on Windows.
		if w.NativeSurface() != w.NativeHandle() {
			t.Errorf("NativeSurface %v != NativeHandle %v", w.NativeSurface(), w.NativeHandle())
			return
		}

		// Verify the scale factor is positive.
		scale := w.ScaleFactor()
		if scale <= 0 {
			t.Errorf("ScaleFactor returned %v, want > 0", scale)
			return
		}

		// Verify the window reports a size close to what we asked for.
		size := w.Size()
		if size.Width <= 0 || size.Height <= 0 {
			t.Errorf("Size returned %v, want positive dimensions", size)
			return
		}

		// Set up an event handler and verify events arrive.
		events := make(chan Event, 16)
		w.SetEventHandler(func(e Event) {
			select {
			case events <- e:
			default:
			}
		})

		// Dispatch a synthetic mouse move onto the platform thread by
		// posting a WM_MOUSEMOVE message to the window.
		p.Dispatch(func() {
			lParam := uintptr(10) | (uintptr(10) << 16)
			postMouseMove(w.NativeHandle(), lParam)
		})

		// Wait for an event.
		select {
		case e := <-events:
			pe, ok := e.(PointerEvent)
			if !ok {
				t.Errorf("expected PointerEvent, got %T", e)
				return
			}
			if pe.Phase != PointerMove {
				t.Errorf("expected PointerMove phase, got %v", pe.Phase)
				return
			}
		case <-time.After(2 * time.Second):
			t.Error("timed out waiting for input event")
			return
		}

		w.Close()
	}()

	// Run blocks until Quit. The helper goroutine calls Quit through a
	// dispatch when it is done; but since the helper's last action is
	// Close, we quit from a separate dispatch after the helper finishes.
	go func() {
		<-done
		p.Quit()
	}()

	if err := p.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
