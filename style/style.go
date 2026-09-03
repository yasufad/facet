package style

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/layout"
)

// Style contains the fully resolved styling information for an element.
//
// Default() is the only valid constructor to obtain a Style. The zero-value
// Style{} has zero opacity and uninitialised fields, rendering nothing.
type Style struct {
	// Display sets the layout strategy for children.
	Display Display

	// Position sets the positioning strategy.
	Position Position

	// Visibility controls whether the element is rendered.
	Visibility Visibility

	// Overflow controls horizontal and vertical overflow clipping and scrolling.
	Overflow Point[Overflow]

	// ScrollbarWidth is the space reserved for scrollbars.
	ScrollbarWidth geometry.Pixels

	// RestrictScrollToAxis locks scrolling to the initial gesture axis.
	RestrictScrollToAxis bool

	// Inset sets offsets for positioned elements.
	Inset Edges[Length]

	// Margin sets the outer spacing on each side.
	Margin Edges[Length]

	// Padding sets the inner spacing on each side.
	Padding Edges[Length]

	// BorderWidths sets the border line thickness on each side.
	BorderWidths geometry.Edges[geometry.Pixels]

	// BorderColour is the colour of the element border.
	BorderColour colour.Rgba

	// BorderStyle is the border line style (solid, dashed, dotted, none).
	BorderStyle BorderStyle

	// CornerRadii sets the radius for each corner.
	CornerRadii geometry.Corners[geometry.Pixels]

	// Size sets the preferred width and height.
	Size Size[Length]

	// MinSize sets the minimum width and height constraints.
	MinSize Size[Length]

	// MaxSize sets the maximum width and height constraints.
	MaxSize Size[Length]

	// AspectRatio sets the preferred width-to-height ratio.
	AspectRatio *float32

	// Gap sets the spacing between flex items (Height: row gap, Width: column gap).
	Gap Size[Length]

	// AlignItems sets cross-axis alignment for flex children.
	AlignItems *AlignItems

	// AlignSelf sets cross-axis alignment for this specific item.
	AlignSelf *AlignSelf

	// AlignContent sets multi-line content distribution.
	AlignContent *AlignContent

	// JustifyContent sets main-axis distribution for flex children.
	JustifyContent *JustifyContent

	// FlexDirection sets the flexbox main axis direction.
	FlexDirection FlexDirection

	// FlexWrap sets whether flex items wrap onto multiple lines.
	FlexWrap FlexWrap

	// FlexBasis sets the initial main axis size.
	FlexBasis Length

	// FlexGrow controls the relative expansion rate.
	FlexGrow float32

	// FlexShrink controls the relative contraction rate.
	FlexShrink float32

	// Background is the fill colour of the element.
	Background colour.Rgba

	// BoxShadow holds drop and inset shadows.
	BoxShadow []BoxShadow

	// MouseCursor is the pointer cursor shape on hover.
	MouseCursor CursorStyle

	// Text contains typographic styling.
	Text TextStyle
}

// Default returns the default style.
func Default() Style {
	return Style{
		Display:              DisplayFlex,
		Position:             PositionRelative,
		Visibility:           VisibilityVisible,
		Overflow:             NewPoint(OverflowVisible, OverflowVisible),
		ScrollbarWidth:       0,
		RestrictScrollToAxis: false,
		Inset:                NewEdges(Auto(), Auto(), Auto(), Auto()),
		Margin:               NewEdges(Px(0), Px(0), Px(0), Px(0)),
		Padding:              NewEdges(Px(0), Px(0), Px(0), Px(0)),
		BorderWidths:         geometry.NewEdges(geometry.Pixels(0), 0, 0, 0),
		BorderColour:         colour.Rgba{},
		BorderStyle:          BorderStyleSolid,
		CornerRadii:          geometry.NewCorners(geometry.Pixels(0), 0, 0, 0),
		Size:                 NewSize(Auto(), Auto()),
		MinSize:              NewSize(Auto(), Auto()),
		MaxSize:              NewSize(Auto(), Auto()),
		AspectRatio:          nil,
		Gap:                  NewSize(Px(0), Px(0)),
		AlignItems:           nil,
		AlignSelf:            nil,
		AlignContent:         nil,
		JustifyContent:       nil,
		FlexDirection:        FlexDirectionRow,
		FlexWrap:             FlexWrapNoWrap,
		FlexBasis:            Auto(),
		FlexGrow:             0.0,
		FlexShrink:           1.0,
		Background:           colour.Rgba{},
		BoxShadow:            nil,
		MouseCursor:          CursorDefault,
		Text:                 DefaultTextStyle(),
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
		if r.mask.has(propPosition) {
			s.Position = r.position
		}
		if r.mask.has(propVisibility) {
			s.Visibility = r.visibility
		}
		if r.mask.has(propOverflowX) {
			s.Overflow.X = r.overflowX
		}
		if r.mask.has(propOverflowY) {
			s.Overflow.Y = r.overflowY
		}
		if r.mask.has(propScrollbarWidth) {
			s.ScrollbarWidth = r.scrollbarWidth
		}
		if r.mask.has(propRestrictScrollToAxis) {
			s.RestrictScrollToAxis = r.restrictScrollToAxis
		}
		if r.mask.has(propInsetTop) {
			s.Inset.Top = r.insetTop
		}
		if r.mask.has(propInsetRight) {
			s.Inset.Right = r.insetRight
		}
		if r.mask.has(propInsetBottom) {
			s.Inset.Bottom = r.insetBottom
		}
		if r.mask.has(propInsetLeft) {
			s.Inset.Left = r.insetLeft
		}
		if r.mask.has(propMarginTop) {
			s.Margin.Top = r.marginTop
		}
		if r.mask.has(propMarginRight) {
			s.Margin.Right = r.marginRight
		}
		if r.mask.has(propMarginBottom) {
			s.Margin.Bottom = r.marginBottom
		}
		if r.mask.has(propMarginLeft) {
			s.Margin.Left = r.marginLeft
		}
		if r.mask.has(propPaddingTop) {
			s.Padding.Top = r.paddingTop
		}
		if r.mask.has(propPaddingRight) {
			s.Padding.Right = r.paddingRight
		}
		if r.mask.has(propPaddingBottom) {
			s.Padding.Bottom = r.paddingBottom
		}
		if r.mask.has(propPaddingLeft) {
			s.Padding.Left = r.paddingLeft
		}
		if r.mask.has(propBorderWidthTop) {
			s.BorderWidths.Top = r.borderWidthTop
		}
		if r.mask.has(propBorderWidthRight) {
			s.BorderWidths.Right = r.borderWidthRight
		}
		if r.mask.has(propBorderWidthBottom) {
			s.BorderWidths.Bottom = r.borderWidthBottom
		}
		if r.mask.has(propBorderWidthLeft) {
			s.BorderWidths.Left = r.borderWidthLeft
		}
		if r.mask.has(propBorderColour) {
			s.BorderColour = r.borderColour
		}
		if r.mask.has(propBorderStyle) {
			s.BorderStyle = r.borderStyle
		}
		if r.mask.has(propCornerRadiusTopLeft) {
			s.CornerRadii.TopLeft = r.cornerRadiusTopLeft
		}
		if r.mask.has(propCornerRadiusTopRight) {
			s.CornerRadii.TopRight = r.cornerRadiusTopRight
		}
		if r.mask.has(propCornerRadiusBottomRight) {
			s.CornerRadii.BottomRight = r.cornerRadiusBottomRight
		}
		if r.mask.has(propCornerRadiusBottomLeft) {
			s.CornerRadii.BottomLeft = r.cornerRadiusBottomLeft
		}
		if r.mask.has(propSizeWidth) {
			s.Size.Width = r.sizeWidth
		}
		if r.mask.has(propSizeHeight) {
			s.Size.Height = r.sizeHeight
		}
		if r.mask.has(propMinSizeWidth) {
			s.MinSize.Width = r.minSizeWidth
		}
		if r.mask.has(propMinSizeHeight) {
			s.MinSize.Height = r.minSizeHeight
		}
		if r.mask.has(propMaxSizeWidth) {
			s.MaxSize.Width = r.maxSizeWidth
		}
		if r.mask.has(propMaxSizeHeight) {
			s.MaxSize.Height = r.maxSizeHeight
		}
		if r.mask.has(propAspectRatio) {
			s.AspectRatio = r.aspectRatio
		}
		if r.mask.has(propGapRow) {
			s.Gap.Height = r.gapRow
		}
		if r.mask.has(propGapColumn) {
			s.Gap.Width = r.gapColumn
		}
		if r.mask.has(propAlignItems) {
			s.AlignItems = r.alignItems
		}
		if r.mask.has(propAlignSelf) {
			s.AlignSelf = r.alignSelf
		}
		if r.mask.has(propAlignContent) {
			s.AlignContent = r.alignContent
		}
		if r.mask.has(propJustifyContent) {
			s.JustifyContent = r.justifyContent
		}
		if r.mask.has(propFlexDirection) {
			s.FlexDirection = r.flexDirection
		}
		if r.mask.has(propFlexWrap) {
			s.FlexWrap = r.flexWrap
		}
		if r.mask.has(propFlexBasis) {
			s.FlexBasis = r.flexBasis
		}
		if r.mask.has(propFlexShrink) {
			s.FlexShrink = r.flexShrink
		}
		if r.mask.has(propBackground) {
			s.Background = r.background
		}
		if r.mask.has(propBoxShadow) {
			s.BoxShadow = r.boxShadow
		}
		if r.mask.has(propMouseCursor) {
			s.MouseCursor = r.mouseCursor
		}
	}
	if r.mask.hi != 0 {
		if r.mask.has(propFlexGrow) {
			s.FlexGrow = r.flexGrow
		}
		if r.mask.has(propTextColour) {
			s.Text.Colour = r.textColour
		}
		if r.mask.has(propFontFamily) {
			s.Text.FontFamily = r.fontFamily
		}
		if r.mask.has(propFontFeatures) {
			s.Text.FontFeatures = r.fontFeatures
		}
		if r.mask.has(propFontFallbacks) {
			s.Text.FontFallbacks = r.fontFallbacks
		}
		if r.mask.has(propFontSize) {
			s.Text.FontSize = r.fontSize
		}
		if r.mask.has(propLineHeight) {
			s.Text.LineHeight = r.lineHeight
		}
		if r.mask.has(propFontWeight) {
			s.Text.FontWeight = r.fontWeight
		}
		if r.mask.has(propFontStyle) {
			s.Text.FontStyle = r.fontStyle
		}
		if r.mask.has(propTextBackgroundColour) {
			s.Text.BackgroundColour = r.textBackgroundColour
		}
		if r.mask.has(propUnderline) {
			s.Text.Underline = r.underline
		}
		if r.mask.has(propStrikethrough) {
			s.Text.Strikethrough = r.strikethrough
		}
	}
}

// Refined returns a copy of s with all set properties from r applied.
func (s Style) Refined(r Refinement) Style {
	s.Refine(r)
	return s
}

// ToLayout converts the resolved Style into layout's input Style.
func (s Style) ToLayout(remSize geometry.Pixels) layout.Style {
	l := layout.NewStyle()
	l.Display = s.Display.toLayout()
	l.Position = s.Position.toLayout()
	l.Overflow = layout.Point[layout.Overflow]{
		X: s.Overflow.X.toLayout(),
		Y: s.Overflow.Y.toLayout(),
	}
	l.ScrollbarWidth = float32(s.ScrollbarWidth)

	l.Inset = layout.Rect[layout.LengthPercentageAuto]{
		Top:    s.Inset.Top.ToLayoutLPA(remSize),
		Right:  s.Inset.Right.ToLayoutLPA(remSize),
		Bottom: s.Inset.Bottom.ToLayoutLPA(remSize),
		Left:   s.Inset.Left.ToLayoutLPA(remSize),
	}

	l.Margin = layout.Rect[layout.LengthPercentageAuto]{
		Top:    s.Margin.Top.ToLayoutLPA(remSize),
		Right:  s.Margin.Right.ToLayoutLPA(remSize),
		Bottom: s.Margin.Bottom.ToLayoutLPA(remSize),
		Left:   s.Margin.Left.ToLayoutLPA(remSize),
	}

	l.Padding = layout.Rect[layout.LengthPercentage]{
		Top:    s.Padding.Top.ToLayoutLP(remSize),
		Right:  s.Padding.Right.ToLayoutLP(remSize),
		Bottom: s.Padding.Bottom.ToLayoutLP(remSize),
		Left:   s.Padding.Left.ToLayoutLP(remSize),
	}

	l.Border = layout.Rect[layout.LengthPercentage]{
		Top:    layout.LPLength(float32(s.BorderWidths.Top)),
		Right:  layout.LPLength(float32(s.BorderWidths.Right)),
		Bottom: layout.LPLength(float32(s.BorderWidths.Bottom)),
		Left:   layout.LPLength(float32(s.BorderWidths.Left)),
	}

	l.Size = layout.Size[layout.Dimension]{
		Width:  s.Size.Width.ToLayoutDimension(remSize),
		Height: s.Size.Height.ToLayoutDimension(remSize),
	}

	l.MinSize = layout.Size[layout.LengthPercentageAuto]{
		Width:  s.MinSize.Width.ToLayoutLPA(remSize),
		Height: s.MinSize.Height.ToLayoutLPA(remSize),
	}

	l.MaxSize = layout.Size[layout.LengthPercentageAuto]{
		Width:  s.MaxSize.Width.ToLayoutLPA(remSize),
		Height: s.MaxSize.Height.ToLayoutLPA(remSize),
	}

	l.AspectRatio = s.AspectRatio

	l.Gap = layout.Size[layout.LengthPercentage]{
		Width:  s.Gap.Width.ToLayoutLP(remSize),
		Height: s.Gap.Height.ToLayoutLP(remSize),
	}

	if s.AlignItems != nil {
		a := s.AlignItems.toLayout()
		l.AlignItems = &a
	}
	if s.AlignSelf != nil {
		a := s.AlignSelf.toLayout()
		l.AlignSelf = &a
	}
	if s.AlignContent != nil {
		a := s.AlignContent.toLayout()
		l.AlignContent = &a
	}
	if s.JustifyContent != nil {
		a := s.JustifyContent.toLayout()
		l.JustifyContent = &a
	}

	l.FlexDirection = s.FlexDirection.toLayout()
	l.FlexWrap = s.FlexWrap.toLayout()
	l.FlexBasis = s.FlexBasis.ToLayoutDimension(remSize)
	l.FlexGrow = s.FlexGrow
	l.FlexShrink = s.FlexShrink

	return l
}
