//go:build windows && facet_debug

package platform

import (
	"testing"
	"unsafe"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/third_party/w32"
)

// TestWindowStateRoundTrip drives a real window through every WindowState and
// reads it back. Minimized and Maximized are OS-reported and expected to
// round-trip trivially; Fullscreen is the one Windows has no native concept
// of, so this checks its geometry against the monitor bounds independently
// of the production code's own stillFullscreen check, rather than asking the
// code whether it agrees with itself.
//
// It also drives the specific interaction the window package's lead flagged:
// minimizing a fullscreen window and restoring it through a raw ShowWindow
// call (bypassing SetState, to exercise what the OS does on its own) must
// come back fullscreen, not Normal — because minimizing never touches the
// geometry State and SetState(WindowNormal) both rely on.
func TestWindowStateRoundTrip(t *testing.T) {
	p, err := New(Options{Name: "facet-windowstate-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:     "Facet Window State Test",
			Size:      geometry.Size[geometry.Pixels]{Width: 400, Height: 300},
			Visible:   true,
			Resizable: true,
			Decorated: true,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}
		hwnd := w32.HWND(w.NativeHandle())

		if got := w.State(); got != WindowNormal {
			t.Fatalf("initial State() = %v, want WindowNormal", got)
		}

		// Minimized and Maximized: OS-reported, must round-trip.
		w.SetState(WindowMinimized)
		if got := w.State(); got != WindowMinimized {
			t.Errorf("after SetState(WindowMinimized): State() = %v, want WindowMinimized", got)
		}

		w.SetState(WindowNormal)
		if got := w.State(); got != WindowNormal {
			t.Errorf("after SetState(WindowNormal): State() = %v, want WindowNormal", got)
		}

		w.SetState(WindowMaximized)
		if got := w.State(); got != WindowMaximized {
			t.Errorf("after SetState(WindowMaximized): State() = %v, want WindowMaximized", got)
		}

		w.SetState(WindowNormal)
		if got := w.State(); got != WindowNormal {
			t.Errorf("after restoring from maximized: State() = %v, want WindowNormal", got)
		}

		// Fullscreen: not an OS state. Verify the geometry independently —
		// against the monitor bounds queried fresh here, not against
		// whatever the production code recorded for itself.
		w.SetState(WindowFullscreen)
		if got := w.State(); got != WindowFullscreen {
			t.Fatalf("after SetState(WindowFullscreen): State() = %v, want WindowFullscreen", got)
		}
		wantRect := monitorRectFor(t, hwnd)
		if got := *w32.GetWindowRect(hwnd); got != wantRect {
			t.Errorf("fullscreen window rect = %+v, want monitor bounds %+v", got, wantRect)
		}

		// Minimize while fullscreen, then restore through a raw ShowWindow
		// call rather than SetState(WindowNormal) -- this is what a taskbar
		// click actually does, and it must not require Facet's code to
		// intervene for the fullscreen geometry to come back.
		w.SetState(WindowMinimized)
		if got := w.State(); got != WindowMinimized {
			t.Fatalf("after minimizing from fullscreen: State() = %v, want WindowMinimized", got)
		}
		w32.ShowWindow(hwnd, w32.SW_RESTORE)
		if got := w.State(); got != WindowFullscreen {
			t.Errorf("after restoring a minimized fullscreen window: State() = %v, want WindowFullscreen", got)
		}

		// Leaving fullscreen explicitly restores the pre-fullscreen bounds.
		w.SetState(WindowNormal)
		if got := w.State(); got != WindowNormal {
			t.Errorf("after SetState(WindowNormal) from fullscreen: State() = %v, want WindowNormal", got)
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

// TestWindowStateFullscreenFlagClearsOnMismatch covers the case the window
// package's lead specifically asked for: a stored fullscreen flag can go
// stale if something moves the window through a route that doesn't go
// through SetState. Reporting Fullscreen from the flag alone in that case
// would be a lie nothing else contradicts. State must notice the geometry no
// longer matches, clear the flag, and fall through to whatever the window
// actually is.
func TestWindowStateFullscreenFlagClearsOnMismatch(t *testing.T) {
	p, err := New(Options{Name: "facet-windowstate-stale-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:   "Facet Window State Stale Flag Test",
			Size:    geometry.Size[geometry.Pixels]{Width: 400, Height: 300},
			Visible: true,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}
		ww, ok := w.(*windowsWindow)
		if !ok {
			t.Fatalf("NewWindow returned %T, want *windowsWindow", w)
		}

		w.SetState(WindowFullscreen)
		if got := w.State(); got != WindowFullscreen {
			t.Fatalf("after SetState(WindowFullscreen): State() = %v, want WindowFullscreen", got)
		}

		// Move the window without going through SetState -- exactly the
		// route the flag has no way to observe.
		w32.SetWindowPos(ww.hwnd, 0, 10, 10, 300, 200,
			w32.SWP_NOZORDER|w32.SWP_NOACTIVATE)

		if got := w.State(); got == WindowFullscreen {
			t.Errorf("State() still reports WindowFullscreen after the window moved out from under it")
		}
		if ww.fullscreen {
			t.Errorf("fullscreen flag still set after State() found it didn't match reality")
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

// monitorRectFor returns the full bounds (not the work area) of the monitor
// hwnd is on, queried independently of the production fullscreen code.
func monitorRectFor(t *testing.T, hwnd w32.HWND) w32.RECT {
	t.Helper()
	monitor := w32.MonitorFromWindow(hwnd, w32.MONITOR_DEFAULTTONEAREST)
	var mi w32.MONITORINFO
	mi.CbSize = uint32(unsafe.Sizeof(mi))
	if !w32.GetMonitorInfo(monitor, &mi) {
		t.Fatalf("GetMonitorInfo failed for hwnd %v", hwnd)
	}
	return mi.RcMonitor
}
