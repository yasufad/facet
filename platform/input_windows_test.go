//go:build windows && facet_debug

package platform

import "github.com/yasufad/facet/third_party/w32"

// postMouseMove posts a WM_MOUSEMOVE message to the window with the given
// lParam. It is used by the smoke test to synthesise input without moving
// the real mouse.
func postMouseMove(hwnd uintptr, lParam uintptr) {
	w32.PostMessage(w32.HWND(hwnd), w32.WM_MOUSEMOVE, 0, lParam)
}
