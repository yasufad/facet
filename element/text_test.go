package element

import (
	"testing"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/text"
)

func TestTextLifecycleAndMeasurement(t *testing.T) {
	frame := newFakeFrame()
	red := colour.Rgba{R: 1, G: 0, B: 0, A: 1}

	txt := NewText("Facet Text Element").
		FontSize(16).
		LineHeight(24).
		TextColour(red).
		FontWeight(text.WeightBold).
		FontStyle(text.StyleNormal)

	if txt.Content() != "Facet Text Element" {
		t.Fatalf("expected content 'Facet Text Element', got %q", txt.Content())
	}

	// 1. RequestLayout
	frame.phase = phaseLayoutRequested
	nodeID := txt.RequestLayout(frame)

	if len(frame.nodes) != 1 {
		t.Fatalf("expected 1 layout node, got %d", len(frame.nodes))
	}
	measureCb, ok := frame.measureCallbacks[nodeID]
	if !ok || measureCb == nil {
		t.Fatalf("expected measure callback registered for node %v", nodeID)
	}

	// Test measure callback with unconstrained available space
	avail := layout.Size[layout.AvailableSpace]{
		Width:  layout.MaxContent(),
		Height: layout.MaxContent(),
	}
	known := layout.Size[layout.OptF32]{}
	measuredSize := measureCb(known, avail)

	if measuredSize.Width <= 0 {
		t.Fatalf("expected measured width > 0 from text shaping, got %v", measuredSize.Width)
	}
	if measuredSize.Height != 24 {
		t.Fatalf("expected line height 24, got %v", measuredSize.Height)
	}

	// Test measure callback with known width
	knownWithWidth := layout.Size[layout.OptF32]{
		Width: layout.SomeOptF32(120),
	}
	measuredKnown := measureCb(knownWithWidth, avail)
	if measuredKnown.Width != 120 {
		t.Fatalf("expected known width 120, got %v", measuredKnown.Width)
	}

	// 2. Prepaint
	bounds := geometry.NewBounds(
		geometry.NewPoint[geometry.Pixels](10, 20),
		measuredSize,
	)
	frame.phase = phasePrepainted
	txt.Prepaint(frame, bounds)

	// 3. Paint
	frame.phase = phasePainted
	txt.Paint(frame, bounds)

	if len(frame.monoSprites) == 0 {
		t.Fatal("expected monochrome sprites emitted for glyphs, got 0")
	}

	// Verify sprite colour
	for i, sp := range frame.monoSprites {
		if sp.Colour != red {
			t.Errorf("sprite %d: expected colour %v, got %v", i, red, sp.Colour)
		}
		if sp.Bounds.Size.Width <= 0 || sp.Bounds.Size.Height <= 0 {
			t.Errorf("sprite %d: invalid bounds %v", i, sp.Bounds)
		}
	}
}

func TestTextEmptyString(t *testing.T) {
	frame := newFakeFrame()
	txt := NewText("")

	frame.phase = phaseLayoutRequested
	nodeID := txt.RequestLayout(frame)

	measureCb := frame.measureCallbacks[nodeID]
	avail := layout.Size[layout.AvailableSpace]{
		Width:  layout.MaxContent(),
		Height: layout.MaxContent(),
	}
	measured := measureCb(layout.Size[layout.OptF32]{}, avail)
	if measured.Width != 0 {
		t.Fatalf("expected empty text width 0, got %v", measured.Width)
	}

	bounds := geometry.NewBounds(geometry.NewPoint[geometry.Pixels](0, 0), measured)
	frame.phase = phasePrepainted
	txt.Prepaint(frame, bounds)

	frame.phase = phasePainted
	txt.Paint(frame, bounds)

	if len(frame.monoSprites) != 0 {
		t.Fatalf("expected 0 sprites for empty string, got %d", len(frame.monoSprites))
	}
}

func TestTextSetContent(t *testing.T) {
	txt := NewText("Initial")
	txt.SetContent("Updated")
	if txt.Content() != "Updated" {
		t.Fatalf("expected content 'Updated', got %q", txt.Content())
	}
}

func TestTextStyling(t *testing.T) {
	green := colour.Rgba{R: 0, G: 1, B: 0, A: 1}
	hsla := colour.Hsla{H: 120, S: 1, L: 0.5, A: 1}
	bg := colour.Rgba{R: 0, G: 0, B: 0, A: 0.5}
	bgHsla := colour.Hsla{H: 0, S: 0, L: 0, A: 0.5}

	txt := NewText("Styled").
		TextColour(green).
		TextColourHsla(hsla).
		FontFamily("Arial").
		FontFeatures([]text.FontFeature{{Tag: "liga", Value: 1}}).
		FontFallbacks([]string{"Segoe UI"}).
		FontSize(18).
		LineHeight(22).
		FontWeight(text.WeightBold).
		FontStyle(text.StyleItalic).
		TextBackgroundColour(bg).
		TextBackgroundColourHsla(bgHsla).
		Underline(style.UnderlineStyle{Thickness: 1}).
		Strikethrough(style.StrikethroughStyle{Thickness: 1}).
		WhiteSpace(style.WhiteSpaceNowrap).
		TextOverflow(style.TextOverflowEllipsis).
		TextAlign(style.TextAlignCentre).
		LineClamp(2)

	txt.ClearUnderline()
	txt.ClearStrikethrough()

	var r style.Refinement
	r.SetFontSize(20)
	txt.Refine(r)
}

func TestTextPhasePanics(t *testing.T) {
	frame := newFakeFrame()
	txt := NewText("Panic Test")

	// Calling Prepaint before RequestLayout should panic
	assertPanics(t, "Prepaint before RequestLayout", func() {
		txt.Prepaint(frame, geometry.Bounds[geometry.Pixels]{})
	})

	// Calling Paint before Prepaint should panic
	assertPanics(t, "Paint before Prepaint", func() {
		txt.Paint(frame, geometry.Bounds[geometry.Pixels]{})
	})

	// RequestLayout
	frame.phase = phaseLayoutRequested
	_ = txt.RequestLayout(frame)

	// Multiple RequestLayout calls should panic
	assertPanics(t, "duplicate RequestLayout", func() {
		txt.RequestLayout(frame)
	})
}

func assertPanics(t *testing.T, msg string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected %s to panic, but it did not", msg)
		}
	}()
	fn()
}
