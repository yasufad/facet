package text

import (
	"github.com/go-text/typesetting/font"
	"github.com/yasufad/facet/geometry"
)

// Metrics describes a face's vertical measurements at a given size. All fields
// are in geometry.Pixels: ascent and cap height are positive (above the
// baseline), descent is negative (below it), line gap is positive.
type Metrics struct {
	// Ascent is the distance from the baseline to the top of the font's
	// typographic extent.
	Ascent geometry.Pixels
	// Descent is the distance from the baseline to the bottom of the font's
	// typographic extent. It is negative.
	Descent geometry.Pixels
	// LineGap is the font's suggested gap between lines.
	LineGap geometry.Pixels
	// CapHeight is the height of a capital letter above the baseline.
	CapHeight geometry.Pixels
	// XHeight is the height of a lowercase x above the baseline.
	XHeight geometry.Pixels
	// UnitsPerEm is the font's units-per-em, the scale that converts font
	// units to pixels at the requested size.
	UnitsPerEm uint16
}

// Metrics returns the face's metrics at the given size. Font units are scaled
// to pixels by size/UnitsPerEm, the standard OpenType conversion.
func (f Face) Metrics(size geometry.Pixels) Metrics {
	if !f.valid() {
		return Metrics{}
	}
	upem := f.face.Upem()
	scale := float32(size) / float32(upem)

	he, _ := f.face.FontHExtents()
	return Metrics{
		Ascent:     geometry.Pixels(he.Ascender * scale),
		Descent:    geometry.Pixels(he.Descender * scale),
		LineGap:    geometry.Pixels(he.LineGap * scale),
		CapHeight:  geometry.Pixels(f.face.LineMetric(font.CapHeight) * scale),
		XHeight:    geometry.Pixels(f.face.LineMetric(font.XHeight) * scale),
		UnitsPerEm: upem,
	}
}

// LineHeight returns the font's suggested line height at size: ascent minus
// descent plus the line gap. It is the height a single line of text occupies
// before any extra leading is applied.
func (f Face) LineHeight(size geometry.Pixels) geometry.Pixels {
	m := f.Metrics(size)
	return m.Ascent - m.Descent + m.LineGap
}

// BaselineOffset returns the distance from the top of a line of the given
// height to the baseline, centring the font's ascent and descent within the
// line. Text editing and hit testing position the caret against this offset.
func (f Face) BaselineOffset(size, lineHeight geometry.Pixels) geometry.Pixels {
	m := f.Metrics(size)
	paddingTop := (lineHeight - m.Ascent - m.Descent) / 2
	return paddingTop + m.Ascent
}
