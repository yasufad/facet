package style

import "github.com/yasufad/facet/layout"

// Position sets the CSS positioning strategy.
type Position uint8

const (
	// PositionRelative offsets the element relative to its normal layout position.
	PositionRelative Position = iota
	// PositionAbsolute positions the element relative to its closest positioned ancestor.
	PositionAbsolute
)

// Visibility controls whether an element is rendered.
type Visibility uint8

const (
	// VisibilityVisible renders the element normally.
	VisibilityVisible Visibility = iota
	// VisibilityHidden hides the element without removing it from layout calculation.
	VisibilityHidden
)

// Overflow controls how content overflowing an element's container is handled.
type Overflow uint8

const (
	// OverflowVisible allows overflowing content to render outside the container.
	OverflowVisible Overflow = iota
	// OverflowClip clips overflowing content to the container's bounds.
	OverflowClip
	// OverflowHidden hides overflowing content and sets automatic minimum sizing to 0.
	OverflowHidden
	// OverflowScroll enables scrolling and reserves space for scrollbars.
	OverflowScroll
)

// BorderStyle specifies the line style of a quad's border.
type BorderStyle uint8

const (
	// BorderStyleSolid draws a solid line.
	BorderStyleSolid BorderStyle = iota
	// BorderStyleDashed draws a dashed line.
	BorderStyleDashed
	// BorderStyleDotted draws a dotted line.
	BorderStyleDotted
	// BorderStyleNone draws no border line.
	BorderStyleNone
)

// AlignItems controls cross-axis alignment of children within a flex container.
type AlignItems uint8

const (
	// AlignItemsStart aligns items to the start of the cross axis.
	AlignItemsStart AlignItems = iota
	// AlignItemsEnd aligns items to the end of the cross axis.
	AlignItemsEnd
	// AlignItemsFlexStart aligns items to the flex-relative start.
	AlignItemsFlexStart
	// AlignItemsFlexEnd aligns items to the flex-relative end.
	AlignItemsFlexEnd
	// AlignItemsCenter centers items along the cross axis.
	AlignItemsCenter
	// AlignItemsBaseline aligns items along their text baseline.
	AlignItemsBaseline
	// AlignItemsStretch stretches items to fill the cross axis.
	AlignItemsStretch
)

// AlignSelf controls cross-axis alignment for an individual child item.
type AlignSelf = AlignItems

// AlignContent sets the distribution of space between and around lines of content.
type AlignContent uint8

const (
	// AlignContentStart packs items toward the start.
	AlignContentStart AlignContent = iota
	// AlignContentEnd packs items toward the end.
	AlignContentEnd
	// AlignContentFlexStart packs items toward the flex start.
	AlignContentFlexStart
	// AlignContentFlexEnd packs items toward the flex end.
	AlignContentFlexEnd
	// AlignContentCenter centers items within the container.
	AlignContentCenter
	// AlignContentStretch stretches lines to fill available space.
	AlignContentStretch
	// AlignContentSpaceBetween distributes lines evenly with flush ends.
	AlignContentSpaceBetween
	// AlignContentSpaceEvenly distributes lines evenly with equal margins.
	AlignContentSpaceEvenly
	// AlignContentSpaceAround distributes lines with half-size outer margins.
	AlignContentSpaceAround
)

// JustifyContent sets the distribution of space between and around items along the main axis.
type JustifyContent = AlignContent

// FlexDirection specifies the main axis direction in a flexbox container.
type FlexDirection uint8

const (
	// FlexDirectionRow flows children horizontally from left to right.
	FlexDirectionRow FlexDirection = iota
	// FlexDirectionColumn flows children vertically from top to bottom.
	FlexDirectionColumn
	// FlexDirectionRowReverse flows children horizontally from right to left.
	FlexDirectionRowReverse
	// FlexDirectionColumnReverse flows children vertically from bottom to top.
	FlexDirectionColumnReverse
)

// FlexWrap specifies whether flex items wrap onto multiple lines.
type FlexWrap uint8

const (
	// FlexWrapNoWrap forces all items onto a single line.
	FlexWrapNoWrap FlexWrap = iota
	// FlexWrapWrap allows items to wrap onto multiple lines.
	FlexWrapWrap
	// FlexWrapReverse wraps items in the reverse direction.
	FlexWrapReverse
)

// CursorStyle defines the mouse pointer cursor shape.
type CursorStyle uint8

const (
	// CursorDefault is the standard arrow cursor.
	CursorDefault CursorStyle = iota
	// CursorPointer is the hand / pointer cursor.
	CursorPointer
	// CursorText is the I-beam text selection cursor.
	CursorText
	// CursorCrosshair is the crosshair cursor.
	CursorCrosshair
	// CursorNotAllowed is the prohibited action cursor.
	CursorNotAllowed
	// CursorGrab is the open hand cursor.
	CursorGrab
	// CursorGrabbing is the closed hand dragging cursor.
	CursorGrabbing
	// CursorResizeCol indicates a column can be resized horizontally.
	CursorResizeCol
	// CursorResizeRow indicates a row can be resized vertically.
	CursorResizeRow
)

// WhiteSpace controls how whitespace inside an element is handled.
type WhiteSpace uint8

const (
	// WhiteSpaceNormal allows text to wrap normally.
	WhiteSpaceNormal WhiteSpace = iota
	// WhiteSpaceNowrap prevents text from wrapping.
	WhiteSpaceNowrap
)

// TextOverflow controls how overflowing text is truncated.
type TextOverflow uint8

const (
	// TextOverflowClip clips overflowing text at the bounds.
	TextOverflowClip TextOverflow = iota
	// TextOverflowEllipsis truncates overflowing text with an ellipsis.
	TextOverflowEllipsis
)

// TextAlign specifies horizontal alignment of text lines.
type TextAlign uint8

const (
	// TextAlignLeft aligns text to the left.
	TextAlignLeft TextAlign = iota
	// TextAlignCenter centers text horizontally.
	TextAlignCenter
	// TextAlignRight aligns text to the right.
	TextAlignRight
)

// Conversion helpers to layout enums

func (d Display) toLayout() layout.Display {
	switch d {
	case DisplayBlock:
		return layout.DisplayBlock
	case DisplayFlex:
		return layout.DisplayFlex
	case DisplayNone:
		return layout.DisplayNone
	default:
		return layout.DisplayFlex
	}
}

func (p Position) toLayout() layout.Position {
	switch p {
	case PositionRelative:
		return layout.PositionRelative
	case PositionAbsolute:
		return layout.PositionAbsolute
	default:
		return layout.PositionRelative
	}
}

func (o Overflow) toLayout() layout.Overflow {
	switch o {
	case OverflowVisible:
		return layout.OverflowVisible
	case OverflowClip:
		return layout.OverflowClip
	case OverflowHidden:
		return layout.OverflowHidden
	case OverflowScroll:
		return layout.OverflowScroll
	default:
		return layout.OverflowVisible
	}
}

func (f FlexDirection) toLayout() layout.FlexDirection {
	switch f {
	case FlexDirectionRow:
		return layout.FlexRow
	case FlexDirectionColumn:
		return layout.FlexColumn
	case FlexDirectionRowReverse:
		return layout.FlexRowReverse
	case FlexDirectionColumnReverse:
		return layout.FlexColumnReverse
	default:
		return layout.FlexRow
	}
}

func (w FlexWrap) toLayout() layout.FlexWrap {
	switch w {
	case FlexWrapNoWrap:
		return layout.FlexNoWrap
	case FlexWrapWrap:
		return layout.FlexWrapWrap
	case FlexWrapReverse:
		return layout.FlexWrapReverse
	default:
		return layout.FlexNoWrap
	}
}

func (a AlignItems) toLayout() layout.AlignItems {
	switch a {
	case AlignItemsStart:
		return layout.AlignItemsStart
	case AlignItemsEnd:
		return layout.AlignItemsEnd
	case AlignItemsFlexStart:
		return layout.AlignItemsFlexStart
	case AlignItemsFlexEnd:
		return layout.AlignItemsFlexEnd
	case AlignItemsCenter:
		return layout.AlignItemsCenter
	case AlignItemsBaseline:
		return layout.AlignItemsBaseline
	case AlignItemsStretch:
		return layout.AlignItemsStretch
	default:
		return layout.AlignItemsStretch
	}
}

func (a AlignContent) toLayout() layout.AlignContent {
	switch a {
	case AlignContentStart:
		return layout.AlignContentStart
	case AlignContentEnd:
		return layout.AlignContentEnd
	case AlignContentFlexStart:
		return layout.AlignContentFlexStart
	case AlignContentFlexEnd:
		return layout.AlignContentFlexEnd
	case AlignContentCenter:
		return layout.AlignContentCenter
	case AlignContentStretch:
		return layout.AlignContentStretch
	case AlignContentSpaceBetween:
		return layout.AlignContentSpaceBetween
	case AlignContentSpaceEvenly:
		return layout.AlignContentSpaceEvenly
	case AlignContentSpaceAround:
		return layout.AlignContentSpaceAround
	default:
		return layout.AlignContentStretch
	}
}

func (t TextAlign) toLayout() layout.TextAlign {
	switch t {
	case TextAlignLeft:
		return layout.TextAlignLeft
	case TextAlignCenter:
		return layout.TextAlignCenter
	case TextAlignRight:
		return layout.TextAlignRight
	default:
		return layout.TextAlignAuto
	}
}
