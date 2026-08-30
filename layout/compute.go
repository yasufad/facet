// Ported from Taffy src/compute/mod.rs (MIT).
//
// The root, cached, hidden and round entry points. The block-layout branch of
// compute_root_layout is included because the tree dispatches display:block to
// the block algorithm; the block algorithm itself is a separate file.
package layout

// computeRootLayout lays out the root node of a tree.
func computeRootLayout(t LayoutTree, root NodeID, avail Size[availableSpace]) {
	known := sizeNone

	style := t.CoreContainerStyle(root)
	if style.isBlock() {
		aspectRatio := style.aspectRatioVal()
		parentSize := sizeAvailIntoOptions(avail)
		margin := rectLPAResolveOrZeroSize(style.marginVal(), parentSize)
		padding := rectLPResolveOrZeroSize(style.paddingVal(), parentSize)
		border := rectLPResolveOrZeroSize(style.borderVal(), parentSize)
		paddingBorderSize := rectF32SumAxes(rectF32AddPaddingBorder(padding, border))
		var boxSizingAdj Size[float32]
		if style.boxSizingVal() == boxSizingContentBox {
			boxSizingAdj = paddingBorderSize
		} else {
			boxSizingAdj = sizeZeroF32
		}
		minSize := sizeOptMaybeAdd(
			sizeOptMaybeApplyAspectRatio(sizeLPAMaybeResolve(style.minSizeVal(), parentSize), aspectRatio),
			sizeF32ToOpt(boxSizingAdj))
		maxSize := sizeOptMaybeAdd(
			sizeOptMaybeApplyAspectRatio(sizeLPAMaybeResolve(style.maxSizeVal(), parentSize), aspectRatio),
			sizeF32ToOpt(boxSizingAdj))
		clampedStyleSize := sizeOptMaybeClamp(
			sizeOptMaybeAdd(
				sizeOptMaybeApplyAspectRatio(sizeDimMaybeResolve(style.sizeVal(), parentSize), aspectRatio),
				sizeF32ToOpt(boxSizingAdj)),
			minSize, maxSize)
		minMaxDefinite := sizeZipMap(minSize, maxSize, func(lo, hi optF32) optF32 {
			if lo.isSome() && hi.isSome() && hi.v <= lo.v {
				return lo
			}
			return none()
		})
		availBased := Size[optF32]{
			Width:  optSubF32(avail.Width.intoOption(), rectF32HorizontalAxisSum(margin)),
			Height: none(),
		}
		styled := sizeOptOr(known, minMaxDefinite)
		styled = sizeOptOr(styled, clampedStyleSize)
		styled = sizeOptOr(styled, availBased)
		styled = sizeOptMaybeMax(styled, sizeF32ToOpt(paddingBorderSize))
		known = styled
	}

	output := performChildLayout(t, root, known, sizeAvailIntoOptions(avail), avail, sizingInherentSize, lineBoolFalse)
	style = t.CoreContainerStyle(root)
	padding := rectLPResolveOrZeroOpt(style.paddingVal(), avail.Width.intoOption())
	border := rectLPResolveOrZeroOpt(style.borderVal(), avail.Width.intoOption())
	margin := rectLPAResolveOrZeroOpt(style.marginVal(), avail.Width.intoOption())
	scrollbarSize := Size[float32]{
		Width:  scrollWidthFor(style.overflowVal().Y, style.scrollbarWidthVal()),
		Height: scrollWidthFor(style.overflowVal().X, style.scrollbarWidthVal()),
	}
	var locX float32
	if style.directionVal().isRtl() {
		if w := avail.Width.intoOption(); w.isSome() {
			locX = w.v - output.Size.Width
		}
	}
	t.SetUnroundedLayout(root, Layout{
		Order:                  0,
		Location:               Point[float32]{X: locX, Y: 0},
		Size:                   output.Size,
		ScrollableOverflowRect: output.ScrollableOverflowRect,
		ScrollbarSize:          scrollbarSize,
		Padding:                padding,
		Border:                 border,
		Margin:                 margin,
	})
}

func scrollWidthFor(o overflow, w float32) float32 {
	if o == overflowScroll {
		return w
	}
	return 0
}

// rectF32AddPaddingBorder adds padding and border rects component-wise.
func rectF32AddPaddingBorder(padding, border Rect[float32]) Rect[float32] {
	return Rect[float32]{
		Left:   padding.Left + border.Left,
		Right:  padding.Right + border.Right,
		Top:    padding.Top + border.Top,
		Bottom: padding.Bottom + border.Bottom,
	}
}

// computeCachedLayout wraps a layout computation in a cache lookup/store.
func computeCachedLayout(
	t LayoutTree,
	node NodeID,
	in LayoutInput,
	computeUncached func(LayoutTree, NodeID, LayoutInput) LayoutOutput,
) LayoutOutput {
	if cached, ok := t.CacheGet(node, &in); ok {
		return cached
	}
	out := computeUncached(t, node, in)
	t.CacheStore(node, &in, out)
	return out
}

// computeHiddenLayout marks a node and its descendants as hidden (zero size at
// the origin).
func computeHiddenLayout(t LayoutTree, node NodeID) LayoutOutput {
	t.CacheClear(node)
	t.SetUnroundedLayout(node, newLayoutWithOrder(0))
	for i := 0; i < t.ChildCount(node); i++ {
		child := t.ChildID(node, i)
		t.ComputeChildLayout(child, layoutInputHidden)
	}
	return layoutOutputHidden
}

// roundLayout rounds a tree of unrounded float layouts to integer pixels.
func roundLayout(t RoundTree, node NodeID) {
	roundLayoutInner(t, node, 0, 0)
}

func roundLayoutInner(t RoundTree, node NodeID, cumulativeX, cumulativeY float32) {
	unrounded := t.GetUnroundedLayout(node)
	layout := unrounded
	cumulativeX += unrounded.Location.X
	cumulativeY += unrounded.Location.Y
	layout.Location.X = round(unrounded.Location.X)
	layout.Location.Y = round(unrounded.Location.Y)
	layout.Size.Width = round(cumulativeX+unrounded.Size.Width) - round(cumulativeX)
	layout.Size.Height = round(cumulativeY+unrounded.Size.Height) - round(cumulativeY)
	layout.ScrollbarSize.Width = round(unrounded.ScrollbarSize.Width)
	layout.ScrollbarSize.Height = round(unrounded.ScrollbarSize.Height)
	layout.Border.Left = round(cumulativeX+unrounded.Border.Left) - round(cumulativeX)
	layout.Border.Right = round(cumulativeX+unrounded.Size.Width) - round(cumulativeX+unrounded.Size.Width-unrounded.Border.Right)
	layout.Border.Top = round(cumulativeY+unrounded.Border.Top) - round(cumulativeY)
	layout.Border.Bottom = round(cumulativeY+unrounded.Size.Height) - round(cumulativeY+unrounded.Size.Height-unrounded.Border.Bottom)
	layout.Padding.Left = round(cumulativeX+unrounded.Padding.Left) - round(cumulativeX)
	layout.Padding.Right = round(cumulativeX+unrounded.Size.Width) - round(cumulativeX+unrounded.Size.Width-unrounded.Padding.Right)
	layout.Padding.Top = round(cumulativeY+unrounded.Padding.Top) - round(cumulativeY)
	layout.Padding.Bottom = round(cumulativeY+unrounded.Size.Height) - round(cumulativeY+unrounded.Size.Height-unrounded.Padding.Bottom)
	layout.ScrollableOverflowRect.Left = round(cumulativeX+unrounded.ScrollableOverflowRect.Left) - round(cumulativeX)
	layout.ScrollableOverflowRect.Right = round(cumulativeX+unrounded.ScrollableOverflowRect.Right) - round(cumulativeX)
	layout.ScrollableOverflowRect.Top = round(cumulativeY+unrounded.ScrollableOverflowRect.Top) - round(cumulativeY)
	layout.ScrollableOverflowRect.Bottom = round(cumulativeY+unrounded.ScrollableOverflowRect.Bottom) - round(cumulativeY)
	t.SetFinalLayout(node, layout)
	for i := 0; i < t.ChildCount(node); i++ {
		roundLayoutInner(t, t.ChildID(node, i), cumulativeX, cumulativeY)
	}
}
