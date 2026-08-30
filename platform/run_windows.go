//go:build windows

package platform

import (
	"fmt"
	"unsafe"

	"github.com/yasufad/facet/third_party/w32"
)

// Run starts the native event loop and blocks until Quit is called. It must
// be called on the same goroutine that called New, because the dispatcher's
// hidden window was created on that thread.
//
// The loop is a standard Win32 GetMessage/TranslateMessage/DispatchMessage
// loop. The dispatcher's hidden window receives posted closures and runs
// them; Facet windows receive input and paint messages and dispatch them
// through their wndproc.
//
// WM_DISPLAYCHANGE triggers a display re-enumeration and fires the display
// change handler.
func (p *windowsPlatform) Run() error {
	// Override the placeholder Run in platform_windows.go. This file owns
	// the loop; that file owns the constructor and shell.
	return p.runLoop()
}

func (p *windowsPlatform) runLoop() error {
	msg := (*w32.MSG)(unsafe.Pointer(w32.GlobalAlloc(0, uint32(unsafe.Sizeof(w32.MSG{})))))
	defer w32.GlobalFree(w32.HGLOBAL(unsafe.Pointer(msg)))

	for w32.GetMessage(msg, 0, 0, 0) != 0 {
		// Handle display configuration changes here rather than in a window
		// wndproc, because they are system-wide, not per-window.
		if msg.Message == w32.WM_DISPLAYCHANGE {
			p.refreshDisplays()
		}
		w32.TranslateMessage(msg)
		w32.DispatchMessage(msg)
	}

	return nil
}

// _ = fmt.Errorf("") keeps fmt imported; the platform uses it in other files
// in this build-tagged package.
var _ = fmt.Errorf
