package platform

// Cursor is the pointer shape a [Window] displays over itself, set through
// [Window.SetCursor]. The set is deliberately small: the shapes a GUI
// framework needs, not every cursor an OS provides. A backend that lacks a
// shape falls back to [CursorDefault].
type Cursor int

const (
	// CursorDefault is the platform's standard arrow pointer.
	CursorDefault Cursor = iota
	// CursorText is the I-beam used over editable text.
	CursorText
	// CursorPointer is the hand used over clickable elements.
	CursorPointer
	// CursorCrosshair is the crosshair used for precise selection.
	CursorCrosshair
	// CursorNotAllowed is the circle-with-line used to indicate an action is
	// unavailable.
	CursorNotAllowed
	// CursorResizeHorizontal is the left-right double arrow.
	CursorResizeHorizontal
	// CursorResizeVertical is the up-down double arrow.
	CursorResizeVertical
	// CursorResizeNorthEastSouthWest is the diagonal double arrow.
	CursorResizeNorthEastSouthWest
	// CursorResizeNorthWestSouthEast is the other diagonal double arrow.
	CursorResizeNorthWestSouthEast
	// CursorMove is the four-direction arrow used for dragging.
	CursorMove
	// CursorWait is the hourglass or spinner used when busy.
	CursorWait
	// CursorNone hides the cursor entirely.
	CursorNone
)
