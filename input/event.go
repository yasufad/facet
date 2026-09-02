package input

import "github.com/yasufad/facet/platform"

// KeyEvent, PointerEvent, TextEvent and WheelEvent are aliases for their
// platform counterparts, not distinct types. input names the event vocabulary
// for everything above it — element and ui are forbidden platform, since it
// also hands out windows, the clipboard, menus and main-thread dispatch, and a
// widget has no business reaching any of that. Aliasing keeps window passing
// platform values straight through with no conversion, and keeps the
// distinctions platform's normalisation deliberately preserves — such as a
// trackpad's exact pixel delta versus a mouse notch's inexact line delta in
// WheelEvent — intact, which a re-declared struct would risk losing.
type (
	KeyEvent     = platform.KeyEvent
	PointerEvent = platform.PointerEvent
	TextEvent    = platform.TextEvent
	WheelEvent   = platform.WheelEvent
)

// ScrollUnit distinguishes exact pixel deltas from inexact line deltas on a
// WheelEvent's Delta. It is an alias, like the event types above: a caller
// reading event.Delta.Unit needs a name for it that does not require
// importing platform.
type ScrollUnit = platform.ScrollUnit

// ScrollPixels and ScrollLines are platform's ScrollUnit constants, named here
// for the same reason ScrollUnit is aliased above. A constant cannot itself be
// aliased, but because ScrollUnit is the same type as platform.ScrollUnit,
// input.ScrollPixels and platform.ScrollPixels are the same value of the same
// type, and comparing a WheelEvent's Delta.Unit against either compiles and
// behaves identically.
const (
	ScrollPixels = platform.ScrollPixels
	ScrollLines  = platform.ScrollLines
)
