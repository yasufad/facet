package input

import "github.com/yasufad/facet/platform"

// KeyEvent, PointerEvent, TextEvent, WheelEvent and IMECompositionEvent are
// aliases for their platform counterparts, not distinct types. input names the
// event vocabulary for everything above it — element and ui are forbidden
// platform, since it also hands out windows, the clipboard, menus and
// main-thread dispatch, and a widget has no business reaching any of that.
// Aliasing keeps window passing platform values straight through with no
// conversion, and keeps the distinctions platform's normalisation deliberately
// preserves — such as a trackpad's exact pixel delta versus a mouse notch's
// inexact line delta in WheelEvent — intact, which a re-declared struct would
// risk losing.
//
// FocusEvent, ResizeEvent and ScaleChangedEvent are deliberately not aliased.
// All three stop at window: a FocusEvent updates window's focused flag, a
// ResizeEvent updates its size, and a ScaleChangedEvent invalidates its glyph
// atlas and scale factor. None is dispatched to an element, no method on Frame
// hands one to a caller, and window may import platform freely — so no package
// above input is ever handed one of these to name. Aliasing them would be
// vocabulary with no consumer.
type (
	KeyEvent            = platform.KeyEvent
	PointerEvent        = platform.PointerEvent
	TextEvent           = platform.TextEvent
	WheelEvent          = platform.WheelEvent
	IMECompositionEvent = platform.IMECompositionEvent
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

// IMEPhase marks the start, an update, or the end of an IME composition. It is
// an alias, like ScrollUnit above: a handler reading event.Phase on an
// IMECompositionEvent needs a name for the type that does not require
// importing platform.
type IMEPhase = platform.IMEPhase

// IMEStart, IMEUpdate and IMEEnd are platform's IMEPhase constants, named here
// for the same reason IMEPhase is aliased above. A handler switching on
// event.Phase compares against these; without them it would have to import
// platform to name the values its own event type carries.
const (
	IMEStart  = platform.IMEStart
	IMEUpdate = platform.IMEUpdate
	IMEEnd    = platform.IMEEnd
)
