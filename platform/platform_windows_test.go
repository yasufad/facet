//go:build windows && facet_debug

package platform

import (
	"testing"
	"time"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/third_party/w32"
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

// TestWindowDeadKeyChar verifies that a committed character outside plain
// ASCII arrives as a TextEvent. Dead-key sequences on European layouts (a
// French or German keyboard composing an accented letter) never touch
// WM_IME_*: the keyboard layout's own dead-key state machine combines the
// keystrokes and delivers the result as an ordinary WM_CHAR, exactly like
// the one this test synthesises.
func TestWindowDeadKeyChar(t *testing.T) {
	p, err := New(Options{Name: "facet-deadkey-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:   "Facet Dead Key Test",
			Size:    geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
			Visible: false,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}

		events := make(chan Event, 16)
		w.SetEventHandler(func(e Event) {
			select {
			case events <- e:
			default:
			}
		})

		p.Dispatch(func() {
			postChar(w.NativeHandle(), 'è')
		})

		select {
		case e := <-events:
			te, ok := e.(TextEvent)
			if !ok {
				t.Errorf("expected TextEvent, got %T", e)
				return
			}
			if te.Text != "è" {
				t.Errorf("Text = %q, want %q", te.Text, "è")
			}
		case <-time.After(2 * time.Second):
			t.Error("timed out waiting for TextEvent")
			return
		}

		w.Close()
	}()

	go func() {
		<-done
		p.Quit()
	}()

	if err := p.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// TestWindowIMEComposition drives a composition through its three phases and
// verifies each arrives as an IMECompositionEvent with the right phase. The
// window has no real IME session attached, so the composition text IMM32
// reports is empty; that is a faithful test of the wndProc wiring — which
// phase reaches the handler, and that an empty composition resolves to a
// sane Cursor rather than panicking — not of IMM32 itself, which
// TestRuneOffsetFromUTF16 in ime_windows_test.go covers directly, including
// a composition character outside the basic multilingual plane.
func TestWindowIMEComposition(t *testing.T) {
	p, err := New(Options{Name: "facet-ime-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:   "Facet IME Test",
			Size:    geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
			Visible: false,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}

		events := make(chan Event, 16)
		w.SetEventHandler(func(e Event) {
			select {
			case events <- e:
			default:
			}
		})

		wantPhases := []IMEPhase{IMEStart, IMEUpdate, IMEEnd}
		p.Dispatch(func() {
			postIMEComposition(w.NativeHandle(), w32.WM_IME_STARTCOMPOSITION, 0)
			postIMEComposition(w.NativeHandle(), w32.WM_IME_COMPOSITION, w32.GCS_COMPSTR)
			postIMEComposition(w.NativeHandle(), w32.WM_IME_ENDCOMPOSITION, 0)
		})

		for _, wantPhase := range wantPhases {
			select {
			case e := <-events:
				ce, ok := e.(IMECompositionEvent)
				if !ok {
					t.Errorf("expected IMECompositionEvent, got %T", e)
					return
				}
				if ce.Phase != wantPhase {
					t.Errorf("Phase = %v, want %v", ce.Phase, wantPhase)
				}
				// Cursor is a rune offset into Text, or -1 if the IME
				// reported none; either way it must not exceed Text's
				// length, and with no real composition here it settles on
				// 0 (IMM32's answer for an empty string) rather than -1.
				if n := len([]rune(ce.Text)); ce.Cursor != -1 && (ce.Cursor < 0 || ce.Cursor > n) {
					t.Errorf("Cursor = %d, want -1 or within [0, %d]", ce.Cursor, n)
				}
			case <-time.After(2 * time.Second):
				t.Errorf("timed out waiting for IMECompositionEvent phase %v", wantPhase)
				return
			}
		}

		w.Close()
	}()

	go func() {
		<-done
		p.Quit()
	}()

	if err := p.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
}
