package style

// Property indices for the parallel bitset mask.
//
// Inset, margin, padding, border widths, and corner radii use 4 bits each
// (one per edge or corner). Size, min size, max size and gap use 2 bits each
// (width/height or row/column).
const (
	// Low word: layout, box model, visual properties (bits 0–63).
	propDisplay uint8 = iota
	propPosition
	propVisibility
	propOverflowX
	propOverflowY
	propScrollbarWidth
	propAllowConcurrentScroll
	propRestrictScrollToAxis

	propInsetTop
	propInsetRight
	propInsetBottom
	propInsetLeft

	propMarginTop
	propMarginRight
	propMarginBottom
	propMarginLeft

	propPaddingTop
	propPaddingRight
	propPaddingBottom
	propPaddingLeft

	propBorderWidthTop
	propBorderWidthRight
	propBorderWidthBottom
	propBorderWidthLeft

	propBorderColour
	propBorderStyle

	propCornerRadiusTopLeft
	propCornerRadiusTopRight
	propCornerRadiusBottomRight
	propCornerRadiusBottomLeft

	propSizeWidth
	propSizeHeight
	propMinSizeWidth
	propMinSizeHeight
	propMaxSizeWidth
	propMaxSizeHeight
	propAspectRatio

	propGapRow
	propGapColumn

	propAlignItems
	propAlignSelf
	propAlignContent
	propJustifyContent

	propFlexDirection
	propFlexWrap
	propFlexBasis
	propFlexShrink

	propBackground
	propOpacity
	propBoxShadow
	propMouseCursor
)

const (
	// High word: flex grow and typography/text properties (bits 64–127).
	propFlexGrow uint8 = 64 + iota
	propTextColour
	propFontFamily
	propFontFeatures
	propFontFallbacks
	propFontSize
	propLineHeight
	propFontWeight
	propFontStyle
	propTextBackgroundColour
	propUnderline
	propStrikethrough
	propWhiteSpace
	propTextOverflow
	propTextAlign
	propLineClamp
)

// mask is a 128-bit bitset indicating which properties have been explicitly
// configured on a Refinement.
type mask struct {
	lo uint64
	hi uint64
}

// has reports whether the property bit is set.
func (m mask) has(bit uint8) bool {
	if bit < 64 {
		return (m.lo & (uint64(1) << bit)) != 0
	}
	return (m.hi & (uint64(1) << (bit - 64))) != 0
}

// set marks the property bit as set.
func (m *mask) set(bit uint8) {
	if bit < 64 {
		m.lo |= uint64(1) << bit
	} else {
		m.hi |= uint64(1) << (bit - 64)
	}
}

// or returns the union of m and other.
func (m mask) or(other mask) mask {
	return mask{
		lo: m.lo | other.lo,
		hi: m.hi | other.hi,
	}
}

// isEmpty reports whether no property bits are set.
func (m mask) isEmpty() bool {
	return m.lo == 0 && m.hi == 0
}
