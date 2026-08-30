//go:build windows

package platform

import "github.com/yasufad/facet/third_party/w32"

// cursorToResource maps a [Cursor] to a Windows cursor resource ID. A shape
// with no native equivalent falls back to IDC_ARROW.
func cursorToResource(c Cursor) uint16 {
	switch c {
	case CursorDefault, CursorArrow:
		return w32.IDC_ARROW
	case CursorText:
		return w32.IDC_IBEAM
	case CursorPointer:
		return w32.IDC_HAND
	case CursorWait:
		return w32.IDC_WAIT
	case CursorCrosshair:
		return w32.IDC_CROSS
	case CursorNotAllowed:
		return w32.IDC_NO
	case CursorResizeAll:
		return w32.IDC_SIZEALL
	case CursorResizeHorizontal:
		return w32.IDC_SIZEWE
	case CursorResizeVertical:
		return w32.IDC_SIZENS
	case CursorResizeTopLeftBottomRight:
		return w32.IDC_SIZENWSE
	default:
		return w32.IDC_ARROW
	}
}

// setCursor loads and sets the Windows cursor for the given shape.
func setCursor(c Cursor) {
	hcursor := w32.LoadCursorWithResourceID(0, cursorToResource(c))
	w32.SetCursor(hcursor)
}

// showCursor shows or hides the pointer. ShowCursor is reference-counted;
// calling it repeatedly with false hides the cursor only after an equal
// number of true calls. We call once per request, matching the interface's
// per-application semantics.
func showCursor(visible bool) {
	if visible {
		w32.ShowCursor(1)
	} else {
		w32.ShowCursor(0)
	}
}
