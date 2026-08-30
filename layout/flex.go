// Ported from Taffy src/style/flex.rs (MIT).
package layout

// flexDirection controls which absolute axis is the flexbox main axis.
type flexDirection uint8

const (
	FlexRow flexDirection = iota
	FlexColumn
	FlexRowReverse
	FlexColumnReverse
)

// isRow reports whether the direction is Row or RowReverse.
func (d flexDirection) isRow() bool { return d == FlexRow || d == FlexRowReverse }

// isColumn reports whether the direction is Column or ColumnReverse.
func (d flexDirection) isColumn() bool { return d == FlexColumn || d == FlexColumnReverse }

// isReverse reports whether the direction is RowReverse or ColumnReverse.
func (d flexDirection) isReverse() bool { return d == FlexRowReverse || d == FlexColumnReverse }

// mainAxis returns the absolute axis of the main axis.
func (d flexDirection) mainAxis() absoluteAxis {
	if d.isRow() {
		return absoluteHorizontal
	}
	return absoluteVertical
}

// crossAxis returns the absolute axis of the cross axis.
func (d flexDirection) crossAxis() absoluteAxis {
	if d.isRow() {
		return absoluteVertical
	}
	return absoluteHorizontal
}

// flexWrap controls whether flex items wrap onto multiple lines.
type flexWrap uint8

const (
	FlexNoWrap flexWrap = iota
	FlexWrap
	FlexWrapReverse
)

// isMultiLine reports whether items wrap (any value other than NoWrap).
func (w flexWrap) isMultiLine() bool { return w != FlexNoWrap }

// isReverse reports whether lines stack opposite to the cross axis.
func (w flexWrap) isReverse() bool { return w == FlexWrapReverse }
