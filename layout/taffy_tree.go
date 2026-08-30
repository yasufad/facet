// Ported from Taffy src/tree/taffy_tree.rs (MIT).
//
// The default node-tree storage. Taffy uses slotmap for stable IDs across
// removals; Go uses a monotonic counter and a map keyed by uint64, which gives
// the same stable-ID property for the operations the layout path uses. Removal
// deletes the entry rather than leaving a slot hole, which is fine because the
// layout algorithms never see a removed node.
package layout

// nodeData is the per-node storage in a TaffyTree.
type nodeData struct {
	style           Style
	unroundedLayout Layout
	finalLayout     Layout
	hasContext      bool
	cache           Cache
}

func newNodeData(style Style) nodeData {
	return nodeData{
		style:           style,
		cache:           NewCache(),
		unroundedLayout: newLayout(),
		finalLayout:     newLayout(),
	}
}

// markDirty clears the node's cache and reports whether it was already empty.
func (n *nodeData) markDirty() clearState { return n.cache.Clear() }

// TaffyTree is the default tree of UI nodes. It stores nodes, their children,
// their parents, and optional per-node context (used by the measure function).
type TaffyTree struct {
	nodes       map[uint64]*nodeData
	children    map[uint64][]NodeID
	parents     map[uint64]*NodeID
	contexts    map[uint64]any
	nextID      uint64
	useRounding bool
}

// NewTaffyTree creates a new empty TaffyTree with default rounding enabled.
func NewTaffyTree() *TaffyTree {
	return &TaffyTree{
		nodes:       make(map[uint64]*nodeData),
		children:    make(map[uint64][]NodeID),
		parents:     make(map[uint64]*NodeID),
		contexts:    make(map[uint64]any),
		useRounding: true,
	}
}

// EnableRounding turns on layout value rounding (the default).
func (t *TaffyTree) EnableRounding() { t.useRounding = true }

// DisableRounding turns off layout value rounding.
func (t *TaffyTree) DisableRounding() { t.useRounding = false }

// NewLeaf creates and adds an unattached leaf node, returning its id.
func (t *TaffyTree) NewLeaf(style Style) NodeID {
	id := t.alloc(style)
	return id
}

// NewLeafWithContext creates a leaf node carrying per-node context for the
// measure function.
func (t *TaffyTree) NewLeafWithContext(style Style, ctx any) NodeID {
	id := t.alloc(style)
	t.nodes[id.raw].hasContext = true
	t.contexts[id.raw] = ctx
	return id
}

// NewWithChildren creates a node with the given children.
func (t *TaffyTree) NewWithChildren(style Style, children []NodeID) NodeID {
	id := t.alloc(style)
	for _, c := range children {
		t.parents[c.raw] = &id
	}
	t.children[id.raw] = append([]NodeID{}, children...)
	return id
}

func (t *TaffyTree) alloc(style Style) NodeID {
	id := NodeID{raw: t.nextID}
	t.nextID++
	nd := newNodeData(style)
	t.nodes[id.raw] = &nd
	t.children[id.raw] = nil
	t.parents[id.raw] = nil
	return id
}

// AddChild attaches a child under the parent and marks the parent dirty.
func (t *TaffyTree) AddChild(parent, child NodeID) {
	p := parent
	t.parents[child.raw] = &p
	t.children[parent.raw] = append(t.children[parent.raw], child)
	t.MarkDirty(parent)
}

// SetChildren replaces the parent's children.
func (t *TaffyTree) SetChildren(parent NodeID, children []NodeID) {
	for _, c := range t.children[parent.raw] {
		t.parents[c.raw] = nil
	}
	for _, c := range children {
		if prev := t.parents[c.raw]; prev != nil {
			t.removeChildFromParent(*prev, c)
		}
		p := parent
		t.parents[c.raw] = &p
	}
	t.children[parent.raw] = append([]NodeID{}, children...)
	t.MarkDirty(parent)
}

// RemoveChild detaches a child from its parent.
func (t *TaffyTree) RemoveChild(parent, child NodeID) {
	t.removeChildFromParent(parent, child)
	t.MarkDirty(parent)
}

func (t *TaffyTree) removeChildFromParent(parent, child NodeID) {
	cs := t.children[parent.raw]
	for i, c := range cs {
		if c == child {
			t.children[parent.raw] = append(cs[:i], cs[i+1:]...)
			break
		}
	}
	t.parents[child.raw] = nil
}

// SetStyle sets the style of a node and marks it dirty.
func (t *TaffyTree) SetStyle(id NodeID, style Style) {
	t.nodes[id.raw].style = style
	t.MarkDirty(id)
}

// Style returns the style of a node.
func (t *TaffyTree) Style(id NodeID) *Style { return &t.nodes[id.raw].style }

// Layout returns the final (rounded or unrounded) layout of a node.
func (t *TaffyTree) Layout(id NodeID) *Layout {
	if t.useRounding {
		return &t.nodes[id.raw].finalLayout
	}
	return &t.nodes[id.raw].unroundedLayout
}

// UnroundedLayout returns the unrounded layout of a node.
func (t *TaffyTree) UnroundedLayout(id NodeID) *Layout { return &t.nodes[id.raw].unroundedLayout }

// Parent returns the parent of a node, or nil if it has none.
func (t *TaffyTree) Parent(child NodeID) *NodeID { return t.parents[child.raw] }

// Children returns a copy of a node's children.
func (t *TaffyTree) Children(parent NodeID) []NodeID {
	return append([]NodeID{}, t.children[parent.raw]...)
}

// SetNodeContext attaches or detaches per-node context for the measure function.
func (t *TaffyTree) SetNodeContext(id NodeID, ctx any) {
	if ctx == nil {
		t.nodes[id.raw].hasContext = false
		delete(t.contexts, id.raw)
	} else {
		t.nodes[id.raw].hasContext = true
		t.contexts[id.raw] = ctx
	}
	t.MarkDirty(id)
}

// NodeContext returns the per-node context, or nil.
func (t *TaffyTree) NodeContext(id NodeID) any { return t.contexts[id.raw] }

// MarkDirty marks a node and its ancestors as requiring relayout.
func (t *TaffyTree) MarkDirty(id NodeID) {
	t.markDirtyRecursive(id.raw)
}

func (t *TaffyTree) markDirtyRecursive(key uint64) {
	nd := t.nodes[key]
	switch nd.markDirty() {
	case clearStateAlreadyEmpty:
		// Already dirty; ancestors are too.
		return
	case clearStateCleared:
		if p := t.parents[key]; p != nil {
			t.markDirtyRecursive(p.raw)
		}
	}
}

// Dirty reports whether a node's layout needs recomputing.
func (t *TaffyTree) Dirty(id NodeID) bool { return t.nodes[id.raw].cache.IsEmpty() }

// TotalNodeCount returns the number of nodes in the tree.
func (t *TaffyTree) TotalNodeCount() int { return len(t.nodes) }

// --- layoutTree / flexboxTree / roundTree implementation ---

func (t *TaffyTree) childCount(parent NodeID) int { return len(t.children[parent.raw]) }

func (t *TaffyTree) childID(parent NodeID, index int) NodeID {
	return t.children[parent.raw][index]
}

func (t *TaffyTree) coreContainerStyle(id NodeID) *Style { return &t.nodes[id.raw].style }

func (t *TaffyTree) setUnroundedLayout(id NodeID, l Layout) {
	t.nodes[id.raw].unroundedLayout = l
}

func (t *TaffyTree) cacheGet(id NodeID, in *LayoutInput) (LayoutOutput, bool) {
	return t.nodes[id.raw].cache.Get(in)
}

func (t *TaffyTree) cacheStore(id NodeID, in *LayoutInput, out LayoutOutput) {
	t.nodes[id.raw].cache.Store(in, out)
}

func (t *TaffyTree) cacheClear(id NodeID) { t.nodes[id.raw].cache.Clear() }

func (t *TaffyTree) flexboxContainerStyle(id NodeID) *Style { return &t.nodes[id.raw].style }

func (t *TaffyTree) flexboxChildStyle(childID NodeID) *Style { return &t.nodes[childID.raw].style }

func (t *TaffyTree) getUnroundedLayout(id NodeID) Layout { return t.nodes[id.raw].unroundedLayout }

func (t *TaffyTree) setFinalLayout(id NodeID, l Layout) { t.nodes[id.raw].finalLayout = l }

// measureFunc is the currently-installed measure function for a compute pass.
// It is set by ComputeLayoutWithMeasure and read by computeChildLayout.
var (
	taffyTreeMeasure MeasureFunction
	taffyTreeActive  *TaffyTree
)

// computeChildLayout dispatches a child layout request to the appropriate
// algorithm based on the node's display and children, mirroring Taffy's
// TaffyView::compute_child_layout. It is the single entry point the
// layoutTree interface exposes to the algorithms.
func (t *TaffyTree) computeChildLayout(id NodeID, in LayoutInput) LayoutOutput {
	if in.RunMode == runPerformHiddenLayout {
		return computeHiddenLayout(t, id)
	}
	return computeCachedLayout(t, id, in, func(tree layoutTree, id NodeID, in LayoutInput) LayoutOutput {
		// Dispatch on display and whether the node has children.
		nd := t.nodes[id.raw]
		hasChildren := t.childCount(id) > 0
		switch nd.style.Display {
		case displayNone:
			return computeHiddenLayout(t, id)
		case displayFlex:
			if hasChildren {
				return computeFlexboxLayout(t, id, in)
			}
		case displayBlock, displayFlowRoot:
			if hasChildren {
				return computeBlockLayout(t, id, in)
			}
		}
		// Leaf: defer to the measure function if installed, else zero.
		if taffyTreeMeasure != nil && t == taffyTreeActive {
			var ctx any
			if nd.hasContext {
				ctx = t.contexts[id.raw]
			}
			return taffyTreeMeasure(in, id, ctx, &nd.style)
		}
		return computeLeafLayout(in, &nd.style, nil, nil)
	})
}

// ComputeLayoutWithMeasure runs a full layout pass from the root, using the
// supplied measure function for leaf nodes.
func (t *TaffyTree) ComputeLayoutWithMeasure(root NodeID, avail Size[AvailableSpace], measure MeasureFunction) {
	prevMeasure := taffyTreeMeasure
	prevActive := taffyTreeActive
	taffyTreeMeasure = measure
	taffyTreeActive = t
	defer func() {
		taffyTreeMeasure = prevMeasure
		taffyTreeActive = prevActive
	}()
	computeRootLayout(t, root, avail)
	if t.useRounding {
		roundLayout(t, root)
	}
}

// ComputeLayout runs a full layout pass from the root with a default
// zero-size measure function for leaves.
func (t *TaffyTree) ComputeLayout(root NodeID, avail Size[AvailableSpace]) {
	t.ComputeLayoutWithMeasure(root, avail, func(in LayoutInput, id NodeID, ctx any, style *Style) LayoutOutput {
		return computeLeafLayout(in, style, nil, nil)
	})
}
