//go:build windows && facet_debug

package platform

import (
	"testing"
	"time"

	"github.com/yasufad/facet/geometry"
)

// TestNewWithZeroValueOptions drives the whole sequence a user actually
// writes, on one goroutine, with zero-value options:
//
//	p, err := New(Options{})
//	w, err := p.NewWindow(WindowOptions{Title: "hello", Size: ...})
//	w.Show()
//	p.Run()
//
// This is the shape of the first program anyone writes against this package.
// Before the fix that defaulted Options.Name, New(Options{}) panicked because
// the empty name became the Win32 window class name and CreateWindowEx
// failed. Every other test supplied a name, so the default path was the only
// untested case.
func TestNewWithZeroValueOptions(t *testing.T) {
	p, err := New(Options{})
	if err != nil {
		t.Fatalf("New(Options{}): %v", err)
	}

	w, err := p.NewWindow(WindowOptions{
		Title:     "hello",
		Size:      geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
		Visible:   false,
		Resizable: true,
		Decorated: true,
	})
	if err != nil {
		t.Fatalf("NewWindow: %v", err)
	}

	if w.NativeHandle() == 0 {
		t.Fatal("NativeHandle returned 0")
	}

	// Show the window, as a user would before calling Run.
	w.Show()

	// Install a handler so we can confirm the window receives a message
	// once the loop is pumping.
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
