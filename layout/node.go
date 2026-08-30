// Ported from Taffy src/tree/node.rs (MIT).
package layout

// NodeID is the identifier of a single node in a layout tree. Internally it is a
// wrapper around a u64.
type NodeID struct{ raw uint64 }

// NewNodeID creates a NodeID from a u64 value.
func NewNodeID(val uint64) NodeID { return NodeID{raw: val} }

// Raw returns the underlying u64.
func (n NodeID) Raw() uint64 { return n.raw }
