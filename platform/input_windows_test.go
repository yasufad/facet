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

// postChar posts a WM_CHAR message carrying r. It is used by tests to
// synthesise a committed character without a real keyboard — the same
// message a dead-key sequence (e.g. a French or German layout composing an
// accented letter) or an IME's finalised composition delivers.
func postChar(hwnd uintptr, r rune) {
	w32.PostMessage(w32.HWND(hwnd), w32.WM_CHAR, uintptr(r), 0)
}

// postIMEComposition posts a WM_IME_STARTCOMPOSITION, WM_IME_COMPOSITION or
// WM_IME_ENDCOMPOSITION message. gcsFlags is the lParam for
// WM_IME_COMPOSITION (ignored for the other two) and reports which part of
// the composition changed — GCS_COMPSTR for the composition string itself.
func postIMEComposition(hwnd uintptr, msg uint32, gcsFlags uintptr) {
	w32.PostMessage(w32.HWND(hwnd), msg, 0, gcsFlags)
}
