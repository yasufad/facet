package style

import (
	"testing"

	"github.com/yasufad/facet/colour"
)

func TestEmptyRefinement(t *testing.T) {
	empty := Refinement{}
	if !empty.IsEmpty() {
		t.Errorf("Refinement{}.IsEmpty() = false, want true")
	}

	mergedEmpty := empty.Merge(Refinement{})
	if !mergedEmpty.IsEmpty() {
		t.Errorf("empty.Merge(empty).IsEmpty() = false, want true")
	}

	// Refining Default() with an empty refinement must not modify any field.
	base := Default()
	expected := Default()
	base.Refine(empty)

	if base.Display != expected.Display {
		t.Errorf("Display = %v, want %v", base.Display, expected.Display)
	}
	if base.Opacity != expected.Opacity {
		t.Errorf("Opacity = %v, want %v", base.Opacity, expected.Opacity)
	}
	if base.Background != expected.Background {
		t.Errorf("Background = %v, want %v", base.Background, expected.Background)
	}
	if base.FlexGrow != expected.FlexGrow {
		t.Errorf("FlexGrow = %v, want %v", base.FlexGrow, expected.FlexGrow)
	}

	// Refining a zero-initialised Style{} with an empty refinement must leave it zeroed.
	var zeroStyle Style
	zeroStyle.Refine(empty)
	if zeroStyle != (Style{}) {
		t.Errorf("zeroStyle.Refine(empty) = %#v, want zero Style", zeroStyle)
	}
}

func TestZeroValueOverride(t *testing.T) {
	// Base style has Opacity = 1.0. A refinement setting Opacity = 0.0 must
	// override the base, distinguishing zero-value from unset.
	base := Default()
	if base.Opacity != 1.0 {
		t.Fatalf("Default().Opacity = %v, want 1.0", base.Opacity)
	}

	r := Refinement{}.Opacity(0.0)
	if r.IsEmpty() {
		t.Fatalf("Refinement{}.Opacity(0.0).IsEmpty() = true, want false")
	}

	base.Refine(r)
	if base.Opacity != 0.0 {
		t.Errorf("after Refine(Opacity(0.0)), Opacity = %v, want 0.0", base.Opacity)
	}
}

func TestOmittedFieldPreservation(t *testing.T) {
	// Refinements that do not mention a field must not touch that field on the base.
	red := colour.Rgb(0xff0000)
	blue := colour.Rgb(0x0000ff)

	base := Default()
	base.Opacity = 0.75
	base.FlexGrow = 3.0
	base.Background = red.Hsla()

	r := Refinement{}.Bg(blue)
	base.Refine(r)

	if base.Opacity != 0.75 {
		t.Errorf("Opacity = %v, want 0.75", base.Opacity)
	}
	if base.FlexGrow != 3.0 {
		t.Errorf("FlexGrow = %v, want 3.0", base.FlexGrow)
	}
	if base.Background != blue.Hsla() {
		t.Errorf("Background = %v, want %v", base.Background, blue.Hsla())
	}
}

func TestRefinementMerge(t *testing.T) {
	red := colour.Rgb(0xff0000)
	green := colour.Rgb(0x00ff00)

	r1 := Refinement{}.Opacity(0.5).FlexGrow(2.0).Bg(red)
	r2 := Refinement{}.Opacity(0.0).Bg(green)

	// r1 merged with r2: r2 overrides Opacity (to 0.0) and Background (to green),
	// while FlexGrow remains 2.0 from r1.
	merged := r1.Merge(r2)

	base := Default()
	base.Refine(merged)

	if base.Opacity != 0.0 {
		t.Errorf("Opacity = %v, want 0.0 (overridden by r2)", base.Opacity)
	}
	if base.Background != green.Hsla() {
		t.Errorf("Background = %v, want %v (overridden by r2)", base.Background, green.Hsla())
	}
	if base.FlexGrow != 2.0 {
		t.Errorf("FlexGrow = %v, want 2.0 (retained from r1)", base.FlexGrow)
	}

	// Inverse merge: r2 merged with r1: r1 overrides Opacity (to 0.5) and
	// Background (to red), FlexGrow is set to 2.0.
	invMerged := r2.Merge(r1)
	invBase := Default()
	invBase.Refine(invMerged)

	if invBase.Opacity != 0.5 {
		t.Errorf("inverse Opacity = %v, want 0.5 (overridden by r1)", invBase.Opacity)
	}
	if invBase.Background != red.Hsla() {
		t.Errorf("inverse Background = %v, want %v (overridden by r1)", invBase.Background, red.Hsla())
	}
	if invBase.FlexGrow != 2.0 {
		t.Errorf("inverse FlexGrow = %v, want 2.0 (set by r1)", invBase.FlexGrow)
	}
}

func TestFluentBuilder(t *testing.T) {
	yellow := colour.Rgb(0xffff00)

	r := Refinement{}.
		Block().
		Opacity(0.85).
		Bg(yellow).
		FlexGrow(1.5)

	s := Default().Refined(r)

	if s.Display != DisplayBlock {
		t.Errorf("Display = %v, want %v", s.Display, DisplayBlock)
	}
	if s.Opacity != 0.85 {
		t.Errorf("Opacity = %v, want 0.85", s.Opacity)
	}
	if s.Background != yellow.Hsla() {
		t.Errorf("Background = %v, want %v", s.Background, yellow.Hsla())
	}
	if s.FlexGrow != 1.5 {
		t.Errorf("FlexGrow = %v, want 1.5", s.FlexGrow)
	}
}
