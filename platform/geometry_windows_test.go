//go:build windows && facet_debug

package platform

import (
	"math"
	"testing"
	"time"

	"github.com/yasufad/facet/geometry"
)

// TestNewWindowHonoursClientSize asserts that a window created with a
// known client size reports that size back through Size(). Before the fix
// that used AdjustWindowRectExForDpi in the creation path, the client size
// was smaller than asked because the frame was subtracted from it instead
// of added around it.
//
// The test checks two sizes — 640×480 and 300×200 — at whatever scale
// factor the test machine reports. Both fail before the fix.
func TestNewWindowHonoursClientSize(t *testing.T) {
	p, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cases := []struct {
		name string
		size geometry.Size[geometry.Pixels]
	}{
		{"640x480", geometry.Size[geometry.Pixels]{Width: 640, Height: 480}},
		{"300x200", geometry.Size[geometry.Pixels]{Width: 300, Height: 200}},
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		for _, c := range cases {
			w, err := p.NewWindow(WindowOptions{
				Title:     c.name,
				Size:      c.size,
				Visible:   false,
				Resizable: true,
				Decorated: true,
			})
			if err != nil {
				t.Errorf("NewWindow %s: %v", c.name, err)
				continue
			}

			got := w.Size()
			// Allow one logical pixel of rounding: the conversion through
			// device pixels and back can lose a fraction.
			if !approxEqual(got.Width, c.size.Width, 1) || !approxEqual(got.Height, c.size.Height, 1) {
				t.Errorf("client size %s: got %v, want %v", c.name, got, c.size)
			}
			w.Close()
		}
	}()

	go func() {
		<-done
		p.Quit()
	}()

	if err := p.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// TestSetSizePreservesPosition asserts that SetSize does not move the
// window. Before the fix that replaced MoveWindow(-1, -1, ...) with
// SetWindowPos(SWP_NOMOVE), SetSize teleported the window to the
// top-left corner of the screen.
func TestSetSizePreservesPosition(t *testing.T) {
	p, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:     "position test",
			Size:      geometry.Size[geometry.Pixels]{Width: 300, Height: 200},
			Position:  geometry.Point[geometry.Pixels]{X: 200, Y: 150},
			Visible:   false,
			Resizable: true,
			Decorated: true,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}

		before := w.Position()
		w.SetSize(geometry.Size[geometry.Pixels]{Width: 400, Height: 300})
		// Give the window a moment to process the resize.
		time.Sleep(50 * time.Millisecond)
		after := w.Position()

		if !approxEqual(after.X, before.X, 1) || !approxEqual(after.Y, before.Y, 1) {
			t.Errorf("SetSize moved the window: before %v, after %v", before, after)
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

// TestSetSizeRoundTrip asserts that SetSize(s) followed by Size() returns
// s, at whatever scale factor the test machine reports.
func TestSetSizeRoundTrip(t *testing.T) {
	p, err := New(Options{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:     "round-trip test",
			Size:      geometry.Size[geometry.Pixels]{Width: 300, Height: 200},
			Visible:   false,
			Resizable: true,
			Decorated: true,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}

		want := geometry.Size[geometry.Pixels]{Width: 500, Height: 350}
		w.SetSize(want)
		// Give the window a moment to process the resize.
		time.Sleep(50 * time.Millisecond)
		got := w.Size()

		if !approxEqual(got.Width, want.Width, 1) || !approxEqual(got.Height, want.Height, 1) {
			t.Errorf("round-trip: SetSize %v, Size returned %v", want, got)
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

// approxEqual reports whether two pixel values are within tolerance of
// each other. The conversion through device pixels and back can lose a
// fraction, so a tolerance of 1 logical pixel is the tightest assertion
// that is not flaky.
func approxEqual(a, b geometry.Pixels, tolerance geometry.Pixels) bool {
	return math.Abs(float64(a-b)) <= float64(tolerance)
}
