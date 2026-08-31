package elementtest

import (
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/text"
)

// Phase represents the lifecycle phase of the test frame.
type Phase uint8

const (
	// PhaseLayout is the layout request phase (elements call RequestLayout).
	PhaseLayout Phase = iota
	// PhasePrepaint is the prepaint phase (elements call Prepaint).
	PhasePrepaint
	// PhasePaint is the paint phase (elements call Paint).
	PhasePaint
)

// RecordedHitRegion records a hit region registered with the frame.
type RecordedHitRegion struct {
	ID     element.HitRegionID
	Bounds geometry.Bounds[geometry.Pixels]
	NodeID input.DispatchNodeID
}

// PointerPhase indicates the pointer interaction phase in test dispatches.
type PointerPhase uint8

const (
	// PointerDown indicates a pointer press.
	PointerDown PointerPhase = iota
	// PointerUp indicates a pointer release.
	PointerUp
	// PointerMove indicates a pointer movement.
	PointerMove
)

// Frame is an exported test double implementing element.Frame for unit testing
// elements and widgets without importing internal rendering or platform packages.
type Frame struct {
	phase       Phase
	scaleFactor float32
	remSize     geometry.Pixels

	tree             *layout.TaffyTree
	nodes            []element.NodeID
	styles           map[element.NodeID]layout.Style
	children         map[element.NodeID][]element.NodeID
	bounds           map[element.NodeID]geometry.Bounds[geometry.Pixels]
	measureCallbacks map[element.NodeID]element.MeasureFunc

	dispatchTree      *input.DispatchTree
	dispatchNodes     []element.DispatchNode
	dispatchNodeStack []input.DispatchNodeID
	clickListeners    map[input.DispatchNodeID][]func(element.ClickEvent) bool

	nextHitRegionID   element.HitRegionID
	hitRegions        []RecordedHitRegion
	hoveredHitRegions map[element.HitRegionID]bool
	activeHitRegions  map[element.HitRegionID]bool
	focusedIDs        map[input.FocusID]bool

	downHitRegion element.HitRegionID
	downNodeID    input.DispatchNodeID

	quads          []scene.Quad
	shadows        []scene.Shadow
	paths          []scene.Path[geometry.ScaledPixels]
	underlines     []scene.Underline
	monoSprites    []scene.MonochromeSprite
	polySprites    []scene.PolychromeSprite
	textSys        *text.System
	textStyleStack []style.TextStyle
}

// Ensure Frame implements element.Frame.
var _ element.Frame = (*Frame)(nil)

// NewFrame constructs an initialised test Frame ready for element lifecycle testing.
func NewFrame() *Frame {
	txtSys, _ := text.NewSystem()
	return &Frame{
		phase:             PhaseLayout,
		scaleFactor:       2.0,
		remSize:           16.0,
		tree:              layout.NewTaffyTree(),
		styles:            make(map[element.NodeID]layout.Style),
		children:          make(map[element.NodeID][]element.NodeID),
		bounds:            make(map[element.NodeID]geometry.Bounds[geometry.Pixels]),
		measureCallbacks:  make(map[element.NodeID]element.MeasureFunc),
		dispatchTree:      input.NewDispatchTree(nil, nil),
		clickListeners:    make(map[input.DispatchNodeID][]func(element.ClickEvent) bool),
		hoveredHitRegions: make(map[element.HitRegionID]bool),
		activeHitRegions:  make(map[element.HitRegionID]bool),
		focusedIDs:        make(map[input.FocusID]bool),
		textSys:           txtSys,
		textStyleStack:    []style.TextStyle{style.DefaultTextStyle()},
	}
}

// SetPhase updates the active lifecycle phase of the test frame.
func (f *Frame) SetPhase(p Phase) {
	f.phase = p
}

// SetScaleFactor sets the display scale factor.
func (f *Frame) SetScaleFactor(sf float32) {
	f.scaleFactor = sf
}

// SetRemSize sets the root rem size in logical pixels.
func (f *Frame) SetRemSize(rem geometry.Pixels) {
	f.remSize = rem
}

// SetHovered sets whether a hit region is considered hovered.
func (f *Frame) SetHovered(id element.HitRegionID, hovered bool) {
	f.hoveredHitRegions[id] = hovered
}

// SetActive sets whether a hit region is considered active/pressed.
func (f *Frame) SetActive(id element.HitRegionID, active bool) {
	f.activeHitRegions[id] = active
}

// SetFocused sets whether a focus identifier is considered focused.
func (f *Frame) SetFocused(id input.FocusID, focused bool) {
	f.focusedIDs[id] = focused
}

// ClearPrimitives empties all recorded scene primitives.
func (f *Frame) ClearPrimitives() {
	f.quads = f.quads[:0]
	f.shadows = f.shadows[:0]
	f.paths = f.paths[:0]
	f.underlines = f.underlines[:0]
	f.monoSprites = f.monoSprites[:0]
	f.polySprites = f.polySprites[:0]
}

// Quads returns all quads emitted into the frame.
func (f *Frame) Quads() []scene.Quad {
	return f.quads
}

// Shadows returns all shadows emitted into the frame.
func (f *Frame) Shadows() []scene.Shadow {
	return f.shadows
}

// Paths returns all vector paths emitted into the frame.
func (f *Frame) Paths() []scene.Path[geometry.ScaledPixels] {
	return f.paths
}

// Underlines returns all underlines emitted into the frame.
func (f *Frame) Underlines() []scene.Underline {
	return f.underlines
}

// MonochromeSprites returns all monochrome sprites emitted into the frame.
func (f *Frame) MonochromeSprites() []scene.MonochromeSprite {
	return f.monoSprites
}

// PolychromeSprites returns all polychrome sprites emitted into the frame.
func (f *Frame) PolychromeSprites() []scene.PolychromeSprite {
	return f.polySprites
}

// HitRegions returns all hit regions registered during prepaint.
func (f *Frame) HitRegions() []RecordedHitRegion {
	return f.hitRegions
}

// DispatchNodes returns all dispatch nodes registered during prepaint.
func (f *Frame) DispatchNodes() []element.DispatchNode {
	return f.dispatchNodes
}

// Nodes returns all layout node identifiers allocated.
func (f *Frame) Nodes() []element.NodeID {
	return f.nodes
}

// ChildrenOf returns the child layout node identifiers of parent.
func (f *Frame) ChildrenOf(parent element.NodeID) []element.NodeID {
	return f.children[parent]
}

// StyleOf returns the layout style of node.
func (f *Frame) StyleOf(node element.NodeID) layout.Style {
	return f.styles[node]
}

// DispatchPointer simulates delivering a pointer event to the frame and synthesising clicks.
func (f *Frame) DispatchPointer(phase PointerPhase, pt geometry.Point[geometry.Pixels], button element.MouseButton, modifiers element.Modifiers) {
	var hitID element.HitRegionID
	var nodeID input.DispatchNodeID = -1
	for i := len(f.hitRegions) - 1; i >= 0; i-- {
		if f.hitRegions[i].Bounds.Contains(pt) {
			hitID = f.hitRegions[i].ID
			nodeID = f.hitRegions[i].NodeID
			break
		}
	}

	if phase == PointerDown {
		f.downHitRegion = hitID
		f.downNodeID = nodeID
	} else if phase == PointerUp {
		if hitID != 0 && hitID == f.downHitRegion {
			var hitBounds geometry.Bounds[geometry.Pixels]
			for _, hr := range f.hitRegions {
				if hr.ID == hitID {
					hitBounds = hr.Bounds
					break
				}
			}
			localPos := geometry.NewPoint[geometry.Pixels](
				pt.X-hitBounds.Origin.X,
				pt.Y-hitBounds.Origin.Y,
			)
			clickEvt := element.ClickEvent{
				Position:      pt,
				LocalPosition: localPos,
				Button:        button,
				Modifiers:     modifiers,
			}
			if listeners, ok := f.clickListeners[nodeID]; ok {
				for _, l := range listeners {
					if l(clickEvt) {
						break
					}
				}
			}
		}
		f.downHitRegion = 0
		f.downNodeID = 0
	}
}

// SimulateClick invokes click listeners registered on nodeID with a synthesised ClickEvent.
func (f *Frame) SimulateClick(nodeID input.DispatchNodeID, pos geometry.Point[geometry.Pixels], button element.MouseButton, modifiers element.Modifiers) bool {
	var hitBounds geometry.Bounds[geometry.Pixels]
	for _, hr := range f.hitRegions {
		if hr.NodeID == nodeID {
			hitBounds = hr.Bounds
			break
		}
	}
	localPos := geometry.NewPoint[geometry.Pixels](
		pos.X-hitBounds.Origin.X,
		pos.Y-hitBounds.Origin.Y,
	)
	evt := element.ClickEvent{
		Position:      pos,
		LocalPosition: localPos,
		Button:        button,
		Modifiers:     modifiers,
	}
	handled := false
	if listeners, ok := f.clickListeners[nodeID]; ok {
		for _, l := range listeners {
			if l(evt) {
				handled = true
				break
			}
		}
	}
	return handled
}

// Solve computes flexbox layout using Taffy and stores resolved bounds on all nodes.
func (f *Frame) Solve(root element.NodeID, availableWidth, availableHeight float32) {
	avail := layout.Size[layout.AvailableSpace]{
		Width:  layout.Definite(availableWidth),
		Height: layout.Definite(availableHeight),
	}
	measureFn := func(inputs layout.LayoutInput, id layout.NodeID, ctx any, s *layout.Style) layout.LayoutOutput {
		cb, ok := f.measureCallbacks[id]
		if !ok || cb == nil {
			return layout.ComputeLeafLayout(inputs, s, nil)
		}
		leafMeasure := func(known layout.Size[layout.OptF32], avail layout.Size[layout.AvailableSpace]) layout.Size[float32] {
			contentSize := cb(known, avail)
			return layout.Size[float32]{
				Width:  float32(contentSize.Width),
				Height: float32(contentSize.Height),
			}
		}
		return layout.ComputeLeafLayout(inputs, s, leafMeasure)
	}

	f.tree.ComputeLayoutWithMeasure(root, avail, measureFn)

	rootLayout := f.tree.Layout(root)
	rootOrigin := geometry.Point[geometry.Pixels]{X: 0, Y: 0}
	f.bounds[root] = geometry.Bounds[geometry.Pixels]{
		Origin: rootOrigin,
		Size: geometry.Size[geometry.Pixels]{
			Width:  geometry.Pixels(rootLayout.Size.Width),
			Height: geometry.Pixels(rootLayout.Size.Height),
		},
	}
	f.populateBounds(root, rootOrigin)
}

func (f *Frame) populateBounds(parentID element.NodeID, parentOrigin geometry.Point[geometry.Pixels]) {
	for _, childID := range f.tree.Children(parentID) {
		childLayout := f.tree.Layout(childID)
		childOrigin := parentOrigin.Add(geometry.Point[geometry.Pixels]{
			X: geometry.Pixels(childLayout.Location.X),
			Y: geometry.Pixels(childLayout.Location.Y),
		})
		f.bounds[childID] = geometry.Bounds[geometry.Pixels]{
			Origin: childOrigin,
			Size: geometry.Size[geometry.Pixels]{
				Width:  geometry.Pixels(childLayout.Size.Width),
				Height: geometry.Pixels(childLayout.Size.Height),
			},
		}
		f.populateBounds(childID, childOrigin)
	}
}

// --- Frame Interface Implementation ---

// RequestLayout adds a node with the given style and children to the layout tree.
func (f *Frame) RequestLayout(s layout.Style, children []element.NodeID) element.NodeID {
	if f.phase != PhaseLayout {
		panic("elementtest: RequestLayout called outside layout phase")
	}
	var id layout.NodeID
	if len(children) == 0 {
		id = f.tree.NewLeaf(s)
	} else {
		id = f.tree.NewWithChildren(s, children)
	}
	f.nodes = append(f.nodes, id)
	f.styles[id] = s
	f.children[id] = append([]element.NodeID(nil), children...)
	return id
}

// RequestMeasuredLayout adds a measured leaf node to the layout tree.
func (f *Frame) RequestMeasuredLayout(s layout.Style, measure element.MeasureFunc) element.NodeID {
	if f.phase != PhaseLayout {
		panic("elementtest: RequestMeasuredLayout called outside layout phase")
	}
	id := f.tree.NewLeaf(s)
	f.nodes = append(f.nodes, id)
	f.styles[id] = s
	if measure != nil {
		f.measureCallbacks[id] = measure
	}
	return id
}

// LayoutBounds returns the computed bounds for node id.
func (f *Frame) LayoutBounds(id element.NodeID) geometry.Bounds[geometry.Pixels] {
	return f.bounds[id]
}

// PushDispatchNode opens an input dispatch node during prepaint.
func (f *Frame) PushDispatchNode(node element.DispatchNode) input.DispatchNodeID {
	if f.phase != PhasePrepaint {
		panic("elementtest: PushDispatchNode called outside prepaint phase")
	}
	nodeID := f.dispatchTree.PushNode()
	if node.KeyContext != nil {
		f.dispatchTree.SetContext(*node.KeyContext)
	}
	if node.FocusID != 0 {
		f.dispatchTree.SetFocusID(node.FocusID)
	}
	for _, ab := range node.ActionBindings {
		f.dispatchTree.OnAction(ab.ActionName, ab.Handler)
	}
	for _, kl := range node.KeyListeners {
		f.dispatchTree.OnKeyEvent(kl)
	}
	for _, pl := range node.PointerListeners {
		f.dispatchTree.OnPointerEvent(pl)
	}
	for _, wl := range node.WheelListeners {
		f.dispatchTree.OnWheelEvent(wl)
	}
	for _, tl := range node.TextListeners {
		f.dispatchTree.OnTextEvent(tl)
	}
	if len(node.ClickListeners) > 0 {
		f.clickListeners[nodeID] = append([]func(element.ClickEvent) bool(nil), node.ClickListeners...)
	}
	f.dispatchNodes = append(f.dispatchNodes, node)
	f.dispatchNodeStack = append(f.dispatchNodeStack, nodeID)
	return nodeID
}

// PopDispatchNode closes the active dispatch node.
func (f *Frame) PopDispatchNode() {
	if f.phase != PhasePrepaint {
		panic("elementtest: PopDispatchNode called outside prepaint phase")
	}
	f.dispatchTree.PopNode()
	if len(f.dispatchNodeStack) > 0 {
		f.dispatchNodeStack = f.dispatchNodeStack[:len(f.dispatchNodeStack)-1]
	}
}

// RegisterHitRegion registers a hit region during prepaint.
func (f *Frame) RegisterHitRegion(bounds geometry.Bounds[geometry.Pixels], nodeID input.DispatchNodeID) element.HitRegionID {
	if f.phase != PhasePrepaint {
		panic("elementtest: RegisterHitRegion called outside prepaint phase")
	}
	f.nextHitRegionID++
	id := f.nextHitRegionID
	f.hitRegions = append(f.hitRegions, RecordedHitRegion{
		ID:     id,
		Bounds: bounds,
		NodeID: nodeID,
	})
	return id
}

// IsHovered reports whether id is hovered.
func (f *Frame) IsHovered(id element.HitRegionID) bool {
	if f.phase != PhasePaint {
		panic("elementtest: IsHovered called outside paint phase")
	}
	return f.hoveredHitRegions[id]
}

// IsActive reports whether id is actively pressed.
func (f *Frame) IsActive(id element.HitRegionID) bool {
	if f.phase != PhasePaint {
		panic("elementtest: IsActive called outside paint phase")
	}
	return f.activeHitRegions[id]
}

// IsFocused reports whether id currently holds focus.
func (f *Frame) IsFocused(id input.FocusID) bool {
	if f.phase != PhasePaint {
		panic("elementtest: IsFocused called outside paint phase")
	}
	return f.focusedIDs[id]
}

// InsertQuad adds a quad to the scene.
func (f *Frame) InsertQuad(q scene.Quad) {
	if f.phase != PhasePaint {
		panic("elementtest: InsertQuad called outside paint phase")
	}
	f.quads = append(f.quads, q)
}

// InsertShadow adds a shadow to the scene.
func (f *Frame) InsertShadow(sh scene.Shadow) {
	if f.phase != PhasePaint {
		panic("elementtest: InsertShadow called outside paint phase")
	}
	f.shadows = append(f.shadows, sh)
}

// InsertPath adds a vector path to the scene.
func (f *Frame) InsertPath(p scene.Path[geometry.ScaledPixels]) {
	if f.phase != PhasePaint {
		panic("elementtest: InsertPath called outside paint phase")
	}
	f.paths = append(f.paths, p)
}

// InsertUnderline adds an underline to the scene.
func (f *Frame) InsertUnderline(u scene.Underline) {
	if f.phase != PhasePaint {
		panic("elementtest: InsertUnderline called outside paint phase")
	}
	f.underlines = append(f.underlines, u)
}

// InsertMonochromeSprite adds a monochrome sprite to the scene.
func (f *Frame) InsertMonochromeSprite(sp scene.MonochromeSprite) {
	if f.phase != PhasePaint {
		panic("elementtest: InsertMonochromeSprite called outside paint phase")
	}
	f.monoSprites = append(f.monoSprites, sp)
}

// InsertPolychromeSprite adds a polychrome sprite to the scene.
func (f *Frame) InsertPolychromeSprite(sp scene.PolychromeSprite) {
	if f.phase != PhasePaint {
		panic("elementtest: InsertPolychromeSprite called outside paint phase")
	}
	f.polySprites = append(f.polySprites, sp)
}

// ShapeLine shapes a single line of text.
func (f *Frame) ShapeLine(str string, runs []text.StyleRun) (text.ShapedLine, error) {
	if f.textSys != nil {
		return f.textSys.ShapeLine(str, runs)
	}
	return text.ShapedLine{}, nil
}

// RasteriseGlyph returns a mock tile and device bounds for testing.
func (f *Frame) RasteriseGlyph(face text.Face, gid text.GlyphID, size geometry.Pixels, subpixel text.SubpixelOffset) (scene.AtlasTile, geometry.Bounds[geometry.DevicePixels], bool) {
	if f.phase != PhasePaint {
		panic("elementtest: RasteriseGlyph called outside paint phase")
	}
	tile := scene.AtlasTile{
		TextureID: scene.AtlasTextureID{Index: 0, Kind: scene.TextureMonochrome},
		TileID:    1,
		Bounds:    geometry.NewBounds(geometry.Point[geometry.DevicePixels]{X: 0, Y: 0}, geometry.Size[geometry.DevicePixels]{Width: 10, Height: 10}),
	}
	glyphBounds := geometry.NewBounds(
		geometry.Point[geometry.DevicePixels]{X: 0, Y: -10},
		geometry.Size[geometry.DevicePixels]{Width: 10, Height: 10},
	)
	return tile, glyphBounds, true
}

// ScaleFactor returns the display scale factor.
func (f *Frame) ScaleFactor() float32 {
	return f.scaleFactor
}

// RemSize returns the root rem size in logical pixels.
func (f *Frame) RemSize() geometry.Pixels {
	return f.remSize
}

// PushTextStyle pushes a text style refinement onto the text style stack.
func (f *Frame) PushTextStyle(refinement style.Refinement) {
	current := f.TextStyle()
	var s style.Style
	s.Text = current
	s.Refine(refinement)
	f.textStyleStack = append(f.textStyleStack, s.Text)
}

// PopTextStyle pops the top text style refinement from the stack.
func (f *Frame) PopTextStyle() {
	if len(f.textStyleStack) > 1 {
		f.textStyleStack = f.textStyleStack[:len(f.textStyleStack)-1]
	}
}

// TextStyle returns the current inherited text style.
func (f *Frame) TextStyle() style.TextStyle {
	if len(f.textStyleStack) == 0 {
		return style.DefaultTextStyle()
	}
	return f.textStyleStack[len(f.textStyleStack)-1]
}
