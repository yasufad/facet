package element

import (
	"math"

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

	// Cached layout & shaping state
	shapedLine *text.ShapedLine
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

	runs := []text.StyleRun{
		{
			ByteLen: len(t.content),
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

	measure := func(known layout.Size[layout.OptF32], avail layout.Size[layout.AvailableSpace]) geometry.Size[geometry.Pixels] {
		// ShapeLine shapes one line with no wrapping, so its output for this
		// content and style is the same at every available width. The solver
		// calls measure several times per solve with different constraints;
		// reshaping on each call would repeat the same work for no reason.
		// Content and style are fixed for the lifetime of this element (they
		// can only change before RequestLayout, which runs once), so shaping
		// once and keeping it for every later call is the whole cache.
		if t.shapedLine == nil {
			line, err := f.ShapeLine(t.content, runs)
			if err == nil {
				t.shapedLine = &line
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

	if t.shapedLine == nil {
		runs := []text.StyleRun{
			{
				ByteLen: len(t.content),
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
		line, err := f.ShapeLine(t.content, runs)
		if err != nil {
			return
		}
		t.shapedLine = &line
	}

	scale := f.ScaleFactor()
	lineOrigin := bounds.Origin

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
}
