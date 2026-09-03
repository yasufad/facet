package style

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/text"
)

// UnderlineStyle describes the appearance of an underline beneath text.
type UnderlineStyle struct {
	// Thickness is the line height in pixels.
	Thickness geometry.Pixels

	// Colour is the underline colour. If zero, defaults to the text colour.
	Colour colour.Rgba

	// Wavy draws a wavy spellchecker-like line.
	Wavy bool
}

// StrikethroughStyle describes the appearance of a strikethrough line across text.
type StrikethroughStyle struct {
	// Thickness is the line height in pixels.
	Thickness geometry.Pixels

	// Colour is the line colour. If zero, defaults to the text colour.
	Colour colour.Rgba
}

// TextStyle holds the typographic styling for text elements and runs.
type TextStyle struct {
	// Colour is the primary text colour.
	Colour colour.Rgba

	// FontFamily is the primary font family name.
	FontFamily string

	// FontFeatures are OpenType feature overrides.
	FontFeatures []text.FontFeature

	// FontFallbacks are fallback font family names.
	FontFallbacks []string

	// FontSize is the font size in logical pixels.
	FontSize geometry.Pixels

	// LineHeight is the line height in logical pixels.
	LineHeight geometry.Pixels

	// FontWeight is the stroke weight.
	FontWeight text.Weight

	// FontStyle selects between upright and italic.
	FontStyle text.Style

	// BackgroundColour is the highlight colour beneath the text.
	BackgroundColour colour.Rgba

	// Underline configures underline styling.
	Underline *UnderlineStyle

	// Strikethrough configures strikethrough styling.
	Strikethrough *StrikethroughStyle
}

// DefaultTextStyle returns the default text styling.
func DefaultTextStyle() TextStyle {
	return TextStyle{
		Colour:     colour.Rgb(0x000000),
		FontFamily: "",
		FontSize:   16,
		LineHeight: 20,
		FontWeight: text.WeightNormal,
		FontStyle:  text.StyleNormal,
	}
}
