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
//
// The field types of the aliased events are aliased too. A caller that can name
// KeyEvent but cannot name KeyCode or KeyPhase can only construct a zero value,
// and a handler that receives a KeyEvent but cannot name its field types can
// only ignore them. The rule is the same as for the event types themselves:
// alias what a caller above has to write, and a non-zero KeyEvent or
// PointerEvent is written with the same vocabulary the type carries. Each field
// type's constants are named here alongside it, because a type without its
// values is a name a caller cannot use.
//
// ScrollDelta is deliberately not aliased. It is reached only through a
// WheelEvent's Delta field, never named as a type by a caller above input, so
// no package above input has to write it. A handler sets event.Delta.Unit and
// event.Delta.DeltaY through the field, not by constructing a ScrollDelta
// literal — and if one ever does, the alias is added then.
type (
	KeyEvent            = platform.KeyEvent
	PointerEvent        = platform.PointerEvent
	TextEvent           = platform.TextEvent
	WheelEvent          = platform.WheelEvent
	IMECompositionEvent = platform.IMECompositionEvent
)

// KeyCode identifies a physical key, independent of keyboard layout. It is the
// type of KeyEvent.Code and of Keystroke.Code, so a caller above input that
// constructs a KeyEvent or reads a Keystroke needs to name it. The constants
// below are the full set platform declares; they are re-exported in full
// because a caller writing a keybinding against any one of them has to name it
// from input, and scoping the re-export to the keys one caller happens to need
// today leaves the next caller blocked.
type KeyCode = platform.KeyCode

// KeyCode constants. These are every value platform declares; see KeyCode
// above for why the set is complete rather than selective.
const (
	KeyUnknown = platform.KeyUnknown

	KeyA = platform.KeyA
	KeyB = platform.KeyB
	KeyC = platform.KeyC
	KeyD = platform.KeyD
	KeyE = platform.KeyE
	KeyF = platform.KeyF
	KeyG = platform.KeyG
	KeyH = platform.KeyH
	KeyI = platform.KeyI
	KeyJ = platform.KeyJ
	KeyK = platform.KeyK
	KeyL = platform.KeyL
	KeyM = platform.KeyM
	KeyN = platform.KeyN
	KeyO = platform.KeyO
	KeyP = platform.KeyP
	KeyQ = platform.KeyQ
	KeyR = platform.KeyR
	KeyS = platform.KeyS
	KeyT = platform.KeyT
	KeyU = platform.KeyU
	KeyV = platform.KeyV
	KeyW = platform.KeyW
	KeyX = platform.KeyX
	KeyY = platform.KeyY
	KeyZ = platform.KeyZ

	Key0 = platform.Key0
	Key1 = platform.Key1
	Key2 = platform.Key2
	Key3 = platform.Key3
	Key4 = platform.Key4
	Key5 = platform.Key5
	Key6 = platform.Key6
	Key7 = platform.Key7
	Key8 = platform.Key8
	Key9 = platform.Key9

	KeyF1  = platform.KeyF1
	KeyF2  = platform.KeyF2
	KeyF3  = platform.KeyF3
	KeyF4  = platform.KeyF4
	KeyF5  = platform.KeyF5
	KeyF6  = platform.KeyF6
	KeyF7  = platform.KeyF7
	KeyF8  = platform.KeyF8
	KeyF9  = platform.KeyF9
	KeyF10 = platform.KeyF10
	KeyF11 = platform.KeyF11
	KeyF12 = platform.KeyF12
	KeyF13 = platform.KeyF13
	KeyF14 = platform.KeyF14
	KeyF15 = platform.KeyF15
	KeyF16 = platform.KeyF16
	KeyF17 = platform.KeyF17
	KeyF18 = platform.KeyF18
	KeyF19 = platform.KeyF19
	KeyF20 = platform.KeyF20
	KeyF21 = platform.KeyF21
	KeyF22 = platform.KeyF22
	KeyF23 = platform.KeyF23
	KeyF24 = platform.KeyF24

	KeyArrowLeft  = platform.KeyArrowLeft
	KeyArrowRight = platform.KeyArrowRight
	KeyArrowUp    = platform.KeyArrowUp
	KeyArrowDown  = platform.KeyArrowDown
	KeyHome       = platform.KeyHome
	KeyEnd        = platform.KeyEnd
	KeyPageUp     = platform.KeyPageUp
	KeyPageDown   = platform.KeyPageDown

	KeyBackspace = platform.KeyBackspace
	KeyEnter     = platform.KeyEnter
	KeyTab       = platform.KeyTab
	KeyEscape    = platform.KeyEscape
	KeySpace     = platform.KeySpace
	KeyDelete    = platform.KeyDelete
	KeyInsert    = platform.KeyInsert
	KeyCapsLock  = platform.KeyCapsLock

	KeyShiftLeft    = platform.KeyShiftLeft
	KeyShiftRight   = platform.KeyShiftRight
	KeyControlLeft  = platform.KeyControlLeft
	KeyControlRight = platform.KeyControlRight
	KeyAltLeft      = platform.KeyAltLeft
	KeyAltRight     = platform.KeyAltRight
	KeySuperLeft    = platform.KeySuperLeft
	KeySuperRight   = platform.KeySuperRight

	KeyMinus        = platform.KeyMinus
	KeyEqual        = platform.KeyEqual
	KeyLeftBracket  = platform.KeyLeftBracket
	KeyRightBracket = platform.KeyRightBracket
	KeyBackslash    = platform.KeyBackslash
	KeySemicolon    = platform.KeySemicolon
	KeyApostrophe   = platform.KeyApostrophe
	KeyGraveAccent  = platform.KeyGraveAccent
	KeyComma        = platform.KeyComma
	KeyPeriod       = platform.KeyPeriod
	KeySlash        = platform.KeySlash
)

// KeyPhase distinguishes a key press from a release or an auto-repeat. It is
// the type of KeyEvent.Phase, so a caller constructing or inspecting a
// KeyEvent needs to name it.
type KeyPhase = platform.KeyPhase

// KeyPhase constants.
const (
	KeyDown   = platform.KeyDown
	KeyUp     = platform.KeyUp
	KeyRepeat = platform.KeyRepeat
)

// Modifiers is a bitfield of the modifier keys held during an event. It is the
// type of the Modifiers field on KeyEvent, PointerEvent and WheelEvent, and of
// Keystroke.Modifiers, so a caller above input that constructs or inspects any
// of those needs to name it. Its Has, IsEmpty and String methods are available
// on the alias because it is the same type.
type Modifiers = platform.Modifiers

// Modifiers constants.
const (
	Shift   = platform.Shift
	Control = platform.Control
	Alt     = platform.Alt
	Super   = platform.Super
)

// PointerPhase distinguishes movement from button press and release. It is the
// type of PointerEvent.Phase.
type PointerPhase = platform.PointerPhase

// PointerPhase constants.
const (
	PointerMove = platform.PointerMove
	PointerDown = platform.PointerDown
	PointerUp   = platform.PointerUp
)

// PointerButton identifies a single mouse or trackpad button. It is the type
// of PointerEvent.Button — the button that changed on a Down or Up.
type PointerButton = platform.PointerButton

// PointerButton constants.
const (
	PointerNone   = platform.PointerNone
	PointerLeft   = platform.PointerLeft
	PointerRight  = platform.PointerRight
	PointerMiddle = platform.PointerMiddle
	PointerX1     = platform.PointerX1
	PointerX2     = platform.PointerX2
)

// PointerButtons is a bitfield of all buttons held during a pointer event. It
// is the type of PointerEvent.Buttons. Its Has method is available on the
// alias because it is the same type.
type PointerButtons = platform.PointerButtons

// PointerButtons constants.
const (
	ButtonLeft   = platform.ButtonLeft
	ButtonRight  = platform.ButtonRight
	ButtonMiddle = platform.ButtonMiddle
	ButtonX1     = platform.ButtonX1
	ButtonX2     = platform.ButtonX2
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

// ScrollPhase tracks the lifecycle of a scroll gesture. It is the type of
// WheelEvent.Phase, so a caller constructing or inspecting a WheelEvent needs
// to name it.
type ScrollPhase = platform.ScrollPhase

// ScrollPhase constants.
const (
	ScrollStarted   = platform.ScrollStarted
	ScrollMoved     = platform.ScrollMoved
	ScrollEnded     = platform.ScrollEnded
	ScrollCancelled = platform.ScrollCancelled
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
