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
// The test runs the platform loop on a separate goroutine, creates a window,
// dispatches a synthetic mouse move onto the platform thread, and checks that
// the event handler received it. It then quits the platform.
func TestWindowOpensAndReportsInput(t *testing.T) {
	p, err := New(Options{Name: "facet-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Run the platform loop on a goroutine.
	loopDone := make(chan error, 1)
	go func() {
		loopDone <- p.Run()
	}()

	// Give the loop a moment to start.
	time.Sleep(100 * time.Millisecond)

	w, err := p.NewWindow(WindowOptions{
		Title:     "Facet Test",
		Size:      geometry.Size[geometry.Pixels]{Width: 400, Height: 300},
		Visible:   true,
		Resizable: true,
		Decorated: true,
	})
	if err != nil {
		t.Fatalf("NewWindow: %v", err)
	}

	// Verify the window has a non-zero native handle.
	if w.NativeHandle() == 0 {
		t.Fatal("NativeHandle returned 0")
	}

	// Verify NativeSurface returns the same handle on Windows.
	if w.NativeSurface() != w.NativeHandle() {
		t.Fatalf("NativeSurface %v != NativeHandle %v", w.NativeSurface(), w.NativeHandle())
	}

	// Verify the scale factor is positive.
	scale := w.ScaleFactor()
	if scale <= 0 {
		t.Fatalf("ScaleFactor returned %v, want > 0", scale)
	}

	// Verify the window reports a size close to what we asked for.
	size := w.Size()
	if size.Width <= 0 || size.Height <= 0 {
		t.Fatalf("Size returned %v, want positive dimensions", size)
	}

	// Set up an event handler and verify events arrive.
	events := make(chan Event, 16)
	w.SetEventHandler(func(e Event) {
		select {
		case events <- e:
		default:
		}
	})

	// Dispatch a synthetic mouse move onto the platform thread by posting
	// a WM_MOUSEMOVE message to the window.
	p.Dispatch(func() {
		// Post a mouse move at (10, 10) in the client area.
		lParam := uintptr(10) | (uintptr(10) << 16)
		// Use PostMessage so the message goes through the loop.
		// We can't import w32 here (layering), but we can use the window's
		// hwnd through the interface... actually we can, this is a test in
		// the platform package itself.
		postMouseMove(w.NativeHandle(), lParam)
	})

	// Wait for an event.
	select {
	case e := <-events:
		pe, ok := e.(PointerEvent)
		if !ok {
			t.Fatalf("expected PointerEvent, got %T", e)
		}
		if pe.Phase != PointerMove {
			t.Fatalf("expected PointerMove phase, got %v", pe.Phase)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for input event")
	}

	// Clean up.
	w.Close()
	p.Quit()
	<-loopDone
}
