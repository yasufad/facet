package input_test

import (
	"testing"

	"github.com/yasufad/facet/input"
)

// A caller above input must be able to build and inspect a wheel event
// without importing platform. This file imports only input; if it stops
// compiling, some part of the vocabulary a handler needs has gone missing.
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

// TestAliasedEventVocabularyNamesNoPlatform constructs a value of every
// aliased event type using only input names. If any alias or its associated
// constant goes missing, this stops compiling — which is the signal that the
// vocabulary a caller above input needs is incomplete. This file imports only
// input; a platform import here would defeat the purpose.
func TestAliasedEventVocabularyNamesNoPlatform(t *testing.T) {
	// KeyEvent — the type is nameable; field types a handler inspects
	// (KeyCode, KeyPhase, Modifiers) are not yet aliased, so a zero value
	// is all a caller can construct from input alone today.
	var key input.KeyEvent
	_ = key

	// PointerEvent — same situation as KeyEvent.
	var ptr input.PointerEvent
	_ = ptr

	// TextEvent — Text is a string, so a caller can build a full value.
	txt := input.TextEvent{Text: "Facet"}
	if txt.Text != "Facet" {
		t.Fatalf("Text = %q, want %q", txt.Text, "Facet")
	}

	// WheelEvent — exercised in detail above; included here for completeness.
	var wheel input.WheelEvent
	wheel.Delta.Unit = input.ScrollPixels
	_ = wheel

	// IMECompositionEvent — the new alias. A handler switches on Phase, so
	// the constants have to be nameable from input too.
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
}
