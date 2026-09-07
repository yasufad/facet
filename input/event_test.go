package input_test

import (
	"testing"

	"github.com/yasufad/facet/geometry"

	"github.com/yasufad/facet/input"
)

// A caller above input must be able to build and inspect a wheel event
// without importing platform. This file imports only input (and geometry,
// which every layer above input may import); if it stops compiling, some part
// of the vocabulary a handler needs has gone missing.
func TestWheelEventVocabularyNamesNoPlatform(t *testing.T) {
	var pixel input.WheelEvent
	pixel.Delta.Unit = input.ScrollPixels
	pixel.Delta.DeltaY = 12
	if pixel.Delta.Unit != input.ScrollPixels {
		t.Fatalf("Unit = %v, want ScrollPixels", pixel.Delta.Unit)
	}

	var line input.WheelEvent
	line.Delta.Unit = input.ScrollLines
	line.Delta.DeltaY = 3
	if line.Delta.Unit != input.ScrollLines {
		t.Fatalf("Unit = %v, want ScrollLines", line.Delta.Unit)
	}
	if line.Delta.Unit == input.ScrollPixels {
		t.Fatalf("ScrollLines compared equal to ScrollPixels")
	}
}

// TestAliasedEventVocabularyNamesNoPlatform constructs a non-zero value of
// every aliased event type using only input names — every field type and
// every constant a caller needs to write a real value, not just a zero. Each
// field type is named explicitly in a variable declaration, so removing the
// type alias (not just the constants) breaks compilation. If any alias or its
// associated constant goes missing, this stops compiling, which is the signal
// that the vocabulary a caller above input needs is incomplete. This file
// imports only input and geometry; a platform import here would defeat the
// purpose.
func TestAliasedEventVocabularyNamesNoPlatform(t *testing.T) {
	// KeyEvent — every field is set with an input name: Phase, Code and
	// Modifiers. A zero value would prove nothing about the vocabulary.
	// The field types are named in declarations below the literal, so a
	// caller writing a handler signature or a local variable has the type
	// name as well as the constants.
	key := input.KeyEvent{
		Phase:     input.KeyDown,
		Code:      input.KeyA,
		Modifiers: input.Shift | input.Control,
	}
	if key.Phase != input.KeyDown {
		t.Fatalf("Phase = %v, want KeyDown", key.Phase)
	}
	if key.Code != input.KeyA {
		t.Fatalf("Code = %v, want KeyA", key.Code)
	}
	if !key.Modifiers.Has(input.Shift) || !key.Modifiers.Has(input.Control) {
		t.Fatalf("Modifiers = %v, want Shift|Control", key.Modifiers)
	}
	if key.Phase == input.KeyUp || key.Phase == input.KeyRepeat {
		t.Fatalf("KeyDown compared equal to another KeyPhase")
	}
	// Name the field types explicitly: a handler that declares a local or
	// writes a signature against the field type needs the alias to exist,
	// not just the constants. Without these declarations, removing the type
	// alias would still compile because the constants carry the type
	// implicitly.
	var keyCode input.KeyCode = key.Code
	var keyPhase input.KeyPhase = key.Phase
	var keyMods input.Modifiers = key.Modifiers
	_ = keyCode
	_ = keyPhase
	if !keyMods.Has(input.Shift) {
		t.Fatalf("keyMods = %v, want Shift held", keyMods)
	}

	// PointerEvent — Phase, Position, Button, Buttons and Modifiers all set
	// with input names. geometry is importable by every layer above input.
	ptr := input.PointerEvent{
		Phase:     input.PointerDown,
		Position:  geometry.NewPoint[geometry.DevicePixels](10, 20),
		Button:    input.PointerLeft,
		Buttons:   input.ButtonLeft,
		Modifiers: input.Control,
	}
	if ptr.Phase != input.PointerDown {
		t.Fatalf("Phase = %v, want PointerDown", ptr.Phase)
	}
	if ptr.Button != input.PointerLeft {
		t.Fatalf("Button = %v, want PointerLeft", ptr.Button)
	}
	if !ptr.Buttons.Has(input.PointerLeft) {
		t.Fatalf("Buttons = %v, want ButtonLeft held", ptr.Buttons)
	}
	if ptr.Phase == input.PointerMove || ptr.Phase == input.PointerUp {
		t.Fatalf("PointerDown compared equal to another PointerPhase")
	}
	var ptrPhase input.PointerPhase = ptr.Phase
	var ptrButton input.PointerButton = ptr.Button
	var ptrButtons input.PointerButtons = ptr.Buttons
	_ = ptrPhase
	_ = ptrButton
	if !ptrButtons.Has(input.PointerLeft) {
		t.Fatalf("ptrButtons = %v, want PointerLeft held", ptrButtons)
	}

	// TextEvent — Text is a string, so a caller builds a full value.
	txt := input.TextEvent{Text: "Facet"}
	if txt.Text != "Facet" {
		t.Fatalf("Text = %q, want %q", txt.Text, "Facet")
	}

	// WheelEvent — Phase, Delta (via its fields) and Modifiers all set with
	// input names. ScrollDelta itself is deliberately not aliased: it is
	// reached through the Delta field, never named as a type.
	wheel := input.WheelEvent{
		Phase:     input.ScrollMoved,
		Modifiers: input.Control,
	}
	wheel.Delta.Unit = input.ScrollLines
	wheel.Delta.DeltaY = 3
	if wheel.Phase != input.ScrollMoved {
		t.Fatalf("Phase = %v, want ScrollMoved", wheel.Phase)
	}
	if wheel.Delta.Unit != input.ScrollLines {
		t.Fatalf("Unit = %v, want ScrollLines", wheel.Delta.Unit)
	}
	if wheel.Phase == input.ScrollStarted || wheel.Phase == input.ScrollEnded {
		t.Fatalf("ScrollMoved compared equal to another ScrollPhase")
	}
	var scrollPhase input.ScrollPhase = wheel.Phase
	var scrollUnit input.ScrollUnit = wheel.Delta.Unit
	_ = scrollPhase
	if scrollUnit != input.ScrollLines {
		t.Fatalf("scrollUnit = %v, want ScrollLines", scrollUnit)
	}

	// IMECompositionEvent — Phase, Text and Cursor all set with input names.
	start := input.IMECompositionEvent{Phase: input.IMEStart, Text: "あ", Cursor: 0}
	if start.Phase != input.IMEStart {
		t.Fatalf("Phase = %v, want IMEStart", start.Phase)
	}

	update := input.IMECompositionEvent{Phase: input.IMEUpdate, Text: "あい", Cursor: 1}
	if update.Phase != input.IMEUpdate {
		t.Fatalf("Phase = %v, want IMEUpdate", update.Phase)
	}

	end := input.IMECompositionEvent{Phase: input.IMEEnd, Text: "あい", Cursor: -1}
	if end.Phase != input.IMEEnd {
		t.Fatalf("Phase = %v, want IMEEnd", end.Phase)
	}

	if start.Phase == end.Phase {
		t.Fatalf("IMEStart compared equal to IMEEnd")
	}
	var imePhase input.IMEPhase = start.Phase
	if imePhase != input.IMEStart {
		t.Fatalf("imePhase = %v, want IMEStart", imePhase)
	}
}
