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

// TestTextBackgroundColourPaintsQuadBehindGlyphs gives TextBackgroundColour
// its first producer: a quad spanning the element's full box, painted before
// the glyph sprites so it sits behind them.
func TestTextBackgroundColourPaintsQuadBehindGlyphs(t *testing.T) {
	frame := newFakeFrame()
	yellow := colour.Rgba{R: 1, G: 1, B: 0, A: 1}

	txt := NewText("A").FontSize(16).TextBackgroundColour(yellow)

	frame.phase = phaseLayoutRequested
	nodeID := txt.RequestLayout(frame)
	measured := frame.measureCallbacks[nodeID](layout.Size[layout.OptF32]{}, layout.Size[layout.AvailableSpace]{
		Width:  layout.MaxContent(),
		Height: layout.MaxContent(),
	})
	bounds := geometry.NewBounds(geometry.NewPoint[geometry.Pixels](5, 10), measured)

	frame.phase = phasePrepainted
	txt.Prepaint(frame, bounds)
	frame.phase = phasePainted
	txt.Paint(frame, bounds)

	if len(frame.quads) != 1 {
		t.Fatalf("expected 1 background quad, got %d", len(frame.quads))
	}
	if frame.quads[0].Background != yellow {
		t.Fatalf("quad background = %v, want %v", frame.quads[0].Background, yellow)
	}
	if len(frame.monoSprites) == 0 {
		t.Fatal("expected glyph sprites in addition to the background quad")
	}

	scale := frame.scaleFactor
	wantBounds := geometry.NewBounds(
		geometry.NewPoint[geometry.ScaledPixels](bounds.Origin.X.Scale(scale), bounds.Origin.Y.Scale(scale)),
		geometry.NewSize[geometry.ScaledPixels](bounds.Size.Width.Scale(scale), bounds.Size.Height.Scale(scale)),
	)
	if frame.quads[0].Bounds != wantBounds {
		t.Fatalf("quad bounds = %v, want %v", frame.quads[0].Bounds, wantBounds)
	}
}

// TestTextUnderlinePaintsPrimitive gives scene.Underline one of its first two
// producers in the project's history (Strikethrough is the other): a line
// within the descent, below the baseline, spanning the shaped line's width.
func TestTextUnderlinePaintsPrimitive(t *testing.T) {
	frame := newFakeFrame()
	blue := colour.Rgba{B: 1, A: 1}

	txt := NewText("A").
		FontSize(16).
		Underline(style.UnderlineStyle{Thickness: 2, Colour: blue, Wavy: true})

	frame.phase = phaseLayoutRequested
	nodeID := txt.RequestLayout(frame)
	measured := frame.measureCallbacks[nodeID](layout.Size[layout.OptF32]{}, layout.Size[layout.AvailableSpace]{
		Width:  layout.MaxContent(),
		Height: layout.MaxContent(),
	})
	bounds := geometry.NewBounds(geometry.NewPoint[geometry.Pixels](0, 0), measured)

	frame.phase = phasePrepainted
	txt.Prepaint(frame, bounds)
	frame.phase = phasePainted
	txt.Paint(frame, bounds)

	if len(frame.underlines) != 1 {
		t.Fatalf("expected 1 underline, got %d", len(frame.underlines))
	}
	u := frame.underlines[0]
	if u.Colour != blue {
		t.Fatalf("underline colour = %v, want %v", u.Colour, blue)
	}
	if !u.Wavy {
		t.Fatal("expected wavy underline")
	}

	scale := frame.scaleFactor
	if wantThickness := geometry.Pixels(2).Scale(scale); u.Thickness != wantThickness {
		t.Fatalf("underline thickness = %v, want %v", u.Thickness, wantThickness)
	}

	// Reproduce the half-leading shift TestTextPaintCentresGlyphsOnTallerLineHeight
	// pins: LineHeight defaults to 20 (DefaultTextStyle), which is usually
	// taller than the shaped line's own ascent+descent, so the baseline is
	// not simply at the box's top edge plus ascent.
	firstLine := txt.shapedLines[0]
	lineOriginY := bounds.Origin.Y
	if extra := bounds.Size.Height - firstLine.Height(); extra > 0 {
		lineOriginY += extra / 2
	}
	baselineY := lineOriginY + firstLine.Ascent()
	wantY := (baselineY + firstLine.Descent()*0.618).Scale(scale)
	if u.Bounds.Origin.Y != wantY {
		t.Fatalf("underline Y = %v, want %v", u.Bounds.Origin.Y, wantY)
	}
	if u.Bounds.Origin.Y <= baselineY.Scale(scale) {
		t.Fatalf("underline Y = %v should be below the baseline %v", u.Bounds.Origin.Y, baselineY.Scale(scale))
	}
}

// TestTextUnderlineColourFallsBackToTextColour confirms an unset underline
// colour defaults to the text colour, per style.UnderlineStyle's doc comment.
func TestTextUnderlineColourFallsBackToTextColour(t *testing.T) {
	frame := newFakeFrame()
	green := colour.Rgba{G: 1, A: 1}

	txt := NewText("A").
		FontSize(16).
		TextColour(green).
		Underline(style.UnderlineStyle{Thickness: 1})

	frame.phase = phaseLayoutRequested
	nodeID := txt.RequestLayout(frame)
	measured := frame.measureCallbacks[nodeID](layout.Size[layout.OptF32]{}, layout.Size[layout.AvailableSpace]{
		Width:  layout.MaxContent(),
		Height: layout.MaxContent(),
	})
	bounds := geometry.NewBounds(geometry.NewPoint[geometry.Pixels](0, 0), measured)

	frame.phase = phasePrepainted
	txt.Prepaint(frame, bounds)
	frame.phase = phasePainted
	txt.Paint(frame, bounds)

	if len(frame.underlines) != 1 {
		t.Fatalf("expected 1 underline, got %d", len(frame.underlines))
	}
	if frame.underlines[0].Colour != green {
		t.Fatalf("underline colour = %v, want text colour %v", frame.underlines[0].Colour, green)
	}
}

// TestTextStrikethroughPaintsPrimitiveAboveBaseline gives scene.Underline its
// second producer: a line through the letters, above the baseline, unlike
// an underline which sits below it.
func TestTextStrikethroughPaintsPrimitiveAboveBaseline(t *testing.T) {
	frame := newFakeFrame()
	orange := colour.Rgba{R: 1, G: 0.5, A: 1}

	txt := NewText("A").
		FontSize(16).
		Strikethrough(style.StrikethroughStyle{Thickness: 1.5, Colour: orange})

	frame.phase = phaseLayoutRequested
	nodeID := txt.RequestLayout(frame)
	measured := frame.measureCallbacks[nodeID](layout.Size[layout.OptF32]{}, layout.Size[layout.AvailableSpace]{
		Width:  layout.MaxContent(),
		Height: layout.MaxContent(),
	})
	bounds := geometry.NewBounds(geometry.NewPoint[geometry.Pixels](0, 0), measured)

	frame.phase = phasePrepainted
	txt.Prepaint(frame, bounds)
	frame.phase = phasePainted
	txt.Paint(frame, bounds)

	if len(frame.underlines) != 1 {
		t.Fatalf("expected 1 strikethrough line, got %d", len(frame.underlines))
	}
	line := frame.underlines[0]
	if line.Colour != orange {
		t.Fatalf("strikethrough colour = %v, want %v", line.Colour, orange)
	}
	if line.Wavy {
		t.Fatal("strikethrough must never be wavy")
	}

	scale := frame.scaleFactor
	if wantThickness := geometry.Pixels(1.5).Scale(scale); line.Thickness != wantThickness {
		t.Fatalf("strikethrough thickness = %v, want %v", line.Thickness, wantThickness)
	}

	firstLine := txt.shapedLines[0]
	lineOriginY := bounds.Origin.Y
	if extra := bounds.Size.Height - firstLine.Height(); extra > 0 {
		lineOriginY += extra / 2
	}
	baselineY := lineOriginY + firstLine.Ascent()
	wantY := (baselineY - firstLine.Ascent()*0.25).Scale(scale)
	if line.Bounds.Origin.Y != wantY {
		t.Fatalf("strikethrough Y = %v, want %v", line.Bounds.Origin.Y, wantY)
	}
	if line.Bounds.Origin.Y >= baselineY.Scale(scale) {
		t.Fatalf("strikethrough Y = %v should be above the baseline %v", line.Bounds.Origin.Y, baselineY.Scale(scale))
	}
}

// TestTextUnderlineAndStrikethroughBothPaint confirms the two decorations
// coexist as two distinct primitives rather than one overwriting the other,
// since they share the same construction helper.
func TestTextUnderlineAndStrikethroughBothPaint(t *testing.T) {
	frame := newFakeFrame()

	txt := NewText("A").
		FontSize(16).
		Underline(style.UnderlineStyle{Thickness: 1}).
		Strikethrough(style.StrikethroughStyle{Thickness: 1})

	frame.phase = phaseLayoutRequested
	nodeID := txt.RequestLayout(frame)
	measured := frame.measureCallbacks[nodeID](layout.Size[layout.OptF32]{}, layout.Size[layout.AvailableSpace]{
		Width:  layout.MaxContent(),
		Height: layout.MaxContent(),
	})
	bounds := geometry.NewBounds(geometry.NewPoint[geometry.Pixels](0, 0), measured)

	frame.phase = phasePrepainted
	txt.Prepaint(frame, bounds)
	frame.phase = phasePainted
	txt.Paint(frame, bounds)

	if len(frame.underlines) != 2 {
		t.Fatalf("expected 2 decoration lines (underline + strikethrough), got %d", len(frame.underlines))
	}
	if frame.underlines[0].Bounds.Origin.Y == frame.underlines[1].Bounds.Origin.Y {
		t.Fatalf("expected the underline and strikethrough at different heights, both at %v", frame.underlines[0].Bounds.Origin.Y)
	}
}

// TestTextDoesNotRewrapOnAvailableWidthChange pins the cache key: the wrap
// width comes from the known (definite) width, not from the available space,
// so the solver calling measure several times with different available widths
// but the same known width reuses one wrap. This is the counterpart to
// TestTextRewrapsOnKnownWidthChange, which asserts the opposite — a different
// known width must re-wrap even with identical style runs.
func TestTextDoesNotRewrapOnAvailableWidthChange(t *testing.T) {
	frame := newFakeFrame()
	frame.phase = phaseLayoutRequested

	txt := NewText("The quick brown fox jumps over the lazy dog").
		FontSize(16).
		LineHeight(20)

	nodeID := txt.RequestLayout(frame)
	measureCb := frame.measureCallbacks[nodeID]
	known := layout.Size[layout.OptF32]{}

	for _, w := range []float32{100, 200, 300, 150, 400} {
		avail := layout.Size[layout.AvailableSpace]{
			Width:  layout.Definite(w),
			Height: layout.MaxContent(),
		}
		if sz := measureCb(known, avail); sz.Width <= 0 {
			t.Fatalf("invalid width at avail=%v", w)
		}
	}

	if frame.wrapTextCalls != 1 {
		t.Fatalf("WrapText called %d times across 5 different available widths, want 1: re-wrapping on available-width change is back", frame.wrapTextCalls)
	}
}

// TestTextRewrapsOnKnownWidthChange pins the prompt's explicit requirement: a
// re-layout at a different width must re-wrap even when the style runs are
// identical. Comparing runs alone would silently keep the previous frame's
// line breaks, so the width is part of the cache key.
func TestTextRewrapsOnKnownWidthChange(t *testing.T) {
	frame := newFakeFrame()
	frame.phase = phaseLayoutRequested

	txt := NewText("The quick brown fox jumps over the lazy dog").
		FontSize(16).
		LineHeight(20)

	nodeID := txt.RequestLayout(frame)
	measureCb := frame.measureCallbacks[nodeID]
	avail := layout.Size[layout.AvailableSpace]{
		Width:  layout.MaxContent(),
		Height: layout.MaxContent(),
	}

	for _, w := range []float32{100, 200, 300} {
		known := layout.Size[layout.OptF32]{Width: layout.SomeOptF32(w)}
		measureCb(known, avail)
	}

	if frame.wrapTextCalls != 3 {
		t.Fatalf("WrapText called %d times across 3 different known widths, want 3: width is not part of the wrap cache key", frame.wrapTextCalls)
	}
}

// TestTextWrapsToMultipleLines pins that Text wraps through Frame.WrapText:
// long content at a narrow known width produces more than one shaped line,
// and the measured height grows to cover every line box.
func TestTextWrapsToMultipleLines(t *testing.T) {
	frame := newFakeFrame()
	frame.phase = phaseLayoutRequested

	txt := NewText("The quick brown fox jumps over the lazy dog").
		FontSize(16).
		LineHeight(20)

	nodeID := txt.RequestLayout(frame)
	measureCb := frame.measureCallbacks[nodeID]

	// A narrow known width forces wrapping; the natural (MaxContent) width
	// keeps everything on one line.
	narrow := measureCb(
		layout.Size[layout.OptF32]{Width: layout.SomeOptF32(40)},
		layout.Size[layout.AvailableSpace]{Width: layout.MaxContent(), Height: layout.MaxContent()},
	)
	narrowLineCount := len(txt.shapedLines)

	natural := measureCb(
		layout.Size[layout.OptF32]{},
		layout.Size[layout.AvailableSpace]{Width: layout.MaxContent(), Height: layout.MaxContent()},
	)

	if narrowLineCount <= 1 {
		t.Fatalf("expected wrapped text to produce >1 line at width 40, got %d", narrowLineCount)
	}
	if narrow.Height <= natural.Height {
		t.Fatalf("narrow height %v should exceed natural height %v for wrapped multi-line text", narrow.Height, natural.Height)
	}
	// LineHeight applies per line: height = LineHeight * lineCount.
	wantHeight := geometry.Pixels(20) * geometry.Pixels(narrowLineCount)
	if narrow.Height != wantHeight {
		t.Fatalf("narrow height = %v, want LineHeight*lines = %v", narrow.Height, wantHeight)
	}
}

// TestTextNewlinesProduceMultipleLines pins that explicit newlines flow through
// WrapText: each newline forces a line break even at the no-wrap width used
// for MaxContent, so the element renders one line per source line.
func TestTextNewlinesProduceMultipleLines(t *testing.T) {
	frame := newFakeFrame()
	frame.phase = phaseLayoutRequested

	txt := NewText("line one\nline two\nline three").FontSize(16).LineHeight(20)

	nodeID := txt.RequestLayout(frame)
	measureCb := frame.measureCallbacks[nodeID]
	measured := measureCb(
		layout.Size[layout.OptF32]{},
		layout.Size[layout.AvailableSpace]{Width: layout.MaxContent(), Height: layout.MaxContent()},
	)

	if len(txt.shapedLines) != 3 {
		t.Fatalf("expected 3 lines from 2 newlines, got %d", len(txt.shapedLines))
	}
	wantHeight := geometry.Pixels(20) * 3
	if measured.Height != wantHeight {
		t.Fatalf("height = %v, want 3*LineHeight = %v", measured.Height, wantHeight)
	}
}

// TestTextWhiteSpaceNowrapSuppressesWrapping pins WhiteSpace's real consumer:
// WhiteSpaceNowrap forces the text to a single line even at a narrow known
// width that would otherwise wrap, where WhiteSpaceNormal (the default) wraps.
// Removing the WhiteSpaceNowrap branch from wrapWidthFor collapses this test
// to the wrapped case and the line count assertion fails.
func TestTextWhiteSpaceNowrapSuppressesWrapping(t *testing.T) {
	content := "The quick brown fox jumps over the lazy dog"

	// Normal: wraps to multiple lines at a narrow known width.
	frameNormal := newFakeFrame()
	frameNormal.phase = phaseLayoutRequested
	normal := NewText(content).FontSize(16).LineHeight(20)
	normalID := normal.RequestLayout(frameNormal)
	frameNormal.measureCallbacks[normalID](
		layout.Size[layout.OptF32]{Width: layout.SomeOptF32(40)},
		layout.Size[layout.AvailableSpace]{Width: layout.MaxContent(), Height: layout.MaxContent()},
	)
	if len(normal.shapedLines) <= 1 {
		t.Fatalf("WhiteSpaceNormal: expected >1 line at width 40, got %d", len(normal.shapedLines))
	}

	// Nowrap: stays on one line at the same narrow known width.
	frameNowrap := newFakeFrame()
	frameNowrap.phase = phaseLayoutRequested
	nowrap := NewText(content).FontSize(16).LineHeight(20).WhiteSpace(style.WhiteSpaceNowrap)
	nowrapID := nowrap.RequestLayout(frameNowrap)
	frameNowrap.measureCallbacks[nowrapID](
		layout.Size[layout.OptF32]{Width: layout.SomeOptF32(40)},
		layout.Size[layout.AvailableSpace]{Width: layout.MaxContent(), Height: layout.MaxContent()},
	)
	if len(nowrap.shapedLines) != 1 {
		t.Fatalf("WhiteSpaceNowrap: expected 1 line at width 40, got %d", len(nowrap.shapedLines))
	}
}

// TestTextOverflowEllipsisTruncatesOverflow pins TextOverflow's real consumer:
// under TextOverflowEllipsis, a single line that overflows the known width is
// truncated to fit with an ellipsis appended, where TextOverflowClip (the
// default) leaves the line overflowing. Removing the Ellipsis branch from
// maybeTruncateWithEllipsis collapses this test to the clip case and the
// width assertion fails.
func TestTextOverflowEllipsisTruncatesOverflow(t *testing.T) {
	content := "The quick brown fox jumps over the lazy dog"
	const knownWidth = geometry.Pixels(100)

	// Clip: the line overflows the known width.
	frameClip := newFakeFrame()
	frameClip.phase = phaseLayoutRequested
	clip := NewText(content).FontSize(16).LineHeight(20).
		WhiteSpace(style.WhiteSpaceNowrap).
		TextOverflow(style.TextOverflowClip)
	clipID := clip.RequestLayout(frameClip)
	frameClip.measureCallbacks[clipID](
		layout.Size[layout.OptF32]{Width: layout.SomeOptF32(float32(knownWidth))},
		layout.Size[layout.AvailableSpace]{Width: layout.MaxContent(), Height: layout.MaxContent()},
	)
	if len(clip.shapedLines) != 1 {
		t.Fatalf("Clip: expected 1 line, got %d", len(clip.shapedLines))
	}
	if clip.shapedLines[0].Width() <= knownWidth {
		t.Fatalf("Clip: line width %v should overflow known width %v", clip.shapedLines[0].Width(), knownWidth)
	}

	// Ellipsis: the line is truncated to fit the known width.
	frameEllipsis := newFakeFrame()
	frameEllipsis.phase = phaseLayoutRequested
	ellipsis := NewText(content).FontSize(16).LineHeight(20).
		WhiteSpace(style.WhiteSpaceNowrap).
		TextOverflow(style.TextOverflowEllipsis)
	ellipsisID := ellipsis.RequestLayout(frameEllipsis)
	frameEllipsis.measureCallbacks[ellipsisID](
		layout.Size[layout.OptF32]{Width: layout.SomeOptF32(float32(knownWidth))},
		layout.Size[layout.AvailableSpace]{Width: layout.MaxContent(), Height: layout.MaxContent()},
	)
	if len(ellipsis.shapedLines) != 1 {
		t.Fatalf("Ellipsis: expected 1 line, got %d", len(ellipsis.shapedLines))
	}
	if ellipsis.shapedLines[0].Width() > knownWidth {
		t.Fatalf("Ellipsis: line width %v should fit known width %v (truncation did not happen)", ellipsis.shapedLines[0].Width(), knownWidth)
	}
	// The truncated line must be shorter than the full clip line: the
	// ellipsis replaced the tail of the content.
	if ellipsis.shapedLines[0].Width() >= clip.shapedLines[0].Width() {
		t.Fatalf("Ellipsis line width %v should be less than clip line width %v", ellipsis.shapedLines[0].Width(), clip.shapedLines[0].Width())
	}
	// The truncated line carries fewer bytes than the full content: the
	// ellipsis replaced the tail.
	if ellipsis.shapedLines[0].Len() >= len(content) {
		t.Fatalf("Ellipsis line Len %d should be less than content len %d", ellipsis.shapedLines[0].Len(), len(content))
	}
}

// TestTextLineClampCapsLineCount pins LineClamp's real consumer: a paragraph
// that wraps to more lines than the clamp is reduced to exactly LineClamp
// lines, and the last visible line is truncated with an ellipsis. Removing
// the clamp branch from maybeClampLines collapses this test to the full
// paragraph and the line-count assertion fails.
func TestTextLineClampCapsLineCount(t *testing.T) {
	content := "The quick brown fox jumps over the lazy dog and then keeps running"
	const knownWidth = geometry.Pixels(80)
	const clamp = 2

	// No clamp: the paragraph wraps to many lines.
	frameFull := newFakeFrame()
	frameFull.phase = phaseLayoutRequested
	full := NewText(content).FontSize(16).LineHeight(20)
	fullID := full.RequestLayout(frameFull)
	frameFull.measureCallbacks[fullID](
		layout.Size[layout.OptF32]{Width: layout.SomeOptF32(float32(knownWidth))},
		layout.Size[layout.AvailableSpace]{Width: layout.MaxContent(), Height: layout.MaxContent()},
	)
	if len(full.shapedLines) <= clamp {
		t.Fatalf("unclamped paragraph: expected >%d lines at width %v, got %d", clamp, knownWidth, len(full.shapedLines))
	}

	// Clamped: exactly LineClamp lines, the last truncated with an ellipsis.
	frameClamp := newFakeFrame()
	frameClamp.phase = phaseLayoutRequested
	clamped := NewText(content).FontSize(16).LineHeight(20).LineClamp(clamp)
	clampedID := clamped.RequestLayout(frameClamp)
	frameClamp.measureCallbacks[clampedID](
		layout.Size[layout.OptF32]{Width: layout.SomeOptF32(float32(knownWidth))},
		layout.Size[layout.AvailableSpace]{Width: layout.MaxContent(), Height: layout.MaxContent()},
	)
	if len(clamped.shapedLines) != clamp {
		t.Fatalf("clamped paragraph: expected %d lines, got %d", clamp, len(clamped.shapedLines))
	}
	// The last clamped line carries the ellipsis, so it differs from the
	// corresponding unclamped line: either longer (ellipsis appended to a
	// line that already fit) or shorter (line truncated to make room for the
	// ellipsis). Either way the byte length changes.
	lastClamped := clamped.shapedLines[clamp-1]
	lastFull := full.shapedLines[clamp-1]
	if lastClamped.Len() == lastFull.Len() {
		t.Fatalf("clamped last line Len %d equals unclamped %d (ellipsis not applied)", lastClamped.Len(), lastFull.Len())
	}
}

// TestTextReshapesWhenPaintTimeStyleChangesFontMetrics pins the paint-time
// gap docs/audit.md names: f.TextStyle() carries whatever pseudo-state
// refinements a container merges in between prepaint and paint, so the style
// Paint shapes under is not always the style RequestLayout's measure shaped
// under. A container's Hover changing the child's font weight is a real,
// supported case — Div.Hover accepts any Refinement mutation — and must
// reshape, not silently keep the layout-time glyphs.
func TestTextReshapesWhenPaintTimeStyleChangesFontMetrics(t *testing.T) {
	frame := newFakeFrame()

	label := NewText("A")
	parent := NewDiv().
		FontWeight(text.WeightNormal).
		Hover(func(r *style.Refinement) {
			r.SetFontWeight(text.WeightBold)
		}).
		Child(label)

	frame.phase = phaseLayoutRequested
	rootID := parent.RequestLayout(frame)
	frame.solve(rootID, 100, 30)
	rootBounds := frame.LayoutBounds(rootID)

	frame.phase = phasePrepainted
	parent.Prepaint(frame, rootBounds)

	frame.phase = phasePainted
	parent.Paint(frame, rootBounds)

	if frame.wrapTextCalls != 1 {
		t.Fatalf("wrapTextCalls after first paint = %d, want 1", frame.wrapTextCalls)
	}

	if len(frame.hitRegions) == 0 {
		t.Fatal("expected hit region registered on parent")
	}
	frame.setHovered(frame.hitRegions[0].id, true)

	parent.phase = phasePrepainted
	label.phase = phasePrepainted
	parent.Paint(frame, rootBounds)

	if frame.wrapTextCalls != 2 {
		t.Fatalf("wrapTextCalls after hover paint = %d, want 2: paint-time font weight change did not reshape", frame.wrapTextCalls)
	}
}

// TestTextDoesNotReshapeWhenOnlyColourChanges is the negative control for
// TestTextReshapesWhenPaintTimeStyleChangesFontMetrics: pseudo-state that
// changes only colour, which is applied per-sprite and never reaches
// ShapeLine's input, must not trigger a reshape.
func TestTextDoesNotReshapeWhenOnlyColourChanges(t *testing.T) {
	frame := newFakeFrame()
	red := colour.Rgba{R: 1, G: 0, B: 0, A: 1}
	blue := colour.Rgba{R: 0, G: 0, B: 1, A: 1}

	label := NewText("A")
	parent := NewDiv().
		TextColour(red).
		Hover(func(r *style.Refinement) {
			r.SetTextColour(blue)
		}).
		Child(label)

	frame.phase = phaseLayoutRequested
	rootID := parent.RequestLayout(frame)
	frame.solve(rootID, 100, 30)
	rootBounds := frame.LayoutBounds(rootID)

	frame.phase = phasePrepainted
	parent.Prepaint(frame, rootBounds)

	frame.phase = phasePainted
	parent.Paint(frame, rootBounds)

	if frame.wrapTextCalls != 1 {
		t.Fatalf("wrapTextCalls after first paint = %d, want 1", frame.wrapTextCalls)
	}

	if len(frame.hitRegions) == 0 {
		t.Fatal("expected hit region registered on parent")
	}
	frame.setHovered(frame.hitRegions[0].id, true)

	parent.phase = phasePrepainted
	label.phase = phasePrepainted
	parent.Paint(frame, rootBounds)

	if frame.wrapTextCalls != 1 {
		t.Fatalf("wrapTextCalls after colour-only hover paint = %d, want 1: colour-only pseudo-state should not reshape", frame.wrapTextCalls)
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
		if txt.shapedLines == nil || len(txt.shapedLines) == 0 {
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
		return frame.monoSprites[0].Bounds.Origin.Y, txt.shapedLines[0].Height()
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

// TestTextAlignShiftsLineOrigin pins TextAlign's real consumer: a short line
// in a wide box shifts its x origin with alignment — Left pins to the left
// edge, Centre shifts right by half the slack, Right by all of it. Removing
// the alignment branch from alignedLineX collapses every alignment to Left
// and the Centre/Right assertions fail.
func TestTextAlignShiftsLineOrigin(t *testing.T) {
	spriteX := func(align style.TextAlign) geometry.ScaledPixels {
		frame := newFakeFrame()
		txt := NewText("Hi").FontSize(16).LineHeight(20).TextAlign(align)

		frame.phase = phaseLayoutRequested
		nodeID := txt.RequestLayout(frame)
		measured := frame.measureCallbacks[nodeID](layout.Size[layout.OptF32]{}, layout.Size[layout.AvailableSpace]{
			Width:  layout.MaxContent(),
			Height: layout.MaxContent(),
		})

		// Paint into a box wider than the line so alignment has slack to use.
		bounds := geometry.NewBounds(geometry.NewPoint[geometry.Pixels](0, 0), geometry.NewSize[geometry.Pixels](200, measured.Height))
		frame.phase = phasePrepainted
		txt.Prepaint(frame, bounds)
		frame.phase = phasePainted
		txt.Paint(frame, bounds)

		if len(frame.monoSprites) == 0 {
			t.Fatal("expected at least one glyph sprite")
		}
		return frame.monoSprites[0].Bounds.Origin.X
	}

	leftX := spriteX(style.TextAlignLeft)
	centreX := spriteX(style.TextAlignCentre)
	rightX := spriteX(style.TextAlignRight)

	if centreX <= leftX {
		t.Fatalf("Centre sprite X %v should be right of Left %v", centreX, leftX)
	}
	if rightX <= centreX {
		t.Fatalf("Right sprite X %v should be right of Centre %v", rightX, centreX)
	}
	// Centre moves by half the slack, Right by all of it, so Right - Left is
	// twice Centre - Left (modulo the per-glyph offset being constant).
	if got := (rightX - leftX) - 2*(centreX-leftX); got > geometry.ScaledPixels(0.5) || got < geometry.ScaledPixels(-0.5) {
		t.Fatalf("Right-Left %v should be 2*(Centre-Left %v), diff %v", rightX-leftX, centreX-leftX, got)
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
		Strikethrough(style.StrikethroughStyle{Thickness: 1})

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
