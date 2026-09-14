package element

import (
	"math"
	"slices"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/text"
)

// noWrapMaxWidth is a width large enough that the line wrapper never breaks a
// line of realistic length. It mirrors text.ShapeLine's noWrapWidth: when the
// available width is unconstrained (MaxContent), the text is laid out as a
// single line per explicit newline rather than word-wrapped.
const noWrapMaxWidth = geometry.Pixels(1 << 23)

// Text is an element that renders text, wrapping it to the available width.
//
// It measures its content size during layout by shaping and wrapping through
// Frame.WrapText, caches the resulting []ShapedLine across solver passes keyed
// on both the style runs and the width it was wrapped at, and emits a
// scene.MonochromeSprite for each glyph during the paint phase. Line breaking
// comes from the text package; element only orchestrates layout and painting
// around the result.
type Text struct {
	content    string
	refinement style.Refinement

	// Cached layout & shaping state. shapedFor is the exact input shapedLines
	// was shaped from and shapedForWidth is the width it was wrapped at. Paint
	// compares against both because the style Text paints under can differ
	// from the style it shaped under (a container's pseudo-state can change
	// font metrics between prepaint and paint), and a re-layout at a
	// different width must re-wrap even when the style runs are identical —
	// comparing runs alone would silently keep the previous frame's line
	// breaks.
	shapedLines     []text.ShapedLine
	shapedFor       []text.StyleRun
	shapedForWidth  geometry.Pixels
	layoutID   layout.NodeID
	bounds     geometry.Bounds[geometry.Pixels]
	phase      drawPhase
}

// Ensure Text implements Element.
var _ Element = (*Text)(nil)

// NewText constructs a new Text element displaying the given string content.
func NewText(content string) *Text {
	return &Text{
		content: content,
	}
}

// Content returns the text string displayed by this element.
func (t *Text) Content() string {
	return t.content
}

// SetContent updates the text content displayed by this element.
func (t *Text) SetContent(content string) *Text {
	t.content = content
	t.shapedLines = nil
	t.shapedFor = nil
	t.shapedForWidth = 0
	return t
}

// Refine applies all explicitly set properties from r onto this element.
func (t *Text) Refine(r style.Refinement) *Text {
	t.refinement.MergeFrom(&r)
	return t
}

// TextColour sets the text foreground colour.
func (t *Text) TextColour(c colour.Rgba) *Text {
	t.refinement.SetTextColour(c)
	return t
}

// TextColourHsla sets the text colour from an Hsla value.
func (t *Text) TextColourHsla(c colour.Hsla) *Text {
	t.refinement.SetTextColourHsla(c)
	return t
}

// FontFamily sets the primary font family name.
func (t *Text) FontFamily(family string) *Text {
	t.refinement.SetFontFamily(family)
	return t
}

// FontFeatures sets OpenType font feature overrides.
func (t *Text) FontFeatures(features []text.FontFeature) *Text {
	t.refinement.SetFontFeatures(features)
	return t
}

// FontFallbacks sets fallback font families.
func (t *Text) FontFallbacks(fallbacks []string) *Text {
	t.refinement.SetFontFallbacks(fallbacks)
	return t
}

// FontSize sets the font size in logical pixels.
func (t *Text) FontSize(size geometry.Pixels) *Text {
	t.refinement.SetFontSize(size)
	return t
}

// LineHeight sets the line height in logical pixels.
func (t *Text) LineHeight(height geometry.Pixels) *Text {
	t.refinement.SetLineHeight(height)
	return t
}

// FontWeight sets font stroke weight.
func (t *Text) FontWeight(weight text.Weight) *Text {
	t.refinement.SetFontWeight(weight)
	return t
}

// FontStyle sets font style (normal or italic).
func (t *Text) FontStyle(s text.Style) *Text {
	t.refinement.SetFontStyle(s)
	return t
}

// TextBackgroundColour sets highlight colour behind text.
func (t *Text) TextBackgroundColour(c colour.Rgba) *Text {
	t.refinement.SetTextBackgroundColour(c)
	return t
}

// TextBackgroundColourHsla sets text highlight colour from an Hsla value.
func (t *Text) TextBackgroundColourHsla(c colour.Hsla) *Text {
	t.refinement.SetTextBackgroundColourHsla(c)
	return t
}

// Underline configures underline styling.
func (t *Text) Underline(u style.UnderlineStyle) *Text {
	t.refinement.SetUnderline(u)
	return t
}

// ClearUnderline removes underline styling.
func (t *Text) ClearUnderline() *Text {
	t.refinement.ClearUnderline()
	return t
}

// Strikethrough configures strikethrough styling.
func (t *Text) Strikethrough(s style.StrikethroughStyle) *Text {
	t.refinement.SetStrikethrough(s)
	return t
}

// ClearStrikethrough removes strikethrough styling.
func (t *Text) ClearStrikethrough() *Text {
	t.refinement.ClearStrikethrough()
	return t
}

// WhiteSpace sets whitespace wrapping behaviour.
func (t *Text) WhiteSpace(w style.WhiteSpace) *Text {
	t.refinement.SetWhiteSpace(w)
	return t
}

// TextOverflow sets text overflow truncation behaviour.
func (t *Text) TextOverflow(to style.TextOverflow) *Text {
	t.refinement.SetTextOverflow(to)
	return t
}

// LineClamp sets maximum line count for text.
func (t *Text) LineClamp(lines int) *Text {
	t.refinement.SetLineClamp(lines)
	return t
}

// textStyleRuns builds the single-run WrapText input for content shaped under
// textStyle. Both RequestLayout and Paint need the exact same construction,
// since Paint compares its result against what RequestLayout shaped from to
// decide whether to reshape.
func textStyleRuns(content string, textStyle style.TextStyle) []text.StyleRun {
	return []text.StyleRun{
		{
			ByteLen: len(content),
			Font: text.FontRequest{
				Family:   textStyle.FontFamily,
				Families: textStyle.FontFallbacks,
				Weight:   textStyle.FontWeight,
				Style:    textStyle.FontStyle,
			},
			Size:      textStyle.FontSize,
			Direction: text.LTR,
			Features:  textStyle.FontFeatures,
		},
	}
}

// styleRunsEqual reports whether a and b would shape identically. StyleRun and
// FontRequest both carry slices, so neither is comparable with ==; this is
// deliberately a manual field comparison rather than reflect.DeepEqual, since
// Paint calls it every frame and AGENTS.md rules reflection out of a per-frame
// path.
func styleRunsEqual(a, b []text.StyleRun) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ByteLen != b[i].ByteLen ||
			a[i].Size != b[i].Size ||
			a[i].Direction != b[i].Direction ||
			a[i].Language != b[i].Language ||
			a[i].Font.Family != b[i].Font.Family ||
			a[i].Font.Weight != b[i].Font.Weight ||
			a[i].Font.Style != b[i].Font.Style ||
			a[i].Font.Stretch != b[i].Font.Stretch ||
			!slices.Equal(a[i].Font.Families, b[i].Font.Families) ||
			!slices.Equal(a[i].Features, b[i].Features) {
			return false
		}
	}
	return true
}

// wrapWidthFor returns the width to wrap text at for a given known-width
// option and whitespace mode: the definite width when one is set and
// positive, or noWrapMaxWidth when the width is unconstrained (MaxContent).
// WhiteSpaceNowrap overrides both — it forces noWrapMaxWidth so the text
// overflows rather than breaking, regardless of the available width. A
// non-positive definite width is degenerate and would force one word per
// line, so it is treated as unconstrained too.
func wrapWidthFor(knownWidth layout.OptF32, whiteSpace style.WhiteSpace) geometry.Pixels {
	if whiteSpace == style.WhiteSpaceNowrap {
		return noWrapMaxWidth
	}
	if knownWidth.IsSome() {
		if w := geometry.Pixels(knownWidth.UnwrapOr(0)); w > 0 {
			return w
		}
	}
	return noWrapMaxWidth
}

// lineBoxHeight returns the height of one line box: the configured line height
// when it is positive, or the shaped line's own ascent plus descent otherwise.
// The line box is what per-line half-leading is computed against.
func lineBoxHeight(line text.ShapedLine, lineHeight geometry.Pixels) geometry.Pixels {
	if lineHeight > 0 {
		return lineHeight
	}
	return line.Height()
}

// ellipsisStr is the ellipsis character appended to truncated text under
// TextOverflowEllipsis and LineClamp. It is U+2026 (HORIZONTAL ELLIPSIS),
// three bytes in UTF-8.
const ellipsisStr = "…"

// truncateLineWithEllipsis shapes content + "…" truncated to fit
// availableWidth, binary-searching the rune count so the result fits, and
// returns the resulting ShapedLine. It is the shared engine behind
// TextOverflowEllipsis (one overflowing line) and LineClamp (the last visible
// line of a clamped paragraph). The ok flag is false when shaping fails or
// even the ellipsis alone does not fit.
func truncateLineWithEllipsis(f Frame, content string, textStyle style.TextStyle, availableWidth geometry.Pixels) (text.ShapedLine, bool) {
	ellipsisRuns := textStyleRuns(ellipsisStr, textStyle)
	ellipsisLines, err := f.WrapText(ellipsisStr, ellipsisRuns, noWrapMaxWidth)
	if err != nil || len(ellipsisLines) == 0 {
		return text.ShapedLine{}, false
	}
	ellipsisWidth := ellipsisLines[0].Width()
	if ellipsisWidth >= availableWidth {
		// Even the ellipsis alone does not fit.
		return text.ShapedLine{}, false
	}

	runes := []rune(content)
	// Binary search the longest rune prefix whose shape plus the ellipsis
	// fits within availableWidth.
	lo, hi := 0, len(runes)
	for lo < hi {
		mid := (lo + hi + 1) / 2
		candidate := string(runes[:mid]) + ellipsisStr
		candRuns := textStyleRuns(candidate, textStyle)
		candLines, err := f.WrapText(candidate, candRuns, noWrapMaxWidth)
		if err != nil || len(candLines) == 0 || candLines[0].Width() > availableWidth {
			hi = mid - 1
		} else {
			lo = mid
		}
	}

	if lo == 0 {
		// Nothing of the content fits; show only the ellipsis.
		return ellipsisLines[0], true
	}

	finalStr := string(runes[:lo]) + ellipsisStr
	finalRuns := textStyleRuns(finalStr, textStyle)
	finalLines, err := f.WrapText(finalStr, finalRuns, noWrapMaxWidth)
	if err != nil || len(finalLines) == 0 {
		return text.ShapedLine{}, false
	}
	return finalLines[0], true
}

// maybeTruncateWithEllipsis implements TextOverflowEllipsis: when the text is
// a single line that overflows availableWidth, it replaces that line with a
// truncated copy plus an ellipsis. This is the CSS single-line text-overflow
// behaviour — it applies to one overflowing line, not to every line of a
// wrapped paragraph (LineClamp handles multi-line). When the text is already
// within availableWidth, or TextOverflow is not Ellipsis, or there is more
// than one line, the input is returned unchanged.
func maybeTruncateWithEllipsis(f Frame, content string, textStyle style.TextStyle, lines []text.ShapedLine, availableWidth geometry.Pixels) []text.ShapedLine {
	if textStyle.TextOverflow != style.TextOverflowEllipsis || len(lines) != 1 || availableWidth <= 0 {
		return lines
	}
	if lines[0].Width() <= availableWidth {
		return lines
	}
	if truncated, ok := truncateLineWithEllipsis(f, content, textStyle, availableWidth); ok {
		return []text.ShapedLine{truncated}
	}
	return lines
}

// lineByteOffsets returns the byte offset of each shaped line's text within
// content. The wrapper breaks at newlines and excludes them from a line's
// text, so the offset of line i is the sum of earlier lines' Len() plus the
// newlines that separated them.
func lineByteOffsets(content string, lines []text.ShapedLine) []int {
	offsets := make([]int, len(lines))
	pos := 0
	for i, l := range lines {
		for pos < len(content) && content[pos] == '\n' {
			pos++
		}
		offsets[i] = pos
		pos += l.Len()
	}
	return offsets
}

// maybeClampLines implements LineClamp: when the wrapped paragraph exceeds
// LineClamp lines, it keeps only the first LineClamp and truncates the last
// visible one with an ellipsis to signal that text was dropped. When
// LineClamp is 0 (no clamping) or the paragraph already fits, the input is
// returned unchanged.
func maybeClampLines(f Frame, content string, textStyle style.TextStyle, lines []text.ShapedLine, availableWidth geometry.Pixels) []text.ShapedLine {
	if textStyle.LineClamp <= 0 || len(lines) <= textStyle.LineClamp {
		return lines
	}
	keep := textStyle.LineClamp
	clamped := make([]text.ShapedLine, keep)
	copy(clamped, lines[:keep])

	// Truncate the last visible line with an ellipsis, using its substring
	// of the content so the binary search truncates the right text.
	offsets := lineByteOffsets(content, lines)
	lastIdx := keep - 1
	lastStart := offsets[lastIdx]
	lastEnd := lastStart + clamped[lastIdx].Len()
	lastContent := content[lastStart:lastEnd]
	if availableWidth > 0 {
		if truncated, ok := truncateLineWithEllipsis(f, lastContent, textStyle, availableWidth); ok {
			clamped[lastIdx] = truncated
		}
	}
	return clamped
}

// RequestLayout resolves text styling, registers a measured layout callback
// that shapes and wraps the text through Frame.WrapText, and adds the leaf
// node to the layout tree.
func (t *Text) RequestLayout(f Frame) NodeID {
	if t.phase != phaseInitial {
		panic("element: RequestLayout called out of order or multiple times")
	}
	t.phase = phaseLayoutRequested

	st := style.Default()
	st.Text = f.TextStyle()
	st.Refine(t.refinement)

	rem := f.RemSize()
	layoutStyle := st.ToLayout(rem)
	textStyle := st.Text
	runs := textStyleRuns(t.content, textStyle)

	measure := func(known layout.Size[layout.OptF32], avail layout.Size[layout.AvailableSpace]) geometry.Size[geometry.Pixels] {
		wrapWidth := wrapWidthFor(known.Width, textStyle.WhiteSpace)

		// Wrap (or re-wrap) when the cache key changes. The width is part of
		// the key, not just the runs: a re-layout at a different width must
		// re-wrap even when the runs are identical, or the previous frame's
		// line breaks are silently retained. Content is fixed for this
		// element's lifetime, and this is the first wrap of it on this frame,
		// so shapedLines==nil covers the cold cache; Paint has to check
		// harder because pseudo-state can change the runs between prepaint
		// and paint.
		if t.shapedLines == nil || !styleRunsEqual(t.shapedFor, runs) || t.shapedForWidth != wrapWidth {
			lines, err := f.WrapText(t.content, runs, wrapWidth)
			if err == nil {
				t.shapedLines = lines
				t.shapedFor = runs
				t.shapedForWidth = wrapWidth
			}
		}

		// TextOverflowEllipsis truncates a single overflowing line to fit the
		// known width, so the measured width is the truncated width rather
		// than the full content width. LineClamp caps the line count and
		// truncates the last visible line with an ellipsis.
		if known.Width.IsSome() {
			availWidth := geometry.Pixels(known.Width.UnwrapOr(0))
			if availWidth > 0 {
				t.shapedLines = maybeTruncateWithEllipsis(f, t.content, textStyle, t.shapedLines, availWidth)
				t.shapedLines = maybeClampLines(f, t.content, textStyle, t.shapedLines, availWidth)
			}
		}

		var width geometry.Pixels
		if known.Width.IsSome() {
			width = geometry.Pixels(known.Width.UnwrapOr(0))
		} else {
			for _, l := range t.shapedLines {
				if l.Width() > width {
					width = l.Width()
				}
			}
		}

		var height geometry.Pixels
		if known.Height.IsSome() {
			height = geometry.Pixels(known.Height.UnwrapOr(0))
		} else if textStyle.LineHeight > 0 {
			height = textStyle.LineHeight * geometry.Pixels(len(t.shapedLines))
		} else {
			for _, l := range t.shapedLines {
				height += l.Height()
			}
		}

		return geometry.NewSize(width, height)
	}

	t.layoutID = f.RequestMeasuredLayout(layoutStyle, measure)
	return t.layoutID
}

// Prepaint commits the solved layout bounds to this element.
func (t *Text) Prepaint(f Frame, bounds geometry.Bounds[geometry.Pixels]) {
	if t.phase != phaseLayoutRequested {
		panic("element: Prepaint called before RequestLayout or out of order")
	}
	t.phase = phasePrepainted
	t.bounds = bounds
}

// Paint draws each glyph in every wrapped line as a scene.MonochromeSprite,
// applying per-line half-leading so a line box taller than the font's own
// metrics centres the glyphs vertically within it.
func (t *Text) Paint(f Frame, bounds geometry.Bounds[geometry.Pixels]) {
	if t.phase != phasePrepainted {
		panic("element: Paint called before Prepaint or out of order")
	}
	t.phase = phasePainted
	t.bounds = bounds

	if len(t.content) == 0 {
		return
	}

	st := style.Default()
	st.Text = f.TextStyle()
	st.Refine(t.refinement)
	textStyle := st.Text

	// f.TextStyle() carries whatever pseudo-state refinements the container
	// merged in between prepaint and paint (docs/packages.md), so the style
	// painted under is not always the style RequestLayout's measure shaped
	// under — a container's Hover changing font weight, family or size is a
	// real, supported case, not a hypothetical one. Reshape whenever the
	// resolved input differs from what shapedLines was actually shaped from,
	// not only when nothing has been shaped yet. The width is the same one
	// measure settled on: the element's bounds are the solver's output for
	// the constraint measure shaped under.
	runs := textStyleRuns(t.content, textStyle)
	wrapWidth := bounds.Size.Width
	if textStyle.WhiteSpace == style.WhiteSpaceNowrap || wrapWidth <= 0 {
		wrapWidth = noWrapMaxWidth
	}
	if t.shapedLines == nil || !styleRunsEqual(t.shapedFor, runs) || t.shapedForWidth != wrapWidth {
		lines, err := f.WrapText(t.content, runs, wrapWidth)
		if err != nil {
			return
		}
		t.shapedLines = lines
		t.shapedFor = runs
		t.shapedForWidth = wrapWidth
	}

	// TextOverflowEllipsis truncates a single overflowing line to fit the
	// bounds width at paint time, mirroring the truncation measure applied.
	// LineClamp caps the line count and truncates the last visible line.
	if bounds.Size.Width > 0 {
		t.shapedLines = maybeTruncateWithEllipsis(f, t.content, textStyle, t.shapedLines, bounds.Size.Width)
		t.shapedLines = maybeClampLines(f, t.content, textStyle, t.shapedLines, bounds.Size.Width)
	}

	scale := f.ScaleFactor()

	// TextBackgroundColour paints as a quad behind the glyphs, spanning the
	// element's full box, the same way Div paints its own background.
	if textStyle.BackgroundColour.A > 0 {
		f.InsertQuad(scene.Quad{
			Bounds:     scaleBounds(bounds, scale),
			Background: textStyle.BackgroundColour,
		})
	}

	// Each line occupies a line box of height lineBoxHeight stacked from the
	// element's top. The half-leading rule — (boxHeight - line.Height()) / 2
	// when the box is taller than the font's metrics — runs per line, not
	// once against the whole element: with several lines, running it once
	// would pin every line to the top of the first box. The caret has to use
	// the same per-line offset or it stops sitting next to the text.
	lineY := bounds.Origin.Y
	for _, line := range t.shapedLines {
		boxH := lineBoxHeight(line, textStyle.LineHeight)
		lineOrigin := geometry.NewPoint(bounds.Origin.X, lineY)
		if extra := boxH - line.Height(); extra > 0 {
			lineOrigin.Y += extra / 2
		}

		baselineY := lineOrigin.Y + line.Ascent()
		lineWidth := line.Width()

		// An underline sits within the descent, below the baseline. 0.618 is
		// the golden-ratio placement GPUI's text_system/line.rs uses for the
		// same line, rather than splitting the descent evenly.
		if u := textStyle.Underline; u != nil {
			underlineY := baselineY + line.Descent()*0.618
			f.InsertUnderline(decorationLine(lineOrigin.X, underlineY, lineWidth, u.Thickness, decorationColour(u.Colour, textStyle.Colour), u.Wavy, scale))
		}

		for _, run := range line.Runs() {
			for _, g := range run.Glyphs {
				penPos := lineOrigin.Add(g.Position)

				// Subpixel bucket calculation from fractional device-pixel pen X
				penXDevice := float32(penPos.X) * scale
				frac := penXDevice - float32(math.Floor(float64(penXDevice)))
				subpixel := text.SubpixelFor(frac)

				tile, glyphBounds, ok := f.RasteriseGlyph(g.Face, g.ID, textStyle.FontSize, subpixel)
				if !ok || glyphBounds.Size.IsZero() {
					continue
				}

				penScaled := geometry.ScalePoint(penPos, scale)
				spOrigin := geometry.Point[geometry.ScaledPixels]{
					X: penScaled.X + glyphBounds.Origin.X.ToScaledPixels(),
					Y: penScaled.Y - glyphBounds.Origin.Y.ToScaledPixels(),
				}
				spSize := geometry.Size[geometry.ScaledPixels]{
					Width:  glyphBounds.Size.Width.ToScaledPixels(),
					Height: glyphBounds.Size.Height.ToScaledPixels(),
				}

				f.InsertMonochromeSprite(scene.MonochromeSprite{
					Bounds:         geometry.NewBounds(spOrigin, spSize),
					Colour:         textStyle.Colour,
					Tile:           tile,
					Transformation: scene.IdentityMatrix,
				})
			}
		}

		// Strikethrough draws last, over the glyphs: it is a line through the
		// letters, not one that sits behind them. It runs through the middle
		// of lowercase glyphs at roughly a quarter of the ascent above the
		// baseline, rather than at the baseline itself.
		if s := textStyle.Strikethrough; s != nil {
			strikeY := baselineY - line.Ascent()*0.25
			f.InsertUnderline(decorationLine(lineOrigin.X, strikeY, lineWidth, s.Thickness, decorationColour(s.Colour, textStyle.Colour), false, scale))
		}

		lineY += boxH
	}
}
