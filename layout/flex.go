// Ported from Taffy src/style/flex.rs (MIT).
package layout

// FlexDirection controls which absolute axis is the flexbox main axis.
type FlexDirection uint8

const (
	FlexRow FlexDirection = iota
	FlexColumn
	FlexRowReverse
	FlexColumnReverse
)

type flexDirection = FlexDirection

// isRow reports whether the direction is Row or RowReverse.
func (d FlexDirection) isRow() bool { return d == FlexRow || d == FlexRowReverse }

// isColumn reports whether the direction is Column or ColumnReverse.
func (d FlexDirection) isColumn() bool { return d == FlexColumn || d == FlexColumnReverse }

// isReverse reports whether the direction is RowReverse or ColumnReverse.
func (d FlexDirection) isReverse() bool { return d == FlexRowReverse || d == FlexColumnReverse }

// mainAxis returns the absolute axis of the main axis.
func (d FlexDirection) mainAxis() absoluteAxis {
	if d.isRow() {
		return absoluteHorizontal
	}
	return absoluteVertical
}

// crossAxis returns the absolute axis of the cross axis.
func (d FlexDirection) crossAxis() absoluteAxis {
	if d.isRow() {
		return absoluteVertical
	}
	return absoluteHorizontal
}

// FlexWrap controls whether flex items wrap onto multiple lines.
type FlexWrap uint8

const (
	FlexNoWrap FlexWrap = iota
	FlexWrapWrap
	FlexWrapReverse
)

var FlexWrapVal = FlexWrapWrap

type flexWrap = FlexWrap

// isMultiLine reports whether items wrap (any value other than NoWrap).
func (w FlexWrap) isMultiLine() bool { return w != FlexNoWrap }

// isReverse reports whether lines stack opposite to the cross axis.
func (w FlexWrap) isReverse() bool { return w == FlexWrapReverse }
