package geometry

import "testing"

func TestAxisInvert(t *testing.T) {
	if Horizontal.Invert() != Vertical {
		t.Fatalf("Horizontal.Invert = %v, want Vertical", Horizontal.Invert())
	}
	if Vertical.Invert() != Horizontal {
		t.Fatalf("Vertical.Invert = %v, want Horizontal", Vertical.Invert())
	}
}

func TestPointAlong(t *testing.T) {
	p := NewPoint(3, 4)
	if got := p.Along(Horizontal); got != 3 {
		t.Fatalf("Along(Horizontal) = %d, want 3", got)
	}
	if got := p.Along(Vertical); got != 4 {
		t.Fatalf("Along(Vertical) = %d, want 4", got)
	}
}

func TestPointApplyAlong(t *testing.T) {
	p := NewPoint(3, 4)

	horizontal := p.ApplyAlong(Horizontal, func(v int) int { return v + 10 })
	if horizontal != NewPoint(13, 4) {
		t.Fatalf("ApplyAlong(Horizontal) = %v, want (13,4)", horizontal)
	}
	// The other axis must be left untouched.
	vertical := p.ApplyAlong(Vertical, func(v int) int { return v * 2 })
	if vertical != NewPoint(3, 8) {
		t.Fatalf("ApplyAlong(Vertical) = %v, want (3,8)", vertical)
	}
}

func TestSizeAlongAndApplyAlong(t *testing.T) {
	s := NewSize(6, 8)
	if got := s.Along(Horizontal); got != 6 {
		t.Fatalf("Size.Along(Horizontal) = %d, want 6", got)
	}
	if got := s.Along(Vertical); got != 8 {
		t.Fatalf("Size.Along(Vertical) = %d, want 8", got)
	}

	got := s.ApplyAlong(Vertical, func(v int) int { return v + 1 })
	if got != NewSize(6, 9) {
		t.Fatalf("Size.ApplyAlong(Vertical) = %v, want (6,9)", got)
	}
}
