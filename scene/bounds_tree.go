// Ported from Zed's GPUI: crates/gpui/src/bounds_tree.rs.
//
// Copyright 2022 - 2025 Zed Industries, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License"); you may not
// use this file except in compliance with the License. You may obtain a copy of
// the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.
//
// Modified from the original: the Rust source uses raw pointers for its search
// stack to work around the borrow checker; this Go port uses slice indices,
// which are safe and equally fast. The zero value is an empty, usable tree;
// emptiness is tested with len(nodes) == 0 rather than a sentinel, so a
// zero-value root does not misidentify itself as a valid node index.
package scene

import (
	"github.com/yasufad/facet/geometry"
)

// maxChildren is the branching factor of the R-tree. Higher values mean a
// shorter tree and fewer cache misses, but more work per node.
const maxChildren = 12

// boundsTree is an R-tree variant that assigns draw orders to spatially
// overlapping bounds. Insert returns an order one greater than the maximum
// order of any existing bounds that intersect the new ones, so a primitive
// that overlaps a previously drawn one is always drawn on top of it. Bounds
// that share no screen space may reuse an order, which lets the renderer batch
// them into one instanced call.
type boundsTree struct {
	nodes       []treeNode
	root        int // valid only when len(nodes) > 0
	maxLeaf     int // index of the leaf with the highest order; valid only when len(nodes) > 0
	insertPath  []int
	searchStack []int
}

type treeNode struct {
	bounds   geometry.Bounds[geometry.ScaledPixels]
	maxOrder DrawOrder
	kind     treeKind
}

type treeKind struct {
	leaf     bool
	order    DrawOrder // valid when leaf
	children []int     // valid when not leaf
}

func (t *boundsTree) clear() {
	t.nodes = t.nodes[:0]
	t.root = 0
	t.maxLeaf = 0
	t.insertPath = t.insertPath[:0]
	t.searchStack = t.searchStack[:0]
}

// Insert assigns an order to bounds and returns it. The order is one greater
// than the maximum order of any existing bounds that intersect the new ones.
func (t *boundsTree) Insert(b geometry.Bounds[geometry.ScaledPixels]) DrawOrder {
	maxIntersecting := t.findMaxOrdering(b)
	order := maxIntersecting + 1
	newLeaf := t.insertLeaf(b, order)
	if len(t.nodes) == 0 || t.nodes[t.maxLeaf].maxOrder < order {
		t.maxLeaf = newLeaf
	}
	return order
}

// findMaxOrdering returns the maximum order among all stored bounds that
// intersect the query. It checks the cached max-order leaf first; if that
// misses, it walks the tree with pruning.
func (t *boundsTree) findMaxOrdering(query geometry.Bounds[geometry.ScaledPixels]) DrawOrder {
	if len(t.nodes) == 0 {
		return 0
	}
	// Fast path: the leaf with the global max order intersects the query.
	if t.maxLeaf >= 0 && t.maxLeaf < len(t.nodes) {
		maxNode := &t.nodes[t.maxLeaf]
		if query.Intersects(maxNode.bounds) {
			return maxNode.maxOrder
		}
	}
	// Slow path: depth-first search with order and spatial pruning.
	t.searchStack = t.searchStack[:0]
	t.searchStack = append(t.searchStack, t.root)
	var maxFound DrawOrder
	for len(t.searchStack) > 0 {
		idx := t.searchStack[len(t.searchStack)-1]
		t.searchStack = t.searchStack[:len(t.searchStack)-1]
		node := &t.nodes[idx]
		if node.maxOrder <= maxFound {
			continue
		}
		if !query.Intersects(node.bounds) {
			continue
		}
		if node.kind.leaf {
			if node.kind.order > maxFound {
				maxFound = node.kind.order
			}
			continue
		}
		for _, childIdx := range node.kind.children {
			if t.nodes[childIdx].maxOrder > maxFound {
				t.searchStack = append(t.searchStack, childIdx)
			}
		}
	}
	return maxFound
}

// insertLeaf adds a leaf node for bounds with the given order, growing the tree
// as needed, and returns the new leaf's index.
func (t *boundsTree) insertLeaf(b geometry.Bounds[geometry.ScaledPixels], order DrawOrder) int {
	wasEmpty := len(t.nodes) == 0
	newIdx := len(t.nodes)
	t.nodes = append(t.nodes, treeNode{
		bounds:   b,
		maxOrder: order,
		kind:     treeKind{leaf: true, order: order},
	})

	if wasEmpty {
		t.root = newIdx
		return newIdx
	}

	root := &t.nodes[t.root]
	if root.kind.leaf {
		// Root is a leaf: create an internal node with both leaves. The child
		// with the higher order goes last, maintaining the max-at-end
		// invariant.
		rootOrder := root.maxOrder
		rootBounds := root.bounds
		var children []int
		if order > rootOrder {
			children = []int{t.root, newIdx}
		} else {
			children = []int{newIdx, t.root}
		}
		internalIdx := len(t.nodes)
		t.nodes = append(t.nodes, treeNode{
			bounds:   rootBounds.Union(b),
			maxOrder: max(rootOrder, order),
			kind:     treeKind{children: children},
		})
		t.root = internalIdx
		return newIdx
	}

	// Descend from the root to find the best internal node to insert into,
	// choosing the child whose union with the new bounds grows least. The
	// current node is pushed onto the insertion path before descending or
	// breaking, so the propagation loop at the end updates every ancestor —
	// including the parent in the split case, whose bounds must grow to cover
	// the new internal node.
	t.insertPath = t.insertPath[:0]
	currentIdx := t.root
	for {
		t.insertPath = append(t.insertPath, currentIdx)

		current := &t.nodes[currentIdx]
		children := current.kind.children
		bestChild := children[0]
		bestPos := 0
		bestCost := b.Union(t.nodes[bestChild].bounds).HalfPerimeter()
		for pos, childIdx := range children[1:] {
			cost := b.Union(t.nodes[childIdx].bounds).HalfPerimeter()
			if cost < bestCost {
				bestCost = cost
				bestChild = childIdx
				bestPos = pos + 1
			}
		}

		if t.nodes[bestChild].kind.leaf {
			// Best child is a leaf. If the current node has room, add the new
			// leaf directly; otherwise split.
			current = &t.nodes[currentIdx]
			if len(current.kind.children) < maxChildren {
				current.kind.children = append(current.kind.children, newIdx)
				// Maintain max-at-end: if the new leaf does not have the
				// highest order, swap it one step left so the highest stays
				// last.
				if order <= current.maxOrder {
					last := len(current.kind.children) - 1
					current.kind.children[last-1], current.kind.children[last] =
						current.kind.children[last], current.kind.children[last-1]
				}
				current.bounds = current.bounds.Union(b)
				if order > current.maxOrder {
					current.maxOrder = order
				}
				break
			}

			// Node is full: create a new internal node with the best leaf and
			// the new leaf, and replace the best leaf in the parent.
			siblingBounds := t.nodes[bestChild].bounds
			siblingOrder := t.nodes[bestChild].maxOrder
			var newChildren []int
			if order > siblingOrder {
				newChildren = []int{bestChild, newIdx}
			} else {
				newChildren = []int{newIdx, bestChild}
			}
			internalIdx := len(t.nodes)
			internalMax := max(siblingOrder, order)
			t.nodes = append(t.nodes, treeNode{
				bounds:   siblingBounds.Union(b),
				maxOrder: internalMax,
				kind:     treeKind{children: newChildren},
			})
			parent := &t.nodes[currentIdx]
			parent.kind.children[bestPos] = internalIdx
			if internalMax > parent.maxOrder {
				parent.kind.children[bestPos], parent.kind.children[len(parent.kind.children)-1] =
					parent.kind.children[len(parent.kind.children)-1], parent.kind.children[bestPos]
			}
			break
		}

		// Best child is internal: keep descending.
		currentIdx = bestChild
	}

	// Propagate bounds and maxOrder updates up the insertion path. The path
	// includes the node where the leaf was inserted or the split happened, so
	// its bounds grow to cover the new leaf in every case.
	var updatedChild int
	var hasUpdated bool
	for i := len(t.insertPath) - 1; i >= 0; i-- {
		idx := t.insertPath[i]
		node := &t.nodes[idx]
		node.bounds = node.bounds.Union(b)
		if node.maxOrder < order {
			node.maxOrder = order
			if hasUpdated {
				for pos, c := range node.kind.children {
					if c == updatedChild {
						last := len(node.kind.children) - 1
						if pos != last {
							node.kind.children[pos], node.kind.children[last] =
								node.kind.children[last], node.kind.children[pos]
						}
						break
					}
				}
			}
		}
		updatedChild = idx
		hasUpdated = true
	}

	return newIdx
}
