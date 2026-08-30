// Ported from Taffy src/tree/layout.rs (MIT).
//
// The layout input/output types and the final Layout struct. Block-layout-only
// margin-collapse metadata is kept (it is cheap and the cache logic references
// it) but is unused by the flexbox path, which always reports zero margins and
// margins_can_collapse_through=false.
package layout

// runMode selects between a full layout, a size-only computation, or hidden
// layout.
type runMode uint8

const (
	runPerformLayout runMode = iota
	runComputeSize
	runPerformHiddenLayout
)

// sizingMode selects whether inherent size styles are taken into account.
type sizingMode uint8

const (
	sizingContentSize sizingMode = iota
	sizingInherentSize
)

// requestedAxis is the axis the parent needs a size for.
type requestedAxis uint8

const (
	requestedHorizontal requestedAxis = iota
	requestedVertical
	requestedBoth
)

// fromAbsoluteAxis converts an absolute axis to a requested axis.
func fromAbsoluteAxis(a absoluteAxis) requestedAxis {
	if a == absoluteHorizontal {
		return requestedHorizontal
	}
	return requestedVertical
}

// collapsibleMarginSet is the block-layout margin-collapse accumulator. The
// flexbox path does not collapse margins, so it always reports zero.
type collapsibleMarginSet struct {
	positive float32
	negative float32
}

var collapsibleMarginZero = collapsibleMarginSet{}

func collapsibleMarginFromMargin(m float32) collapsibleMarginSet {
	if m >= 0 {
		return collapsibleMarginSet{positive: m}
	}
	return collapsibleMarginSet{negative: m}
}

func (c collapsibleMarginSet) collapseWithMargin(m float32) collapsibleMarginSet {
	if m >= 0 {
		return collapsibleMarginSet{positive: f32Max(c.positive, m), negative: c.negative}
	}
	return collapsibleMarginSet{positive: c.positive, negative: f32Min(c.negative, m)}
}

func (c collapsibleMarginSet) collapseWithSet(other collapsibleMarginSet) collapsibleMarginSet {
	return collapsibleMarginSet{
		positive: f32Max(c.positive, other.positive),
		negative: f32Min(c.negative, other.negative),
	}
}

func (c collapsibleMarginSet) resolve() float32 { return c.positive + c.negative }

// LayoutInput is the set of constraints and hints passed from a parent to a
// child during layout.
type LayoutInput struct {
	RunMode                       runMode
	SizingMode                    sizingMode
	Axis                          requestedAxis
	KnownDimensions               Size[optF32]
	KnownDimensionsAreDefinite    Size[bool]
	ParentSize                    Size[optF32]
	AvailableSpace                Size[AvailableSpace]
	VerticalMarginsAreCollapsible Line[bool]
}

// layoutInputHidden is the LayoutInput used to request hidden layout.
var layoutInputHidden = LayoutInput{
	RunMode:                       runPerformHiddenLayout,
	KnownDimensions:               sizeNone,
	KnownDimensionsAreDefinite:    Size[bool]{Width: true, Height: true},
	ParentSize:                    sizeNone,
	AvailableSpace:                sizeMaxContent,
	SizingMode:                    sizingInherentSize,
	Axis:                          requestedBoth,
	VerticalMarginsAreCollapsible: lineBoolFalse,
}

// baselines holds the first and last baseline of a node in the horizontal axis.
type baselines struct {
	first optF32
	last  optF32
}

var baselinesNone = baselines{}

func baselinesFromFirst(first optF32) baselines { return baselines{first: first} }

// LayoutOutput is the result of laying out a single node.
type LayoutOutput struct {
	Size                      Size[float32]
	ScrollableOverflowRect    Rect[float32]
	Baselines                 baselines
	TopMargin                 collapsibleMarginSet
	BottomMargin              collapsibleMarginSet
	MarginsCanCollapseThrough bool
}

var layoutOutputHidden = LayoutOutput{
	Size:                      sizeZeroF32,
	ScrollableOverflowRect:    rectZeroF32,
	Baselines:                 baselinesNone,
	TopMargin:                 collapsibleMarginZero,
	BottomMargin:              collapsibleMarginZero,
	MarginsCanCollapseThrough: false,
}

// layoutOutputFromOuterSize constructs a LayoutOutput from just the size.
func layoutOutputFromOuterSize(s Size[float32]) LayoutOutput {
	return layoutOutputFromSizes(s, rectZeroF32)
}

// layoutOutputFromSizes constructs a LayoutOutput from size and overflow rect.
func layoutOutputFromSizes(s Size[float32], r Rect[float32]) LayoutOutput {
	return LayoutOutput{
		Size:                   s,
		ScrollableOverflowRect: r,
		Baselines:              baselinesNone,
	}
}

// layoutOutputFromSizesAndBaselines constructs a LayoutOutput from size, overflow
// rect and baselines.
func layoutOutputFromSizesAndBaselines(s Size[float32], r Rect[float32], b baselines) LayoutOutput {
	return LayoutOutput{
		Size:                   s,
		ScrollableOverflowRect: r,
		Baselines:              b,
	}
}

// Layout is the final result of a layout algorithm for a single node.
type Layout struct {
	Order                  u32
	Location               Point[float32]
	Size                   Size[float32]
	ScrollableOverflowRect Rect[float32]
	ScrollbarSize          Size[float32]
	Border                 Rect[float32]
	Padding                Rect[float32]
	Margin                 Rect[float32]
}

// u32 is an alias for uint32 used by Taffy's Layout.order field.
type u32 = uint32

// newLayout creates a zero Layout.
func newLayout() Layout {
	return Layout{
		Order:                  0,
		Location:               pointZeroF32,
		Size:                   sizeZeroF32,
		ScrollableOverflowRect: rectZeroF32,
		ScrollbarSize:          sizeZeroF32,
		Border:                 rectZeroF32,
		Padding:                rectZeroF32,
		Margin:                 rectZeroF32,
	}
}

// newLayoutWithOrder creates a zero Layout with the supplied order.
func newLayoutWithOrder(order uint32) Layout {
	l := newLayout()
	l.Order = order
	return l
}

// contentBoxWidth returns the width of the node's content box.
func (l Layout) contentBoxWidth() float32 {
	return l.Size.Width - l.Padding.Left - l.Padding.Right - l.Border.Left - l.Border.Right
}

// contentBoxHeight returns the height of the node's content box.
func (l Layout) contentBoxHeight() float32 {
	return l.Size.Height - l.Padding.Top - l.Padding.Bottom - l.Border.Top - l.Border.Bottom
}

// contentBoxSize returns the size of the node's content box.
func (l Layout) contentBoxSize() Size[float32] {
	return Size[float32]{Width: l.contentBoxWidth(), Height: l.contentBoxHeight()}
}

// contentBoxX returns the x offset of the node's content box relative to its
// parent's border box.
func (l Layout) contentBoxX() float32 {
	return l.Location.X + l.Border.Left + l.Padding.Left
}

// contentBoxY returns the y offset of the node's content box relative to its
// parent's border box.
func (l Layout) contentBoxY() float32 {
	return l.Location.Y + l.Border.Top + l.Padding.Top
}

// scrollWidth returns the maximum horizontal scroll offset.
func (l Layout) scrollWidth() float32 {
	return f32Max(0,
		l.ScrollableOverflowRect.Right+f32Min(l.ScrollbarSize.Width, l.Size.Width)-l.Size.Width+
			l.Border.Left+l.Border.Right)
}

// scrollHeight returns the maximum vertical scroll offset.
func (l Layout) scrollHeight() float32 {
	return f32Max(0,
		l.ScrollableOverflowRect.Bottom+f32Min(l.ScrollbarSize.Height, l.Size.Height)-l.Size.Height+
			l.Border.Top+l.Border.Bottom)
}
