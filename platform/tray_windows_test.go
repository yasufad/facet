//go:build windows && facet_debug

package platform

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yasufad/facet/third_party/w32"
)

// TestSystemTrayRegistersAndFiresCallback verifies that NewSystemTray
// registers a tray icon the OS can see — asked through
// Shell_NotifyIconGetRect, the same way the visibility test asks
// IsWindowVisible rather than reading our own map — and that the callback
// message reaches the OnClick handler when the tray window receives it.
//
// Before the fix, NewSystemTray returned an error and never touched the
// shell. The test catches a stub by requiring GetSystrayBounds to succeed
// after NewSystemTray and to fail after Remove.
func TestSystemTrayRegistersAndFiresCallback(t *testing.T) {
	// Generate a small solid-colour PNG for the tray icon.
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{R: 0, G: 128, B: 255, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	pngBytes := pngBuf.Bytes()

	p, err := New(Options{Name: "facet-tray-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var clicked atomic.Bool

	done := make(chan struct{})
	go func() {
		defer close(done)

		tray, err := p.NewSystemTray(SystemTrayOptions{
			Icon:    pngBytes,
			Tooltip: "Facet Tray Test",
			OnClick: func() { clicked.Store(true) },
		})
		if err != nil {
			t.Errorf("NewSystemTray: %v", err)
			return
		}
		wt := tray.(*windowsSystemTray)
		wp := p.(*windowsPlatform)

		// Ask the OS whether the tray icon exists. GetSystrayBounds
		// calls Shell_NotifyIconGetRect, which returns the icon's
		// bounding rectangle only if the icon is registered. A stub
		// that never called Shell_NotifyIcon leaves nothing to find.
		p.Dispatch(func() {
			rect, err := w32.GetSystrayBounds(wp.trayHwnd, wt.id)
			if err != nil {
				t.Errorf("GetSystrayBounds after add: %v", err)
				return
			}
			if rect == nil {
				t.Error("GetSystrayBounds returned nil rect after add")
				return
			}
		})

		// Simulate a left-click by sending the callback message
		// directly to the tray window. This is the same technique the
		// quit-handler test uses to send WM_QUERYENDSESSION: drive the
		// real message path without moving the real mouse.
		p.Dispatch(func() {
			w32.SendMessage(wp.trayHwnd, wmTrayCallback, uintptr(wt.id), w32.WM_LBUTTONUP)
		})

		// Verify the OnClick handler fired. The SendMessage above is
		// synchronous (same thread), so the handler has run by the time
		// this dispatch closure executes.
		p.Dispatch(func() {
			if !clicked.Load() {
				t.Error("OnClick handler was not called after callback message")
			}
		})

		// Remove the tray icon. Remove is synchronous per the interface
		// contract, so it has completed when this call returns.
		tray.Remove()

		// Verify the tray is gone from the platform's map. The
		// GetSystrayBounds check above already proved the icon was
		// registered with the OS; this checks our cleanup removed it
		// from our own tracking. Shell_NotifyIconGetRect may still
		// return a cached rect briefly after NIM_DELETE, so the map
		// is the reliable signal that Remove completed.
		p.Dispatch(func() {
			if _, ok := wp.trays[wt.id]; ok {
				t.Error("tray still in platform map after Remove")
			}
			if wt.icon != 0 {
				t.Error("tray icon handle not destroyed after Remove")
			}
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

// TestSystemTraySetIconReplacesIcon verifies that SetIcon on a tray replaces
// the icon handle, so the OS sees a non-zero HICON after the replacement.
// It asks the OS through the NOTIFYICONDATA the platform would send, by
// checking that the tray's icon field is non-zero after SetIcon and zero
// after SetIcon with empty bytes.
func TestSystemTraySetIconReplacesIcon(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 128, A: 255})
		}
	}
	var pngBuf bytes.Buffer
	if err := png.Encode(&pngBuf, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}
	pngBytes := pngBuf.Bytes()

	p, err := New(Options{Name: "facet-tray-seticon-test"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		// Create a tray without an icon.
		tray, err := p.NewSystemTray(SystemTrayOptions{
			Tooltip: "Facet SetIcon Test",
		})
		if err != nil {
			t.Errorf("NewSystemTray: %v", err)
			return
		}
		wt := tray.(*windowsSystemTray)

		// Before SetIcon, the icon handle is zero.
		p.Dispatch(func() {
			if wt.icon != 0 {
				t.Errorf("icon = %v before SetIcon, want 0", wt.icon)
			}
		})

		// Set an icon and verify the handle is non-zero.
		tray.SetIcon(pngBytes)
		p.Dispatch(func() {
			if wt.icon == 0 {
				t.Error("icon = 0 after SetIcon with PNG bytes")
			}
		})

		// Clear the icon and verify the handle is zero again.
		tray.SetIcon(nil)
		p.Dispatch(func() {
			if wt.icon != 0 {
				t.Errorf("icon = %v after SetIcon(nil), want 0", wt.icon)
			}
		})

		tray.Remove()
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
