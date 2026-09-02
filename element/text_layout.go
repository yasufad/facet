package element

import (
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/text"
)

// TextLayout is a queryable view of a Text element's shaped line, usable
// outside a Frame — in particular during event dispatch, where no Frame
// exists. It wraps text.ShapedLine so that a caller (ui, for a text field's
// caret arithmetic) can query caret geometry from state it stores on its own
// entity, without importing text itself.
//
// The zero value and a TextLayout taken before Text has shaped anything both
// report the line's empty position rather than panicking, since a text field
// may query its layout before the first frame that shapes its content.
type TextLayout struct {
	line *text.ShapedLine
}

// Layout returns a queryable view of this Text element's most recently
// shaped line. Content and style are fixed once RequestLayout runs, so the
// line does not change again until a fresh Text element replaces this one on
// the next frame.
func (t *Text) Layout() TextLayout {
	return TextLayout{line: t.shapedLine}
}

// XForIndex returns the x position, relative to the line's left edge, of the
// caret boundary at the given byte offset. See text.ShapedLine.XForIndex.
func (l TextLayout) XForIndex(byteIndex int) geometry.Pixels {
	if l.line == nil {
		return 0
	}
	return l.line.XForIndex(byteIndex)
}

// IndexForX returns the byte offset, relative to the line's left edge, of
// the caret boundary at the given x position. See text.ShapedLine.IndexForX.
func (l TextLayout) IndexForX(x geometry.Pixels) (int, bool) {
	if l.line == nil {
		return 0, false
	}
	return l.line.IndexForX(x)
}

// ClosestIndexForX returns the byte offset of the caret boundary nearest x.
// See text.ShapedLine.ClosestIndexForX.
func (l TextLayout) ClosestIndexForX(x geometry.Pixels) int {
	if l.line == nil {
		return 0
	}
	return l.line.ClosestIndexForX(x)
}
