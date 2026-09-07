//go:build windows && facet_debug

package platform

import (
	"testing"
	"time"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/third_party/w32"
)

// TestSetApplicationMenuAttachesToWindow creates a platform, opens a
// decorated window, calls SetApplicationMenu, and verifies that a real HMENU
// is attached to the window (GetMenu returns non-zero). Then it posts a
// WM_COMMAND with the first command ID and verifies the matching OnClick
// fires — proving the command map is wired through from the Menu tree to the
// wndproc.
//
// It is behind facet_debug because it opens a real window and drives the
// message loop, matching this package's convention for anything that touches
// real desktop resources.
func TestSetApplicationMenuAttachesToWindow(t *testing.T) {
	p, err := New(Options{Name: "facet-menu-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:     "Facet Menu Test",
			Size:      geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
			Visible:   false,
			Decorated: true,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}
		hwnd := w32.HWND(w.NativeHandle())

		clicked := make(chan struct{}, 1)
		p.SetApplicationMenu(&Menu{
			Items: []MenuItem{
				{Label: "File", Submenu: &Menu{
					Items: []MenuItem{
						{Label: "Quit", OnClick: func() {
							select {
							case clicked <- struct{}{}:
							default:
							}
						}},
					},
				}},
			},
		})

		// SetApplicationMenu dispatches onto the platform thread; give it a
		// moment to land before reading the menu back.
		p.Dispatch(func() {
			// Verify a real HMENU is attached to the decorated window.
			hmenu := w32.GetMenu(hwnd)
			if hmenu == 0 {
				t.Error("GetMenu returned 0 after SetApplicationMenu")
				return
			}

			// The first leaf item ("Quit") gets command ID firstCommandID.
			// Post WM_COMMAND to fire its OnClick.
			w32.PostMessage(hwnd, w32.WM_COMMAND, firstCommandID, 0)
		})

		select {
		case <-clicked:
			// OnClick fired — the command map is wired through.
		case <-time.After(2 * time.Second):
			t.Error("OnClick did not fire after WM_COMMAND")
			return
		}

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

// TestSetApplicationMenuSkipsUndecoratedWindow verifies that an undecorated
// window gets no menu bar. A window created with Decorated: false asked for
// no OS chrome, and a menu bar is OS chrome.
func TestSetApplicationMenuSkipsUndecoratedWindow(t *testing.T) {
	p, err := New(Options{Name: "facet-menu-undecorated-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:     "Facet Undecorated Menu Test",
			Size:      geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
			Visible:   false,
			Decorated: false,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}
		hwnd := w32.HWND(w.NativeHandle())

		p.SetApplicationMenu(&Menu{
			Items: []MenuItem{
				{Label: "File", Submenu: &Menu{
					Items: []MenuItem{
						{Label: "Quit", OnClick: func() {}},
					},
				}},
			},
		})

		p.Dispatch(func() {
			// An undecorated window must have no menu bar.
			if hmenu := w32.GetMenu(hwnd); hmenu != 0 {
				t.Error("GetMenu returned non-zero for an undecorated window")
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
