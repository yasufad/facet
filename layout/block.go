// Ported from Taffy src/compute/block.rs (MIT).
//
// The block layout algorithm is not the focus of this port (the assignment is
// the flexbox solver), but the tree dispatches display:block and display:flow-root
// containers to it, and the root-layout path uses block-style sizing for block
// roots. This is a minimal implementation that handles the common cases the
// ported flexbox tests exercise: a block container stacks its children
// vertically, sizing its content height to the sum of child heights.
package layout

// computeBlockLayout lays out a block container.
func computeBlockLayout(t layoutTree, node NodeID, in LayoutInput) LayoutOutput {
	style := t.coreContainerStyle(node)
	known := in.KnownDimensions
	parentSize := in.ParentSize
	avail := in.AvailableSpace

	margin := rectLPAResolveOrZeroOpt(style.marginVal(), parentSize.Width)
	padding := rectLPResolveOrZeroOpt(style.paddingVal(), parentSize.Width)
	border := rectLPResolveOrZeroOpt(style.borderVal(), parentSize.Width)
	paddingBorder := rectF32Add(padding, border)
	pbSum := rectF32SumAxes(paddingBorder)

	var boxSizingAdj Size[float32]
	if style.boxSizingVal() == boxSizingContentBox {
		boxSizingAdj = pbSum
	}

	aspectRatio := style.aspectRatioVal()
	minSize := sizeOptMaybeAdd(
		sizeOptMaybeApplyAspectRatio(sizeLPAMaybeResolve(style.minSizeVal(), parentSize), aspectRatio),
		sizeF32ToOpt(boxSizingAdj))
	maxSize := sizeOptMaybeAdd(
		sizeOptMaybeApplyAspectRatio(sizeLPAMaybeResolve(style.maxSizeVal(), parentSize), aspectRatio),
		sizeF32ToOpt(boxSizingAdj))

	var clampedStyleSize Size[optF32]
	if in.SizingMode == sizingInherentSize {
		clampedStyleSize = sizeOptMaybeClamp(
			sizeOptMaybeAdd(
				sizeOptMaybeApplyAspectRatio(sizeDimMaybeResolve(style.sizeVal(), parentSize), aspectRatio),
				sizeF32ToOpt(boxSizingAdj)),
			minSize, maxSize)
	}

	minMaxDefinite := sizeZipMap(minSize, maxSize, func(lo, hi optF32) optF32 {
		if lo.isSome() && hi.isSome() && hi.v <= lo.v {
			return lo
		}
		return none()
	})

	styledKnown := sizeOptOr(known, minMaxDefinite)
	styledKnown = sizeOptOr(styledKnown, clampedStyleSize)
	styledKnown = sizeOptMaybeMax(styledKnown, sizeF32ToOpt(pbSum))

	// Short-circuit for ComputeSize when both dimensions are known.
	if in.RunMode == runComputeSize {
		if styledKnown.Width.isSome() && styledKnown.Height.isSome() {
			return layoutOutputFromOuterSize(Size[float32]{Width: styledKnown.Width.v, Height: styledKnown.Height.v})
		}
	}

	// Determine available space for children.
	innerWidth := styledKnown.Width
	if !innerWidth.isSome() {
		w := avail.Width.maybeSubF32(rectF32HorizontalAxisSum(margin))
		w = w.maybeSubF32(rectF32HorizontalAxisSum(paddingBorder))
		innerWidth = w.intoOption()
	}
	// Inner height is definite only when the container's height is both
	// known and definite (KnownDimensionsAreDefinite). A flex-grown height
	// with an auto basis is not definite for percentage resolution.
	innerHeight := none()
	if styledKnown.Height.isSome() && in.KnownDimensionsAreDefinite.Height {
		innerHeight = optSubF32(styledKnown.Height, pbSum.Height)
	}

	childAvail := Size[AvailableSpace]{
		Width:  fromOptF32(innerWidth),
		Height: avail.Height.maybeSubF32(rectF32VerticalAxisSum(margin)).maybeSubF32(rectF32VerticalAxisSum(paddingBorder)),
	}

	// Lay out children stacked vertically.
	childCount := t.childCount(node)
	totalHeight := float32(0)
	maxWidth := float32(0)
	for i := 0; i < childCount; i++ {
		child := t.childID(node, i)
		childStyle := t.coreContainerStyle(child)
		if childStyle.boxGenerationMode() == boxGenNone {
			t.setUnroundedLayout(child, newLayoutWithOrder(uint32(i)))
			t.computeChildLayout(child, layoutInputHidden)
			continue
		}
		if childStyle.positionVal() == positionAbsolute {
			continue
		}
		// Block children stretch to fill the container width when their
		// width is auto. Children with an explicit width use that width.
		childKnown := sizeNone
		if childStyle.sizeVal().Width.isAuto() {
			childKnown.Width = innerWidth
		}
		childOut := performChildLayout(t, child,
			childKnown,
			Size[optF32]{Width: innerWidth, Height: innerHeight},
			childAvail,
			sizingInherentSize, lineBoolFalse)
		childLayout := t.coreContainerStyle(child)
		_ = childLayout
		// Set the child's position.
		cl := newLayoutWithOrder(uint32(i))
		cl.Location.Y = totalHeight
		cl.Size = childOut.Size
		// In RTL, position children from the right edge.
		if style.directionVal().isRtl() && innerWidth.isSome() {
			cl.Location.X = innerWidth.v - childOut.Size.Width
		}
		t.setUnroundedLayout(child, cl)
		totalHeight += childOut.Size.Height
		if childOut.Size.Width > maxWidth {
			maxWidth = childOut.Size.Width
		}
	}

	// Determine the container's size.
	width := styledKnown.Width
	if !width.isSome() {
		width = some(f32Max(maxWidth, pbSum.Width))
	}
	height := styledKnown.Height
	if !height.isSome() {
		height = some(f32Max(totalHeight, pbSum.Height))
	}
	size := Size[float32]{Width: width.v, Height: height.v}
	size = sizeOptUnwrapOr(sizeOptMaybeClamp(sizeF32ToOpt(size), minSize, maxSize), size)
	size = sizeF32Max(size, pbSum)

	return layoutOutputFromOuterSize(size)
}
