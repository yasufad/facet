// Ported from Taffy src/compute/common/scrollable_overflow.rs (MIT).
package layout

// computeScrollableOverflowContribution determines the rectangle that a given
// node contributes to its parent's scrollable overflow rectangle.
func computeScrollableOverflowContribution(
	location Point[float32],
	size Size[float32],
	scrollableOverflowRect Rect[float32],
	overflow Point[overflow],
	contain contain,
	parentIsScrollContainer bool,
) Rect[float32] {
	isScrollContainer := overflow.X.isScrollContainer() || overflow.Y.isScrollContainer()
	overflowIsContained := contain.containsScrollableOverflow()
	propagates := Point[bool]{
		X: !isScrollContainer && !overflowIsContained && overflow.X == overflowVisible,
		Y: !isScrollContainer && !overflowIsContained && overflow.Y == overflowVisible,
	}
	endExtent := Size[float32]{
		Width:  size.Width,
		Height: size.Height,
	}
	if propagates.X {
		endExtent.Width = f32Max(size.Width, scrollableOverflowRect.Right)
	}
	if propagates.Y {
		endExtent.Height = f32Max(size.Height, scrollableOverflowRect.Bottom)
	}
	if endExtent.Width <= 0 || endExtent.Height <= 0 {
		return rectZeroF32
	}
	startExtent := Point[float32]{X: 0, Y: 0}
	if propagates.X {
		startExtent.X = f32Min(0, scrollableOverflowRect.Left)
	}
	if propagates.Y {
		startExtent.Y = f32Min(0, scrollableOverflowRect.Top)
	}
	contribution := Rect[float32]{
		Left:   location.X + startExtent.X,
		Right:  location.X + endExtent.Width,
		Top:    location.Y + startExtent.Y,
		Bottom: location.Y + endExtent.Height,
	}
	isWhollyUnreachable := contribution.Right <= 0 || contribution.Bottom <= 0
	if parentIsScrollContainer && isWhollyUnreachable {
		return rectZeroF32
	}
	return contribution
}
