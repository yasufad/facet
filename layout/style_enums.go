// Ported from Taffy src/style/mod.rs and src/style/block.rs (MIT).
//
// The non-alignment, non-flex style enums: display, position, box-sizing,
// overflow, direction, containment and text-align. Grid-only display variants
// are absent.
package layout

// display sets the layout used for the children of a node.
type display uint8

const (
	displayBlock display = iota
	displayFlowRoot
	displayFlex
	displayNone
)

// displayDefault is the default display mode (flex, matching Taffy's flexbox
// feature being enabled).
var displayDefault = displayFlex

// boxGenerationMode is the abstracted display property: Normal or None.
type boxGenerationMode uint8

const (
	boxGenNormal boxGenerationMode = iota
	boxGenNone
)

// position is the CSS positioning strategy.
type position uint8

const (
	positionRelative position = iota
	positionAbsolute
)

// boxSizing controls whether size styles apply to the content or border box.
type boxSizing uint8

const (
	boxSizingBorderBox boxSizing = iota
	boxSizingContentBox
)

// overflow controls how children overflowing their container affect layout.
type overflow uint8

const (
	overflowVisible overflow = iota
	overflowClip
	overflowHidden
	overflowScroll
)

// isScrollContainer reports whether the overflow mode contains its contents.
func (o overflow) isScrollContainer() bool {
	return o == overflowHidden || o == overflowScroll
}

// maybeIntoAutomaticMinSize returns Some(0) when the overflow mode forces the
// automatic minimum size of a flex/grid item to 0, else none.
func (o overflow) maybeIntoAutomaticMinSize() optF32 {
	if o.isScrollContainer() {
		return some(0)
	}
	return none()
}

// direction sets the text direction.
type direction uint8

const (
	directionLtr direction = iota
	directionRtl
)

// isRtl reports whether the direction is right-to-left.
func (d direction) isRtl() bool { return d == directionRtl }

// contain is the layout-affecting part of the CSS contain property.
type contain uint8

const (
	containNone    contain = 0
	containLayout  contain = 1 << 0
	containPaint   contain = 1 << 1
	containContent         = containLayout | containPaint
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

// textAlign controls inline-axis alignment of block children.
type textAlign uint8

const (
	textAlignAuto textAlign = iota
	textAlignStart
	textAlignEnd
	textAlignLeft
	textAlignRight
	textAlignCenter
	textAlignJustify
)
