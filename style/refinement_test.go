package style

import (
	"testing"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/text"
)

func TestEmptyRefinement(t *testing.T) {
	empty := Refinement{}
	if !empty.IsEmpty() {
		t.Errorf("Refinement{}.IsEmpty() = false, want true")
	}

	mergedEmpty := Refinement{}
	mergedEmpty.MergeFrom(&empty)
	if !mergedEmpty.IsEmpty() {
		t.Errorf("mergedEmpty.IsEmpty() = false, want true")
	}

	// Refining Default() with an empty refinement must not modify any field.
	base := Default()
	expected := Default()
	base.Refine(empty)

	if base.Display != expected.Display {
		t.Errorf("Display = %v, want %v", base.Display, expected.Display)
	}
	if base.FlexShrink != expected.FlexShrink {
		t.Errorf("FlexShrink = %v, want %v", base.FlexShrink, expected.FlexShrink)
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
	if zeroStyle.FlexShrink != 0 {
		t.Errorf("zeroStyle.Refine(empty) = %#v, want zero Style", zeroStyle)
	}
}

func TestZeroValueOverride(t *testing.T) {
	// Base style has FlexShrink = 1.0. A refinement setting FlexShrink = 0.0
	// must override the base, distinguishing zero-value from unset.
	base := Default()
	if base.FlexShrink != 1.0 {
		t.Fatalf("Default().FlexShrink = %v, want 1.0", base.FlexShrink)
	}

	var r Refinement
	r.SetFlexShrink(0.0)
	if r.IsEmpty() {
		t.Fatalf("Refinement.SetFlexShrink(0.0).IsEmpty() = true, want false")
	}

	base.Refine(r)
	if base.FlexShrink != 0.0 {
		t.Errorf("after Refine(FlexShrink(0.0)), FlexShrink = %v, want 0.0", base.FlexShrink)
	}
}

func TestPerEdgeGranularity(t *testing.T) {
	// Refinements must be able to override a single edge without touching the others.
	base := Default()
	base.Padding = NewEdges(Px(10), Px(20), Px(30), Px(40))

	var r Refinement
	r.SetPaddingLeft(Px(99))

	base.Refine(r)

	if base.Padding.Top != Px(10) {
		t.Errorf("Padding.Top = %v, want 10", base.Padding.Top)
	}
	if base.Padding.Right != Px(20) {
		t.Errorf("Padding.Right = %v, want 20", base.Padding.Right)
	}
	if base.Padding.Bottom != Px(30) {
		t.Errorf("Padding.Bottom = %v, want 30", base.Padding.Bottom)
	}
	if base.Padding.Left != Px(99) {
		t.Errorf("Padding.Left = %v, want 99", base.Padding.Left)
	}
}

func TestOmittedFieldPreservation(t *testing.T) {
	// Refinements that do not mention a field must not touch that field on the base.
	red := colour.Rgb(0xff0000)
	blue := colour.Rgb(0x0000ff)

	base := Default()
	base.FlexShrink = 0.75
	base.FlexGrow = 3.0
	base.Background = red

	var r Refinement
	r.SetBackground(blue)
	base.Refine(r)

	if base.FlexShrink != 0.75 {
		t.Errorf("FlexShrink = %v, want 0.75", base.FlexShrink)
	}
	if base.FlexGrow != 3.0 {
		t.Errorf("FlexGrow = %v, want 3.0", base.FlexGrow)
	}
	if base.Background != blue {
		t.Errorf("Background = %v, want %v", base.Background, blue)
	}
}

func TestRefinementMergeFrom(t *testing.T) {
	red := colour.Rgb(0xff0000)
	green := colour.Rgb(0x00ff00)

	var r1 Refinement
	r1.SetFlexShrink(0.5)
	r1.SetFlexGrow(2.0)
	r1.SetBackground(red)

	var r2 Refinement
	r2.SetFlexShrink(0.0)
	r2.SetBackground(green)

	// r1 merged with r2: r2 overrides FlexShrink (to 0.0) and Background (to green),
	// while FlexGrow remains 2.0 from r1.
	merged := r1
	merged.MergeFrom(&r2)

	base := Default()
	base.Refine(merged)

	if base.FlexShrink != 0.0 {
		t.Errorf("FlexShrink = %v, want 0.0 (overridden by r2)", base.FlexShrink)
	}
	if base.Background != green {
		t.Errorf("Background = %v, want %v (overridden by r2)", base.Background, green)
	}
	if base.FlexGrow != 2.0 {
		t.Errorf("FlexGrow = %v, want 2.0 (retained from r1)", base.FlexGrow)
	}

	// Inverse merge: r2 merged with r1: r1 overrides FlexShrink (to 0.5) and
	// Background (to red), FlexGrow is set to 2.0.
	invMerged := r2
	invMerged.MergeFrom(&r1)
	invBase := Default()
	invBase.Refine(invMerged)

	if invBase.FlexShrink != 0.5 {
		t.Errorf("inverse FlexShrink = %v, want 0.5 (overridden by r1)", invBase.FlexShrink)
	}
	if invBase.Background != red {
		t.Errorf("inverse Background = %v, want %v (overridden by r1)", invBase.Background, red)
	}
	if invBase.FlexGrow != 2.0 {
		t.Errorf("inverse FlexGrow = %v, want 2.0 (set by r1)", invBase.FlexGrow)
	}
}

func TestHighWordProperty(t *testing.T) {
	// FlexGrow is at bit 64 (high word). Test that Refine and MergeFrom copy it correctly.
	var r Refinement
	r.SetFlexGrow(2.5)
	r.SetFontSize(24)
	r.SetFontFamily("Fira Code")

	base := Default()
	base.Refine(r)
	if base.FlexGrow != 2.5 {
		t.Errorf("Refine skipped high-word FlexGrow = %v, want 2.5", base.FlexGrow)
	}
	if base.Text.FontSize != 24 {
		t.Errorf("Refine skipped high-word FontSize = %v, want 24", base.Text.FontSize)
	}
	if base.Text.FontFamily != "Fira Code" {
		t.Errorf("Refine skipped high-word FontFamily = %v, want Fira Code", base.Text.FontFamily)
	}

	// MergeFrom into a non-empty receiver.
	var r1 Refinement
	r1.SetFlexShrink(0.5)

	var r2 Refinement
	r2.SetFlexGrow(2.5)
	r2.SetFontFamily("Fira Code")

	r1.MergeFrom(&r2)
	if r1.flexGrow != 2.5 {
		t.Errorf("MergeFrom skipped high-word FlexGrow = %v, want 2.5", r1.flexGrow)
	}
	if r1.fontFamily != "Fira Code" {
		t.Errorf("MergeFrom skipped high-word FontFamily = %v, want Fira Code", r1.fontFamily)
	}
	if r1.flexShrink != 0.5 {
		t.Errorf("MergeFrom clobbered low-word FlexShrink = %v, want 0.5", r1.flexShrink)
	}
}

func TestToLayoutConversion(t *testing.T) {
	var r Refinement
	r.SetDisplay(DisplayFlex)
	r.SetPosition(PositionRelative)
	r.SetFlexDirection(FlexDirectionColumn)
	r.SetFlexWrap(FlexWrapWrap)
	r.SetFlexGrow(1.5)
	r.SetFlexShrink(0.5)
	r.SetFlexBasis(Px(100))
	r.SetWidth(Px(200))
	r.SetHeight(Rem(2)) // 2 rems = 32 pixels at remSize = 16
	r.SetPadding(Px(8))
	r.SetMarginTop(Px(4))
	r.SetBorderWidth(2)
	r.SetGapRow(Px(12))
	r.SetGapCol(Px(16))
	r.SetAlignItems(AlignItemsCentre)
	r.SetJustifyContent(AlignContentSpaceBetween)
	r.SetAspectRatio(1.77)

	s := Default().Refined(r)
	remSize := geometry.Pixels(16)
	l := s.ToLayout(remSize)

	if l.Display != layout.DisplayFlex {
		t.Errorf("l.Display = %v, want %v", l.Display, layout.DisplayFlex)
	}
	if l.FlexDirection != layout.FlexColumn {
		t.Errorf("l.FlexDirection = %v, want %v", l.FlexDirection, layout.FlexColumn)
	}
	if l.FlexWrap != layout.FlexWrapWrap {
		t.Errorf("l.FlexWrap = %v, want %v", l.FlexWrap, layout.FlexWrapWrap)
	}
	if l.FlexGrow != 1.5 {
		t.Errorf("l.FlexGrow = %v, want 1.5", l.FlexGrow)
	}
	if l.FlexShrink != 0.5 {
		t.Errorf("l.FlexShrink = %v, want 0.5", l.FlexShrink)
	}
	if l.AspectRatio == nil || *l.AspectRatio != 1.77 {
		t.Errorf("l.AspectRatio = %v, want 1.77", l.AspectRatio)
	}
	if l.AlignItems == nil || l.AlignItems.Keyword != layout.AlignItemsCentre.Keyword {
		t.Errorf("l.AlignItems = %v, want Centre", l.AlignItems)
	}
	if l.JustifyContent == nil || l.JustifyContent.Keyword != layout.AlignContentSpaceBetween.Keyword {
		t.Errorf("l.JustifyContent = %v, want SpaceBetween", l.JustifyContent)
	}
}

func TestTypographyRefinement(t *testing.T) {
	var r Refinement
	r.SetTextColour(colour.Rgb(0x123456))
	r.SetFontFamily("Inter")
	r.SetFontSize(18)
	r.SetLineHeight(24)
	r.SetFontWeight(text.WeightBold)
	r.SetFontStyle(text.StyleItalic)
	r.SetUnderline(UnderlineStyle{Thickness: 2, Wavy: true})

	s := Default().Refined(r)

	if s.Text.Colour != colour.Rgb(0x123456) {
		t.Errorf("Text.Colour = %v, want 0x123456", s.Text.Colour)
	}
	if s.Text.FontFamily != "Inter" {
		t.Errorf("Text.FontFamily = %v, want Inter", s.Text.FontFamily)
	}
	if s.Text.FontSize != 18 {
		t.Errorf("Text.FontSize = %v, want 18", s.Text.FontSize)
	}
	if s.Text.LineHeight != 24 {
		t.Errorf("Text.LineHeight = %v, want 24", s.Text.LineHeight)
	}
	if s.Text.FontWeight != text.WeightBold {
		t.Errorf("Text.FontWeight = %v, want Bold", s.Text.FontWeight)
	}
	if s.Text.FontStyle != text.StyleItalic {
		t.Errorf("Text.FontStyle = %v, want Italic", s.Text.FontStyle)
	}
	if s.Text.Underline == nil || s.Text.Underline.Thickness != 2 || !s.Text.Underline.Wavy {
		t.Errorf("Text.Underline = %v, want thickness 2 wavy", s.Text.Underline)
	}
}

func TestDistinctPropertyIndices(t *testing.T) {
	props := []struct {
		name  string
		index uint8
	}{
		{"propDisplay", propDisplay},
		{"propPosition", propPosition},
		{"propVisibility", propVisibility},
		{"propOverflowX", propOverflowX},
		{"propOverflowY", propOverflowY},
		{"propScrollbarWidth", propScrollbarWidth},
		{"propInsetTop", propInsetTop},
		{"propInsetRight", propInsetRight},
		{"propInsetBottom", propInsetBottom},
		{"propInsetLeft", propInsetLeft},
		{"propMarginTop", propMarginTop},
		{"propMarginRight", propMarginRight},
		{"propMarginBottom", propMarginBottom},
		{"propMarginLeft", propMarginLeft},
		{"propPaddingTop", propPaddingTop},
		{"propPaddingRight", propPaddingRight},
		{"propPaddingBottom", propPaddingBottom},
		{"propPaddingLeft", propPaddingLeft},
		{"propBorderWidthTop", propBorderWidthTop},
		{"propBorderWidthRight", propBorderWidthRight},
		{"propBorderWidthBottom", propBorderWidthBottom},
		{"propBorderWidthLeft", propBorderWidthLeft},
		{"propBorderColour", propBorderColour},
		{"propBorderStyle", propBorderStyle},
		{"propCornerRadiusTopLeft", propCornerRadiusTopLeft},
		{"propCornerRadiusTopRight", propCornerRadiusTopRight},
		{"propCornerRadiusBottomRight", propCornerRadiusBottomRight},
		{"propCornerRadiusBottomLeft", propCornerRadiusBottomLeft},
		{"propSizeWidth", propSizeWidth},
		{"propSizeHeight", propSizeHeight},
		{"propMinSizeWidth", propMinSizeWidth},
		{"propMinSizeHeight", propMinSizeHeight},
		{"propMaxSizeWidth", propMaxSizeWidth},
		{"propMaxSizeHeight", propMaxSizeHeight},
		{"propAspectRatio", propAspectRatio},
		{"propGapRow", propGapRow},
		{"propGapColumn", propGapColumn},
		{"propAlignItems", propAlignItems},
		{"propAlignSelf", propAlignSelf},
		{"propAlignContent", propAlignContent},
		{"propJustifyContent", propJustifyContent},
		{"propFlexDirection", propFlexDirection},
		{"propFlexWrap", propFlexWrap},
		{"propFlexBasis", propFlexBasis},
		{"propFlexShrink", propFlexShrink},
		{"propBackground", propBackground},
		{"propBoxShadow", propBoxShadow},
		{"propMouseCursor", propMouseCursor},
		{"propFlexGrow", propFlexGrow},
		{"propTextColour", propTextColour},
		{"propFontFamily", propFontFamily},
		{"propFontFeatures", propFontFeatures},
		{"propFontFallbacks", propFontFallbacks},
		{"propFontSize", propFontSize},
		{"propLineHeight", propLineHeight},
		{"propFontWeight", propFontWeight},
		{"propFontStyle", propFontStyle},
		{"propTextBackgroundColour", propTextBackgroundColour},
		{"propUnderline", propUnderline},
		{"propStrikethrough", propStrikethrough},
	}

	seen := make(map[uint8]string)
	for _, p := range props {
		if p.index >= 128 {
			t.Errorf("property %s index %d exceeds mask capacity of 128 bits", p.name, p.index)
		}
		if other, exists := seen[p.index]; exists {
			t.Errorf("property index collision: %s and %s share bit index %d", p.name, other, p.index)
		}
		seen[p.index] = p.name
	}
}

func TestTypographyPropertyIsolation(t *testing.T) {
	// Refinements that set a single typography property must not clobber
	// other typography properties on the base style.
	base := Default()
	base.Text.Colour = colour.Rgb(0xff0000)
	base.Text.FontFamily = "Inter"
	base.Text.FontWeight = text.WeightBold
	base.Text.LineHeight = 28
	base.Text.FontSize = 16

	var r Refinement
	r.SetFontSize(12)

	base.Refine(r)

	if base.Text.FontSize != 12 {
		t.Errorf("FontSize = %v, want 12", base.Text.FontSize)
	}
	if base.Text.Colour != colour.Rgb(0xff0000) {
		t.Errorf("refining FontSize clobbered Text.Colour: got %v, want %v", base.Text.Colour, colour.Rgb(0xff0000))
	}
	if base.Text.FontFamily != "Inter" {
		t.Errorf("refining FontSize clobbered Text.FontFamily: got %q, want \"Inter\"", base.Text.FontFamily)
	}
	if base.Text.FontWeight != text.WeightBold {
		t.Errorf("refining FontSize clobbered Text.FontWeight: got %v, want Bold", base.Text.FontWeight)
	}
	if base.Text.LineHeight != 28 {
		t.Errorf("refining FontSize clobbered Text.LineHeight: got %v, want 28", base.Text.LineHeight)
	}
}
