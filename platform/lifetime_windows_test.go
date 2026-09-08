//go:build windows && facet_debug

package platform

import (
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/third_party/w32"
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

// TestLastWindowCloseStopsRun verifies that closing the last registered
// window stops the event loop and Run returns — without anyone calling
// Quit. Before the fix, WM_DESTROY removed the HWND from the map but never
// posted WM_QUIT, so the loop kept pumping an empty window map and Run never
// returned; the process stayed alive until Ctrl-C.
//
// The test fails by watchdog, not by the framework's 10-minute timeout: a
// helper goroutine waits five seconds for Run to return, and if it has not,
// marks the test failed and calls Quit to unblock Run so the test ends in
// seconds rather than minutes. With the fix, Run returns as soon as the
// posted WM_CLOSE destroys the window and unregisterWindow finds the map
// empty.
func TestLastWindowCloseStopsRun(t *testing.T) {
	p, err := New(Options{Name: "facet-last-window-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	runReturned := make(chan struct{})

	go func() {
		w, err := p.NewWindow(WindowOptions{
			Title:   "Facet Last Window Test",
			Size:    geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
			Visible: false,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}

		// Close the window by posting WM_CLOSE to its HWND. This drives the
		// real close path: WM_CLOSE -> DestroyWindow -> WM_DESTROY ->
		// unregisterWindow -> (with the fix) dispatcher.Quit.
		p.Dispatch(func() {
			postClose(w.NativeHandle())
		})

		// Watchdog: if Run does not return within five seconds, the
		// last-window fix is absent. Mark the test failed and force Quit so
		// Run unblocks and the test ends promptly.
		select {
		case <-runReturned:
			// Run returned on its own — the fix works.
		case <-time.After(5 * time.Second):
			t.Error("Run did not return after the last window was closed")
			p.Quit()
		}
	}()

	if err := p.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	close(runReturned)
}

// TestQuitHandlerVetoesSessionEnd verifies that SetQuitHandler is consulted on
// WM_QUERYENDSESSION — the Windows quit-request event — and that returning
// false vetoes the session end. Before the fix, SetQuitHandler stored a
// handler no code read, so this test would have no way to observe a veto.
//
// WM_QUERYENDSESSION's return value is the veto signal: 0 means veto,
// non-zero means allow. The test sends WM_QUERYENDSESSION directly to the
// window's HWND and reads the result from SendMessage, which is the
// synchronous return value of the wndproc. With a handler returning false,
// the result must be 0; with no handler, non-zero.
func TestQuitHandlerVetoesSessionEnd(t *testing.T) {
	p, err := New(Options{Name: "facet-quit-handler-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var called atomic.Bool

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:   "Facet Quit Handler Test",
			Size:    geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
			Visible: false,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}
		hwnd := w32.HWND(w.NativeHandle())

		p.SetQuitHandler(func() bool {
			called.Store(true)
			return false // veto
		})

		p.Dispatch(func() {
			// SendMessage returns the wndproc's return value directly.
			// WM_QUERYENDSESSION with a vetoing handler must return 0.
			result := w32.SendMessage(hwnd, w32.WM_QUERYENDSESSION, 0, 0)
			if result != 0 {
				t.Errorf("WM_QUERYENDSESSION returned %v, want 0 (veto)", result)
			}
			if !called.Load() {
				t.Error("quit handler was not called for WM_QUERYENDSESSION")
			}

			// Now remove the handler: with no handler,
			// WM_QUERYENDSESSION must allow (return non-zero).
			p.SetQuitHandler(nil)
			result = w32.SendMessage(hwnd, w32.WM_QUERYENDSESSION, 0, 0)
			if result == 0 {
				t.Error("WM_QUERYENDSESSION returned 0 with no handler, want non-zero (allow)")
			}
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
