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

// WheelEvent reports scroll-wheel input. Delta is normalised to lines: a
// positive Y delta scrolls toward the bottom of the content, a positive X
// delta toward the right. The platform converts from its native units —
// Windows multiples of 120, macOS pixel-based deltas — so nothing above this
// layer has to know them.
type WheelEvent struct {
	Position  geometry.Point[geometry.DevicePixels]
	DeltaX    float32
	DeltaY    float32
	Modifiers Modifiers
	Time      time.Time
}

func (WheelEvent) isEvent() {}

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
// size is included so the handler need not query the window.
type ResizeEvent struct {
	Size geometry.Size[geometry.DevicePixels]
	Time time.Time
}

func (ResizeEvent) isEvent() {}

// ScaleChangedEvent reports that the display scale factor for the window's
// display changed — the window moved to a different display, or the user
// changed the DPI setting. The new factor is included; the window package
// uses it to invalidate the glyph atlas and relayout.
type ScaleChangedEvent struct {
	ScaleFactor float32
	Time        time.Time
}

func (ScaleChangedEvent) isEvent() {}
