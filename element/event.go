package element

import (
	"github.com/yasufad/facet/geometry"
)

// MouseButton identifies the mouse button in a pointer interaction.
type MouseButton uint8

const (
	// MouseButtonLeft represents the primary/left mouse button.
	MouseButtonLeft MouseButton = iota
	// MouseButtonMiddle represents the middle mouse button / scroll wheel button.
	MouseButtonMiddle
	// MouseButtonRight represents the secondary/right mouse button.
	MouseButtonRight
)

// Modifiers is a bitfield of modifier keys held during an interaction.
type Modifiers uint8

const (
	// ModShift indicates the Shift key is held.
	ModShift Modifiers = 1 << iota
	// ModControl indicates the Control key is held.
	ModControl
	// ModAlt indicates the Alt key is held.
	ModAlt
	// ModSuper indicates the Super/Command/Windows key is held.
	ModSuper
)

// Has reports whether all modifiers in other are set in m.
func (m Modifiers) Has(other Modifiers) bool {
	return m&other == other
}

// ClickEvent represents a synthesised click interaction (pointer pressed and
// released within the bounds of the same element).
type ClickEvent struct {
	// Position is the window-relative click position in logical pixels.
	Position geometry.Point[geometry.Pixels]

	// LocalPosition is the element-relative click position in logical pixels.
	LocalPosition geometry.Point[geometry.Pixels]

	Button     MouseButton
	Modifiers  Modifiers
	ClickCount int
}
