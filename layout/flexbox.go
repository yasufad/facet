// Ported from Taffy src/compute/flexbox.rs (MIT).
//
// The complete Taffy flexbox algorithm, preserving ordering and edge cases.
// The flexbox_balance feature (flex-wrap: balance) is not ported.
package layout

// flexItem is the intermediate result of a flexbox calculation for a single item.
type flexItem struct {
	node  NodeID
	order uint32

	size        Size[optF32] // base size
	sizeStyle   Size[Dimension]
	minSize     Size[optF32]
	maxSize     Size[optF32]
	aspectRatio *float32
	alignSelf   AlignItems

	overflow            Point[overflow]
	contain             contain
	scrollbarWidth      float32
	flexShrink          float32
	flexGrow            float32
	flexBasisIsDefinite bool

	resolvedMinimumMainSize float32

	inset        Rect[optF32]
	margin       Rect[float32]
	marginIsAuto Rect[bool]
	padding      Rect[float32]
	border       Rect[float32]

	flexBasis      float32
	innerFlexBasis float32
	violation      float32
	frozen         bool

	contentFlexFraction float32

	hypotheticalInnerSize Size[float32]
	hypotheticalOuterSize Size[float32]
	targetSize            Size[float32]
	outerTargetSize       Size[float32]

	baseline float32

	offsetMain  float32
	offsetCross float32
}

// isScrollContainer reports whether the item is a scroll container.
func (i *flexItem) isScrollContainer() bool {
	return i.overflow.X.isScrollContainer() || i.overflow.Y.isScrollContainer()
}

// participatesInBaselineAlignment reports whether the item participates in
// baseline alignment.
func (i *flexItem) participatesInBaselineAlignment(dir flexDirection) bool {
	return i.alignSelf.Keyword == alignItemsBaseline &&
		!rectBoolCrossStart(i.marginIsAuto, dir) &&
		!rectBoolCrossEnd(i.marginIsAuto, dir)
}

// flexLine is a line of flex items.
type flexLine struct {
	items       []*flexItem
	crossSize   float32
	offsetCross float32
}

// algoConstants holds values cached during the flexbox algorithm.
type algoConstants struct {
	dir             flexDirection
	layoutDirection direction
	isRow           bool
	isColumn        bool
	isWrap          bool
	isWrapReverse   bool

	minSize           Size[optF32]
	maxSize           Size[optF32]
	margin            Rect[float32]
	border            Rect[float32]
	contentBoxInset   Rect[float32]
	scrollbarGutter   Point[float32]
	isScrollContainer bool
	gap               Size[float32]
	alignItems        AlignItems
	alignContent      AlignContent
	justifyContent    *AlignContent

	nodeOuterSize                     Size[optF32]
	nodeInnerSize                     Size[optF32]
	knownMainSizeIsDefinite           bool
	hasDefiniteMainSize               bool
	hasDefiniteCrossSize              bool
	crossAxisAvailableSpaceIsDefinite bool

	containerSize      Size[float32]
	innerContainerSize Size[float32]
}

// computeFlexboxLayout computes the layout of a box according to the flexbox algorithm.
func computeFlexboxLayout(t flexboxTree, node NodeID, in LayoutInput) LayoutOutput {
	known := in.KnownDimensions
	parentSize := in.ParentSize
	runMode := in.RunMode
	style := t.flexboxContainerStyle(node)

	contain := style.containVal()
	aspectRatio := style.aspectRatioVal()
	padding := rectLPResolveOrZeroOpt(style.paddingVal(), parentSize.Width)
	border := rectLPResolveOrZeroOpt(style.borderVal(), parentSize.Width)
	paddingBorderSum := sizeF32Add(rectF32SumAxes(padding), rectF32SumAxes(border))
	var boxSizingAdj Size[float32]
	if style.boxSizingVal() == boxSizingContentBox {
		boxSizingAdj = paddingBorderSum
	}

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
	_ = minMaxDefinite // used below

	styledKnown := sizeOptOr(known, minMaxDefinite)
	styledKnown = sizeOptOr(styledKnown, clampedStyleSize)
	styledKnown = sizeOptMaybeMax(styledKnown, sizeF32ToOpt(paddingBorderSum))

	// Short-circuit for ComputeSize.
	if runMode == runComputeSize {
		if styledKnown.Width.isSome() && styledKnown.Height.isSome() {
			return layoutOutputFromOuterSize(Size[float32]{Width: styledKnown.Width.v, Height: styledKnown.Height.v})
		}
		if in.Axis == requestedHorizontal {
			if styledKnown.Width.isSome() {
				return layoutOutputFromOuterSize(Size[float32]{Width: styledKnown.Width.v, Height: 0})
			}
		}
	}

	knownDimensionsAreDefinite := sizeZipMap(in.KnownDimensionsAreDefinite, known,
		func(isDefinite bool, kd optF32) bool { return isDefinite || !kd.isSome() })

	inputCopy := in
	inputCopy.KnownDimensions = styledKnown
	inputCopy.KnownDimensionsAreDefinite = knownDimensionsAreDefinite

	output := computePreliminary(t, node, inputCopy)

	if contain.suppressesBaseline() {
		output.Baselines = baselinesNone
	}

	return output
}

// computePreliminary computes a preliminary size for an item.
func computePreliminary(t flexboxTree, node NodeID, in LayoutInput) LayoutOutput {
	known := in.KnownDimensions
	parentSize := in.ParentSize
	avail := in.AvailableSpace
	runMode := in.RunMode

	constants := computeConstants(t, t.flexboxContainerStyle(node), known, in.KnownDimensionsAreDefinite, parentSize, avail)

	// 9.1. Generate anonymous flex items.
	flexItems := generateAnonymousFlexItems(t, node, &constants)

	// 9.2. Determine the available main and cross space.
	availSpace := determineAvailableSpace(known, avail, &constants)

	// 3. Determine the flex base size and hypothetical main size.
	determineFlexBaseSize(t, &constants, availSpace, flexItems)

	// 5. Collect flex items into flex lines.
	flexLines := collectFlexLines(&constants, availSpace, flexItems)

	// Determine the container's main size if not already known.
	if mainSize := sizeMain(constants.nodeInnerSize, constants.dir); mainSize.isSome() {
		innerMain := mainSize.v
		outerMain := innerMain + rectF32MainAxisSum(constants.contentBoxInset, constants.dir)
		sizeSetMain(&constants.innerContainerSize, constants.dir, innerMain)
		sizeSetMain(&constants.containerSize, constants.dir, outerMain)
	} else {
		determineContainerMainSize(t, availSpace, flexLines, &constants)
		sizeSetMain(&constants.nodeInnerSize, constants.dir, some(sizeMain(constants.innerContainerSize, constants.dir)))
		sizeSetMain(&constants.nodeOuterSize, constants.dir, some(sizeMain(constants.containerSize, constants.dir)))
		// Re-resolve percentage gaps.
		style := t.flexboxContainerStyle(node)
		innerMain := sizeMain(constants.innerContainerSize, constants.dir)
		newGap := lpMaybeResolve(sizeMain(style.gapVal(), constants.dir), some(innerMain))
		sizeSetMain(&constants.gap, constants.dir, newGap.unwrapOr(0))
	}

	// 6. Resolve the flexible lengths.
	for i := range flexLines {
		resolveFlexibleLengths(&flexLines[i], &constants)
	}

	// 9.4. Cross Size Determination.
	// 7. Determine the hypothetical cross size of each item.
	for i := range flexLines {
		determineHypotheticalCrossSize(t, &flexLines[i], &constants, availSpace)
	}

	// Calculate child baselines.
	calculateChildrenBaseLines(t, known, availSpace, flexLines, &constants)

	// 8. Calculate the cross size of each flex line.
	calculateCrossSize(flexLines, known, &constants)

	// 9. Handle align-content: stretch.
	handleAlignContentStretch(flexLines, known, &constants)

	// 11. Determine the used cross size of each flex item.
	determineUsedCrossSize(t, flexLines, &constants)

	// 9.5. Main-Axis Alignment.
	// 12. Distribute any remaining free space.
	distributeRemainingFreeSpace(flexLines, &constants)

	// 9.6. Cross-Axis Alignment.
	// 13. Resolve cross-axis auto margins.
	resolveCrossAxisAutoMargins(flexLines, &constants)

	// 15. Determine the flex container's used cross size.
	totalLineCrossSize := determineContainerCrossSize(flexLines, known, &constants)

	if runMode == runComputeSize {
		return layoutOutputFromOuterSize(constants.containerSize)
	}

	// 16. Align all flex lines per align-content.
	alignFlexLinesPerAlignContent(flexLines, &constants, totalLineCrossSize)

	// Final layout pass.
	inflowOverflowRect := finalLayoutPass(t, flexLines, &constants)

	// Absolute layout on absolutely positioned children.
	absoluteOverflowRect := performAbsoluteLayoutOnAbsoluteChildren(t, node, &constants)

	// Hidden layout for display:none children.
	childCount := t.childCount(node)
	for order := 0; order < childCount; order++ {
		child := t.childID(node, order)
		if t.flexboxChildStyle(child).boxGenerationMode() == boxGenNone {
			t.setUnroundedLayout(child, newLayoutWithOrder(uint32(order)))
			t.computeChildLayout(child, layoutInputHidden)
		}
	}

	// 8.5. Flex Container Baselines.
	var firstLine *flexLine
	if constants.isWrapReverse {
		if len(flexLines) > 0 {
			firstLine = &flexLines[len(flexLines)-1]
		}
	} else {
		if len(flexLines) > 0 {
			firstLine = &flexLines[0]
		}
	}
	var firstVerticalBaseline optF32
	if firstLine != nil {
		if constants.isColumn {
			var item *flexItem
			if constants.dir.isReverse() {
				if len(firstLine.items) > 0 {
					item = firstLine.items[len(firstLine.items)-1]
				}
			} else {
				if len(firstLine.items) > 0 {
					item = firstLine.items[0]
				}
			}
			if item != nil {
				firstVerticalBaseline = some(item.baseline)
			}
		} else {
			var found *flexItem
			for _, item := range firstLine.items {
				if item.participatesInBaselineAlignment(constants.dir) {
					found = item
					break
				}
			}
			if found == nil && len(firstLine.items) > 0 {
				found = firstLine.items[0]
			}
			if found != nil {
				firstVerticalBaseline = some(found.baseline)
			}
		}
	}

	return layoutOutputFromSizesAndBaselines(
		constants.containerSize,
		rectF32Union(inflowOverflowRect, absoluteOverflowRect),
		baselinesFromFirst(firstVerticalBaseline),
	)
}

// computeConstants computes constants that can be reused during the algorithm.
func computeConstants(
	t flexboxTree,
	style *Style,
	known Size[optF32],
	knownAreDefinite Size[bool],
	parentSize Size[optF32],
	avail Size[AvailableSpace],
) algoConstants {
	dir := style.flexDirectionVal()
	isRow := dir.isRow()
	isColumn := dir.isColumn()
	fw := style.flexWrapVal()
	isWrap := fw.isMultiLine()
	isWrapReverse := fw.isReverse()

	aspectRatio := style.aspectRatioVal()
	margin := rectLPAResolveOrZeroOpt(style.marginVal(), parentSize.Width)
	padding := rectLPResolveOrZeroOpt(style.paddingVal(), parentSize.Width)
	border := rectLPResolveOrZeroOpt(style.borderVal(), parentSize.Width)
	paddingBorderSum := sizeF32Add(rectF32SumAxes(padding), rectF32SumAxes(border))
	var boxSizingAdj Size[float32]
	if style.boxSizingVal() == boxSizingContentBox {
		boxSizingAdj = paddingBorderSum
	}

	alignItems := AlignItemsStretch
	if ai := style.alignItemsVal(); ai != nil {
		alignItems = *ai
	}
	alignContent := AlignContentStretch
	if ac := style.alignContentVal(); ac != nil {
		alignContent = *ac
	}
	justifyContent := style.justifyContentVal()
	layoutDirection := style.directionVal()

	// Scrollbar gutters: axes are transposed.
	overflow := style.overflowVal()
	scrollbarGutter := Point[float32]{
		X: scrollGutter(overflow.Y, style.scrollbarWidthVal()),
		Y: scrollGutter(overflow.X, style.scrollbarWidthVal()),
	}
	isScrollContainer := overflow.X.isScrollContainer() || overflow.Y.isScrollContainer()

	contentBoxInset := rectF32Add(padding, border)
	contentBoxInset.Bottom += scrollbarGutter.Y
	if layoutDirection == directionLtr {
		contentBoxInset.Right += scrollbarGutter.X
	} else {
		contentBoxInset.Left += scrollbarGutter.X
	}

	nodeOuterSize := known
	nodeInnerSize := sizeOptMaybeSub(nodeOuterSize, sizeF32ToOpt(rectF32SumAxes(contentBoxInset)))
	knownMainSizeIsDefinite := sizeMain(knownAreDefinite, dir)
	hasDefiniteMainSize := knownMainSizeIsDefinite && sizeMain(known, dir).isSome()
	hasDefiniteCrossSize := sizeCross(knownAreDefinite, dir) && sizeCross(known, dir).isSome()
	crossAxisAvailIsDefinite := hasDefiniteCrossSize || sizeCross(avail, dir).isDefinite()

	gap := sizeLPResolveOrZeroSize(style.gapVal(), sizeOptOr(nodeInnerSize, sizeNone))

	return algoConstants{
		dir:             dir,
		layoutDirection: layoutDirection,
		isRow:           isRow,
		isColumn:        isColumn,
		isWrap:          isWrap,
		isWrapReverse:   isWrapReverse,
		minSize: sizeOptMaybeAdd(
			sizeOptMaybeApplyAspectRatio(sizeLPAMaybeResolve(style.minSizeVal(), parentSize), aspectRatio),
			sizeF32ToOpt(boxSizingAdj)),
		maxSize: sizeOptMaybeAdd(
			sizeOptMaybeApplyAspectRatio(sizeLPAMaybeResolve(style.maxSizeVal(), parentSize), aspectRatio),
			sizeF32ToOpt(boxSizingAdj)),
		margin:                            margin,
		border:                            border,
		gap:                               gap,
		contentBoxInset:                   contentBoxInset,
		scrollbarGutter:                   scrollbarGutter,
		isScrollContainer:                 isScrollContainer,
		alignItems:                        alignItems,
		alignContent:                      alignContent,
		justifyContent:                    justifyContent,
		nodeOuterSize:                     nodeOuterSize,
		nodeInnerSize:                     nodeInnerSize,
		knownMainSizeIsDefinite:           knownMainSizeIsDefinite,
		hasDefiniteMainSize:               hasDefiniteMainSize,
		hasDefiniteCrossSize:              hasDefiniteCrossSize,
		crossAxisAvailableSpaceIsDefinite: crossAxisAvailIsDefinite,
		containerSize:                     sizeZeroF32,
		innerContainerSize:                sizeZeroF32,
	}
}

// generateAnonymousFlexItems generates anonymous flex items.
func generateAnonymousFlexItems(t flexboxTree, node NodeID, constants *algoConstants) []*flexItem {
	var percentResolutionSize Size[optF32]
	if constants.knownMainSizeIsDefinite {
		percentResolutionSize = constants.nodeInnerSize
	} else {
		percentResolutionSize = sizeWithMain(constants.nodeInnerSize, constants.dir, none())
	}

	var items []*flexItem
	childCount := t.childCount(node)
	for index := 0; index < childCount; index++ {
		child := t.childID(node, index)
		childStyle := t.flexboxChildStyle(child)
		if childStyle.positionVal() == positionAbsolute {
			continue
		}
		if childStyle.boxGenerationMode() == boxGenNone {
			continue
		}

		aspectRatio := childStyle.aspectRatioVal()
		padding := rectLPResolveOrZeroOpt(childStyle.paddingVal(), constants.nodeInnerSize.Width)
		border := rectLPResolveOrZeroOpt(childStyle.borderVal(), constants.nodeInnerSize.Width)
		pbSum := rectF32SumAxes(rectF32Add(padding, border))
		var boxSizingAdj Size[float32]
		if childStyle.boxSizingVal() == boxSizingContentBox {
			boxSizingAdj = pbSum
		}

		alignSelf := constants.alignItems
		if as := childStyle.alignSelfVal(); as != nil {
			alignSelf = *as
		}
		alignSelf = alignSelf.resolveSelfRelative(childStyle.directionVal(), constants.layoutDirection, constants.isColumn)

		item := &flexItem{
			node:        child,
			order:       uint32(index),
			size:        sizeOptMaybeAdd(sizeOptMaybeApplyAspectRatio(sizeDimMaybeResolve(childStyle.sizeVal(), percentResolutionSize), aspectRatio), sizeF32ToOpt(boxSizingAdj)),
			sizeStyle:   childStyle.sizeVal(),
			minSize:     sizeOptMaybeAdd(sizeLPAMaybeResolve(childStyle.minSizeVal(), percentResolutionSize), sizeF32ToOpt(boxSizingAdj)),
			maxSize:     sizeOptMaybeAdd(sizeLPAMaybeResolve(childStyle.maxSizeVal(), percentResolutionSize), sizeF32ToOpt(boxSizingAdj)),
			aspectRatio: aspectRatio,
			inset: rectZipSize(childStyle.insetVal(), constants.nodeInnerSize,
				func(p LengthPercentageAuto, s optF32) optF32 { return lpaMaybeResolve(p, s) }),
			margin:         rectLPAResolveOrZeroOpt(childStyle.marginVal(), constants.nodeInnerSize.Width),
			marginIsAuto:   rectMap(childStyle.marginVal(), func(lpa LengthPercentageAuto) bool { return lpa.isAuto() }),
			padding:        padding,
			border:         border,
			alignSelf:      alignSelf,
			overflow:       childStyle.overflowVal(),
			contain:        childStyle.containVal(),
			scrollbarWidth: childStyle.scrollbarWidthVal(),
			flexGrow:       childStyle.flexGrowVal(),
			flexShrink:     childStyle.flexShrinkVal(),
		}
		items = append(items, item)
	}
	return items
}

// determineAvailableSpace determines the available main and cross space.
func determineAvailableSpace(known Size[optF32], outerAvail Size[AvailableSpace], c *algoConstants) Size[AvailableSpace] {
	var width, height AvailableSpace
	if known.Width.isSome() {
		width = definiteAvail(known.Width.v - rectF32HorizontalAxisSum(c.contentBoxInset))
	} else {
		width = outerAvail.Width.maybeSubF32(rectF32HorizontalAxisSum(c.margin)).maybeSubF32(rectF32HorizontalAxisSum(c.contentBoxInset))
	}
	if known.Height.isSome() {
		height = definiteAvail(known.Height.v - rectF32VerticalAxisSum(c.contentBoxInset))
	} else {
		height = outerAvail.Height.maybeSubF32(rectF32VerticalAxisSum(c.margin)).maybeSubF32(rectF32VerticalAxisSum(c.contentBoxInset))
	}
	return Size[AvailableSpace]{Width: width, Height: height}
}

// determineFlexBaseSize determines the flex base size and hypothetical main size.
func determineFlexBaseSize(t flexboxTree, c *algoConstants, avail Size[AvailableSpace], items []*flexItem) {
	dir := c.dir

	for _, child := range items {
		childStyle := t.flexboxChildStyle(child.node)

		crossAxisParentSize := sizeCross(c.nodeInnerSize, dir)
		childParentSize := sizeFromCross(dir, crossAxisParentSize)

		crossAxisMarginSum := rectF32CrossAxisSum(c.margin, dir)
		transferredMinSize := sizeOptMaybeApplyAspectRatio(child.minSize, child.aspectRatio)
		transferredMaxSize := sizeOptMaybeApplyAspectRatio(child.maxSize, child.aspectRatio)
		childMinCross := optMaybeAdd(sizeCross(transferredMinSize, dir), some(crossAxisMarginSum))
		childMaxCross := optMaybeAdd(sizeCross(transferredMaxSize, dir), some(crossAxisMarginSum))

		// Clamp available space by min/max.
		var crossAxisAvail AvailableSpace
		switch crossAvail := sizeCross(avail, dir); crossAvail.kind {
		case availableDefinite:
			val := c.dividedCrossSpace(crossAxisParentSize.unwrapOr(crossAvail.val))
			crossAxisAvail = availMaybeClampF32(definiteAvail(val), childMinCross.unwrapOr(0), childMaxCross.unwrapOr(float32(mathInf())))
		case availableMinContent:
			if childMinCross.isSome() {
				crossAxisAvail = definiteAvail(childMinCross.v)
			} else {
				crossAxisAvail = minContent
			}
		case availableMaxContent:
			if childMaxCross.isSome() {
				crossAxisAvail = definiteAvail(childMaxCross.v)
			} else {
				crossAxisAvail = maxContent
			}
		}

		// Known dimensions for child sizing.
		childCrossSizeIsDefinite := sizeCross(child.size, dir).isSome()
		childKnown := sizeWithMain(child.size, dir, none())
		// Clamp the definite cross size by transferred min/max.
		clampedCross := optMaybeClamp(sizeCross(childKnown, dir), sizeCross(transferredMinSize, dir), sizeCross(transferredMaxSize, dir))
		sizeSetCross(&childKnown, dir, clampedCross)
		if child.alignSelf.Keyword == alignItemsStretch &&
			!rectBoolCrossStart(child.marginIsAuto, c.dir) &&
			!rectBoolCrossEnd(child.marginIsAuto, c.dir) &&
			!sizeCross(childKnown, dir).isSome() {
			sizeSetCross(&childKnown, dir, optMaybeSub(crossAxisAvail.intoOption(), some(rectF32CrossAxisSum(child.margin, dir))))
			childCrossSizeIsDefinite = !c.isWrap && c.hasDefiniteCrossSize && crossAxisParentSize.isSome()
		}

		containerWidth := sizeMain(c.nodeInnerSize, dir)
		var boxSizingAdjMain float32
		if childStyle.boxSizingVal() == boxSizingContentBox {
			pb := rectF32SumAxes(rectF32Add(
				rectLPResolveOrZeroOpt(childStyle.paddingVal(), containerWidth),
				rectLPResolveOrZeroOpt(childStyle.borderVal(), containerWidth)))
			boxSizingAdjMain = sizeMain(pb, dir)
		}

		percentResolutionMainSize := none()
		if c.knownMainSizeIsDefinite {
			percentResolutionMainSize = sizeMain(c.nodeInnerSize, dir)
		}
		flexBasisStyle := childStyle.flexBasisVal()
		flexBasis := optMaybeAdd(
			dimMaybeResolve(flexBasisStyle, percentResolutionMainSize),
			some(boxSizingAdjMain))

		// Compute flex_basis.
		child.flexBasis = func() float32 {
			mainSize := sizeMain(child.size, dir)
			mainStretchSize := optMaybeSub(percentResolutionMainSize, some(rectF32MainAxisSum(child.margin, dir)))

			var keywordMainAvail *AvailableSpace
			if flexBasisStyle.isContent() {
				keywordMainAvail = nil
			} else if flexBasisStyle.isSizingKeyword() {
				res := resolveSizingKeyword(flexBasisStyle, mainStretchSize, percentResolutionMainSize)
				if res != nil {
					if res.kind == sizingExact {
						child.flexBasisIsDefinite = true
						return res.exact
					}
					keywordMainAvail = &res.value
				}
			} else {
				if fb := optOr(flexBasis, mainSize); fb.isSome() {
					child.flexBasisIsDefinite = true
					return fb.v
				}
				res := resolveSizingKeyword(sizeMain(child.sizeStyle, dir), mainStretchSize, percentResolutionMainSize)
				if res != nil {
					if res.kind == sizingExact {
						child.flexBasisIsDefinite = true
						return res.exact
					}
					keywordMainAvail = &res.value
				}
			}

			// If the item has an aspect ratio and a definite cross size.
			if childCrossSizeIsDefinite {
				if child.aspectRatio != nil {
					cross := sizeCross(childKnown, dir)
					if cross.isSome() {
						child.flexBasisIsDefinite = true
						if dir.isRow() {
							return cross.v * *child.aspectRatio
						}
						return cross.v / *child.aspectRatio
					}
				}
			}

			// E. Size the item into the available space.
			mainAvail := maxContent
			if keywordMainAvail != nil {
				mainAvail = *keywordMainAvail
			} else if sizeMain(avail, dir).kind == availableMinContent {
				mainAvail = minContent
			}
			childAvail := sizeWithCross(sizeWithMain(sizeMaxContent, dir, mainAvail), dir, crossAxisAvail)

			return measureChildSize(t, child.node, childKnown, childParentSize, childAvail,
				sizingContentSize, dir.mainAxis(), lineBoolFalse)
		}()

		// Floor flex-basis by padding_border_sum.
		pbMainSum := rectF32MainAxisSum(child.padding, c.dir) + rectF32MainAxisSum(child.border, c.dir)
		child.flexBasis = f32Max(child.flexBasis, pbMainSum)

		child.innerFlexBasis = child.flexBasis - rectF32MainAxisSum(child.padding, c.dir) - rectF32MainAxisSum(child.border, c.dir)

		pbAxesSums := sizeF32ToOpt(rectF32SumAxes(rectF32Add(child.padding, child.border)))

		// Resolve minimum main size.
		styleMinMain := optOr(
			sizeMain(child.minSize, dir),
			pointMain(child.overflow, dir).maybeIntoAutomaticMinSize())

		child.resolvedMinimumMainSize = styleMinMain.unwrapOrFunc(func() float32 {
			minContentAvail := sizeWithCross(sizeMaxContent, dir, crossAxisAvail)
			// Override main with MinContent.
			sizeSetMain(&minContentAvail, dir, minContent)
			minContentMainSize := measureChildSize(t, child.node, childKnown, childParentSize, minContentAvail,
				sizingContentSize, dir.mainAxis(), lineBoolFalse)

			clampedMinContent := f32MaybeMin(minContentMainSize, sizeMain(child.size, dir))
			clampedMinContent = f32MaybeMin(clampedMinContent, sizeMain(transferredMaxSize, dir))
			return f32MaybeMax(clampedMinContent, sizeMain(pbAxesSums, dir))
		})

		// Hypothetical main size.
		hypInnerMinMain := f32Max(child.resolvedMinimumMainSize,
			f32Max(sizeMain(transferredMinSize, c.dir).unwrapOr(0), sizeMain(pbAxesSums, c.dir).unwrapOr(0)))
		hypInnerSize := f32MaybeClamp(child.flexBasis, some(hypInnerMinMain), sizeMain(transferredMaxSize, c.dir))
		hypOuterSize := hypInnerSize + rectF32MainAxisSum(child.margin, c.dir)

		sizeSetMain(&child.hypotheticalInnerSize, c.dir, hypInnerSize)
		sizeSetMain(&child.hypotheticalOuterSize, c.dir, hypOuterSize)
	}
}

// collectFlexLines collects flex items into flex lines.
func collectFlexLines(c *algoConstants, avail Size[AvailableSpace], items []*flexItem) []flexLine {
	if !c.isWrap || !c.knownMainSizeIsDefinite {
		return []flexLine{{items: items, crossSize: 0, offsetCross: 0}}
	}

	mainAxisAvail := sizeMain(avail, c.dir)
	if maxMain := sizeMain(c.maxSize, c.dir); maxMain.isSome() {
		availVal := sizeMain(avail, c.dir).intoOption().unwrapOr(maxMain.v)
		if !c.hasDefiniteMainSize {
			availVal = f32Min(availVal, maxMain.v)
		}
		availVal = f32MaybeMax(availVal, sizeMain(c.minSize, c.dir))
		mainAxisAvail = definiteAvail(availVal)
	}

	switch mainAxisAvail.kind {
	case availableMaxContent:
		return []flexLine{{items: items, crossSize: 0, offsetCross: 0}}
	case availableMinContent:
		lines := make([]flexLine, 0, len(items))
		for _, item := range items {
			lines = append(lines, flexLine{items: []*flexItem{item}, crossSize: 0, offsetCross: 0})
		}
		return lines
	case availableDefinite:
		mainAvailVal := mainAxisAvail.val
		mainGap := sizeMain(c.gap, c.dir)
		lines := make([]flexLine, 0, 1)
		remaining := items
		for len(remaining) > 0 {
			lineLength := float32(0)
			index := len(remaining)
			for i, child := range remaining {
				gapContrib := float32(0)
				if i != 0 {
					gapContrib = mainGap
				}
				lineLength += sizeMain(child.hypotheticalOuterSize, c.dir) + gapContrib
				if lineLength > mainAvailVal && i != 0 {
					index = i
					break
				}
			}
			lines = append(lines, flexLine{items: remaining[:index], crossSize: 0, offsetCross: 0})
			remaining = remaining[index:]
		}
		return lines
	}
	return []flexLine{{items: items, crossSize: 0, offsetCross: 0}}
}

// itemKnownDimensionDefiniteness computes whether each known dimension should be
// treated as definite.
func itemKnownDimensionDefiniteness(c *algoConstants, item *flexItem) Size[bool] {
	dir := c.dir
	mainIsDefinite := c.hasDefiniteMainSize || item.flexBasisIsDefinite

	hasCrossAutoMargins := rectBoolCrossStart(item.marginIsAuto, dir) || rectBoolCrossEnd(item.marginIsAuto, dir)
	crossSize := sizeCross(item.sizeStyle, dir)
	isStretched := !hasCrossAutoMargins &&
		(crossSize.isStretch() || (item.alignSelf.Keyword == alignItemsStretch && crossSize.isAuto()))
	crossIsDefinite := isStretched ||
		sizeCross(item.size, dir).isSome() ||
		(!dir.isRow() && c.crossAxisAvailableSpaceIsDefinite)

	result := sizeWithMain(Size[bool]{Width: true, Height: true}, dir, mainIsDefinite)
	return sizeWithCross(result, dir, crossIsDefinite)
}

// determineContainerMainSize determines the container's main size.
func determineContainerMainSize(t flexboxTree, avail Size[AvailableSpace], lines []flexLine, c *algoConstants) {
	dir := c.dir
	mainContentBoxInset := rectF32MainAxisSum(c.contentBoxInset, c.dir)

	outerMainSize := sizeMain(c.nodeOuterSize, c.dir).unwrapOrFunc(func() float32 {
		switch mainAvail := sizeMain(avail, dir); mainAvail.kind {
		case availableDefinite:
			mainAxisAvail := mainAvail.val
			mainGap := sizeMain(c.gap, c.dir)
			itemMainLen := func(child *flexItem) float32 {
				pbSum := rectF32MainAxisSum(rectF32Add(child.padding, child.border), c.dir)
				return f32Max(
					f32MaybeMax(child.flexBasis, sizeMain(child.minSize, c.dir))+rectF32MainAxisSum(child.margin, c.dir),
					pbSum)
			}
			longestLine := float32(0)
			for _, line := range lines {
				lineGap := sumAxisGaps(mainGap, len(line.items))
				total := float32(0)
				for _, child := range line.items {
					total += itemMainLen(child)
				}
				total += lineGap
				if total > longestLine {
					longestLine = total
				}
			}
			size := longestLine + mainContentBoxInset
			if len(lines) > 1 {
				return f32Max(size, mainAxisAvail)
			}
			return size
		case availableMinContent:
			if c.isWrap {
				longestLine := float32(0)
				for _, line := range lines {
					lineGap := sumAxisGaps(sizeMain(c.gap, c.dir), len(line.items))
					total := float32(0)
					for _, child := range line.items {
						pbSum := rectF32MainAxisSum(rectF32Add(child.padding, child.border), c.dir)
						total += f32Max(
							f32MaybeMax(child.flexBasis, sizeMain(child.minSize, c.dir))+rectF32MainAxisSum(child.margin, c.dir),
							pbSum)
					}
					total += lineGap
					if total > longestLine {
						longestLine = total
					}
				}
				return longestLine + mainContentBoxInset
			}
			fallthrough
		default:
			// MinContent (non-wrap) or MaxContent.
			mainSize := float32(0)
			for _, line := range lines {
				for _, item := range line.items {
					styleMin := sizeMain(item.minSize, c.dir)
					stylePref := sizeMain(item.size, c.dir)
					styleMax := sizeMain(item.maxSize, c.dir)

					clampingBasis := f32MaybeMax(item.flexBasis, stylePref)
					var flexBasisMin, flexBasisMax optF32
					if item.flexShrink == 0 {
						flexBasisMin = some(clampingBasis)
					}
					if item.flexGrow == 0 {
						flexBasisMax = some(clampingBasis)
					}

					minMainSize := f32Max(
						optOr(optMaybeMax(styleMin, flexBasisMin), flexBasisMin).unwrapOr(item.resolvedMinimumMainSize),
						item.resolvedMinimumMainSize)
					maxMainSize := optOr(optMaybeMin(styleMax, flexBasisMax), flexBasisMax).unwrapOrFunc(func() float32 { return float32(mathInf()) })

					var contentContribution float32
					if maxMainSize <= minMainSize || (stylePref.isSome() && maxMainSize <= stylePref.v) {
						contentContribution = f32Max(f32Min(stylePref.unwrapOr(0), maxMainSize), minMainSize) + rectF32MainAxisSum(item.margin, c.dir)
					} else if maxMainSize <= minMainSize {
						contentContribution = minMainSize + rectF32MainAxisSum(item.margin, c.dir)
					} else if item.isScrollContainer() {
						contentContribution = item.flexBasis + rectF32MainAxisSum(item.margin, c.dir)
					} else if stylePref.isSome() {
						itemPBMain := rectF32MainAxisSum(item.padding, c.dir) + rectF32MainAxisSum(item.border, c.dir)
						innerMain := f32Max(stylePref.v, itemPBMain)
						if c.isRow {
							contentContribution = f32MaybeClamp(innerMain+rectF32MainAxisSum(item.margin, c.dir), styleMin, styleMax)
						} else {
							contentContribution = f32MaybeClamp(f32Max(innerMain, item.flexBasis)+rectF32MainAxisSum(item.margin, c.dir), styleMin, styleMax)
						}
					} else {
						// Measure.
						crossAxisParentSize := sizeCross(c.nodeInnerSize, dir)
						crossAxisMarginSum := rectF32CrossAxisSum(c.margin, dir)
						childMinCross := optMaybeAdd(sizeCross(item.minSize, dir), some(crossAxisMarginSum))
						childMaxCross := optMaybeAdd(sizeCross(item.maxSize, dir), some(crossAxisMarginSum))
						crossAvail := sizeCross(avail, dir).mapDefiniteValue(func(val float32) float32 {
							return c.dividedCrossSpace(crossAxisParentSize.unwrapOr(val))
						})
						crossAvail = availMaybeClampOpt(crossAvail, childMinCross, childMaxCross)
						childAvail := sizeWithCross(avail, dir, crossAvail)

						childKnown := sizeWithMain(item.size, dir, none())
						if item.alignSelf.Keyword == alignItemsStretch && !sizeCross(childKnown, dir).isSome() {
							sizeSetCross(&childKnown, dir, optMaybeSub(crossAvail.intoOption(), some(rectF32CrossAxisSum(item.margin, dir))))
						}

						measuredMainSize := measureChildSize(t, item.node, childKnown, c.nodeInnerSize, childAvail,
							sizingContentSize, dir.mainAxis(), lineBoolFalse)

						var transferredMain optF32
						if item.aspectRatio != nil {
							cross := sizeCross(childKnown, dir)
							if cross.isSome() {
								if c.isRow {
									transferredMain = some(cross.v * *item.aspectRatio)
								} else {
									transferredMain = some(cross.v / *item.aspectRatio)
								}
							}
						}
						innerMain := f32MaybeMax(measuredMainSize, transferredMain)
						if c.isRow {
							contentContribution = f32MaybeClamp(innerMain+rectF32MainAxisSum(item.margin, c.dir), styleMin, styleMax)
						} else {
							contentContribution = f32MaybeClamp(f32Max(innerMain, item.flexBasis)+rectF32MainAxisSum(item.margin, c.dir), styleMin, styleMax)
						}
					}

					diff := contentContribution - item.flexBasis
					if diff > 0 {
						item.contentFlexFraction = diff / f32Max(1, item.flexGrow)
					} else if diff < 0 {
						scaledShrink := f32Max(1, item.flexShrink) * item.innerFlexBasis
						item.contentFlexFraction = diff / scaledShrink
					} else {
						item.contentFlexFraction = 0
					}
				}

				// Sum item main sizes.
				itemMainSum := float32(0)
				for _, item := range line.items {
					flexFraction := item.contentFlexFraction
					var flexContrib float32
					if item.contentFlexFraction > 0 {
						flexContrib = f32Max(1, item.flexGrow) * flexFraction
					} else if item.contentFlexFraction < 0 {
						scaledShrink := f32Max(1, item.flexShrink) * item.innerFlexBasis
						if scaledShrink != 0 {
							flexContrib = scaledShrink * flexFraction
						}
					}
					size := item.flexBasis + flexContrib
					sizeSetMain(&item.outerTargetSize, c.dir, size)
					sizeSetMain(&item.targetSize, c.dir, size)
					itemMainSum += size
				}
				gapSum := sumAxisGaps(sizeMain(c.gap, c.dir), len(line.items))
				mainSize = f32Max(mainSize, itemMainSum+gapSum)
			}
			return mainSize + mainContentBoxInset
		}
	})

	outerMainSize = f32MaybeClamp(outerMainSize, sizeMain(c.minSize, c.dir), sizeMain(c.maxSize, c.dir))
	outerMainSize = f32Max(outerMainSize, mainContentBoxInset-pointMain(c.scrollbarGutter, c.dir))
	innerMainSize := f32Max(outerMainSize-mainContentBoxInset, 0)
	sizeSetMain(&c.containerSize, c.dir, outerMainSize)
	sizeSetMain(&c.innerContainerSize, c.dir, innerMainSize)
	sizeSetMain(&c.nodeInnerSize, c.dir, some(innerMainSize))
}

// resolveFlexibleLengths resolves the flexible lengths of items within a line.
func resolveFlexibleLengths(line *flexLine, c *algoConstants) {
	totalMainGap := sumAxisGaps(sizeMain(c.gap, c.dir), len(line.items))

	totalHypOuterMain := float32(0)
	for _, child := range line.items {
		totalHypOuterMain += sizeMain(child.hypotheticalOuterSize, c.dir)
	}
	usedFlexFactor := totalMainGap + totalHypOuterMain
	innerMain := sizeMain(c.nodeInnerSize, c.dir).unwrapOr(0)
	growing := usedFlexFactor < innerMain
	shrinking := usedFlexFactor > innerMain
	exactlySized := !growing && !shrinking

	// 2. Size inflexible items.
	for _, child := range line.items {
		innerTarget := sizeMain(child.hypotheticalInnerSize, c.dir)
		sizeSetMain(&child.targetSize, c.dir, innerTarget)
		if exactlySized ||
			(child.flexGrow == 0 && child.flexShrink == 0) ||
			(growing && child.flexBasis > sizeMain(child.hypotheticalInnerSize, c.dir)) ||
			(shrinking && child.flexBasis < sizeMain(child.hypotheticalInnerSize, c.dir)) {
			child.frozen = true
			sizeSetMain(&child.outerTargetSize, c.dir, innerTarget+rectF32MainAxisSum(child.margin, c.dir))
		}
	}

	if exactlySized {
		return
	}

	// 3. Calculate initial free space.
	usedSpace := totalMainGap
	for _, child := range line.items {
		if child.frozen {
			usedSpace += sizeMain(child.outerTargetSize, c.dir)
		} else {
			usedSpace += child.flexBasis + rectF32MainAxisSum(child.margin, c.dir)
		}
	}
	initialFreeSpace := optMaybeSub(sizeMain(c.nodeInnerSize, c.dir), some(usedSpace)).unwrapOr(0)

	// 4. Loop.
	for {
		// a. Check for flexible items.
		allFrozen := true
		for _, child := range line.items {
			if !child.frozen {
				allFrozen = false
				break
			}
		}
		if allFrozen {
			break
		}

		// b. Calculate remaining free space.
		usedSpace = totalMainGap
		for _, child := range line.items {
			if child.frozen {
				usedSpace += sizeMain(child.outerTargetSize, c.dir)
			} else {
				usedSpace += child.flexBasis + rectF32MainAxisSum(child.margin, c.dir)
			}
		}

		var sumGrow, sumShrink float32
		for _, child := range line.items {
			if !child.frozen {
				sumGrow += child.flexGrow
				sumShrink += child.flexShrink
			}
		}

		var freeSpace float32
		if growing && sumGrow < 1 {
			freeSpace = f32MaybeMin(
				initialFreeSpace*sumGrow-totalMainGap,
				optMaybeSub(sizeMain(c.nodeInnerSize, c.dir), some(usedSpace)))
		} else if shrinking && sumShrink < 1 {
			freeSpace = f32MaybeMax(
				initialFreeSpace*sumShrink-totalMainGap,
				optMaybeSub(sizeMain(c.nodeInnerSize, c.dir), some(usedSpace)))
		} else {
			sub := optMaybeSub(sizeMain(c.nodeInnerSize, c.dir), some(usedSpace))
			if sub.isSome() {
				freeSpace = sub.v
			} else {
				freeSpace = usedFlexFactor - usedSpace
			}
		}

		// c. Distribute free space.
		if freeSpace != 0 && !isNaN(freeSpace) {
			if growing && sumGrow > 0 {
				for _, child := range line.items {
					if !child.frozen {
						sizeSetMain(&child.targetSize, c.dir, child.flexBasis+freeSpace*(child.flexGrow/sumGrow))
					}
				}
			} else if shrinking && sumShrink > 0 {
				sumScaledShrink := float32(0)
				for _, child := range line.items {
					if !child.frozen {
						sumScaledShrink += child.innerFlexBasis * child.flexShrink
					}
				}
				if sumScaledShrink > 0 {
					for _, child := range line.items {
						if !child.frozen {
							scaledShrink := child.innerFlexBasis * child.flexShrink
							sizeSetMain(&child.targetSize, c.dir, child.flexBasis+freeSpace*(scaledShrink/sumScaledShrink))
						}
					}
				}
			}
		}

		// d. Fix min/max violations.
		totalViolation := float32(0)
		for _, child := range line.items {
			if child.frozen {
				continue
			}
			clamped := f32MaybeClamp(sizeMain(child.targetSize, c.dir), some(child.resolvedMinimumMainSize), sizeMain(child.maxSize, c.dir))
			clamped = f32Max(clamped, 0)
			child.violation = clamped - sizeMain(child.targetSize, c.dir)
			sizeSetMain(&child.targetSize, c.dir, clamped)
			sizeSetMain(&child.outerTargetSize, c.dir, clamped+rectF32MainAxisSum(child.margin, c.dir))
			totalViolation += child.violation
		}

		// e. Freeze over-flexed items.
		for _, child := range line.items {
			if child.frozen {
				continue
			}
			if totalViolation > 0 {
				child.frozen = child.violation > 0
			} else if totalViolation < 0 {
				child.frozen = child.violation < 0
			} else {
				child.frozen = true
			}
		}
	}
}

// determineHypotheticalCrossSize determines the hypothetical cross size of each item.
func determineHypotheticalCrossSize(t flexboxTree, line *flexLine, c *algoConstants, avail Size[AvailableSpace]) {
	for _, child := range line.items {
		pbCrossSum := rectF32CrossAxisSum(rectF32Add(child.padding, child.border), c.dir)

		childKnownMain := some(sizeMain(c.containerSize, c.dir))

		transferredMinCross := sizeCross(sizeOptMaybeApplyAspectRatio(child.minSize, child.aspectRatio), c.dir)
		transferredMaxCross := sizeCross(sizeOptMaybeApplyAspectRatio(child.maxSize, child.aspectRatio), c.dir)

		childCross := optMaybeMax(
			optMaybeClamp(sizeCross(child.size, c.dir), transferredMinCross, transferredMaxCross),
			some(pbCrossSum))

		childAvailCross := availMaybeMaxOpt(
			availMaybeClampOpt(
				sizeCross(avail, c.dir).mapDefiniteValue(func(val float32) float32 { return c.dividedCrossSpace(val) }),
				transferredMinCross, transferredMaxCross),
			some(pbCrossSum))

		crossStretchSizeInner := sizeCross(c.nodeInnerSize, c.dir)
		var crossStretchResolved optF32
		if crossStretchSizeInner.isSome() {
			crossStretchResolved = some(c.dividedCrossSpace(crossStretchSizeInner.v))
		} else {
			crossStretchResolved = none()
		}
		crossStretchSize := optMaybeSub(
			crossStretchResolved,
			some(rectF32CrossAxisSum(child.margin, c.dir)))
		if res := resolveSizingKeyword(sizeCross(child.sizeStyle, c.dir), crossStretchSize, sizeCross(c.nodeInnerSize, c.dir)); res != nil && res.kind == sizingMeasure {
			childAvailCross = res.value
		}

		var childInnerCross float32
		if childCross.isSome() {
			childInnerCross = childCross.v
		} else {
			// Measure.
			var knownDims Size[optF32]
			if c.isRow {
				knownDims = Size[optF32]{Width: some(child.targetSize.Width), Height: childCross}
			} else {
				knownDims = Size[optF32]{Width: childCross, Height: some(child.targetSize.Height)}
			}
			var childAvailSize Size[AvailableSpace]
			if c.isRow {
				childAvailSize = Size[AvailableSpace]{Width: fromOptF32(childKnownMain), Height: childAvailCross}
			} else {
				childAvailSize = Size[AvailableSpace]{Width: childAvailCross, Height: fromOptF32(childKnownMain)}
			}
			out := t.computeChildLayout(child.node, LayoutInput{
				RunMode:                       runComputeSize,
				SizingMode:                    sizingContentSize,
				Axis:                          fromAbsoluteAxis(c.dir.crossAxis()),
				KnownDimensions:               knownDims,
				KnownDimensionsAreDefinite:    itemKnownDimensionDefiniteness(c, child),
				ParentSize:                    c.nodeInnerSize,
				AvailableSpace:                childAvailSize,
				VerticalMarginsAreCollapsible: lineBoolFalse,
			})
			measuredCross := sizeGetAbs(out.Size, c.dir.crossAxis())
			childInnerCross = f32Max(
				f32MaybeClamp(measuredCross, transferredMinCross, transferredMaxCross),
				pbCrossSum)
		}
		childOuterCross := childInnerCross + rectF32CrossAxisSum(child.margin, c.dir)

		sizeSetCross(&child.hypotheticalInnerSize, c.dir, childInnerCross)
		sizeSetCross(&child.hypotheticalOuterSize, c.dir, childOuterCross)
	}
}

// calculateChildrenBaseLines calculates the baselines of children.
func calculateChildrenBaseLines(t flexboxTree, nodeSize Size[optF32], avail Size[AvailableSpace], lines []flexLine, c *algoConstants) {
	if !c.isRow {
		return
	}
	for _, line := range lines {
		baselineCount := 0
		for _, child := range line.items {
			if child.participatesInBaselineAlignment(c.dir) {
				baselineCount++
			}
		}
		if baselineCount <= 1 {
			continue
		}
		for _, child := range line.items {
			if !child.participatesInBaselineAlignment(c.dir) {
				continue
			}
			var knownDims Size[optF32]
			if c.isRow {
				knownDims = Size[optF32]{Width: some(child.targetSize.Width), Height: some(child.hypotheticalInnerSize.Height)}
			} else {
				knownDims = Size[optF32]{Width: some(child.hypotheticalInnerSize.Width), Height: some(child.targetSize.Height)}
			}
			var childAvail Size[AvailableSpace]
			if c.isRow {
				childAvail = Size[AvailableSpace]{
					Width:  fromOptF32(some(c.containerSize.Width)),
					Height: avail.Height.maybeSet(nodeSize.Height),
				}
			} else {
				childAvail = Size[AvailableSpace]{
					Width:  avail.Width.maybeSet(nodeSize.Width),
					Height: fromOptF32(some(c.containerSize.Height)),
				}
			}
			out := t.computeChildLayout(child.node, LayoutInput{
				RunMode:                       runPerformLayout,
				SizingMode:                    sizingContentSize,
				Axis:                          requestedBoth,
				KnownDimensions:               knownDims,
				KnownDimensionsAreDefinite:    itemKnownDimensionDefiniteness(c, child),
				ParentSize:                    c.nodeInnerSize,
				AvailableSpace:                childAvail,
				VerticalMarginsAreCollapsible: lineBoolFalse,
			})
			baseline := out.Baselines.first
			height := out.Size.Height
			var resolvedBaseline float32
			if child.overflow.Y.isScrollContainer() {
				resolvedBaseline = f32Max(f32Min(baseline.unwrapOr(height), height), 0)
			} else {
				resolvedBaseline = baseline.unwrapOr(height)
			}
			child.baseline = resolvedBaseline + child.margin.Top
		}
	}
}

// calculateCrossSize calculates the cross size of each flex line.
func calculateCrossSize(lines []flexLine, nodeSize Size[optF32], c *algoConstants) {
	if !c.isWrap && sizeCross(nodeSize, c.dir).isSome() {
		crossPB := rectF32CrossAxisSum(c.contentBoxInset, c.dir)
		crossMin := sizeCross(c.minSize, c.dir)
		crossMax := sizeCross(c.maxSize, c.dir)
		lines[0].crossSize = optMaybeMax(
			optMaybeSub(
				optMaybeClamp(sizeCross(nodeSize, c.dir), crossMin, crossMax),
				some(crossPB)),
			some(0)).unwrapOr(0)
	} else {
		for i := range lines {
			line := &lines[i]
			maxBaseline := float32(0)
			for _, child := range line.items {
				if child.baseline > maxBaseline {
					maxBaseline = child.baseline
				}
			}
			lineCrossSize := float32(0)
			for _, child := range line.items {
				var contrib float32
				if child.participatesInBaselineAlignment(c.dir) {
					contrib = maxBaseline - child.baseline + sizeCross(child.hypotheticalOuterSize, c.dir)
				} else {
					contrib = sizeCross(child.hypotheticalOuterSize, c.dir)
				}
				if contrib > lineCrossSize {
					lineCrossSize = contrib
				}
			}
			line.crossSize = lineCrossSize
		}
		if !c.isWrap {
			crossPB := rectF32CrossAxisSum(c.contentBoxInset, c.dir)
			crossMin := sizeCross(c.minSize, c.dir)
			crossMax := sizeCross(c.maxSize, c.dir)
			lines[0].crossSize = f32MaybeClamp(lines[0].crossSize,
				optMaybeSub(crossMin, some(crossPB)),
				optMaybeSub(crossMax, some(crossPB)))
		}
	}
}

// handleAlignContentStretch handles align-content: stretch.
func handleAlignContentStretch(lines []flexLine, nodeSize Size[optF32], c *algoConstants) {
	if c.alignContent.Keyword != alignContentStretch {
		return
	}
	crossPB := rectF32CrossAxisSum(c.contentBoxInset, c.dir)
	crossMin := sizeCross(c.minSize, c.dir)
	crossMax := sizeCross(c.maxSize, c.dir)
	containerMinInnerCross := optMaybeMax(
		optMaybeSub(
			optMaybeClamp(
				optOr(sizeCross(nodeSize, c.dir), crossMin),
				crossMin, crossMax),
			some(crossPB)),
		some(0)).unwrapOr(0)

	totalCrossGap := sumAxisGaps(sizeCross(c.gap, c.dir), len(lines))
	linesTotalCross := float32(0)
	for _, line := range lines {
		linesTotalCross += line.crossSize
	}
	linesTotalCross += totalCrossGap

	if linesTotalCross < containerMinInnerCross {
		remaining := containerMinInnerCross - linesTotalCross
		addition := remaining / float32(len(lines))
		for i := range lines {
			lines[i].crossSize += addition
		}
	}
}

// determineUsedCrossSize determines the used cross size of each flex item.
func determineUsedCrossSize(t flexboxTree, lines []flexLine, c *algoConstants) {
	for _, line := range lines {
		lineCrossSize := line.crossSize
		for _, child := range line.items {
			childStyle := t.flexboxChildStyle(child.node)
			crossIsStretch := sizeCross(child.sizeStyle, c.dir).isStretch()
			var crossTarget float32
			if !rectBoolCrossStart(child.marginIsAuto, c.dir) &&
				!rectBoolCrossEnd(child.marginIsAuto, c.dir) &&
				(crossIsStretch || (child.alignSelf.Keyword == alignItemsStretch &&
					sizeCross(childStyle.sizeVal(), c.dir).isAuto())) {
				padding := rectLPResolveOrZeroSize(childStyle.paddingVal(), c.nodeInnerSize)
				border := rectLPResolveOrZeroSize(childStyle.borderVal(), c.nodeInnerSize)
				pbSum := rectF32SumAxes(rectF32Add(padding, border))
				var boxSizingAdj Size[float32]
				if childStyle.boxSizingVal() == boxSizingContentBox {
					boxSizingAdj = pbSum
				}
				maxSizeIgnoringAR := sizeOptMaybeAdd(
					sizeLPAMaybeResolve(childStyle.maxSizeVal(), c.nodeInnerSize),
					sizeF32ToOpt(boxSizingAdj))
				crossTarget = f32MaybeClamp(
					lineCrossSize-rectF32CrossAxisSum(child.margin, c.dir),
					sizeCross(child.minSize, c.dir),
					sizeCross(maxSizeIgnoringAR, c.dir))
			} else {
				crossTarget = sizeCross(child.hypotheticalInnerSize, c.dir)
			}
			sizeSetCross(&child.targetSize, c.dir, crossTarget)
			sizeSetCross(&child.outerTargetSize, c.dir, crossTarget+rectF32CrossAxisSum(child.margin, c.dir))
		}
	}
}

// distributeRemainingFreeSpace distributes any remaining free space.
func distributeRemainingFreeSpace(lines []flexLine, c *algoConstants) {
	for _, line := range lines {
		totalMainGap := sumAxisGaps(sizeMain(c.gap, c.dir), len(line.items))
		usedSpace := totalMainGap
		for _, child := range line.items {
			usedSpace += sizeMain(child.outerTargetSize, c.dir)
		}
		freeSpace := sizeMain(c.innerContainerSize, c.dir) - usedSpace
		numAutoMargins := 0
		for _, child := range line.items {
			if rectBoolMainStart(child.marginIsAuto, c.dir) {
				numAutoMargins++
			}
			if rectBoolMainEnd(child.marginIsAuto, c.dir) {
				numAutoMargins++
			}
		}

		if freeSpace > 0 && numAutoMargins > 0 {
			marginVal := freeSpace / float32(numAutoMargins)
			for _, child := range line.items {
				if rectBoolMainStart(child.marginIsAuto, c.dir) {
					if c.isRow {
						child.margin.Left = marginVal
					} else {
						child.margin.Top = marginVal
					}
				}
				if rectBoolMainEnd(child.marginIsAuto, c.dir) {
					if c.isRow {
						child.margin.Right = marginVal
					} else {
						child.margin.Bottom = marginVal
					}
				}
			}
			freeSpace = 0
		}

		numItems := len(line.items)
		layoutReverse := c.dir.isReverse()
		gap := sizeMain(c.gap, c.dir)
		rawJustify := AlignContentFlexStart
		if c.justifyContent != nil {
			rawJustify = *c.justifyContent
		}
		justifyMode := applyAlignmentFallback(freeSpace, numItems, rawJustify)

		justifyItem := func(i int, child *flexItem) {
			child.offsetMain = computeAlignmentOffset(freeSpace, numItems, gap, justifyMode, layoutReverse, i == 0)
		}
		if layoutReverse {
			for i := len(line.items) - 1; i >= 0; i-- {
				justifyItem(len(line.items)-1-i, line.items[i])
			}
		} else {
			for i, child := range line.items {
				justifyItem(i, child)
			}
		}
	}
}

// resolveCrossAxisAutoMargins resolves cross-axis auto margins.
func resolveCrossAxisAutoMargins(lines []flexLine, c *algoConstants) {
	for _, line := range lines {
		lineCrossSize := line.crossSize
		maxBaseline := float32(0)
		for _, child := range line.items {
			if child.baseline > maxBaseline {
				maxBaseline = child.baseline
			}
		}
		maxBaselineToBottom := float32(0)
		for _, child := range line.items {
			if child.participatesInBaselineAlignment(c.dir) {
				dist := sizeCross(child.outerTargetSize, c.dir) - child.baseline
				if dist > maxBaselineToBottom {
					maxBaselineToBottom = dist
				}
			}
		}

		for _, child := range line.items {
			freeSpace := lineCrossSize - sizeCross(child.outerTargetSize, c.dir)
			if rectBoolCrossStart(child.marginIsAuto, c.dir) && rectBoolCrossEnd(child.marginIsAuto, c.dir) {
				if c.isRow {
					child.margin.Top = freeSpace / 2
					child.margin.Bottom = freeSpace / 2
				} else {
					child.margin.Left = freeSpace / 2
					child.margin.Right = freeSpace / 2
				}
			} else if rectBoolCrossStart(child.marginIsAuto, c.dir) {
				if c.isRow {
					child.margin.Top = freeSpace
				} else {
					child.margin.Left = freeSpace
				}
			} else if rectBoolCrossEnd(child.marginIsAuto, c.dir) {
				if c.isRow {
					child.margin.Bottom = freeSpace
				} else {
					child.margin.Right = freeSpace
				}
			} else {
				child.offsetCross = alignFlexItemsAlongCrossAxis(child, freeSpace, maxBaseline, maxBaselineToBottom, c)
			}
		}
	}
}

// alignFlexItemsAlongCrossAxis aligns all flex items along the cross-axis.
func alignFlexItemsAlongCrossAxis(child *flexItem, freeSpace, maxBaseline, maxBaselineToBottom float32, c *algoConstants) float32 {
	crossAxisShouldReverse := c.isColumn && c.layoutDirection == directionRtl

	var alignKeyword alignItemsKeyword
	if child.alignSelf.isSafe() && freeSpace < 0 {
		alignKeyword = alignItemsStart
	} else {
		alignKeyword = child.alignSelf.Keyword
	}

	switch alignKeyword {
	case alignItemsStart:
		if crossAxisShouldReverse {
			return freeSpace
		}
		return 0
	case alignItemsFlexStart:
		if c.isWrapReverse != crossAxisShouldReverse {
			return freeSpace
		}
		return 0
	case alignItemsEnd:
		if crossAxisShouldReverse {
			return 0
		}
		return freeSpace
	case alignItemsFlexEnd:
		if c.isWrapReverse != crossAxisShouldReverse {
			return 0
		}
		return freeSpace
	case alignItemsCentre:
		return freeSpace / 2
	case alignItemsBaseline:
		if c.isRow {
			if c.isWrapReverse {
				lineCrossSize := freeSpace + sizeCross(child.outerTargetSize, c.dir)
				return lineCrossSize - maxBaselineToBottom - child.baseline
			}
			return maxBaseline - child.baseline
		}
		baselineColShouldReverse := crossAxisShouldReverse && !c.isWrap
		if c.isWrapReverse != baselineColShouldReverse {
			return freeSpace
		}
		return 0
	case alignItemsStretch:
		if c.isWrapReverse != crossAxisShouldReverse {
			return freeSpace
		}
		return 0
	}
	return 0
}

// determineContainerCrossSize determines the flex container's used cross size.
func determineContainerCrossSize(lines []flexLine, nodeSize Size[optF32], c *algoConstants) float32 {
	totalCrossGap := sumAxisGaps(sizeCross(c.gap, c.dir), len(lines))
	totalLineCross := float32(0)
	for _, line := range lines {
		totalLineCross += line.crossSize
	}

	pbSum := rectF32CrossAxisSum(c.contentBoxInset, c.dir)
	crossScrollbarGutter := pointCross(c.scrollbarGutter, c.dir)
	crossMin := sizeCross(c.minSize, c.dir)
	crossMax := sizeCross(c.maxSize, c.dir)

	outerContainerSize := f32MaybeClamp(
		sizeCross(nodeSize, c.dir).unwrapOr(totalLineCross+totalCrossGap+pbSum),
		crossMin, crossMax)
	outerContainerSize = f32Max(outerContainerSize, pbSum-crossScrollbarGutter)
	innerContainerSize := f32Max(outerContainerSize-pbSum, 0)

	sizeSetCross(&c.containerSize, c.dir, outerContainerSize)
	sizeSetCross(&c.innerContainerSize, c.dir, innerContainerSize)

	return totalLineCross
}

// alignFlexLinesPerAlignContent aligns all flex lines per align-content.
func alignFlexLinesPerAlignContent(lines []flexLine, c *algoConstants, totalCrossSize float32) {
	numLines := len(lines)
	gap := sizeCross(c.gap, c.dir)
	totalCrossGap := sumAxisGaps(gap, numLines)
	freeSpace := sizeCross(c.innerContainerSize, c.dir) - totalCrossSize - totalCrossGap

	alignMode := applyAlignmentFallback(freeSpace, numLines, c.alignContent)

	alignLine := func(i int, line *flexLine) {
		line.offsetCross = computeAlignmentOffset(freeSpace, numLines, gap, alignMode, c.isWrapReverse, i == 0)
	}
	if c.isWrapReverse {
		for i := len(lines) - 1; i >= 0; i-- {
			alignLine(len(lines)-1-i, &lines[i])
		}
	} else {
		for i := range lines {
			alignLine(i, &lines[i])
		}
	}
}

// finalLayoutPass does a final layout pass and collects the resulting layouts.
func finalLayoutPass(t flexboxTree, lines []flexLine, c *algoConstants) Rect[float32] {
	var totalOffsetCross float32
	if c.isColumn && c.layoutDirection == directionRtl {
		totalOffsetCross = c.containerSize.Width - rectCrossEnd(c.contentBoxInset, c.dir)
	} else {
		totalOffsetCross = rectCrossStart(c.contentBoxInset, c.dir)
	}

	var overflowRect Rect[float32]

	processLine := func(line *flexLine) {
		calculateLayoutLine(t, line, &totalOffsetCross, &overflowRect, c)
	}

	if c.isWrapReverse {
		for i := len(lines) - 1; i >= 0; i-- {
			processLine(&lines[i])
		}
	} else {
		for i := range lines {
			processLine(&lines[i])
		}
	}

	if c.isScrollContainer {
		if c.layoutDirection == directionRtl {
			overflowRect.Right += c.contentBoxInset.Left - c.border.Left - c.scrollbarGutter.X
		} else {
			overflowRect.Right += c.contentBoxInset.Right - c.border.Right - c.scrollbarGutter.X
		}
		overflowRect.Bottom += c.contentBoxInset.Bottom - c.border.Bottom - c.scrollbarGutter.Y
	}

	return overflowRect
}

// calculateLayoutLine calculates the layout for a flex line.
func calculateLayoutLine(t flexboxTree, line *flexLine, totalOffsetCross *float32, overflowRect *Rect[float32], c *algoConstants) {
	var totalOffsetMain float32
	if c.layoutDirection == directionRtl && c.dir.isRow() {
		totalOffsetMain = c.containerSize.Width - rectMainEnd(c.contentBoxInset, c.dir)
	} else {
		totalOffsetMain = rectMainStart(c.contentBoxInset, c.dir)
	}
	lineOffsetCross := line.offsetCross

	isRtlColumn := c.layoutDirection == directionRtl && c.dir.isColumn()
	if isRtlColumn {
		*totalOffsetCross -= lineOffsetCross + line.crossSize
	}

	processItem := func(item *flexItem) {
		calculateFlexItem(t, item, &totalOffsetMain, *totalOffsetCross, lineOffsetCross, overflowRect, c)
	}

	if c.dir.isReverse() {
		for i := len(line.items) - 1; i >= 0; i-- {
			processItem(line.items[i])
		}
	} else {
		for _, item := range line.items {
			processItem(item)
		}
	}

	if !isRtlColumn {
		*totalOffsetCross += lineOffsetCross + line.crossSize
	}
}

// calculateFlexItem calculates the layout for a flex item.
func calculateFlexItem(t flexboxTree, item *flexItem, totalOffsetMain *float32, totalOffsetCross, lineOffsetCross float32, overflowRect *Rect[float32], c *algoConstants) {
	itemKnownDefiniteness := itemKnownDimensionDefiniteness(c, item)
	layoutOutput := t.computeChildLayout(item.node, LayoutInput{
		RunMode:                       runPerformLayout,
		SizingMode:                    sizingContentSize,
		Axis:                          requestedBoth,
		KnownDimensions:               sizeMap(item.targetSize, func(v float32) optF32 { return some(v) }),
		KnownDimensionsAreDefinite:    itemKnownDefiniteness,
		ParentSize:                    c.nodeInnerSize,
		AvailableSpace:                sizeMap(c.containerSize, func(v float32) AvailableSpace { return definiteAvail(v) }),
		VerticalMarginsAreCollapsible: lineBoolFalse,
	})
	size := layoutOutput.Size

	isRtlRow := c.dir.isRow() && c.layoutDirection == directionRtl
	isRtlColumn := c.dir.isColumn() && c.layoutDirection == directionRtl

	var mainRelativeInset float32
	if isRtlRow {
		mainRelativeInset = rectMainEnd(item.inset, c.dir).unwrapOrFunc(func() float32 {
			if s := rectMainStart(item.inset, c.dir); s.isSome() {
				return -s.v
			}
			return 0
		})
	} else {
		mainRelativeInset = rectMainStart(item.inset, c.dir).unwrapOrFunc(func() float32 {
			if e := rectMainEnd(item.inset, c.dir); e.isSome() {
				return -e.v
			}
			return 0
		})
	}

	var crossRelativeInset float32
	if isRtlColumn {
		crossRelativeInset = func() float32 {
			if e := rectCrossEnd(item.inset, c.dir); e.isSome() {
				return -e.v
			}
			return rectCrossStart(item.inset, c.dir).unwrapOr(0)
		}()
	} else {
		crossRelativeInset = rectCrossStart(item.inset, c.dir).unwrapOrFunc(func() float32 {
			if e := rectCrossEnd(item.inset, c.dir); e.isSome() {
				return -e.v
			}
			return 0
		})
	}

	effectiveLineOffsetCross := float32(0)
	if !isRtlColumn {
		effectiveLineOffsetCross = lineOffsetCross
	}

	var offsetMain float32
	if isRtlRow {
		offsetMain = *totalOffsetMain - item.offsetMain - rectMainEnd(item.margin, c.dir) - mainRelativeInset - size.Width
	} else {
		offsetMain = *totalOffsetMain + item.offsetMain + rectMainStart(item.margin, c.dir) + mainRelativeInset
	}

	offsetCross := totalOffsetCross + item.offsetCross + effectiveLineOffsetCross + rectCrossStart(item.margin, c.dir) + crossRelativeInset

	if c.dir.isRow() {
		baselineOffsetCross := totalOffsetCross + item.offsetCross + effectiveLineOffsetCross + rectCrossStart(item.margin, c.dir)
		innerBaseline := layoutOutput.Baselines.first.unwrapOr(size.Height)
		if item.overflow.Y.isScrollContainer() {
			innerBaseline = f32Max(f32Min(innerBaseline, size.Height), 0)
		}
		item.baseline = baselineOffsetCross + innerBaseline
	} else {
		baselineOffsetMain := *totalOffsetMain + item.offsetMain + rectMainStart(item.margin, c.dir)
		innerBaseline := layoutOutput.Baselines.first.unwrapOr(size.Height)
		item.baseline = baselineOffsetMain + innerBaseline
	}

	var location Point[float32]
	if c.dir.isRow() {
		location = Point[float32]{X: offsetMain, Y: offsetCross}
	} else {
		location = Point[float32]{X: offsetCross, Y: offsetMain}
	}

	scrollbarSize := Size[float32]{
		Width:  scrollWidthFor(item.overflow.Y, item.scrollbarWidth),
		Height: scrollWidthFor(item.overflow.X, item.scrollbarWidth),
	}

	t.setUnroundedLayout(item.node, Layout{
		Order:                  item.order,
		Size:                   size,
		ScrollableOverflowRect: layoutOutput.ScrollableOverflowRect,
		ScrollbarSize:          scrollbarSize,
		Location:               location,
		Padding:                item.padding,
		Border:                 item.border,
		Margin:                 item.margin,
	})

	if isRtlRow {
		*totalOffsetMain -= item.offsetMain + rectF32MainAxisSum(item.margin, c.dir) + sizeMain(size, c.dir)
	} else {
		*totalOffsetMain += item.offsetMain + rectF32MainAxisSum(item.margin, c.dir) + sizeMain(size, c.dir)
	}

	// Scrollable overflow contribution.
	var contributionLocation Point[float32]
	if c.layoutDirection == directionRtl {
		contributionLocation = Point[float32]{
			X: c.containerSize.Width - (location.X + size.Width) - c.border.Right,
			Y: location.Y - c.border.Top,
		}
	} else {
		contributionLocation = Point[float32]{
			X: location.X - c.border.Left,
			Y: location.Y - c.border.Top,
		}
	}
	*overflowRect = rectF32Union(*overflowRect, computeScrollableOverflowContribution(
		contributionLocation, size, layoutOutput.ScrollableOverflowRect, item.overflow, item.contain, c.isScrollContainer))
}

// performAbsoluteLayoutOnAbsoluteChildren performs absolute layout on all
// absolutely positioned children.
func performAbsoluteLayoutOnAbsoluteChildren(t flexboxTree, node NodeID, c *algoConstants) Rect[float32] {
	containerWidth := c.containerSize.Width
	containerHeight := c.containerSize.Height
	insetRelativeSize := Size[float32]{
		Width:  c.containerSize.Width - rectF32HorizontalAxisSum(c.border) - c.scrollbarGutter.X,
		Height: c.containerSize.Height - rectF32VerticalAxisSum(c.border) - c.scrollbarGutter.Y,
	}

	var overflowRect Rect[float32]

	childCount := t.childCount(node)
	for order := 0; order < childCount; order++ {
		child := t.childID(node, order)
		childStyle := t.flexboxChildStyle(child)
		if childStyle.boxGenerationMode() == boxGenNone || childStyle.positionVal() != positionAbsolute {
			continue
		}

		overflow := childStyle.overflowVal()
		contain := childStyle.containVal()
		scrollbarWidth := childStyle.scrollbarWidthVal()
		aspectRatio := childStyle.aspectRatioVal()
		alignSelf := c.alignItems
		if as := childStyle.alignSelfVal(); as != nil {
			alignSelf = *as
		}
		alignSelf = alignSelf.resolveSelfRelative(childStyle.directionVal(), c.layoutDirection, c.isColumn)

		marginOpt := rectMap(childStyle.marginVal(), func(lpa LengthPercentageAuto) optF32 {
			return lpa.resolveToOption(some(insetRelativeSize.Width))
		})
		padding := rectLPResolveOrZeroSize(childStyle.paddingVal(), Size[optF32]{Width: some(insetRelativeSize.Width), Height: some(insetRelativeSize.Height)})
		border := rectLPResolveOrZeroSize(childStyle.borderVal(), Size[optF32]{Width: some(insetRelativeSize.Width), Height: some(insetRelativeSize.Height)})
		pbSum := rectF32SumAxes(rectF32Add(padding, border))
		var boxSizingAdj Size[float32]
		if childStyle.boxSizingVal() == boxSizingContentBox {
			boxSizingAdj = pbSum
		}

		// Resolve inset.
		left := lpaMaybeResolve(childStyle.insetVal().Left, some(insetRelativeSize.Width))
		right := lpaMaybeResolve(childStyle.insetVal().Right, some(insetRelativeSize.Width))
		top := lpaMaybeResolve(childStyle.insetVal().Top, some(insetRelativeSize.Height))
		bottom := lpaMaybeResolve(childStyle.insetVal().Bottom, some(insetRelativeSize.Height))

		// Compute known dimensions.
		sizeStyle := childStyle.sizeVal()
		styleSize := sizeOptMaybeAdd(
			sizeOptMaybeApplyAspectRatio(sizeDimMaybeResolve(sizeStyle, Size[optF32]{Width: some(insetRelativeSize.Width), Height: some(insetRelativeSize.Height)}), aspectRatio),
			sizeF32ToOpt(boxSizingAdj))
		minSize := sizeOptMaybeMax(
			sizeOptOr(
				sizeOptMaybeAdd(
					sizeOptMaybeApplyAspectRatio(sizeLPAMaybeResolve(childStyle.minSizeVal(), Size[optF32]{Width: some(insetRelativeSize.Width), Height: some(insetRelativeSize.Height)}), aspectRatio),
					sizeF32ToOpt(boxSizingAdj)),
				sizeF32ToOpt(pbSum)),
			sizeF32ToOpt(pbSum))
		maxSize := sizeOptMaybeAdd(
			sizeOptMaybeApplyAspectRatio(sizeLPAMaybeResolve(childStyle.maxSizeVal(), Size[optF32]{Width: some(insetRelativeSize.Width), Height: some(insetRelativeSize.Height)}), aspectRatio),
			sizeF32ToOpt(boxSizingAdj))
		known := sizeOptMaybeClamp(styleSize, minSize, maxSize)

		// Resolve sizing keywords.
		if sizeStyle.Width.isSizingKeyword() || sizeStyle.Height.isSizingKeyword() {
			resolveAbsoluteSizingKeywords(t, child, &known, sizeStyle, insetRelativeSize,
				Rect[optF32]{Left: left, Right: right, Top: top, Bottom: bottom}, marginOpt, sizingContentSize)
			known = sizeOptMaybeClamp(
				sizeOptMaybeApplyAspectRatio(known, aspectRatio),
				minSize, maxSize)
		}

		// Fill in width from left/right.
		if !known.Width.isSome() && left.isSome() && right.isSome() {
			newWidth := insetRelativeSize.Width - marginOpt.Left.unwrapOr(0) - marginOpt.Right.unwrapOr(0) - left.v - right.v
			known.Width = some(f32Max(newWidth, 0))
			known = sizeOptMaybeClamp(sizeOptMaybeApplyAspectRatio(known, aspectRatio), minSize, maxSize)
		}
		// Fill in height from top/bottom.
		if !known.Height.isSome() && top.isSome() && bottom.isSome() {
			newHeight := insetRelativeSize.Height - marginOpt.Top.unwrapOr(0) - marginOpt.Bottom.unwrapOr(0) - top.v - bottom.v
			known.Height = some(f32Max(newHeight, 0))
			known = sizeOptMaybeClamp(sizeOptMaybeApplyAspectRatio(known, aspectRatio), minSize, maxSize)
		}

		var finalSize Size[float32]
		if known.Width.isSome() && known.Height.isSome() {
			finalSize = Size[float32]{Width: known.Width.v, Height: known.Height.v}
		} else {
			measured := measureChildSizeBoth(t, child, known, c.nodeInnerSize,
				Size[AvailableSpace]{
					Width:  definiteAvail(f32MaybeClamp(containerWidth, minSize.Width, maxSize.Width)),
					Height: definiteAvail(f32MaybeClamp(containerHeight, minSize.Height, maxSize.Height)),
				}, sizingContentSize, lineBoolFalse)
			finalSize = sizeOptUnwrapOr(known, measured)
		}
		finalSize = sizeOptUnwrapOr(sizeOptMaybeClamp(sizeF32ToOpt(finalSize), minSize, maxSize), finalSize)

		layoutOutput := performChildLayout(t, child,
			sizeF32ToOpt(finalSize), c.nodeInnerSize,
			Size[AvailableSpace]{
				Width:  definiteAvail(f32MaybeClamp(containerWidth, minSize.Width, maxSize.Width)),
				Height: definiteAvail(f32MaybeClamp(containerHeight, minSize.Height, maxSize.Height)),
			}, sizingContentSize, lineBoolFalse)

		nonAutoMargin := rectMap(marginOpt, func(m optF32) float32 { return m.unwrapOr(0) })

		freeSpace := Size[float32]{
			Width:  f32Max(containerWidth-finalSize.Width-rectF32HorizontalAxisSum(nonAutoMargin), 0),
			Height: f32Max(containerHeight-finalSize.Height-rectF32VerticalAxisSum(nonAutoMargin), 0),
		}

		// Expand auto margins.
		autoMarginW := 0
		if !marginOpt.Left.isSome() {
			autoMarginW++
		}
		if !marginOpt.Right.isSome() {
			autoMarginW++
		}
		autoMarginH := 0
		if !marginOpt.Top.isSome() {
			autoMarginH++
		}
		if !marginOpt.Bottom.isSome() {
			autoMarginH++
		}
		autoMarginSize := Size[float32]{Width: 0, Height: 0}
		if autoMarginW > 0 && left.isSome() && right.isSome() {
			autoMarginSize.Width = freeSpace.Width / float32(autoMarginW)
		}
		if autoMarginH > 0 && top.isSome() && bottom.isSome() {
			autoMarginSize.Height = freeSpace.Height / float32(autoMarginH)
		}
		resolvedMargin := Rect[float32]{
			Left:   marginOpt.Left.unwrapOr(autoMarginSize.Width),
			Right:  marginOpt.Right.unwrapOr(autoMarginSize.Width),
			Top:    marginOpt.Top.unwrapOr(autoMarginSize.Height),
			Bottom: marginOpt.Bottom.unwrapOr(autoMarginSize.Height),
		}

		// Determine flex-relative insets.
		var startMain, endMain, startCross, endCross optF32
		if c.isRow {
			startMain, endMain = left, right
			startCross, endCross = top, bottom
		} else {
			startMain, endMain = top, bottom
			startCross, endCross = left, right
		}
		mainAxisIsHorizontal := c.isRow
		crossAxisIsHorizontal := !c.isRow
		mainIsRtl := mainAxisIsHorizontal && c.layoutDirection == directionRtl
		crossIsRtl := crossAxisIsHorizontal && c.layoutDirection == directionRtl
		mainAxisFlexStartReversed := c.dir.isReverse() != mainIsRtl
		crossAxisFlexStartReversed := c.isWrapReverse != crossIsRtl
		mainStartScrollbarOffset := float32(0)
		if mainIsRtl {
			mainStartScrollbarOffset = pointMain(c.scrollbarGutter, c.dir)
		}
		crossStartScrollbarOffset := float32(0)
		if crossIsRtl {
			crossStartScrollbarOffset = pointCross(c.scrollbarGutter, c.dir)
		}
		mainEndScrollbarOffset := float32(0)
		if !mainIsRtl {
			mainEndScrollbarOffset = pointMain(c.scrollbarGutter, c.dir)
		}
		crossEndScrollbarOffset := float32(0)
		if !crossIsRtl {
			crossEndScrollbarOffset = pointCross(c.scrollbarGutter, c.dir)
		}

		// Apply main-axis alignment.
		var offsetMain float32
		if startMain.isSome() || endMain.isSome() {
			if mainIsRtl && endMain.isSome() {
				offsetMain = sizeMain(c.containerSize, c.dir) - rectMainEnd(c.border, c.dir) - mainEndScrollbarOffset -
					sizeMain(finalSize, c.dir) - endMain.unwrapOr(0) - rectMainEnd(resolvedMargin, c.dir)
			} else if startMain.isSome() {
				offsetMain = startMain.v + rectMainStart(c.border, c.dir) + mainStartScrollbarOffset + rectMainStart(resolvedMargin, c.dir)
			} else {
				offsetMain = sizeMain(c.containerSize, c.dir) - rectMainEnd(c.border, c.dir) - mainEndScrollbarOffset -
					sizeMain(finalSize, c.dir) - endMain.unwrapOr(0) - rectMainEnd(resolvedMargin, c.dir)
			}
		} else {
			rawJustify := AlignContentFlexStart
			if c.justifyContent != nil {
				rawJustify = *c.justifyContent
			}
			justifyKw := rawJustify.Keyword
			startPosition := true
			switch justifyKw {
			case alignContentStart:
				startPosition = !mainIsRtl
			case alignContentEnd:
				startPosition = mainIsRtl
			}
			switch justifyKw {
			case alignContentSpaceBetween, alignContentStretch, alignContentFlexStart:
				if !mainAxisFlexStartReversed {
					offsetMain = rectMainStart(c.contentBoxInset, c.dir) + rectMainStart(resolvedMargin, c.dir)
				} else {
					offsetMain = sizeMain(c.containerSize, c.dir) - rectMainEnd(c.contentBoxInset, c.dir) - sizeMain(finalSize, c.dir) - rectMainEnd(resolvedMargin, c.dir)
				}
			case alignContentStart, alignContentEnd:
				if startPosition {
					offsetMain = rectMainStart(c.contentBoxInset, c.dir) + rectMainStart(resolvedMargin, c.dir)
				} else {
					offsetMain = sizeMain(c.containerSize, c.dir) - rectMainEnd(c.contentBoxInset, c.dir) - sizeMain(finalSize, c.dir) - rectMainEnd(resolvedMargin, c.dir)
				}
			case alignContentFlexEnd:
				if !mainAxisFlexStartReversed {
					offsetMain = sizeMain(c.containerSize, c.dir) - rectMainEnd(c.contentBoxInset, c.dir) - sizeMain(finalSize, c.dir) - rectMainEnd(resolvedMargin, c.dir)
				} else {
					offsetMain = rectMainStart(c.contentBoxInset, c.dir) + rectMainStart(resolvedMargin, c.dir)
				}
			case alignContentSpaceEvenly, alignContentSpaceAround, alignContentCentre:
				offsetMain = (sizeMain(c.containerSize, c.dir) + rectMainStart(c.contentBoxInset, c.dir) - rectMainEnd(c.contentBoxInset, c.dir) -
					sizeMain(finalSize, c.dir) + rectMainStart(resolvedMargin, c.dir) - rectMainEnd(resolvedMargin, c.dir)) / 2
			}
		}

		// Apply cross-axis alignment.
		var offsetCross float32
		if startCross.isSome() || endCross.isSome() {
			if crossIsRtl && endCross.isSome() {
				offsetCross = sizeCross(c.containerSize, c.dir) - rectCrossEnd(c.border, c.dir) - crossEndScrollbarOffset -
					sizeCross(finalSize, c.dir) - endCross.unwrapOr(0) - rectCrossEnd(resolvedMargin, c.dir)
			} else if startCross.isSome() {
				offsetCross = startCross.v + rectCrossStart(c.border, c.dir) + crossStartScrollbarOffset + rectCrossStart(resolvedMargin, c.dir)
			} else {
				offsetCross = sizeCross(c.containerSize, c.dir) - rectCrossEnd(c.border, c.dir) - crossEndScrollbarOffset -
					sizeCross(finalSize, c.dir) - endCross.unwrapOr(0) - rectCrossEnd(resolvedMargin, c.dir)
			}
		} else {
			crossOverflows := sizeCross(finalSize, c.dir)+rectF32CrossAxisSum(resolvedMargin, c.dir) >
				sizeCross(c.containerSize, c.dir)-rectF32CrossAxisSum(c.contentBoxInset, c.dir)
			crossKw := resolveSelfAlignmentSafety(alignSelf, crossOverflows)
			startPosition := true
			switch crossKw {
			case alignItemsStart, alignItemsBaseline:
				startPosition = !crossIsRtl
			case alignItemsEnd:
				startPosition = crossIsRtl
			}
			switch crossKw {
			case alignItemsStart, alignItemsEnd, alignItemsBaseline:
				if startPosition {
					offsetCross = rectCrossStart(c.contentBoxInset, c.dir) + rectCrossStart(resolvedMargin, c.dir)
				} else {
					offsetCross = sizeCross(c.containerSize, c.dir) - rectCrossEnd(c.contentBoxInset, c.dir) - sizeCross(finalSize, c.dir) - rectCrossEnd(resolvedMargin, c.dir)
				}
			case alignItemsStretch, alignItemsFlexStart:
				if !crossAxisFlexStartReversed {
					offsetCross = rectCrossStart(c.contentBoxInset, c.dir) + rectCrossStart(resolvedMargin, c.dir)
				} else {
					offsetCross = sizeCross(c.containerSize, c.dir) - rectCrossEnd(c.contentBoxInset, c.dir) - sizeCross(finalSize, c.dir) - rectCrossEnd(resolvedMargin, c.dir)
				}
			case alignItemsFlexEnd:
				if !crossAxisFlexStartReversed {
					offsetCross = sizeCross(c.containerSize, c.dir) - rectCrossEnd(c.contentBoxInset, c.dir) - sizeCross(finalSize, c.dir) - rectCrossEnd(resolvedMargin, c.dir)
				} else {
					offsetCross = rectCrossStart(c.contentBoxInset, c.dir) + rectCrossStart(resolvedMargin, c.dir)
				}
			case alignItemsCentre:
				offsetCross = (sizeCross(c.containerSize, c.dir) + rectCrossStart(c.contentBoxInset, c.dir) - rectCrossEnd(c.contentBoxInset, c.dir) -
					sizeCross(finalSize, c.dir) + rectCrossStart(resolvedMargin, c.dir) - rectCrossEnd(resolvedMargin, c.dir)) / 2
			}
		}

		var location Point[float32]
		if c.isRow {
			location = Point[float32]{X: offsetMain, Y: offsetCross}
		} else {
			location = Point[float32]{X: offsetCross, Y: offsetMain}
		}
		scrollbarSize := Size[float32]{
			Width:  scrollWidthFor(overflow.Y, scrollbarWidth),
			Height: scrollWidthFor(overflow.X, scrollbarWidth),
		}
		t.setUnroundedLayout(child, Layout{
			Order:                  uint32(order),
			Size:                   finalSize,
			ScrollableOverflowRect: layoutOutput.ScrollableOverflowRect,
			ScrollbarSize:          scrollbarSize,
			Location:               location,
			Padding:                padding,
			Border:                 border,
			Margin:                 resolvedMargin,
		})

		// Scrollable overflow contribution.
		absoluteAreaOffset := Point[float32]{
			X: c.border.Left + func() float32 {
				if c.layoutDirection == directionRtl {
					return c.scrollbarGutter.X
				}
				return 0
			}(),
			Y: c.border.Top,
		}
		relativeLocation := Point[float32]{X: location.X - absoluteAreaOffset.X, Y: location.Y - absoluteAreaOffset.Y}
		var contributionLocation Point[float32]
		if c.layoutDirection == directionRtl {
			contributionLocation = Point[float32]{X: insetRelativeSize.Width - relativeLocation.X - finalSize.Width, Y: relativeLocation.Y}
		} else {
			contributionLocation = relativeLocation
		}
		overflowRect = rectF32Union(overflowRect, computeScrollableOverflowContribution(
			contributionLocation, finalSize, layoutOutput.ScrollableOverflowRect, overflow, contain, c.isScrollContainer))
	}

	return overflowRect
}

// sumAxisGaps computes the total space taken up by gaps.
func sumAxisGaps(gap float32, numItems int) float32 {
	if numItems <= 1 {
		return 0
	}
	return gap * float32(numItems-1)
}

// dividedCrossSpace divides the cross available space (no-op without balance).
func (c *algoConstants) dividedCrossSpace(crossAvail float32) float32 {
	return crossAvail
}
