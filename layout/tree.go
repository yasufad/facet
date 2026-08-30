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

// LayoutTree is the interface the layout algorithms consume. It combines
// Taffy's TraversePartialTree, LayoutPartialTree, CacheTree and the
// flexbox-specific style getters. RoundTree and PrintTree are split off because
// they require full recursive traversal and are only used by the rounding and
// printing entry points.
type LayoutTree interface {
	// TraversePartialTree
	ChildCount(parent NodeID) int
	ChildID(parent NodeID, index int) NodeID

	// LayoutPartialTree
	CoreContainerStyle(id NodeID) *Style
	SetUnroundedLayout(id NodeID, l Layout)
	ComputeChildLayout(id NodeID, in LayoutInput) LayoutOutput

	// CacheTree
	CacheGet(id NodeID, in *LayoutInput) (LayoutOutput, bool)
	CacheStore(id NodeID, in *LayoutInput, out LayoutOutput)
	CacheClear(id NodeID)
}

// FlexboxTree extends LayoutTree with the flexbox container/item style getters.
// Separating it lets non-flexbox algorithms skip implementing it.
type FlexboxTree interface {
	LayoutTree
	FlexboxContainerStyle(id NodeID) *Style
	FlexboxChildStyle(childID NodeID) *Style
}

// RoundTree is the interface for rounding a tree of unrounded layouts to
// integer pixels. It requires full recursive access.
type RoundTree interface {
	ChildCount(parent NodeID) int
	ChildID(parent NodeID, index int) NodeID
	GetUnroundedLayout(id NodeID) Layout
	SetFinalLayout(id NodeID, l Layout)
}

// measureChildSize computes the size of a node along a single axis, used by
// parents that need to know a child's extent before laying it out fully. This
// mirrors Taffy's LayoutPartialTreeExt::measure_child_size.
func measureChildSize(
	t LayoutTree,
	id NodeID,
	known Size[optF32],
	parentSize Size[optF32],
	avail Size[availableSpace],
	sizing sizingMode,
	axis absoluteAxis,
	verticalMarginsCollapsible Line[bool],
) float32 {
	out := t.ComputeChildLayout(id, LayoutInput{
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
	t LayoutTree,
	id NodeID,
	known Size[optF32],
	parentSize Size[optF32],
	avail Size[availableSpace],
	sizing sizingMode,
	verticalMarginsCollapsible Line[bool],
) Size[float32] {
	out := t.ComputeChildLayout(id, LayoutInput{
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
	t LayoutTree,
	id NodeID,
	known Size[optF32],
	parentSize Size[optF32],
	avail Size[availableSpace],
	sizing sizingMode,
	verticalMarginsCollapsible Line[bool],
) LayoutOutput {
	return t.ComputeChildLayout(id, LayoutInput{
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
