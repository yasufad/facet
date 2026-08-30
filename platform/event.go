package platform

import (
	"time"

	"github.com/yasufad/facet/geometry"
)

// Event is a single input event from the native event stream. The platform
// translates each native message into one of the concrete types below and
// delivers it through [Window.SetEventHandler]. Events arrive synchronously on
// the platform thread, in the order the native stream produced them.
//
// The concrete types are value types: they carry no behaviour and are copied
// freely. The unexported isEvent method seals the set so that a new event type
// is a visible change to this package, not something a consumer silently
// misses in a type switch.
type Event interface {
	isEvent()
}

// PointerEvent reports pointer movement or a button press and release.
// Position is in the window's client area, in device pixels relative to the
// top-left corner.
type PointerEvent struct {
	Phase     PointerPhase
	Position  geometry.Point[geometry.DevicePixels]
	Button    PointerButton  // the button that changed, for Down and Up
	Buttons   PointerButtons // all buttons currently held
	Modifiers Modifiers
	Time      time.Time
}

func (PointerEvent) isEvent() {}

// WheelEvent reports scroll-wheel or trackpad scroll input. The delta carries
// its unit so the consumer can apply it correctly: pixel deltas from a
// trackpad or precision touchpad are exact and applied directly; line deltas
// from a mouse wheel are inexact and multiplied by a line height. The phase
// tracks the gesture lifecycle, which a mouse-wheel notch does not have — it
// arrives as [ScrollMoved] with no Started/Ended pair.
//
// Positive Y scrolls toward the bottom of the content; positive X toward the
// right. The platform converts from its native units — Windows multiples of
// 120 for a mouse wheel, pixel deltas for a precision touchpad — but does not
// flatten the distinction, because precision is unrecoverable once lost.
type WheelEvent struct {
	Position  geometry.Point[geometry.DevicePixels]
	Delta     ScrollDelta
	Phase     ScrollPhase
	Modifiers Modifiers
	Time      time.Time
}

func (WheelEvent) isEvent() {}

// ScrollDelta carries a scroll distance in one of two units. A mouse wheel
// emits discrete notches measured in lines; a trackpad or precision touchpad
// emits pixel-exact deltas with momentum. The consumer applies pixel deltas
// directly and multiplies line deltas by a line height. Hiding the unit would
// destroy precision this layer received — the two inputs are different, and
// the consumer has to know which arrived.
type ScrollDelta struct {
	// Unit distinguishes pixel-exact from line-based deltas.
	Unit ScrollUnit

	// DeltaX is the horizontal scroll distance. Positive is toward the right.
	DeltaX float32

	// DeltaY is the vertical scroll distance. Positive is toward the bottom.
	DeltaY float32
}

// ScrollUnit distinguishes exact pixel deltas from inexact line deltas.
type ScrollUnit int

const (
	// ScrollPixels is an exact delta in physical pixels, from a trackpad or
	// precision touchpad. The consumer applies it directly.
	ScrollPixels ScrollUnit = iota

	// ScrollLines is an inexact delta in lines, from a mouse wheel. The
	// consumer multiplies by a line height before applying.
	ScrollLines
)

// ScrollPhase tracks the lifecycle of a scroll gesture — typically a trackpad
// or touch gesture with momentum. A mouse-wheel notch has no lifecycle: it
// arrives as [ScrollMoved] with no [ScrollStarted]/[ScrollEnded] pair.
//
// The phases mirror winit's TouchPhase, which mirrors GPUI's.
type ScrollPhase int

const (
	// ScrollStarted is the first event of a gesture.
	ScrollStarted ScrollPhase = iota

	// ScrollMoved is a continuing event, or the only event for a discrete
	// mouse-wheel notch.
	ScrollMoved

	// ScrollEnded is the last event of a gesture, before inertia.
	ScrollEnded

	// ScrollCancelled is sent when the gesture is interrupted — a finger
	// landing on the trackpad, or the system cancelling the touch.
	ScrollCancelled
)

// KeyEvent reports a key press, release or auto-repeat. Code identifies the
// physical key for keybindings; the character it produced, if any, arrives as
// a separate [TextEvent] so that text input and key identity stay decoupled.
type KeyEvent struct {
	Phase     KeyPhase
	Code      KeyCode
	Modifiers Modifiers
	Time      time.Time
}

func (KeyEvent) isEvent() {}

// TextEvent delivers a committed character — from a key press, an IME
// composition that completed, or a paste. It carries the runes the platform
// has finalised, not the raw key that produced them. A key press that
// produces no text (a modifier, an arrow key) generates a [KeyEvent] but no
// TextEvent.
type TextEvent struct {
	Text string
	Time time.Time
}

func (TextEvent) isEvent() {}

// IMECompositionEvent reports a change in an in-progress IME composition.
// The platform surfaces the composed text as it stands during composition so
// the framework can render the candidate inline; the finalised text arrives
// as a [TextEvent] when composition ends.
//
// On macOS, composition flows through the NSTextInputClient protocol, which
// has no direct equivalent on Windows; the Windows backend uses WM_IME_*
// messages. The interface abstracts both to the same three phases.
type IMECompositionEvent struct {
	Phase  IMEPhase
	Text   string // the current composition string
	Cursor int    // cursor position within Text, in runes; -1 if none
	Time   time.Time
}

// IMEPhase marks the start, an update, or the end of a composition.
type IMEPhase int

const (
	IMEStart IMEPhase = iota
	IMEUpdate
	IMEEnd
)

func (IMECompositionEvent) isEvent() {}

// FocusEvent reports that the window gained or lost keyboard focus.
type FocusEvent struct {
	Focused bool
	Time    time.Time
}

func (FocusEvent) isEvent() {}

// ResizeEvent reports that the window's client area changed size. The new
// size is in logical pixels, matching [Window.Size], so the handler need not
// query the window or convert units.
type ResizeEvent struct {
	Size geometry.Size[geometry.Pixels]
	Time time.Time
}

func (ResizeEvent) isEvent() {}

// ScaleChangedEvent reports that the display scale factor for the window's
// display changed — the window moved to a different display, or the user
// changed the DPI setting. It is the authoritative per-window signal for a
// scale-factor change: the window package uses it to invalidate the glyph
// atlas and relayout. The global display-set signal, for monitors attached or
// removed, is [Platform.SetDisplayChangeHandler].
type ScaleChangedEvent struct {
	ScaleFactor float32
	Time        time.Time
}

func (ScaleChangedEvent) isEvent() {}
