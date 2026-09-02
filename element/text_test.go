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

// TestTextPaintCentresGlyphsOnTallerLineHeight pins the half-leading fix: a
// LineHeight taller than the font's own ascent+descent must split the extra
// space evenly above and below the glyphs (CSS's rule), not leave it all
// below.
//
// It compares the painted sprite position for the same glyph at two line
// heights rather than asserting an absolute pixel, since the absolute
// position depends on font metrics this test does not control. Holding
// content, font and fakeFrame's fixed mock glyph bounds constant between the
// two paints isolates the shift to exactly the half-leading term: the
// sprite's Y must move down by scale * extra/2, where extra is how much
// taller the second line height is than the shaped line's own height.
func TestTextPaintCentresGlyphsOnTallerLineHeight(t *testing.T) {
	topSpriteY := func(lineHeight geometry.Pixels) (y geometry.ScaledPixels, shapedHeight geometry.Pixels) {
		frame := newFakeFrame()
		txt := NewText("A").FontSize(16).LineHeight(lineHeight)

		frame.phase = phaseLayoutRequested
		nodeID := txt.RequestLayout(frame)
		measured := frame.measureCallbacks[nodeID](layout.Size[layout.OptF32]{}, layout.Size[layout.AvailableSpace]{
			Width:  layout.MaxContent(),
			Height: layout.MaxContent(),
		})
		if txt.shapedLine == nil {
			t.Fatal("expected the measure pass to shape the line")
		}

		bounds := geometry.NewBounds(geometry.NewPoint[geometry.Pixels](0, 0), measured)
		frame.phase = phasePrepainted
		txt.Prepaint(frame, bounds)
		frame.phase = phasePainted
		txt.Paint(frame, bounds)

		if len(frame.monoSprites) == 0 {
			t.Fatal("expected at least one glyph sprite")
		}
		return frame.monoSprites[0].Bounds.Origin.Y, txt.shapedLine.Height()
	}

	naturalY, shapedHeight := topSpriteY(0) // LineHeight(0) falls back to the shaped line's own height.
	const extra = geometry.Pixels(40)
	tallY, _ := topSpriteY(shapedHeight + extra)

	const scale = float32(2.0) // fakeFrame's default ScaleFactor.
	want := geometry.ScaledPixels(float32(extra) / 2 * scale)
	got := tallY - naturalY

	const epsilon = geometry.ScaledPixels(0.01)
	if diff := got - want; diff < -epsilon || diff > epsilon {
		t.Fatalf("sprite shifted by %v when line height grew by %v, want %v: glyphs are not centred in the taller box", got, extra, want)
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

func TestTextInheritsParentStyleAndHover(t *testing.T) {
	frame := newFakeFrame()
	normalColour := colour.Rgba{R: 0.1, G: 0.2, B: 0.3, A: 1}
	hoverColour := colour.Rgba{R: 0.9, G: 0.8, B: 0.1, A: 1}

	label := NewText("Button Label")
	parent := NewDiv().
		TextColour(normalColour).
		Hover(func(r *style.Refinement) {
			r.SetTextColour(hoverColour)
		}).
		Child(label)

	// Step 1: Request layout
	frame.phase = phaseLayoutRequested
	rootID := parent.RequestLayout(frame)
	frame.solve(rootID, 120, 30)
	rootBounds := frame.LayoutBounds(rootID)

	// Step 2: Prepaint
	frame.phase = phasePrepainted
	parent.Prepaint(frame, rootBounds)

	// Test Normal Paint (not hovered)
	frame.phase = phasePainted
	parent.Paint(frame, rootBounds)

	if len(frame.monoSprites) == 0 {
		t.Fatal("expected monochrome sprites emitted for label, got 0")
	}
	for i, sp := range frame.monoSprites {
		if sp.Colour != normalColour {
			t.Errorf("sprite %d: expected normal colour %v, got %v", i, normalColour, sp.Colour)
		}
	}

	// Test Hover Paint (hovered)
	frame.monoSprites = frame.monoSprites[:0]
	if len(frame.hitRegions) == 0 {
		t.Fatal("expected hit region registered on parent")
	}
	parentHitID := frame.hitRegions[0].id
	frame.setHovered(parentHitID, true)

	// Reset phase for second paint test
	parent.phase = phasePrepainted
	label.phase = phasePrepainted
	parent.Paint(frame, rootBounds)

	if len(frame.monoSprites) == 0 {
		t.Fatal("expected monochrome sprites emitted for hovered label, got 0")
	}
	for i, sp := range frame.monoSprites {
		if sp.Colour != hoverColour {
			t.Errorf("sprite %d: expected hover colour %v, got %v", i, hoverColour, sp.Colour)
		}
	}
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
