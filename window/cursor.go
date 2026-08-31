package window

import (
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/style"
)

// cursorStyleToPlatform converts a style.CursorStyle into a platform.Cursor.
func cursorStyleToPlatform(c style.CursorStyle) platform.Cursor {
	switch c {
	case style.CursorPointer:
		return platform.CursorPointer
	case style.CursorText:
		return platform.CursorText
	case style.CursorCrosshair:
		return platform.CursorCrosshair
	case style.CursorNotAllowed:
		return platform.CursorNotAllowed
	case style.CursorGrab, style.CursorGrabbing:
		return platform.CursorMove
	case style.CursorResizeCol:
		return platform.CursorResizeHorizontal
	case style.CursorResizeRow:
		return platform.CursorResizeVertical
	default:
		return platform.CursorDefault
	}
}
