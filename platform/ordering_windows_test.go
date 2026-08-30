//go:build windows && facet_debug

package platform

import (
	"testing"
	"time"

	"github.com/yasufad/facet/geometry"
)

// TestWindowCreatedBeforeRunWorks verifies the natural ordering every user
// will write first: New, then NewWindow, then Run. Before the fix that
// moved the dispatcher's hidden window back into New, NewWindow before Run
// blocked forever — there was no hidden window to post to.
//
// New and Run are called on the test goroutine. NewWindow is called before
// Run, on the same goroutine, and the window must exist and receive a
// message once the loop is pumping.
func TestWindowCreatedBeforeRunWorks(t *testing.T) {
	p, err := New(Options{Name: "facet-ordering-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Create the window before Run. This is the ordering the assignment
	// asks for, and the first thing anyone tries.
	w, err := p.NewWindow(WindowOptions{
		Title:     "Facet Ordering Test",
		Size:      geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
		Visible:   false,
		Resizable: true,
		Decorated: true,
	})
	if err != nil {
		t.Fatalf("NewWindow before Run: %v", err)
	}

	if w.NativeHandle() == 0 {
		t.Fatal("NativeHandle returned 0")
	}

	// Install a handler so we can confirm the window receives a message
	// once the loop is running.
	events := make(chan Event, 16)
	w.SetEventHandler(func(e Event) {
		select {
		case events <- e:
		default:
		}
	})

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Post a mouse move now that the loop is pumping.
		p.Dispatch(func() {
			lParam := uintptr(10) | (uintptr(10) << 16)
			postMouseMove(w.NativeHandle(), lParam)
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
			t.Error("timed out waiting for input event")
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

// TestRunFromWrongGoroutinePanics verifies that calling Run from a different
// goroutine than the one that called New panics with a legible message,
// rather than hanging silently. A clear panic beats a silent hang.
func TestRunFromWrongGoroutinePanics(t *testing.T) {
	p, err := New(Options{Name: "facet-panic-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() {
			r := recover()
			if r == nil {
				t.Error("Run from wrong goroutine did not panic")
				return
			}
			msg, ok := r.(string)
			if !ok {
				t.Errorf("panic value is %T, want string", r)
				return
			}
			// The message must name the mistake so the user can fix it.
			if !contains(msg, "same goroutine") {
				t.Errorf("panic message %q does not mention the same-goroutine requirement", msg)
			}
		}()

		// Run on a different goroutine than New. This must panic.
		_ = p.Run()
	}()

	select {
	case <-done:
		// Good: the panic happened and was recovered.
	case <-time.After(2 * time.Second):
		t.Fatal("Run from wrong goroutine hung instead of panicking")
	}
}

// contains is a minimal strings.Contains for test use, avoiding an import.
func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
