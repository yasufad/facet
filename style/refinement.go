package style

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/text"
)

// Refinement represents a set of sparse style property overrides.
//
// Unset properties are distinguished from properties explicitly set to their
// zero values via an internal bitset mask.
type Refinement struct {
	mask mask

	display               Display
	position              Position
	visibility            Visibility
	overflowX             Overflow
	overflowY             Overflow
	scrollbarWidth        geometry.Pixels
	allowConcurrentScroll bool
	restrictScrollToAxis  bool

	insetTop    Length
	insetRight  Length
	insetBottom Length
	insetLeft   Length

	marginTop    Length
	marginRight  Length
	marginBottom Length
	marginLeft   Length

	paddingTop    Length
	paddingRight  Length
	paddingBottom Length
	paddingLeft   Length

	borderWidthTop    geometry.Pixels
	borderWidthRight  geometry.Pixels
	borderWidthBottom geometry.Pixels
	borderWidthLeft   geometry.Pixels

	borderColour colour.Rgba
	borderStyle  BorderStyle

	cornerRadiusTopLeft     geometry.Pixels
	cornerRadiusTopRight    geometry.Pixels
	cornerRadiusBottomRight geometry.Pixels
	cornerRadiusBottomLeft  geometry.Pixels

	sizeWidth     Length
	sizeHeight    Length
	minSizeWidth  Length
	minSizeHeight Length
	maxSizeWidth  Length
	maxSizeHeight Length
	aspectRatio   *float32

	gapRow    Length
	gapColumn Length

	alignItems     *AlignItems
	alignSelf      *AlignSelf
	alignContent   *AlignContent
	justifyContent *JustifyContent

	flexDirection FlexDirection
	flexWrap      FlexWrap
	flexBasis     Length
	flexShrink    float32

	background  colour.Rgba
	boxShadow   []BoxShadow
	mouseCursor CursorStyle

	flexGrow             float32
	textColour           colour.Rgba
	fontFamily           string
	fontFeatures         []text.FontFeature
	fontFallbacks        []string
	fontSize             geometry.Pixels
	lineHeight           geometry.Pixels
	fontWeight           text.Weight
	fontStyle            text.Style
	textBackgroundColour colour.Rgba
	underline            *UnderlineStyle
	strikethrough        *StrikethroughStyle
}

// IsEmpty reports whether no properties have been set in this refinement.
func (r Refinement) IsEmpty() bool {
	return r.mask.isEmpty()
}

// MergeFrom applies all explicitly configured properties in other onto r in place,
// overriding any existing property values in r.
func (r *Refinement) MergeFrom(other *Refinement) {
	if other == nil || other.mask.isEmpty() {
		return
	}
	r.mask = r.mask.or(other.mask)
	if other.mask.lo != 0 {
		if other.mask.has(propDisplay) {
			r.display = other.display
		}
		if other.mask.has(propPosition) {
			r.position = other.position
		}
		if other.mask.has(propVisibility) {
			r.visibility = other.visibility
		}
		if other.mask.has(propOverflowX) {
			r.overflowX = other.overflowX
		}
		if other.mask.has(propOverflowY) {
			r.overflowY = other.overflowY
		}
		if other.mask.has(propScrollbarWidth) {
			r.scrollbarWidth = other.scrollbarWidth
		}
		if other.mask.has(propAllowConcurrentScroll) {
			r.allowConcurrentScroll = other.allowConcurrentScroll
		}
		if other.mask.has(propRestrictScrollToAxis) {
			r.restrictScrollToAxis = other.restrictScrollToAxis
		}
		if other.mask.has(propInsetTop) {
			r.insetTop = other.insetTop
		}
		if other.mask.has(propInsetRight) {
			r.insetRight = other.insetRight
		}
		if other.mask.has(propInsetBottom) {
			r.insetBottom = other.insetBottom
		}
		if other.mask.has(propInsetLeft) {
			r.insetLeft = other.insetLeft
		}
		if other.mask.has(propMarginTop) {
			r.marginTop = other.marginTop
		}
		if other.mask.has(propMarginRight) {
			r.marginRight = other.marginRight
		}
		if other.mask.has(propMarginBottom) {
			r.marginBottom = other.marginBottom
		}
		if other.mask.has(propMarginLeft) {
			r.marginLeft = other.marginLeft
		}
		if other.mask.has(propPaddingTop) {
			r.paddingTop = other.paddingTop
		}
		if other.mask.has(propPaddingRight) {
			r.paddingRight = other.paddingRight
		}
		if other.mask.has(propPaddingBottom) {
			r.paddingBottom = other.paddingBottom
		}
		if other.mask.has(propPaddingLeft) {
			r.paddingLeft = other.paddingLeft
		}
		if other.mask.has(propBorderWidthTop) {
			r.borderWidthTop = other.borderWidthTop
		}
		if other.mask.has(propBorderWidthRight) {
			r.borderWidthRight = other.borderWidthRight
		}
		if other.mask.has(propBorderWidthBottom) {
			r.borderWidthBottom = other.borderWidthBottom
		}
		if other.mask.has(propBorderWidthLeft) {
			r.borderWidthLeft = other.borderWidthLeft
		}
		if other.mask.has(propBorderColour) {
			r.borderColour = other.borderColour
		}
		if other.mask.has(propBorderStyle) {
			r.borderStyle = other.borderStyle
		}
		if other.mask.has(propCornerRadiusTopLeft) {
			r.cornerRadiusTopLeft = other.cornerRadiusTopLeft
		}
		if other.mask.has(propCornerRadiusTopRight) {
			r.cornerRadiusTopRight = other.cornerRadiusTopRight
		}
		if other.mask.has(propCornerRadiusBottomRight) {
			r.cornerRadiusBottomRight = other.cornerRadiusBottomRight
		}
		if other.mask.has(propCornerRadiusBottomLeft) {
			r.cornerRadiusBottomLeft = other.cornerRadiusBottomLeft
		}
		if other.mask.has(propSizeWidth) {
			r.sizeWidth = other.sizeWidth
		}
		if other.mask.has(propSizeHeight) {
			r.sizeHeight = other.sizeHeight
		}
		if other.mask.has(propMinSizeWidth) {
			r.minSizeWidth = other.minSizeWidth
		}
		if other.mask.has(propMinSizeHeight) {
			r.minSizeHeight = other.minSizeHeight
		}
		if other.mask.has(propMaxSizeWidth) {
			r.maxSizeWidth = other.maxSizeWidth
		}
		if other.mask.has(propMaxSizeHeight) {
			r.maxSizeHeight = other.maxSizeHeight
		}
		if other.mask.has(propAspectRatio) {
			r.aspectRatio = other.aspectRatio
		}
		if other.mask.has(propGapRow) {
			r.gapRow = other.gapRow
		}
		if other.mask.has(propGapColumn) {
			r.gapColumn = other.gapColumn
		}
		if other.mask.has(propAlignItems) {
			r.alignItems = other.alignItems
		}
		if other.mask.has(propAlignSelf) {
			r.alignSelf = other.alignSelf
		}
		if other.mask.has(propAlignContent) {
			r.alignContent = other.alignContent
		}
		if other.mask.has(propJustifyContent) {
			r.justifyContent = other.justifyContent
		}
		if other.mask.has(propFlexDirection) {
			r.flexDirection = other.flexDirection
		}
		if other.mask.has(propFlexWrap) {
			r.flexWrap = other.flexWrap
		}
		if other.mask.has(propFlexBasis) {
			r.flexBasis = other.flexBasis
		}
		if other.mask.has(propFlexShrink) {
			r.flexShrink = other.flexShrink
		}
		if other.mask.has(propBackground) {
			r.background = other.background
		}
		if other.mask.has(propBoxShadow) {
			r.boxShadow = other.boxShadow
		}
		if other.mask.has(propMouseCursor) {
			r.mouseCursor = other.mouseCursor
		}
	}
	if other.mask.hi != 0 {
		if other.mask.has(propFlexGrow) {
			r.flexGrow = other.flexGrow
		}
		if other.mask.has(propTextColour) {
			r.textColour = other.textColour
		}
		if other.mask.has(propFontFamily) {
			r.fontFamily = other.fontFamily
		}
		if other.mask.has(propFontFeatures) {
			r.fontFeatures = other.fontFeatures
		}
		if other.mask.has(propFontFallbacks) {
			r.fontFallbacks = other.fontFallbacks
		}
		if other.mask.has(propFontSize) {
			r.fontSize = other.fontSize
		}
		if other.mask.has(propLineHeight) {
			r.lineHeight = other.lineHeight
		}
		if other.mask.has(propFontWeight) {
			r.fontWeight = other.fontWeight
		}
		if other.mask.has(propFontStyle) {
			r.fontStyle = other.fontStyle
		}
		if other.mask.has(propTextBackgroundColour) {
			r.textBackgroundColour = other.textBackgroundColour
		}
		if other.mask.has(propUnderline) {
			r.underline = other.underline
		}
		if other.mask.has(propStrikethrough) {
			r.strikethrough = other.strikethrough
		}
	}
}

// Mutators on *Refinement

// SetDisplay sets the layout strategy for children.
func (r *Refinement) SetDisplay(d Display) {
	r.mask.set(propDisplay)
	r.display = d
}

// SetPosition sets the CSS positioning strategy.
func (r *Refinement) SetPosition(p Position) {
	r.mask.set(propPosition)
	r.position = p
}

// SetVisibility sets whether the element is rendered.
func (r *Refinement) SetVisibility(v Visibility) {
	r.mask.set(propVisibility)
	r.visibility = v
}

// SetOverflow sets both horizontal and vertical overflow handling.
func (r *Refinement) SetOverflow(o Overflow) {
	r.SetOverflowX(o)
	r.SetOverflowY(o)
}

// SetOverflowX sets horizontal overflow handling.
func (r *Refinement) SetOverflowX(o Overflow) {
	r.mask.set(propOverflowX)
	r.overflowX = o
}

// SetOverflowY sets vertical overflow handling.
func (r *Refinement) SetOverflowY(o Overflow) {
	r.mask.set(propOverflowY)
	r.overflowY = o
}

// SetScrollbarWidth sets the space reserved for scrollbars.
func (r *Refinement) SetScrollbarWidth(w geometry.Pixels) {
	r.mask.set(propScrollbarWidth)
	r.scrollbarWidth = w
}

// SetAllowConcurrentScroll sets whether both axes can scroll concurrently.
func (r *Refinement) SetAllowConcurrentScroll(allow bool) {
	r.mask.set(propAllowConcurrentScroll)
	r.allowConcurrentScroll = allow
}

// SetRestrictScrollToAxis sets whether scroll is locked to the dominant gesture axis.
func (r *Refinement) SetRestrictScrollToAxis(restrict bool) {
	r.mask.set(propRestrictScrollToAxis)
	r.restrictScrollToAxis = restrict
}

// SetInset sets all four inset offsets.
func (r *Refinement) SetInset(l Length) {
	r.SetInsetTop(l)
	r.SetInsetRight(l)
	r.SetInsetBottom(l)
	r.SetInsetLeft(l)
}

// SetInsetTop sets the top inset offset.
func (r *Refinement) SetInsetTop(l Length) {
	r.mask.set(propInsetTop)
	r.insetTop = l
}

// SetInsetRight sets the right inset offset.
func (r *Refinement) SetInsetRight(l Length) {
	r.mask.set(propInsetRight)
	r.insetRight = l
}

// SetInsetBottom sets the bottom inset offset.
func (r *Refinement) SetInsetBottom(l Length) {
	r.mask.set(propInsetBottom)
	r.insetBottom = l
}

// SetInsetLeft sets the left inset offset.
func (r *Refinement) SetInsetLeft(l Length) {
	r.mask.set(propInsetLeft)
	r.insetLeft = l
}

// SetMargin sets margin on all four sides.
func (r *Refinement) SetMargin(l Length) {
	r.SetMarginTop(l)
	r.SetMarginRight(l)
	r.SetMarginBottom(l)
	r.SetMarginLeft(l)
}

// SetMarginTop sets top margin.
func (r *Refinement) SetMarginTop(l Length) {
	r.mask.set(propMarginTop)
	r.marginTop = l
}

// SetMarginRight sets right margin.
func (r *Refinement) SetMarginRight(l Length) {
	r.mask.set(propMarginRight)
	r.marginRight = l
}

// SetMarginBottom sets bottom margin.
func (r *Refinement) SetMarginBottom(l Length) {
	r.mask.set(propMarginBottom)
	r.marginBottom = l
}

// SetMarginLeft sets left margin.
func (r *Refinement) SetMarginLeft(l Length) {
	r.mask.set(propMarginLeft)
	r.marginLeft = l
}

// SetPadding sets padding on all four sides.
func (r *Refinement) SetPadding(l Length) {
	r.SetPaddingTop(l)
	r.SetPaddingRight(l)
	r.SetPaddingBottom(l)
	r.SetPaddingLeft(l)
}

// SetPaddingTop sets top padding.
func (r *Refinement) SetPaddingTop(l Length) {
	r.mask.set(propPaddingTop)
	r.paddingTop = l
}

// SetPaddingRight sets right padding.
func (r *Refinement) SetPaddingRight(l Length) {
	r.mask.set(propPaddingRight)
	r.paddingRight = l
}

// SetPaddingBottom sets bottom padding.
func (r *Refinement) SetPaddingBottom(l Length) {
	r.mask.set(propPaddingBottom)
	r.paddingBottom = l
}

// SetPaddingLeft sets left padding.
func (r *Refinement) SetPaddingLeft(l Length) {
	r.mask.set(propPaddingLeft)
	r.paddingLeft = l
}

// SetBorderWidth sets border line thickness on all four sides.
func (r *Refinement) SetBorderWidth(w geometry.Pixels) {
	r.SetBorderWidthTop(w)
	r.SetBorderWidthRight(w)
	r.SetBorderWidthBottom(w)
	r.SetBorderWidthLeft(w)
}

// SetBorderWidthTop sets top border width.
func (r *Refinement) SetBorderWidthTop(w geometry.Pixels) {
	r.mask.set(propBorderWidthTop)
	r.borderWidthTop = w
}

// SetBorderWidthRight sets right border width.
func (r *Refinement) SetBorderWidthRight(w geometry.Pixels) {
	r.mask.set(propBorderWidthRight)
	r.borderWidthRight = w
}

// SetBorderWidthBottom sets bottom border width.
func (r *Refinement) SetBorderWidthBottom(w geometry.Pixels) {
	r.mask.set(propBorderWidthBottom)
	r.borderWidthBottom = w
}

// SetBorderWidthLeft sets left border width.
func (r *Refinement) SetBorderWidthLeft(w geometry.Pixels) {
	r.mask.set(propBorderWidthLeft)
	r.borderWidthLeft = w
}

// SetBorderColour sets the border colour.
func (r *Refinement) SetBorderColour(c colour.Rgba) {
	r.mask.set(propBorderColour)
	r.borderColour = c
}

// SetBorderColourHsla sets the border colour from an Hsla value.
func (r *Refinement) SetBorderColourHsla(c colour.Hsla) {
	r.SetBorderColour(c.Rgba())
}

// SetBorderStyle sets the border line style.
func (r *Refinement) SetBorderStyle(s BorderStyle) {
	r.mask.set(propBorderStyle)
	r.borderStyle = s
}

// SetCornerRadius sets corner radius on all four corners.
func (r *Refinement) SetCornerRadius(radius geometry.Pixels) {
	r.SetCornerRadiusTopLeft(radius)
	r.SetCornerRadiusTopRight(radius)
	r.SetCornerRadiusBottomRight(radius)
	r.SetCornerRadiusBottomLeft(radius)
}

// SetCornerRadiusTopLeft sets the top-left corner radius.
func (r *Refinement) SetCornerRadiusTopLeft(radius geometry.Pixels) {
	r.mask.set(propCornerRadiusTopLeft)
	r.cornerRadiusTopLeft = radius
}

// SetCornerRadiusTopRight sets the top-right corner radius.
func (r *Refinement) SetCornerRadiusTopRight(radius geometry.Pixels) {
	r.mask.set(propCornerRadiusTopRight)
	r.cornerRadiusTopRight = radius
}

// SetCornerRadiusBottomRight sets the bottom-right corner radius.
func (r *Refinement) SetCornerRadiusBottomRight(radius geometry.Pixels) {
	r.mask.set(propCornerRadiusBottomRight)
	r.cornerRadiusBottomRight = radius
}

// SetCornerRadiusBottomLeft sets the bottom-left corner radius.
func (r *Refinement) SetCornerRadiusBottomLeft(radius geometry.Pixels) {
	r.mask.set(propCornerRadiusBottomLeft)
	r.cornerRadiusBottomLeft = radius
}

// SetSize sets both preferred width and height.
func (r *Refinement) SetSize(width, height Length) {
	r.SetWidth(width)
	r.SetHeight(height)
}

// SetWidth sets the preferred width.
func (r *Refinement) SetWidth(w Length) {
	r.mask.set(propSizeWidth)
	r.sizeWidth = w
}

// SetHeight sets the preferred height.
func (r *Refinement) SetHeight(h Length) {
	r.mask.set(propSizeHeight)
	r.sizeHeight = h
}

// SetMinSize sets both minimum width and height constraints.
func (r *Refinement) SetMinSize(width, height Length) {
	r.SetMinWidth(width)
	r.SetMinHeight(height)
}

// SetMinWidth sets the minimum width constraint.
func (r *Refinement) SetMinWidth(w Length) {
	r.mask.set(propMinSizeWidth)
	r.minSizeWidth = w
}

// SetMinHeight sets the minimum height constraint.
func (r *Refinement) SetMinHeight(h Length) {
	r.mask.set(propMinSizeHeight)
	r.minSizeHeight = h
}

// SetMaxSize sets both maximum width and height constraints.
func (r *Refinement) SetMaxSize(width, height Length) {
	r.SetMaxWidth(width)
	r.SetMaxHeight(height)
}

// SetMaxWidth sets the maximum width constraint.
func (r *Refinement) SetMaxWidth(w Length) {
	r.mask.set(propMaxSizeWidth)
	r.maxSizeWidth = w
}

// SetMaxHeight sets the maximum height constraint.
func (r *Refinement) SetMaxHeight(h Length) {
	r.mask.set(propMaxSizeHeight)
	r.maxSizeHeight = h
}

// SetAspectRatio sets the preferred width-to-height ratio.
func (r *Refinement) SetAspectRatio(ratio float32) {
	r.mask.set(propAspectRatio)
	r.aspectRatio = &ratio
}

// SetGap sets both row and column gap between flex items.
func (r *Refinement) SetGap(row, col Length) {
	r.SetGapRow(row)
	r.SetGapCol(col)
}

// SetGapRow sets row gap.
func (r *Refinement) SetGapRow(row Length) {
	r.mask.set(propGapRow)
	r.gapRow = row
}

// SetGapCol sets column gap.
func (r *Refinement) SetGapCol(col Length) {
	r.mask.set(propGapColumn)
	r.gapColumn = col
}

// SetAlignItems sets cross-axis alignment for children.
func (r *Refinement) SetAlignItems(a AlignItems) {
	r.mask.set(propAlignItems)
	r.alignItems = &a
}

// SetAlignSelf sets cross-axis alignment for this item.
func (r *Refinement) SetAlignSelf(a AlignSelf) {
	r.mask.set(propAlignSelf)
	r.alignSelf = &a
}

// SetAlignContent sets multi-line content distribution.
func (r *Refinement) SetAlignContent(a AlignContent) {
	r.mask.set(propAlignContent)
	r.alignContent = &a
}

// SetJustifyContent sets main-axis distribution for children.
func (r *Refinement) SetJustifyContent(j JustifyContent) {
	r.mask.set(propJustifyContent)
	r.justifyContent = &j
}

// SetFlexDirection sets the flexbox main axis direction.
func (r *Refinement) SetFlexDirection(d FlexDirection) {
	r.mask.set(propFlexDirection)
	r.flexDirection = d
}

// SetFlexWrap sets flex wrap mode.
func (r *Refinement) SetFlexWrap(w FlexWrap) {
	r.mask.set(propFlexWrap)
	r.flexWrap = w
}

// SetFlexBasis sets the initial main-axis size.
func (r *Refinement) SetFlexBasis(b Length) {
	r.mask.set(propFlexBasis)
	r.flexBasis = b
}

// SetFlexGrow sets the flex grow factor.
func (r *Refinement) SetFlexGrow(grow float32) {
	r.mask.set(propFlexGrow)
	r.flexGrow = grow
}

// SetFlexShrink sets the flex shrink factor.
func (r *Refinement) SetFlexShrink(shrink float32) {
	r.mask.set(propFlexShrink)
	r.flexShrink = shrink
}

// SetBackground sets the background fill colour.
func (r *Refinement) SetBackground(c colour.Rgba) {
	r.mask.set(propBackground)
	r.background = c
}

// SetBackgroundHsla sets the background fill colour from an Hsla value.
func (r *Refinement) SetBackgroundHsla(c colour.Hsla) {
	r.SetBackground(c.Rgba())
}

// SetBoxShadow sets the box shadow slice.
func (r *Refinement) SetBoxShadow(shadows []BoxShadow) {
	r.mask.set(propBoxShadow)
	r.boxShadow = shadows
}

// SetMouseCursor sets the hover pointer cursor shape.
func (r *Refinement) SetMouseCursor(c CursorStyle) {
	r.mask.set(propMouseCursor)
	r.mouseCursor = c
}

// Typography / Text Mutators

// SetTextColour sets the text colour.
func (r *Refinement) SetTextColour(c colour.Rgba) {
	r.mask.set(propTextColour)
	r.textColour = c
}

// SetTextColourHsla sets the text colour from an Hsla value.
func (r *Refinement) SetTextColourHsla(c colour.Hsla) {
	r.SetTextColour(c.Rgba())
}

// SetFontFamily sets the primary font family name.
func (r *Refinement) SetFontFamily(family string) {
	r.mask.set(propFontFamily)
	r.fontFamily = family
}

// SetFontFeatures sets OpenType font features.
func (r *Refinement) SetFontFeatures(features []text.FontFeature) {
	r.mask.set(propFontFeatures)
	r.fontFeatures = features
}

// SetFontFallbacks sets fallback font families.
func (r *Refinement) SetFontFallbacks(fallbacks []string) {
	r.mask.set(propFontFallbacks)
	r.fontFallbacks = fallbacks
}

// SetFontSize sets the font size in logical pixels.
func (r *Refinement) SetFontSize(size geometry.Pixels) {
	r.mask.set(propFontSize)
	r.fontSize = size
}

// SetLineHeight sets the line height in logical pixels.
func (r *Refinement) SetLineHeight(height geometry.Pixels) {
	r.mask.set(propLineHeight)
	r.lineHeight = height
}

// SetFontWeight sets the stroke weight.
func (r *Refinement) SetFontWeight(weight text.Weight) {
	r.mask.set(propFontWeight)
	r.fontWeight = weight
}

// SetFontStyle sets upright or italic.
func (r *Refinement) SetFontStyle(style text.Style) {
	r.mask.set(propFontStyle)
	r.fontStyle = style
}

// SetTextBackgroundColour sets the background highlight colour behind text.
func (r *Refinement) SetTextBackgroundColour(c colour.Rgba) {
	r.mask.set(propTextBackgroundColour)
	r.textBackgroundColour = c
}

// SetTextBackgroundColourHsla sets the text background colour from an Hsla value.
func (r *Refinement) SetTextBackgroundColourHsla(c colour.Hsla) {
	r.SetTextBackgroundColour(c.Rgba())
}

// SetUnderline configures an underline.
func (r *Refinement) SetUnderline(u UnderlineStyle) {
	r.mask.set(propUnderline)
	r.underline = &u
}

// ClearUnderline removes underline styling.
func (r *Refinement) ClearUnderline() {
	r.mask.set(propUnderline)
	r.underline = nil
}

// SetStrikethrough configures a strikethrough line.
func (r *Refinement) SetStrikethrough(s StrikethroughStyle) {
	r.mask.set(propStrikethrough)
	r.strikethrough = &s
}

// ClearStrikethrough removes strikethrough styling.
func (r *Refinement) ClearStrikethrough() {
	r.mask.set(propStrikethrough)
	r.strikethrough = nil
}
