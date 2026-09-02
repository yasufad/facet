package element

import (
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/text"
)

// HitRegionID uniquely identifies a hit-testable region registered during prepaint.
type HitRegionID uint64

// ActionBinding associates an action name with an ActionHandler.
type ActionBinding struct {
	ActionName string
	Handler    input.ActionHandler
}

// DispatchNode encapsulates the complete input configuration and event listeners
// attached to an input dispatch node atomically during prepaint.
type DispatchNode struct {
	KeyContext       *input.KeyContext
	FocusID          input.FocusID
	TabStop          bool
	TabIndex         int
	Cursor           style.CursorStyle
	ActionBindings   []ActionBinding
	KeyListeners     []input.KeyEventHandler
	PointerListeners []input.PointerEventHandler
	WheelListeners   []input.WheelEventHandler
	TextListeners    []input.TextEventHandler
	ClickListeners   []func(event ClickEvent) bool
}

// MeasureFunc computes the intrinsic content size of an element given known
// dimensions and available space constraints provided by the layout solver.
type MeasureFunc func(known layout.Size[layout.OptF32], available layout.Size[layout.AvailableSpace]) geometry.Size[geometry.Pixels]

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

	// RequestMeasuredLayout adds a leaf node with the given style and content
	// measurement callback to the layout engine, returning its layout identifier.
	RequestMeasuredLayout(style layout.Style, measure MeasureFunc) layout.NodeID

	// LayoutBounds returns the computed bounds of a layout node in logical pixels
	// after the layout pass has solved.
	LayoutBounds(id layout.NodeID) geometry.Bounds[geometry.Pixels]

	// PushDispatchNode opens a new input dispatch node configured with the given
	// key context, focus handle, and event listeners, returning its identifier.
	PushDispatchNode(node DispatchNode) input.DispatchNodeID

	// PopDispatchNode closes the top active dispatch node.
	PopDispatchNode()

	// RegisterHitRegion registers a hit-testable bounding rectangle keyed by an
	// input dispatch node identifier during the prepaint phase.
	RegisterHitRegion(bounds geometry.Bounds[geometry.Pixels], nodeID input.DispatchNodeID) HitRegionID

	// IsHovered reports whether the given hit region is hovered by the pointer
	// in the current frame (evaluated during step 5 between prepaint and paint).
	// Valid during the paint phase only.
	IsHovered(id HitRegionID) bool

	// IsActive reports whether the given hit region is actively pressed by the
	// pointer in the current frame. Valid during the paint phase only.
	IsActive(id HitRegionID) bool

	// IsFocused reports whether the given focus identifier currently holds
	// keyboard focus. Valid during the paint phase only.
	IsFocused(id input.FocusID) bool

	// RequestFocus moves keyboard focus to the node identified by id.
	RequestFocus(id input.FocusID)

	// PushClip pushes a content clip mask for the current phase. Valid during
	// prepaint, where it confines the bounds RegisterHitRegion intersects
	// against, and during paint, where it also confines painted primitives via
	// the scene clip stack. The two stacks are independent, so a prepaint push
	// does not touch the scene.
	PushClip(bounds geometry.Bounds[geometry.Pixels])

	// PopClip removes the top content clip mask for the current phase. Valid
	// during prepaint and paint; see PushClip.
	PopClip()

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

	// RasteriseGlyph returns the atlas tile and device-pixel bounding box relative to
	// the pen position for the specified glyph, rasterising and uploading on miss.
	RasteriseGlyph(face text.Face, gid text.GlyphID, size geometry.Pixels, subpixel text.SubpixelOffset) (scene.AtlasTile, geometry.Bounds[geometry.DevicePixels], bool)

	// ScaleFactor returns the display scale factor for snapping to physical device pixels.
	ScaleFactor() float32

	// RemSize returns the current root font size in logical pixels for resolving rem units.
	RemSize() geometry.Pixels

	// PushTextStyle pushes a text style refinement onto the inherited text style stack.
	PushTextStyle(refinement style.Refinement)

	// PopTextStyle pops the top text style refinement from the stack.
	PopTextStyle()

	// TextStyle returns the current inherited text style computed by layering all
	// pushed refinements from root to the current element.
	TextStyle() style.TextStyle
}
