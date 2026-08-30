// Ported from Taffy src/style/mod.rs and src/style/block.rs (MIT).
//
// The non-alignment, non-flex style enums: display, position, box-sizing,
// overflow, direction, containment and text-align. Grid-only display variants
// are absent.
package layout

// Display sets the layout used for the children of a node.
type Display uint8

const (
	DisplayBlock Display = iota
	DisplayFlowRoot
	DisplayFlex
	DisplayNone
)

type display = Display

const (
	displayBlock    = DisplayBlock
	displayFlowRoot = DisplayFlowRoot
	displayFlex     = DisplayFlex
	displayNone     = DisplayNone
)

// displayDefault is the default display mode (flex, matching Taffy's flexbox
// feature being enabled).
var displayDefault = DisplayFlex

// boxGenerationMode is the abstracted display property: Normal or None.
type boxGenerationMode uint8

const (
	boxGenNormal boxGenerationMode = iota
	boxGenNone
)

// Position is the CSS positioning strategy.
type Position uint8

const (
	PositionRelative Position = iota
	PositionAbsolute
)

type position = Position

const (
	positionRelative = PositionRelative
	positionAbsolute = PositionAbsolute
)

// BoxSizing controls whether size styles apply to the content or border box.
type BoxSizing uint8

const (
	BoxSizingBorderBox BoxSizing = iota
	BoxSizingContentBox
)

type boxSizing = BoxSizing

const (
	boxSizingBorderBox  = BoxSizingBorderBox
	boxSizingContentBox = BoxSizingContentBox
)

// Overflow controls how children overflowing their container affect layout.
type Overflow uint8

const (
	OverflowVisible Overflow = iota
	OverflowClip
	OverflowHidden
	OverflowScroll
)

type overflow = Overflow

const (
	overflowVisible = OverflowVisible
	overflowClip    = OverflowClip
	overflowHidden  = OverflowHidden
	overflowScroll  = OverflowScroll
)

// isScrollContainer reports whether the overflow mode contains its contents.
func (o Overflow) isScrollContainer() bool {
	return o == OverflowHidden || o == OverflowScroll
}

// maybeIntoAutomaticMinSize returns Some(0) when the overflow mode forces the
// automatic minimum size of a flex/grid item to 0, else none.
func (o overflow) maybeIntoAutomaticMinSize() optF32 {
	if o.isScrollContainer() {
		return some(0)
	}
	return none()
}

// Direction sets the text direction.
type Direction uint8

const (
	DirectionLtr Direction = iota
	DirectionRtl
)

type direction = Direction

const (
	directionLtr = DirectionLtr
	directionRtl = DirectionRtl
)

// isRtl reports whether the direction is right-to-left.
func (d direction) isRtl() bool { return d == DirectionRtl }

// Contain is the layout-affecting part of the CSS contain property.
type Contain uint8

const (
	ContainNone    Contain = 0
	ContainLayout  Contain = 1 << 0
	ContainPaint   Contain = 1 << 1
	ContainContent         = ContainLayout | ContainPaint
)

type contain = Contain

const (
	containNone    = ContainNone
	containLayout  = ContainLayout
	containPaint   = ContainPaint
	containContent = ContainContent
)

// contains reports whether self contains all of other's containment types.
func (c contain) contains(other contain) bool { return c&other == other }

// intersects reports whether self shares any containment type with other.
func (c contain) intersects(other contain) bool { return c&other != 0 }

// union returns the union of self and other.
func (c contain) union(other contain) contain { return c | other }

// establishesIndependentFormattingContext reports whether layout or paint
// containment is set.
func (c contain) establishesIndependentFormattingContext() bool {
	return c.intersects(containLayout | containPaint)
}

// suppressesBaseline reports whether layout containment suppresses the baseline.
func (c contain) suppressesBaseline() bool { return c.contains(containLayout) }

// containsScrollableOverflow reports whether containment treats overflowing
// content as not contributing to an ancestor's scrollable overflow.
func (c contain) containsScrollableOverflow() bool {
	return c.intersects(containLayout | containPaint)
}

// TextAlign controls inline-axis alignment of block children.
type TextAlign uint8

const (
	TextAlignAuto TextAlign = iota
	TextAlignStart
	TextAlignEnd
	TextAlignLeft
	TextAlignRight
	TextAlignCentre
	TextAlignJustify
)

type textAlign = TextAlign

const (
	textAlignAuto    = TextAlignAuto
	textAlignStart   = TextAlignStart
	textAlignEnd     = TextAlignEnd
	textAlignLeft    = TextAlignLeft
	textAlignRight   = TextAlignRight
	textAlignCentre  = TextAlignCentre
	textAlignJustify = TextAlignJustify
)
