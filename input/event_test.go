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
