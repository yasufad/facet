package window

import (
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/style"
)

// hitRegion records a hit-testable bounding rectangle registered during prepaint.
type hitRegion struct {
	id      element.HitRegionID
	bounds  geometry.Bounds[geometry.Pixels]
	nodeID  input.DispatchNodeID
	focusID input.FocusID
	cursor  style.CursorStyle
}

// hitTest resolves pt against regions in reverse insertion order (back to front),
// returning the topmost matching hit region and dispatch node identifier.
func hitTest(regions []hitRegion, pt geometry.Point[geometry.Pixels]) (element.HitRegionID, input.DispatchNodeID, bool) {
	for i := len(regions) - 1; i >= 0; i-- {
		r := regions[i]
		if r.bounds.Contains(pt) {
			return r.id, r.nodeID, true
		}
	}
	return 0, -1, false
}
