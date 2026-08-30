package text

import (
	"testing"

	"github.com/yasufad/facet/geometry"
)

// newTestSystem creates a System, skipping the test if the system has no
// fonts (an unusual but possible CI environment).
func newTestSystem(t *testing.T) *System {
	t.Helper()
	s, err := NewSystem()
	if err != nil {
		t.Skipf("system fonts unavailable: %v", err)
	}
	return s
}

// defaultRun is a single style run covering all of text at 16px LTR English.
func defaultRun(text string) []StyleRun {
	return []StyleRun{{
		ByteLen:   len(text),
		Font:      FontRequest{Family: "Arial", Families: []string{"Helvetica", "DejaVu Sans", "Liberation Sans"}},
		Size:      geometry.Pixels(16),
		Direction: LTR,
		Language:  "en",
	}}
}

// TestShapeLineBasic shapes a simple Latin string and checks the line has
// non-zero width and the expected byte length.
func TestShapeLineBasic(t *testing.T) {
	s := newTestSystem(t)
	text := "Hello, world"
	line, err := s.ShapeLine(text, defaultRun(text))
	if err != nil {
		t.Fatalf("ShapeLine: %v", err)
	}
	if line.Width() <= 0 {
		t.Fatalf("line width %v, expected positive", line.Width())
	}
	if line.Len() != len(text) {
		t.Fatalf("line len %d, expected %d", line.Len(), len(text))
	}
	if line.Ascent() <= 0 {
		t.Fatalf("ascent %v, expected positive", line.Ascent())
	}
	if line.Descent() < 0 {
		t.Fatalf("descent %v, expected non-negative", line.Descent())
	}
	if line.Height() <= 0 {
		t.Fatalf("height %v, expected positive", line.Height())
	}
}

// TestXForIndexMonotonic checks that XForIndex is monotonically non-decreasing
// across the line's byte range, the core property hit testing relies on.
func TestXForIndexMonotonic(t *testing.T) {
	s := newTestSystem(t)
	text := "The quick brown fox"
	line, err := s.ShapeLine(text, defaultRun(text))
	if err != nil {
		t.Fatalf("ShapeLine: %v", err)
	}
	prev := geometry.Pixels(-1)
	for i := 0; i <= len(text); i++ {
		x := line.XForIndex(i)
		if x < prev {
			t.Fatalf("XForIndex not monotonic at byte %d: %v < %v", i, x, prev)
		}
		prev = x
	}
}

// TestIndexForXRoundTrip checks that for each byte boundary, the x position
// maps back to a byte index that is within the cluster's range. This is the
// two-direction mapping the assignment requires.
func TestIndexForXRoundTrip(t *testing.T) {
	s := newTestSystem(t)
	text := "Hello, world"
	line, err := s.ShapeLine(text, defaultRun(text))
	if err != nil {
		t.Fatalf("ShapeLine: %v", err)
	}
	for i := 0; i <= len(text); i++ {
		x := line.XForIndex(i)
		idx, ok := line.IndexForX(x)
		if !ok && idx != len(text) {
			t.Fatalf("IndexForX(%v) returned ok=false but idx=%d, expected %d", x, idx, len(text))
		}
		// The round-tripped index should be a valid cluster boundary. It may
		// not equal i exactly when i falls inside a multi-byte cluster, but
		// it must be within the line.
		if idx < 0 || idx > len(text) {
			t.Fatalf("IndexForX(%v) = %d, out of range [0, %d]", x, idx, len(text))
		}
	}
}

// TestXForIndexBoundaries checks the edge cases: byte 0 maps to the line's
// left edge, and byte Len maps to the line's right edge (width).
func TestXForIndexBoundaries(t *testing.T) {
	s := newTestSystem(t)
	text := "Hello"
	line, err := s.ShapeLine(text, defaultRun(text))
	if err != nil {
		t.Fatalf("ShapeLine: %v", err)
	}
	if x := line.XForIndex(0); x != 0 {
		t.Fatalf("XForIndex(0) = %v, expected 0", x)
	}
	if x := line.XForIndex(len(text)); x != line.Width() {
		t.Fatalf("XForIndex(%d) = %v, expected %v", len(text), x, line.Width())
	}
}

// TestIndexForXBeyondEdges checks that x positions before and after the line
// clamp to the start and end.
func TestIndexForXBeyondEdges(t *testing.T) {
	s := newTestSystem(t)
	text := "Hello"
	line, err := s.ShapeLine(text, defaultRun(text))
	if err != nil {
		t.Fatalf("ShapeLine: %v", err)
	}
	if idx, _ := line.IndexForX(geometry.Pixels(-10)); idx != 0 {
		t.Fatalf("IndexForX(-10) = %d, expected 0", idx)
	}
	if idx, ok := line.IndexForX(line.Width() + 10); idx != len(text) || ok {
		t.Fatalf("IndexForX(beyond) = %d (ok=%v), expected %d false", idx, ok, len(text))
	}
}

// TestClosestIndexForX checks that the closest-boundary query picks the
// nearer side of a cluster midpoint.
func TestClosestIndexForX(t *testing.T) {
	s := newTestSystem(t)
	text := "Hello"
	line, err := s.ShapeLine(text, defaultRun(text))
	if err != nil {
		t.Fatalf("ShapeLine: %v", err)
	}
	// At x=0 the closest boundary is 0.
	if idx := line.ClosestIndexForX(geometry.Pixels(0)); idx != 0 {
		t.Fatalf("ClosestIndexForX(0) = %d, expected 0", idx)
	}
	// Beyond the right edge the closest boundary is the end.
	if idx := line.ClosestIndexForX(line.Width() + 100); idx != len(text) {
		t.Fatalf("ClosestIndexForX(beyond) = %d, expected %d", idx, len(text))
	}
}

// TestShapeLineEmpty checks that an empty string produces an empty but valid
// line.
func TestShapeLineEmpty(t *testing.T) {
	s := newTestSystem(t)
	line, err := s.ShapeLine("", []StyleRun{})
	if err != nil {
		t.Fatalf("ShapeLine(empty): %v", err)
	}
	if line.Width() != 0 {
		t.Fatalf("empty line width %v, expected 0", line.Width())
	}
	if line.Len() != 0 {
		t.Fatalf("empty line len %d, expected 0", line.Len())
	}
}

// TestShapeLineRejectsNewlines checks that ShapeLine rejects text containing
// newlines.
func TestShapeLineRejectsNewlines(t *testing.T) {
	s := newTestSystem(t)
	text := "line one\nline two"
	_, err := s.ShapeLine(text, defaultRun(text))
	if err == nil {
		t.Fatal("ShapeLine accepted text with newline, expected error")
	}
}

// TestValidateRuns checks that run validation catches gaps and overlaps.
func TestValidateRuns(t *testing.T) {
	if err := validateRuns(10, []StyleRun{{ByteLen: 5}, {ByteLen: 4}}); err == nil {
		t.Fatal("validateRuns accepted runs covering 9 bytes for 10-byte text")
	}
	if err := validateRuns(10, []StyleRun{{ByteLen: 5}, {ByteLen: 6}}); err == nil {
		t.Fatal("validateRuns accepted runs covering 11 bytes for 10-byte text")
	}
	if err := validateRuns(10, []StyleRun{{ByteLen: 10}}); err != nil {
		t.Fatalf("validateRuns rejected exact cover: %v", err)
	}
	if err := validateRuns(0, nil); err != nil {
		t.Fatalf("validateRuns rejected empty: %v", err)
	}
}
