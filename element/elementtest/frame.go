package elementtest

import (
	"cmp"
	"slices"

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

type tabStopEntry struct {
	focusID  input.FocusID
	tabIndex int
	order    int
}

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

	focusTree         *input.FocusTree
	dispatchTree      *input.DispatchTree
	dispatchNodes     []element.DispatchNode
	dispatchNodeStack []input.DispatchNodeID
	clickListeners    map[input.DispatchNodeID][]func(element.ClickEvent) bool

	nextHitRegionID   element.HitRegionID
	hitRegions        []RecordedHitRegion
	hoveredHitRegions map[element.HitRegionID]bool
	activeHitRegions  map[element.HitRegionID]bool
	focusedIDs        map[input.FocusID]bool

	tabStops []tabStopEntry
	tabOrder []input.FocusID

	downHitRegion element.HitRegionID
	downNodeID    input.DispatchNodeID

	scene          *scene.Scene
	clipDepth      int
	textSys        *text.System
	textStyleStack []style.TextStyle
}

// Ensure Frame implements element.Frame.
var _ element.Frame = (*Frame)(nil)

// NewFrame constructs an initialised test Frame ready for element lifecycle testing.
func NewFrame() *Frame {
	txtSys, _ := text.NewSystem()
	ft := input.NewFocusTree()
	return &Frame{
		phase:             PhaseLayout,
		scaleFactor:       2.0,
		remSize:           16.0,
		tree:              layout.NewTaffyTree(),
		styles:            make(map[element.NodeID]layout.Style),
		children:          make(map[element.NodeID][]element.NodeID),
		bounds:            make(map[element.NodeID]geometry.Bounds[geometry.Pixels]),
		measureCallbacks:  make(map[element.NodeID]element.MeasureFunc),
		focusTree:         ft,
		dispatchTree:      input.NewDispatchTree(nil, ft),
		clickListeners:    make(map[input.DispatchNodeID][]func(element.ClickEvent) bool),
		hoveredHitRegions: make(map[element.HitRegionID]bool),
		activeHitRegions:  make(map[element.HitRegionID]bool),
		focusedIDs:        make(map[input.FocusID]bool),
		scene:             scene.New(),
		textSys:           txtSys,
		textStyleStack:    []style.TextStyle{style.DefaultTextStyle()},
	}
}

// SetPhase updates the active lifecycle phase of the test frame.
func (f *Frame) SetPhase(p Phase) {
	f.phase = p
	if p == PhasePaint || p == PhasePrepaint {
		f.tabOrder = sortTabStops(f.tabStops)
	}
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
	if focused {
		f.focusTree.Focus(id)
	} else {
		f.focusTree.Blur()
	}
}

// Focused returns the currently focused FocusID and true, or (0, false) if nothing is focused.
func (f *Frame) Focused() (input.FocusID, bool) {
	return f.focusTree.Focused()
}

// ClearPrimitives empties all recorded scene primitives.
func (f *Frame) ClearPrimitives() {
	f.scene.Clear()
	f.clipDepth = 0
}

// Quads returns all quads emitted into the frame.
func (f *Frame) Quads() []scene.Quad {
	return f.scene.Quads()
}

// Shadows returns all shadows emitted into the frame.
func (f *Frame) Shadows() []scene.Shadow {
	return f.scene.Shadows()
}

// Paths returns all vector paths emitted into the frame.
func (f *Frame) Paths() []scene.Path[geometry.ScaledPixels] {
	return f.scene.Paths()
}

// Underlines returns all underlines emitted into the frame.
func (f *Frame) Underlines() []scene.Underline {
	return f.scene.Underlines()
}

// MonochromeSprites returns all monochrome sprites emitted into the frame.
func (f *Frame) MonochromeSprites() []scene.MonochromeSprite {
	return f.scene.MonochromeSprites()
}

// PolychromeSprites returns all polychrome sprites emitted into the frame.
func (f *Frame) PolychromeSprites() []scene.PolychromeSprite {
	return f.scene.PolychromeSprites()
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

// TabOrder returns the ordered list of focusable identifiers in tab navigation order.
func (f *Frame) TabOrder() []input.FocusID {
	if len(f.tabOrder) == 0 && len(f.tabStops) > 0 {
		f.tabOrder = sortTabStops(f.tabStops)
	}
	return f.tabOrder
}

// FocusNext moves focus to the next focusable element in tab order, wrapping around.
func (f *Frame) FocusNext() {
	tabOrder := f.TabOrder()
	if len(tabOrder) == 0 {
		return
	}
	curr, ok := f.focusTree.Focused()
	if !ok || curr == 0 {
		f.RequestFocus(tabOrder[0])
		return
	}
	idx := slices.Index(tabOrder, curr)
	if idx < 0 {
		f.RequestFocus(tabOrder[0])
		return
	}
	nextIdx := (idx + 1) % len(tabOrder)
	f.RequestFocus(tabOrder[nextIdx])
}

// FocusPrev moves focus to the previous focusable element in tab order, wrapping around.
func (f *Frame) FocusPrev() {
	tabOrder := f.TabOrder()
	if len(tabOrder) == 0 {
		return
	}
	curr, ok := f.focusTree.Focused()
	if !ok || curr == 0 {
		f.RequestFocus(tabOrder[len(tabOrder)-1])
		return
	}
	idx := slices.Index(tabOrder, curr)
	if idx < 0 {
		f.RequestFocus(tabOrder[len(tabOrder)-1])
		return
	}
	prevIdx := (idx - 1 + len(tabOrder)) % len(tabOrder)
	f.RequestFocus(tabOrder[prevIdx])
}

// SimulateTab simulates pressing Tab (or Shift+Tab when shift is true) to navigate focus along tab order.
func (f *Frame) SimulateTab(shift bool) {
	if shift {
		f.FocusPrev()
	} else {
		f.FocusNext()
	}
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
		if node.TabStop || node.TabIndex != 0 {
			f.tabStops = append(f.tabStops, tabStopEntry{
				focusID:  node.FocusID,
				tabIndex: node.TabIndex,
				order:    len(f.tabStops),
			})
		}
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
	if focused, ok := f.focusTree.Focused(); ok {
		return focused == id || f.focusTree.Contains(id, focused)
	}
	return f.focusedIDs[id]
}

// RequestFocus moves keyboard focus to the node identified by id.
func (f *Frame) RequestFocus(id input.FocusID) {
	if id == 0 {
		f.focusTree.Blur()
		clear(f.focusedIDs)
	} else {
		f.focusTree.Focus(id)
		clear(f.focusedIDs)
		f.focusedIDs[id] = true
	}
}

// PushClip pushes a content clip mask onto the scene clip stack.
func (f *Frame) PushClip(bounds geometry.Bounds[geometry.Pixels]) {
	if f.phase != PhasePaint {
		panic("elementtest: PushClip called outside paint phase")
	}
	scale := f.scaleFactor
	scaledBounds := geometry.Bounds[geometry.ScaledPixels]{
		Origin: geometry.Point[geometry.ScaledPixels]{
			X: bounds.Origin.X.Scale(scale),
			Y: bounds.Origin.Y.Scale(scale),
		},
		Size: geometry.Size[geometry.ScaledPixels]{
			Width:  bounds.Size.Width.Scale(scale),
			Height: bounds.Size.Height.Scale(scale),
		},
	}
	f.clipDepth++
	f.scene.PushClip(scene.ContentMask[geometry.ScaledPixels]{
		Bounds: scaledBounds,
	})
}

// PopClip pops the top content clip mask from the scene clip stack.
func (f *Frame) PopClip() {
	if f.phase != PhasePaint {
		panic("elementtest: PopClip called outside paint phase")
	}
	if f.clipDepth > 0 {
		f.clipDepth--
	}
	f.scene.PopClip()
}

// InsertQuad adds a quad to the scene.
func (f *Frame) InsertQuad(q scene.Quad) {
	if f.phase != PhasePaint {
		panic("elementtest: InsertQuad called outside paint phase")
	}
	f.scene.InsertQuad(q)
}

// InsertShadow adds a shadow to the scene.
func (f *Frame) InsertShadow(sh scene.Shadow) {
	if f.phase != PhasePaint {
		panic("elementtest: InsertShadow called outside paint phase")
	}
	f.scene.InsertShadow(sh)
}

// InsertPath adds a vector path to the scene.
func (f *Frame) InsertPath(p scene.Path[geometry.ScaledPixels]) {
	if f.phase != PhasePaint {
		panic("elementtest: InsertPath called outside paint phase")
	}
	f.scene.InsertPath(p)
}

// InsertUnderline adds an underline to the scene.
func (f *Frame) InsertUnderline(u scene.Underline) {
	if f.phase != PhasePaint {
		panic("elementtest: InsertUnderline called outside paint phase")
	}
	f.scene.InsertUnderline(u)
}

// InsertMonochromeSprite adds a monochrome sprite to the scene.
func (f *Frame) InsertMonochromeSprite(sp scene.MonochromeSprite) {
	if f.phase != PhasePaint {
		panic("elementtest: InsertMonochromeSprite called outside paint phase")
	}
	f.scene.InsertMonochromeSprite(sp)
}

// InsertPolychromeSprite adds a polychrome sprite to the scene.
func (f *Frame) InsertPolychromeSprite(sp scene.PolychromeSprite) {
	if f.phase != PhasePaint {
		panic("elementtest: InsertPolychromeSprite called outside paint phase")
	}
	f.scene.InsertPolychromeSprite(sp)
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

func sortTabStops(entries []tabStopEntry) []input.FocusID {
	if len(entries) == 0 {
		return nil
	}
	var valid []tabStopEntry
	for _, e := range entries {
		if e.tabIndex >= 0 {
			valid = append(valid, e)
		}
	}
	slices.SortStableFunc(valid, func(a, b tabStopEntry) int {
		if a.tabIndex > 0 && b.tabIndex > 0 {
			if a.tabIndex != b.tabIndex {
				return cmp.Compare(a.tabIndex, b.tabIndex)
			}
			return cmp.Compare(a.order, b.order)
		}
		if a.tabIndex > 0 && b.tabIndex == 0 {
			return -1
		}
		if a.tabIndex == 0 && b.tabIndex > 0 {
			return 1
		}
		return cmp.Compare(a.order, b.order)
	})

	order := make([]input.FocusID, len(valid))
	for i, e := range valid {
		order[i] = e.focusID
	}
	return order
}
