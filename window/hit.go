package window

import (
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/style"
)

// hitRegion records a hit-testable bounding rectangle registered during prepaint.
// bounds is already clipped to the prepaint clip mask in force at registration.
type hitRegion struct {
	id      element.HitRegionID
	bounds  geometry.Bounds[geometry.Pixels]
	nodeID  input.DispatchNodeID
	focusID input.FocusID
	cursor  style.CursorStyle
}

// hitTest resolves pt against regions in reverse insertion order (back to front),
// returning the topmost matching hit region.
func hitTest(regions []hitRegion, pt geometry.Point[geometry.Pixels]) (hitRegion, bool) {
	for i := len(regions) - 1; i >= 0; i-- {
		if regions[i].bounds.Contains(pt) {
			return regions[i], true
		}
	}
	return hitRegion{}, false
}
