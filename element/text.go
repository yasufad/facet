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

// Text is an element that renders a run of single-line, left-aligned text.
//
// It measures its content size during layout by shaping via Frame.ShapeLine,
// caches the resulting ShapedLine across solver passes, and emits a
// scene.MonochromeSprite for each glyph during the paint phase.
type Text struct {
	content    string
	refinement style.Refinement

	// Cached layout & shaping state. shapedFor is the exact input shapedLine
	// was shaped from; Paint compares against it because the style Text
	// paints under can differ from the style it shaped under — see Paint.
	shapedLine *text.ShapedLine
	shapedFor  []text.StyleRun
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
	t.shapedLine = nil
	t.shapedFor = nil
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

// TextOverflow sets text overflow truncation behaviour.
func (t *Text) TextOverflow(to style.TextOverflow) *Text {
	t.refinement.SetTextOverflow(to)
	return t
}

// TextAlign sets text alignment.
func (t *Text) TextAlign(a style.TextAlign) *Text {
	t.refinement.SetTextAlign(a)
	return t
}

// LineClamp sets maximum line count for text.
func (t *Text) LineClamp(lines int) *Text {
	t.refinement.SetLineClamp(lines)
	return t
}

// textStyleRuns builds the single-run ShapeLine input for content shaped
// under textStyle. Both RequestLayout and Paint need the exact same
// construction, since Paint compares its result against what RequestLayout
// shaped from to decide whether to reshape.
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

// RequestLayout resolves text styling, registers a measured layout callback
// that shapes the text run through Frame.ShapeLine, and adds the leaf node
// to the layout tree.
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
		// ShapeLine shapes one line with no wrapping, so its output for a
		// given content and style is the same at every available width. The
		// solver calls measure several times per solve with different width
		// constraints; reshaping on each call would repeat the same work for
		// no reason. Content is fixed for this element's lifetime, and this
		// is the first shape of it, so nothing else can have changed since
		// the call above built runs — shapedLine==nil is the whole cache
		// this measure closure needs. Paint has to check harder; see there.
		if t.shapedLine == nil {
			line, err := f.ShapeLine(t.content, runs)
			if err == nil {
				t.shapedLine = &line
				t.shapedFor = runs
			}
		}

		var width geometry.Pixels
		if known.Width.IsSome() {
			width = geometry.Pixels(known.Width.UnwrapOr(0))
		} else if t.shapedLine != nil {
			width = t.shapedLine.Width()
		}

		var height geometry.Pixels
		if known.Height.IsSome() {
			height = geometry.Pixels(known.Height.UnwrapOr(0))
		} else if textStyle.LineHeight > 0 {
			height = textStyle.LineHeight
		} else if t.shapedLine != nil {
			height = t.shapedLine.Height()
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

// Paint draws each glyph in the shaped text run as a scene.MonochromeSprite.
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
	// resolved input differs from what shapedLine was actually shaped from,
	// not only when nothing has been shaped yet.
	runs := textStyleRuns(t.content, textStyle)
	if t.shapedLine == nil || !styleRunsEqual(t.shapedFor, runs) {
		line, err := f.ShapeLine(t.content, runs)
		if err != nil {
			return
		}
		t.shapedLine = &line
		t.shapedFor = runs
	}

	scale := f.ScaleFactor()
	lineOrigin := bounds.Origin

	// TextBackgroundColour paints as a quad behind the glyphs, spanning the
	// element's full box, the same way Div paints its own background.
	if textStyle.BackgroundColour.A > 0 {
		f.InsertQuad(scene.Quad{
			Bounds:     scaleBounds(bounds, scale),
			Background: textStyle.BackgroundColour,
		})
	}

	// A glyph's Position.Y is the baseline's offset from the top of the
	// line's own tight box (ascent plus descent). When the box painted into
	// is taller than that — LineHeight set higher than the font's own
	// metrics, as an editor's 1.5 line height does — CSS's rule is half the
	// extra space above and half below, not all of it below. Skipping this
	// pins every line's glyphs to the top of its box instead of centring
	// them, and the caret has to use the same offset or it stops lining up
	// with the text it sits next to.
	if extra := bounds.Size.Height - t.shapedLine.Height(); extra > 0 {
		lineOrigin.Y += extra / 2
	}

	baselineY := lineOrigin.Y + t.shapedLine.Ascent()
	lineWidth := t.shapedLine.Width()

	// An underline sits within the descent, below the baseline. 0.618 is the
	// golden-ratio placement GPUI's text_system/line.rs uses for the same
	// line, rather than splitting the descent evenly.
	if u := textStyle.Underline; u != nil {
		underlineY := baselineY + t.shapedLine.Descent()*0.618
		f.InsertUnderline(decorationLine(bounds.Origin.X, underlineY, lineWidth, u.Thickness, decorationColour(u.Colour, textStyle.Colour), u.Wavy, scale))
	}

	for _, run := range t.shapedLine.Runs() {
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
	// letters, not one that sits behind them. It runs through the middle of
	// lowercase glyphs at roughly a quarter of the ascent above the
	// baseline, rather than at the baseline itself.
	if s := textStyle.Strikethrough; s != nil {
		strikeY := baselineY - t.shapedLine.Ascent()*0.25
		f.InsertUnderline(decorationLine(bounds.Origin.X, strikeY, lineWidth, s.Thickness, decorationColour(s.Colour, textStyle.Colour), false, scale))
	}
}
