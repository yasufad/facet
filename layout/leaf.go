// Ported from Taffy src/compute/leaf.rs (MIT).
//
// Leaf layout: applies padding, border, scrollbar gutter, aspect ratio and
// min/max clamping around a measure function that reports the content size.
package layout

// measureContentSize reports the intrinsic content size of a leaf given the
// known dimensions and available space. Taffy passes this as a closure; here it
// is a function value so the algorithm and tests can supply their own.
type measureContentSize func(known Size[optF32], avail Size[availableSpace]) Size[float32]

// computeLeafLayout lays out a leaf node (a node with no children).
//
// resolveCalc is the calc resolver (nil means no calc support, returning 0).
// measure is the content-size measure function (nil means zero content size).
func computeLeafLayout(
	in LayoutInput,
	style *Style,
	resolveCalc func(val uintptr, basis float32) float32,
	measure measureContentSize,
) LayoutOutput {
	if measure == nil {
		measure = func(Size[optF32], Size[availableSpace]) Size[float32] { return sizeZeroF32 }
	}
	if resolveCalc == nil {
		resolveCalc = func(uintptr, float32) float32 { return 0 }
	}

	known := in.KnownDimensions
	parentSize := in.ParentSize
	avail := in.AvailableSpace
	sizing := in.SizingMode
	runMode := in.RunMode

	// Both horizontal and vertical percentage padding/borders resolve against
	// the container's inline size (width). This is how CSS is specified.
	margin := rectLPAResolveOrZeroOpt(style.marginVal(), parentSize.Width)
	padding := rectLPResolveOrZeroOpt(style.paddingVal(), parentSize.Width)
	border := rectLPResolveOrZeroOpt(style.borderVal(), parentSize.Width)
	paddingBorder := Rect[float32]{
		Left:   padding.Left + border.Left,
		Right:  padding.Right + border.Right,
		Top:    padding.Top + border.Top,
		Bottom: padding.Bottom + border.Bottom,
	}
	pbSum := rectF32SumAxes(paddingBorder)
	var boxSizingAdj Size[float32]
	if style.boxSizingVal() == boxSizingContentBox {
		boxSizingAdj = pbSum
	}

	// Resolve preferred/min/max sizes. In ContentSize mode, size styles are
	// ignored.
	var nodeSize, nodeMinSize, nodeMaxSize Size[optF32]
	var aspectRatio *float32
	switch sizing {
	case sizingContentSize:
		nodeSize = known
		nodeMinSize = sizeNone
		nodeMaxSize = sizeNone
		aspectRatio = nil
	case sizingInherentSize:
		aspectRatio = style.aspectRatioVal()
		styleSize := sizeOptMaybeApplyAspectRatio(
			sizeDimMaybeResolve(style.sizeVal(), parentSize), aspectRatio)
		styleSize = sizeOptMaybeAdd(styleSize, sizeF32ToOpt(boxSizingAdj))
		styleMinSize := sizeOptMaybeApplyAspectRatio(
			sizeLPAMaybeResolve(style.minSizeVal(), parentSize), aspectRatio)
		styleMinSize = sizeOptMaybeAdd(styleMinSize, sizeF32ToOpt(boxSizingAdj))
		styleMaxSize := sizeOptMaybeAdd(
			sizeLPAMaybeResolve(style.maxSizeVal(), parentSize), sizeF32ToOpt(boxSizingAdj))
		nodeSize = sizeOptOr(known, styleSize)
		nodeMinSize = styleMinSize
		nodeMaxSize = styleMaxSize
	}

	// Scrollbar gutters are reserved when overflow is Scroll. The axes are
	// transposed: a node that scrolls vertically needs horizontal space.
	overflow := style.overflowVal()
	scrollbarGutter := Point[float32]{
		X: scrollGutter(overflow.Y, style.scrollbarWidthVal()),
		Y: scrollGutter(overflow.X, style.scrollbarWidthVal()),
	}
	contentBoxInset := paddingBorder
	contentBoxInset.Right += scrollbarGutter.X
	contentBoxInset.Bottom += scrollbarGutter.Y

	hasStylesPreventingCollapseThrough := !style.isBlock() ||
		overflow.X.isScrollContainer() ||
		overflow.Y.isScrollContainer() ||
		style.positionVal() == positionAbsolute ||
		style.containVal().establishesIndependentFormattingContext() ||
		padding.Top > 0 || padding.Bottom > 0 ||
		border.Top > 0 || border.Bottom > 0 ||
		(nodeSize.Height.isSome() && nodeSize.Height.v > 0) ||
		(nodeMinSize.Height.isSome() && nodeMinSize.Height.v > 0)

	// Early return when both dimensions are known and the node can't be
	// collapsed through.
	if runMode == runComputeSize && hasStylesPreventingCollapseThrough {
		if nodeSize.Width.isSome() && nodeSize.Height.isSome() {
			s := Size[float32]{Width: nodeSize.Width.v, Height: nodeSize.Height.v}
			s = sizeOptUnwrapOr(sizeOptMaybeClamp(sizeF32ToOpt(s), nodeMinSize, nodeMaxSize), s)
			s = sizeF32Max(s, pbSum)
			return layoutOutputFromOuterSize(s)
		}
	}

	// Compute available space for the measure function.
	availWidth := fromOptF32(known.Width)
	if !known.Width.isSome() {
		availWidth = avail.Width
	}
	availWidth = availMaybeSubOpt(availWidth, some(rectF32HorizontalAxisSum(margin)))
	availWidth = availWidth.maybeSet(known.Width).maybeSet(nodeSize.Width)
	availWidth = availWidth.mapDefiniteValue(func(size float32) float32 {
		return f32MaybeClamp(size, nodeMinSize.Width, nodeMaxSize.Width) - rectF32HorizontalAxisSum(contentBoxInset)
	})
	availHeight := fromOptF32(known.Height)
	if !known.Height.isSome() {
		availHeight = avail.Height
	}
	availHeight = availMaybeSubOpt(availHeight, some(rectF32VerticalAxisSum(margin)))
	availHeight = availHeight.maybeSet(known.Height).maybeSet(nodeSize.Height)
	availHeight = availHeight.mapDefiniteValue(func(size float32) float32 {
		return f32MaybeClamp(size, nodeMinSize.Height, nodeMaxSize.Height) - rectF32VerticalAxisSum(contentBoxInset)
	})
	measureAvail := Size[availableSpace]{Width: availWidth, Height: availHeight}

	// Measure.
	var measureKnown Size[optF32]
	if runMode == runComputeSize {
		measureKnown = known
	}
	measuredSize := measure(measureKnown, measureAvail)

	// Clamp and combine.
	clamped := sizeOptOr(known, nodeSize)
	clampedF32 := sizeOptUnwrapOr(clamped, sizeF32Add(measuredSize, rectF32SumAxes(contentBoxInset)))
	clampedF32 = sizeOptUnwrapOr(sizeOptMaybeClamp(sizeF32ToOpt(clampedF32), nodeMinSize, nodeMaxSize), clampedF32)
	heightFloor := float32(0)
	if aspectRatio != nil {
		heightFloor = clampedF32.Width / *aspectRatio
	}
	size := Size[float32]{
		Width:  clampedF32.Width,
		Height: f32Max(clampedF32.Height, heightFloor),
	}
	size = sizeF32Max(size, pbSum)

	// Scrollable overflow rect.
	isScrollContainer := overflow.X.isScrollContainer() || overflow.Y.isScrollContainer()
	isRtl := style.directionVal().isRtl()
	startPadding := padding.Left
	if isRtl {
		startPadding = padding.Right
	}
	endPadding := padding.Right
	if isRtl {
		endPadding = padding.Left
	}
	endPaddingAdd := float32(0)
	if isScrollContainer {
		endPaddingAdd = endPadding
	}
	paddingBottomAdd := float32(0)
	if isScrollContainer {
		paddingBottomAdd = padding.Bottom
	}
	scrollableOverflow := Rect[float32]{
		Left:   0,
		Right:  startPadding + measuredSize.Width + endPaddingAdd,
		Top:    0,
		Bottom: padding.Top + measuredSize.Height + paddingBottomAdd,
	}

	marginsCanCollapseThrough := !hasStylesPreventingCollapseThrough &&
		size.Height == 0 && measuredSize.Height == 0

	return LayoutOutput{
		Size:                      size,
		ScrollableOverflowRect:    scrollableOverflow,
		Baselines:                 baselinesNone,
		MarginsCanCollapseThrough: marginsCanCollapseThrough,
	}
}

func scrollGutter(o overflow, scrollbarWidth float32) float32 {
	if o == overflowScroll {
		return scrollbarWidth
	}
	return 0
}

// sizeF32ToOpt lifts a Size[float32] into a Size[optF32].
func sizeF32ToOpt(s Size[float32]) Size[optF32] {
	return Size[optF32]{Width: some(s.Width), Height: some(s.Height)}
}

// sizeF32Add adds two Size[float32]s component-wise.
func sizeF32Add(a, b Size[float32]) Size[float32] {
	return Size[float32]{Width: a.Width + b.Width, Height: a.Height + b.Height}
}
