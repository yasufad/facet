package element

import (
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/text"
)

type recordedHitRegion struct {
	id     HitRegionID
	bounds geometry.Bounds[geometry.Pixels]
	nodeID input.DispatchNodeID
}

type fakeFrame struct {
	phase       drawPhase
	scaleFactor float32
	remSize     geometry.Pixels

	tree             *layout.TaffyTree
	nodes            []layout.NodeID
	styles           map[layout.NodeID]layout.Style
	children         map[layout.NodeID][]layout.NodeID
	bounds           map[layout.NodeID]geometry.Bounds[geometry.Pixels]
	measureCallbacks map[layout.NodeID]MeasureFunc

	dispatchTree      *input.DispatchTree
	dispatchNodes     []DispatchNode
	dispatchNodeStack []input.DispatchNodeID

	nextHitRegionID   HitRegionID
	hitRegions        []recordedHitRegion
	hoveredHitRegions map[HitRegionID]bool
	activeHitRegions  map[HitRegionID]bool
	focusedIDs        map[input.FocusID]bool

	quads             []scene.Quad
	shadows           []scene.Shadow
	paths             []scene.Path[geometry.ScaledPixels]
	underlines        []scene.Underline
	monoSprites       []scene.MonochromeSprite
	polySprites       []scene.PolychromeSprite
	textSys           *text.System
	textStyleStack    []style.TextStyle
	clips             []geometry.Bounds[geometry.Pixels]
	pushedClips       []geometry.Bounds[geometry.Pixels]
	popClipCount      int
	prepaintClipStack []geometry.Bounds[geometry.Pixels]
}

func newFakeFrame() *fakeFrame {
	txtSys, _ := text.NewSystem()
	return &fakeFrame{
		phase:             phaseLayoutRequested,
		scaleFactor:       2.0,
		remSize:           16.0,
		tree:              layout.NewTaffyTree(),
		styles:            make(map[layout.NodeID]layout.Style),
		children:          make(map[layout.NodeID][]layout.NodeID),
		bounds:            make(map[layout.NodeID]geometry.Bounds[geometry.Pixels]),
		measureCallbacks:  make(map[layout.NodeID]MeasureFunc),
		dispatchTree:      input.NewDispatchTree(nil, nil),
		hoveredHitRegions: make(map[HitRegionID]bool),
		activeHitRegions:  make(map[HitRegionID]bool),
		focusedIDs:        make(map[input.FocusID]bool),
		textSys:           txtSys,
		textStyleStack:    []style.TextStyle{style.DefaultTextStyle()},
	}
}

func (f *fakeFrame) RequestLayout(s layout.Style, children []layout.NodeID) layout.NodeID {
	if f.phase != phaseLayoutRequested {
		panic("fakeFrame: RequestLayout called in wrong phase")
	}
	var id layout.NodeID
	if len(children) == 0 {
		id = f.tree.NewLeaf(s)
	} else {
		id = f.tree.NewWithChildren(s, children)
	}
	f.nodes = append(f.nodes, id)
	f.styles[id] = s
	f.children[id] = append([]layout.NodeID(nil), children...)
	return id
}

func (f *fakeFrame) RequestMeasuredLayout(s layout.Style, measure MeasureFunc) layout.NodeID {
	if f.phase != phaseLayoutRequested {
		panic("fakeFrame: RequestMeasuredLayout called in wrong phase")
	}
	id := f.tree.NewLeaf(s)
	f.nodes = append(f.nodes, id)
	f.styles[id] = s
	if measure != nil {
		f.measureCallbacks[id] = measure
	}
	return id
}

func (f *fakeFrame) LayoutBounds(id layout.NodeID) geometry.Bounds[geometry.Pixels] {
	return f.bounds[id]
}

func (f *fakeFrame) PushDispatchNode(node DispatchNode) input.DispatchNodeID {
	if f.phase != phasePrepainted {
		panic("fakeFrame: PushDispatchNode called outside prepaint phase")
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

	f.dispatchNodes = append(f.dispatchNodes, node)
	f.dispatchNodeStack = append(f.dispatchNodeStack, nodeID)
	return nodeID
}

func (f *fakeFrame) PopDispatchNode() {
	if f.phase != phasePrepainted {
		panic("fakeFrame: PopDispatchNode called outside prepaint phase")
	}
	f.dispatchTree.PopNode()
	if len(f.dispatchNodeStack) > 0 {
		f.dispatchNodeStack = f.dispatchNodeStack[:len(f.dispatchNodeStack)-1]
	}
}

func (f *fakeFrame) RegisterHitRegion(bounds geometry.Bounds[geometry.Pixels], nodeID input.DispatchNodeID) HitRegionID {
	if f.phase != phasePrepainted {
		panic("fakeFrame: RegisterHitRegion called outside prepaint phase")
	}
	if len(f.prepaintClipStack) > 0 {
		bounds = bounds.Intersect(f.prepaintClipStack[len(f.prepaintClipStack)-1])
	}
	f.nextHitRegionID++
	id := f.nextHitRegionID
	f.hitRegions = append(f.hitRegions, recordedHitRegion{
		id:     id,
		bounds: bounds,
		nodeID: nodeID,
	})
	return id
}

func (f *fakeFrame) IsHovered(id HitRegionID) bool {
	if f.phase != phasePainted {
		panic("fakeFrame: IsHovered called outside paint phase")
	}
	return f.hoveredHitRegions[id]
}

func (f *fakeFrame) IsActive(id HitRegionID) bool {
	if f.phase != phasePainted {
		panic("fakeFrame: IsActive called outside paint phase")
	}
	return f.activeHitRegions[id]
}

func (f *fakeFrame) IsFocused(id input.FocusID) bool {
	if f.phase != phasePainted {
		panic("fakeFrame: IsFocused called outside paint phase")
	}
	return f.focusedIDs[id]
}

func (f *fakeFrame) RequestFocus(id input.FocusID) {
	f.focusedIDs = make(map[input.FocusID]bool)
	if id != 0 {
		f.focusedIDs[id] = true
	}
}

func (f *fakeFrame) PushClip(bounds geometry.Bounds[geometry.Pixels]) {
	switch f.phase {
	case phasePrepainted:
		if len(f.prepaintClipStack) > 0 {
			bounds = bounds.Intersect(f.prepaintClipStack[len(f.prepaintClipStack)-1])
		}
		f.prepaintClipStack = append(f.prepaintClipStack, bounds)
	case phasePainted:
		f.clips = append(f.clips, bounds)
		f.pushedClips = append(f.pushedClips, bounds)
	default:
		panic("fakeFrame: PushClip called outside prepaint or paint phase")
	}
}

func (f *fakeFrame) PopClip() {
	switch f.phase {
	case phasePrepainted:
		if len(f.prepaintClipStack) > 0 {
			f.prepaintClipStack = f.prepaintClipStack[:len(f.prepaintClipStack)-1]
		}
	case phasePainted:
		f.popClipCount++
		if len(f.clips) > 0 {
			f.clips = f.clips[:len(f.clips)-1]
		}
	default:
		panic("fakeFrame: PopClip called outside prepaint or paint phase")
	}
}

func (f *fakeFrame) setHovered(id HitRegionID, hovered bool) {
	f.hoveredHitRegions[id] = hovered
}

func (f *fakeFrame) setActive(id HitRegionID, active bool) {
	f.activeHitRegions[id] = active
}

func (f *fakeFrame) setFocused(id input.FocusID, focused bool) {
	f.focusedIDs[id] = focused
}

func (f *fakeFrame) InsertQuad(q scene.Quad) {
	if f.phase != phasePainted {
		panic("fakeFrame: InsertQuad called outside paint phase")
	}
	f.quads = append(f.quads, q)
}

func (f *fakeFrame) InsertShadow(sh scene.Shadow) {
	if f.phase != phasePainted {
		panic("fakeFrame: InsertShadow called outside paint phase")
	}
	f.shadows = append(f.shadows, sh)
}

func (f *fakeFrame) InsertPath(p scene.Path[geometry.ScaledPixels]) {
	if f.phase != phasePainted {
		panic("fakeFrame: InsertPath called outside paint phase")
	}
	f.paths = append(f.paths, p)
}

func (f *fakeFrame) InsertUnderline(u scene.Underline) {
	if f.phase != phasePainted {
		panic("fakeFrame: InsertUnderline called outside paint phase")
	}
	f.underlines = append(f.underlines, u)
}

func (f *fakeFrame) InsertMonochromeSprite(sp scene.MonochromeSprite) {
	if f.phase != phasePainted {
		panic("fakeFrame: InsertMonochromeSprite called outside paint phase")
	}
	f.monoSprites = append(f.monoSprites, sp)
}

func (f *fakeFrame) InsertPolychromeSprite(sp scene.PolychromeSprite) {
	if f.phase != phasePainted {
		panic("fakeFrame: InsertPolychromeSprite called outside paint phase")
	}
	f.polySprites = append(f.polySprites, sp)
}

func (f *fakeFrame) ShapeLine(str string, runs []text.StyleRun) (text.ShapedLine, error) {
	if f.textSys != nil {
		return f.textSys.ShapeLine(str, runs)
	}
	return text.ShapedLine{}, nil
}

func (f *fakeFrame) RasteriseGlyph(face text.Face, gid text.GlyphID, size geometry.Pixels, subpixel text.SubpixelOffset) (scene.AtlasTile, geometry.Bounds[geometry.DevicePixels], bool) {
	if f.phase != phasePainted {
		panic("fakeFrame: RasteriseGlyph called outside paint phase")
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

func (f *fakeFrame) ScaleFactor() float32 {
	return f.scaleFactor
}

func (f *fakeFrame) RemSize() geometry.Pixels {
	return f.remSize
}

func (f *fakeFrame) PushTextStyle(refinement style.Refinement) {
	current := f.TextStyle()
	var s style.Style
	s.Text = current
	s.Refine(refinement)
	f.textStyleStack = append(f.textStyleStack, s.Text)
}

func (f *fakeFrame) PopTextStyle() {
	if len(f.textStyleStack) > 1 {
		f.textStyleStack = f.textStyleStack[:len(f.textStyleStack)-1]
	}
}

func (f *fakeFrame) TextStyle() style.TextStyle {
	if len(f.textStyleStack) == 0 {
		return style.DefaultTextStyle()
	}
	return f.textStyleStack[len(f.textStyleStack)-1]
}

// solve computes layout on the underlying layout tree and derives window-relative bounds.
func (f *fakeFrame) solve(root layout.NodeID, availableWidth, availableHeight float32) {
	// For testing with mock layout without available space constraints, we assign mock sizes or compute via Taffy.
	f.computeRecursiveBounds(root, geometry.Point[geometry.Pixels]{X: 0, Y: 0}, geometry.Size[geometry.Pixels]{
		Width:  geometry.Pixels(availableWidth),
		Height: geometry.Pixels(availableHeight),
	})
}

func (f *fakeFrame) computeRecursiveBounds(node layout.NodeID, origin geometry.Point[geometry.Pixels], size geometry.Size[geometry.Pixels]) {
	f.bounds[node] = geometry.Bounds[geometry.Pixels]{
		Origin: origin,
		Size:   size,
	}
	children := f.children[node]
	if len(children) == 0 {
		return
	}
	// Simple horizontal flex distribution for test verification.
	childWidth := size.Width / geometry.Pixels(len(children))
	currentX := origin.X
	for _, childID := range children {
		childOrigin := geometry.Point[geometry.Pixels]{X: currentX, Y: origin.Y}
		childSize := geometry.Size[geometry.Pixels]{Width: childWidth, Height: size.Height}
		f.computeRecursiveBounds(childID, childOrigin, childSize)
		currentX += childWidth
	}
}
