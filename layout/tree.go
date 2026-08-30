// Ported from Taffy src/tree/traits.rs and src/tree/taffy_tree.rs (MIT).
//
// Taffy's traits (TraversePartialTree, LayoutPartialTree, CacheTree, RoundTree,
// PrintTree, LayoutFlexboxContainer) become a single Go interface, LayoutTree,
// that the algorithm package consumes. The default TaffyTree storage implements
// it directly. The measure-function hook is a callback set on the tree rather
// than a trait method, matching Taffy's TaffyView pattern.
package layout

// MeasureFunction computes the intrinsic size of a leaf node. It receives the
// layout inputs, the node id, the node's context (if any) and the node's style.
// Text nodes use this to report their own size; nodes without a measure
// function fall back to a zero size.
type MeasureFunction func(inputs LayoutInput, id NodeID, ctx any, style *Style) LayoutOutput

// layoutTree is the interface the layout algorithms consume. It combines
// Taffy's TraversePartialTree, LayoutPartialTree, CacheTree and the
// flexbox-specific style getters. roundTree and PrintTree are split off because
// they require full recursive traversal and are only used by the rounding and
// printing entry points.
type layoutTree interface {
	// TraversePartialTree
	childCount(parent NodeID) int
	childID(parent NodeID, index int) NodeID

	// LayoutPartialTree
	coreContainerStyle(id NodeID) *Style
	setUnroundedLayout(id NodeID, l Layout)
	computeChildLayout(id NodeID, in LayoutInput) LayoutOutput

	// CacheTree
	cacheGet(id NodeID, in *LayoutInput) (LayoutOutput, bool)
	cacheStore(id NodeID, in *LayoutInput, out LayoutOutput)
	cacheClear(id NodeID)
}

// flexboxTree extends layoutTree with the flexbox container/item style getters.
// Separating it lets non-flexbox algorithms skip implementing it.
type flexboxTree interface {
	layoutTree
	flexboxContainerStyle(id NodeID) *Style
	flexboxChildStyle(childID NodeID) *Style
}

// roundTree is the interface for rounding a tree of unrounded layouts to
// integer pixels. It requires full recursive access.
type roundTree interface {
	childCount(parent NodeID) int
	childID(parent NodeID, index int) NodeID
	getUnroundedLayout(id NodeID) Layout
	setFinalLayout(id NodeID, l Layout)
}

// measureChildSize computes the size of a node along a single axis, used by
// parents that need to know a child's extent before laying it out fully. This
// mirrors Taffy's LayoutPartialTreeExt::measure_child_size.
func measureChildSize(
	t layoutTree,
	id NodeID,
	known Size[optF32],
	parentSize Size[optF32],
	avail Size[AvailableSpace],
	sizing sizingMode,
	axis absoluteAxis,
	verticalMarginsCollapsible Line[bool],
) float32 {
	out := t.computeChildLayout(id, LayoutInput{
		KnownDimensions:               known,
		KnownDimensionsAreDefinite:    Size[bool]{Width: true, Height: true},
		ParentSize:                    parentSize,
		AvailableSpace:                avail,
		SizingMode:                    sizing,
		Axis:                          fromAbsoluteAxis(axis),
		RunMode:                       runComputeSize,
		VerticalMarginsAreCollapsible: verticalMarginsCollapsible,
	})
	return sizeGetAbs(out.Size, axis)
}

// measureChildSizeBoth computes the size of a node along both axes.
func measureChildSizeBoth(
	t layoutTree,
	id NodeID,
	known Size[optF32],
	parentSize Size[optF32],
	avail Size[AvailableSpace],
	sizing sizingMode,
	verticalMarginsCollapsible Line[bool],
) Size[float32] {
	out := t.computeChildLayout(id, LayoutInput{
		KnownDimensions:               known,
		KnownDimensionsAreDefinite:    Size[bool]{Width: true, Height: true},
		ParentSize:                    parentSize,
		AvailableSpace:                avail,
		SizingMode:                    sizing,
		Axis:                          requestedBoth,
		RunMode:                       runComputeSize,
		VerticalMarginsAreCollapsible: verticalMarginsCollapsible,
	})
	return out.Size
}

// performChildLayout requests a full layout of a node's children.
func performChildLayout(
	t layoutTree,
	id NodeID,
	known Size[optF32],
	parentSize Size[optF32],
	avail Size[AvailableSpace],
	sizing sizingMode,
	verticalMarginsCollapsible Line[bool],
) LayoutOutput {
	return t.computeChildLayout(id, LayoutInput{
		KnownDimensions:               known,
		KnownDimensionsAreDefinite:    Size[bool]{Width: true, Height: true},
		ParentSize:                    parentSize,
		AvailableSpace:                avail,
		SizingMode:                    sizing,
		Axis:                          requestedBoth,
		RunMode:                       runPerformLayout,
		VerticalMarginsAreCollapsible: verticalMarginsCollapsible,
	})
}
