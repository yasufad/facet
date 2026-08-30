package style

import "github.com/yasufad/facet/colour"

// Style contains the fully resolved styling information for an element.
//
// Default() is the only valid constructor to obtain a Style. The zero-value
// Style{} has zero opacity and default-initialised fields, rendering nothing.
type Style struct {
	// Display sets the layout strategy for children.
	Display Display

	// Opacity is the element opacity in [0, 1].
	Opacity float32

	// Background is the fill colour of the element.
	Background colour.Rgba

	// FlexGrow controls the relative rate at which this item expands.
	FlexGrow float32

	// testHigh is a high-word test property (bit 64).
	testHigh float32
}

// Default returns the default style.
func Default() Style {
	return Style{
		Display:    DisplayFlex,
		Opacity:    1.0,
		Background: colour.Rgba{},
		FlexGrow:   0.0,
	}
}

// Refine applies any properties explicitly set in r onto s in place.
func (s *Style) Refine(r Refinement) {
	if r.mask.isEmpty() {
		return
	}
	if r.mask.lo != 0 {
		if r.mask.has(propDisplay) {
			s.Display = r.display
		}
		if r.mask.has(propOpacity) {
			s.Opacity = r.opacity
		}
		if r.mask.has(propBackground) {
			s.Background = r.background
		}
		if r.mask.has(propFlexGrow) {
			s.FlexGrow = r.flexGrow
		}
	}
	if r.mask.hi != 0 {
		if r.mask.has(propTestHigh) {
			s.testHigh = r.testHigh
		}
	}
}

// Refined returns a copy of s with all set properties from r applied.
func (s Style) Refined(r Refinement) Style {
	s.Refine(r)
	return s
}
