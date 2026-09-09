//go:build windows && facet_debug

package platform

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
	"time"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/third_party/w32"
)

// TestSetIconAppliesIconToWindow verifies that SetIcon creates HICONs from
// PNG bytes and applies them to the window via WM_SETICON, so the OS
// reports the icon through WM_GETICON. The test asks the OS directly
// rather than reading our own largeIcon/smallIcon fields, the same way
// TestHideAndShowChangeWindowVisibility asks IsWindowVisible.
//
// Before the fix, SetIcon was a no-op that returned without doing
// anything. The test catches a no-op stub by checking that WM_GETICON
// returns a non-zero HICON after SetIcon — a stub leaves it at zero.
func TestSetIconAppliesIconToWindow(t *testing.T) {
	// Generate a small solid-colour PNG. CreateLargeHIconFromImage and
	// CreateSmallHIconFromImage accept PNG bytes, so a 32x32 PNG is
	// enough to produce a valid HICON.
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 128, B: 0, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	pngBytes := pngBuf.Bytes()

	p, err := New(Options{Name: "facet-icon-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:     "Facet Icon Test",
			Size:      geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
			Visible:   false,
			Decorated: true,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}
		hwnd := w32.HWND(w.NativeHandle())

		// Before SetIcon, WM_GETICON should return 0 because the
		// window class registers no icon.
		p.Dispatch(func() {
			before := w32.SendMessage(hwnd, w32.WM_GETICON, w32.ICON_SMALL, 0)
			if before != 0 {
				t.Errorf("WM_GETICON before SetIcon = %v, want 0", before)
				return
			}
		})

		// SetIcon dispatches onto the platform thread. The Dispatch
		// below runs after it because the dispatcher is FIFO.
		p.SetIcon(pngBytes)
		p.Dispatch(func() {
			// After SetIcon, WM_GETICON should return the HICON we
			// set. A no-op stub leaves it at 0.
			small := w32.SendMessage(hwnd, w32.WM_GETICON, w32.ICON_SMALL, 0)
			if small == 0 {
				t.Error("WM_GETICON ICON_SMALL = 0 after SetIcon")
				return
			}
			large := w32.SendMessage(hwnd, w32.WM_GETICON, w32.ICON_BIG, 0)
			if large == 0 {
				t.Error("WM_GETICON ICON_BIG = 0 after SetIcon")
				return
			}
		})

		// Close by posting WM_CLOSE rather than calling w.Close,
		// which uses SendMessage — a sent message Win32 processes
		// before posted messages, so it would destroy the window
		// before the dispatch closures above run.
		p.Dispatch(func() {
			postClose(hwnd)
		})
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

// TestSetIconWithEmptyBytesClearsIcon verifies that SetIcon with empty
// bytes clears any previously set icon, so WM_GETICON returns 0 again.
// This probes the zero-value / nil-input path the AGENTS.md verification
// rules call out: a defect that only shows when the field is not set.
func TestSetIconWithEmptyBytesClearsIcon(t *testing.T) {
	// Generate a small solid-colour PNG to set first.
	img := image.NewRGBA(image.Rect(0, 0, 32, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 128, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	pngBytes := pngBuf.Bytes()

	p, err := New(Options{Name: "facet-icon-clear-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		w, err := p.NewWindow(WindowOptions{
			Title:     "Facet Icon Clear Test",
			Size:      geometry.Size[geometry.Pixels]{Width: 200, Height: 150},
			Visible:   false,
			Decorated: true,
		})
		if err != nil {
			t.Errorf("NewWindow: %v", err)
			return
		}
		hwnd := w32.HWND(w.NativeHandle())

		// Set an icon first, then clear it with empty bytes.
		p.SetIcon(pngBytes)
		p.SetIcon(nil)
		p.Dispatch(func() {
			small := w32.SendMessage(hwnd, w32.WM_GETICON, w32.ICON_SMALL, 0)
			if small != 0 {
				t.Errorf("WM_GETICON ICON_SMALL = %v after clear, want 0", small)
				return
			}
			large := w32.SendMessage(hwnd, w32.WM_GETICON, w32.ICON_BIG, 0)
			if large != 0 {
				t.Errorf("WM_GETICON ICON_BIG = %v after clear, want 0", large)
				return
			}
		})

		p.Dispatch(func() {
			postClose(hwnd)
		})
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
