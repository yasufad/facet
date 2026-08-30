package element

import (
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/text"
)

// HitRegionID uniquely identifies a hit-testable region registered during prepaint.
type HitRegionID uint64

// Frame is the capability interface that window provides to elements across the
// three lifecycle phases.
//
// Elements never import window directly; this interface breaks the cycle.
// Every method on Frame is an explicit capability that window guarantees
// throughout the frame lifecycle.
type Frame interface {
	// RequestLayout adds a node with the given style and children to the layout
	// engine and returns its layout identifier.
	RequestLayout(style layout.Style, children []layout.NodeID) layout.NodeID

	// LayoutBounds returns the computed bounds of a layout node in logical pixels
	// after the layout pass has solved.
	LayoutBounds(id layout.NodeID) geometry.Bounds[geometry.Pixels]

	// RegisterHitRegion registers a hit-testable bounding rectangle keyed by an
	// input dispatch node identifier during the prepaint phase.
	RegisterHitRegion(bounds geometry.Bounds[geometry.Pixels], nodeID input.DispatchNodeID) HitRegionID

	// InsertQuad adds a quad primitive to the frame's scene.
	InsertQuad(q scene.Quad)

	// InsertShadow adds a shadow primitive to the frame's scene.
	InsertShadow(sh scene.Shadow)

	// InsertPath adds a path primitive to the frame's scene.
	InsertPath(p scene.Path[geometry.ScaledPixels])

	// InsertUnderline adds an underline primitive to the frame's scene.
	InsertUnderline(u scene.Underline)

	// InsertMonochromeSprite adds a monochrome sprite primitive to the frame's scene.
	InsertMonochromeSprite(sp scene.MonochromeSprite)

	// InsertPolychromeSprite adds a polychrome sprite primitive to the frame's scene.
	InsertPolychromeSprite(sp scene.PolychromeSprite)

	// ShapeLine shapes a single line of text with the given style runs at the
	// window's scale factor.
	ShapeLine(str string, runs []text.StyleRun) (text.ShapedLine, error)

	// ScaleFactor returns the display scale factor for snapping to physical device pixels.
	ScaleFactor() float32

	// RemSize returns the current root font size in logical pixels for resolving rem units.
	RemSize() geometry.Pixels
}
