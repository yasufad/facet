package text

import (
	"testing"

	"github.com/yasufad/facet/geometry"
)

// TestWrapTextMultipleLines shapes a long string and checks it wraps into
// more than one line when the width is narrow.
func TestWrapTextMultipleLines(t *testing.T) {
	s := newTestSystem(t)
	text := "The quick brown fox jumps over the lazy dog"
	lines, err := s.WrapText(text, defaultRun(text), geometry.Pixels(80))
	if err != nil {
		t.Fatalf("WrapText: %v", err)
	}
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %d", len(lines))
	}
	// Every line except possibly the last should fit within the width.
	for i, l := range lines {
		if i < len(lines)-1 && l.Width() > geometry.Pixels(80)+1 {
			t.Fatalf("line %d width %v exceeds 80px", i, l.Width())
		}
	}
}

// TestWrapTextSingleLine checks that short text fits on one line.
func TestWrapTextSingleLine(t *testing.T) {
	s := newTestSystem(t)
	text := "Hi"
	lines, err := s.WrapText(text, defaultRun(text), geometry.Pixels(1000))
	if err != nil {
		t.Fatalf("WrapText: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
}

// TestWrapTextAwkwardBoundary wraps text at a width that forces a break in
// the middle of a word, checking the wrapper does not panic and produces
// lines whose byte ranges cover the whole text.
func TestWrapTextAwkwardBoundary(t *testing.T) {
	s := newTestSystem(t)
	text := "supercalifragilistic"
	lines, err := s.WrapText(text, defaultRun(text), geometry.Pixels(30))
	if err != nil {
		t.Fatalf("WrapText: %v", err)
	}
	if len(lines) < 2 {
		t.Fatalf("expected word to wrap at 30px, got %d lines", len(lines))
	}
	// The total byte length of all lines should equal the text length.
	total := 0
	for _, l := range lines {
		total += l.Len()
	}
	if total != len(text) {
		t.Fatalf("wrapped lines cover %d bytes, text is %d bytes", total, len(text))
	}
}

// TestWrapTextEmpty checks that wrapping an empty string produces one empty
// line.
func TestWrapTextEmpty(t *testing.T) {
	s := newTestSystem(t)
	lines, err := s.WrapText("", []StyleRun{}, geometry.Pixels(100))
	if err != nil {
		t.Fatalf("WrapText(empty): %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 empty line, got %d", len(lines))
	}
}

// TestWrapTextNewlines checks that newlines force line breaks.
func TestWrapTextNewlines(t *testing.T) {
	s := newTestSystem(t)
	text := "line one\nline two\nline three"
	run := defaultRun(text)
	lines, err := s.WrapText(text, run, geometry.Pixels(1000))
	if err != nil {
		t.Fatalf("WrapText: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines from newlines, got %d", len(lines))
	}
}
