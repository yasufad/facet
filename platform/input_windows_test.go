//go:build windows && facet_debug

package platform

import "github.com/yasufad/facet/third_party/w32"

// postMouseMove posts a WM_MOUSEMOVE message to the window with the given
// lParam. It is used by the smoke test to synthesise input without moving
// the real mouse.
func postMouseMove(hwnd uintptr, lParam uintptr) {
	w32.PostMessage(w32.HWND(hwnd), w32.WM_MOUSEMOVE, 0, lParam)
}

// postClose posts a WM_CLOSE message to the window, triggering the close
// path (close handler, then DestroyWindow) without holding a Go reference
// to the window. It is used by the lifetime test to clean up a window whose
// Go handle has been dropped.
func postClose(hwnd uintptr) {
	w32.PostMessage(w32.HWND(hwnd), w32.WM_CLOSE, 0, 0)
}
