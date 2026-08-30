//go:build windows && facet_debug

package platform

import (
	"runtime"
	"testing"
	"time"

	"github.com/yasufad/facet/geometry"
)

// TestWindowSurvivesGCAfterHandleDropped verifies that a window is not
// collected while the OS still holds a reference to it through its HWND.
//
// Before the fix that introduced the platform's window map, the only
// reference to a *windowsWindow was a uintptr stored in GWLP_USERDATA —
// invisible to the garbage collector. A caller that dropped its Window
// handle (entirely legal: call Show, keep nothing) would leave the wndproc
// dereferencing freed memory on the next WM_* message. This test fails
// under that scheme and passes once the platform keeps the Go pointer in a
// map the collector can see.
//
// New and Run are called on the test goroutine, as the contract requires.
// The test logic runs on a helper goroutine and marshals onto the platform
// thread through Dispatch.
func TestWindowSurvivesGCAfterHandleDropped(t *testing.T) {
	p, err := New(Options{Name: "facet-lifetime-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:     "Facet Lifetime Test",
			Size:      geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
			Visible:   false,
			Resizable: true,
			Decorated: true,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}

		// Capture the HWND before dropping the reference.
		hwnd := w.NativeHandle()
		if hwnd == 0 {
			t.Error("NativeHandle returned 0")
			return
		}

		// Install a handler that signals when a message arrives.
		events := make(chan Event, 16)
		w.SetEventHandler(func(e Event) {
			select {
			case events <- e:
			default:
			}
		})

		// Drop every Go reference to the window and force collection. If
		// the platform does not keep the window alive, the GC frees it
		// here.
		w = nil
		runtime.GC()
		runtime.GC() // run twice to let finalisers from the first pass complete

		// Post a mouse move to the window by HWND. If the window was
		// collected, this dereferences freed memory inside the wndproc —
		// sometimes a crash, sometimes silent corruption. The handler
		// firing proves the window is alive.
		p.Dispatch(func() {
			lParam := uintptr(5) | (uintptr(5) << 16)
			postMouseMove(hwnd, lParam)
		})

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
			t.Error("handler did not fire after GC; window was collected while the OS still holds its HWND")
			return
		}

		// Clean up by HWND. Close dispatches WM_CLOSE, which destroys the
		// window and removes it from the platform's map.
		p.Dispatch(func() {
			postClose(hwnd)
		})
	}()

	go func() {
		<-done
		// Give the close dispatch a moment to process before quitting.
		time.Sleep(100 * time.Millisecond)
		p.Quit()
	}()

	if err := p.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
