package ui

import (
	"testing"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/text"
)

type recordedHitRegion struct {
	id     element.HitRegionID
	bounds geometry.Bounds[geometry.Pixels]
	nodeID input.DispatchNodeID
}

type testFrame struct {
	phase       drawPhase
	scaleFactor float32
	remSize     geometry.Pixels

	tree             *layout.TaffyTree
	nodes            []layout.NodeID
	styles           map[layout.NodeID]layout.Style
	children         map[layout.NodeID][]layout.NodeID
	bounds           map[layout.NodeID]geometry.Bounds[geometry.Pixels]
	measureCallbacks map[layout.NodeID]element.MeasureFunc

	dispatchTree      *input.DispatchTree
	dispatchNodes     []element.DispatchNode
	dispatchNodeStack []input.DispatchNodeID
	clickListeners    map[input.DispatchNodeID][]func(element.ClickEvent) bool

	nextHitRegionID   element.HitRegionID
	hitRegions        []recordedHitRegion
	hoveredHitRegions map[element.HitRegionID]bool
	activeHitRegions  map[element.HitRegionID]bool
	focusedIDs        map[input.FocusID]bool

	downHitRegion element.HitRegionID
	downNodeID    input.DispatchNodeID

	quads       []scene.Quad
	shadows     []scene.Shadow
	paths       []scene.Path[geometry.ScaledPixels]
	underlines  []scene.Underline
	monoSprites []scene.MonochromeSprite
	polySprites []scene.PolychromeSprite
	textSys     *text.System
}

type drawPhase uint8

const (
	phaseLayoutRequested drawPhase = iota
	phasePrepainted
	phasePainted
)

func newTestFrame() *testFrame {
	txtSys, _ := text.NewSystem()
	return &testFrame{
		phase:             phaseLayoutRequested,
		scaleFactor:       2.0,
		remSize:           16.0,
		tree:              layout.NewTaffyTree(),
		styles:            make(map[layout.NodeID]layout.Style),
		children:          make(map[layout.NodeID][]layout.NodeID),
		bounds:            make(map[layout.NodeID]geometry.Bounds[geometry.Pixels]),
		measureCallbacks:  make(map[layout.NodeID]element.MeasureFunc),
		dispatchTree:      input.NewDispatchTree(nil, nil),
		clickListeners:    make(map[input.DispatchNodeID][]func(element.ClickEvent) bool),
		hoveredHitRegions: make(map[element.HitRegionID]bool),
		activeHitRegions:  make(map[element.HitRegionID]bool),
		focusedIDs:        make(map[input.FocusID]bool),
		textSys:           txtSys,
	}
}

func (f *testFrame) RequestLayout(s layout.Style, children []layout.NodeID) layout.NodeID {
	if f.phase != phaseLayoutRequested {
		panic("testFrame: RequestLayout called in wrong phase")
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

func (f *testFrame) RequestMeasuredLayout(s layout.Style, measure element.MeasureFunc) layout.NodeID {
	if f.phase != phaseLayoutRequested {
		panic("testFrame: RequestMeasuredLayout called in wrong phase")
	}
	id := f.tree.NewLeaf(s)
	f.nodes = append(f.nodes, id)
	f.styles[id] = s
	if measure != nil {
		f.measureCallbacks[id] = measure
	}
	return id
}

func (f *testFrame) LayoutBounds(id layout.NodeID) geometry.Bounds[geometry.Pixels] {
	return f.bounds[id]
}

func (f *testFrame) PushDispatchNode(node element.DispatchNode) input.DispatchNodeID {
	if f.phase != phasePrepainted {
		panic("testFrame: PushDispatchNode called outside prepaint phase")
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

func (f *testFrame) PopDispatchNode() {
	if f.phase != phasePrepainted {
		panic("testFrame: PopDispatchNode called outside prepaint phase")
	}
	f.dispatchTree.PopNode()
	if len(f.dispatchNodeStack) > 0 {
		f.dispatchNodeStack = f.dispatchNodeStack[:len(f.dispatchNodeStack)-1]
	}
}

func (f *testFrame) RegisterHitRegion(bounds geometry.Bounds[geometry.Pixels], nodeID input.DispatchNodeID) element.HitRegionID {
	if f.phase != phasePrepainted {
		panic("testFrame: RegisterHitRegion called outside prepaint phase")
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

func (f *testFrame) IsHovered(id element.HitRegionID) bool {
	if f.phase != phasePainted {
		panic("testFrame: IsHovered called outside paint phase")
	}
	return f.hoveredHitRegions[id]
}

func (f *testFrame) IsActive(id element.HitRegionID) bool {
	if f.phase != phasePainted {
		panic("testFrame: IsActive called outside paint phase")
	}
	return f.activeHitRegions[id]
}

func (f *testFrame) IsFocused(id input.FocusID) bool {
	if f.phase != phasePainted {
		panic("testFrame: IsFocused called outside paint phase")
	}
	return f.focusedIDs[id]
}

func (f *testFrame) setHovered(id element.HitRegionID, hovered bool) {
	f.hoveredHitRegions[id] = hovered
}

func (f *testFrame) setActive(id element.HitRegionID, active bool) {
	f.activeHitRegions[id] = active
}

func (f *testFrame) setFocused(id input.FocusID, focused bool) {
	f.focusedIDs[id] = focused
}

func (f *testFrame) InsertQuad(q scene.Quad) {
	if f.phase != phasePainted {
		panic("testFrame: InsertQuad called outside paint phase")
	}
	f.quads = append(f.quads, q)
}

func (f *testFrame) InsertShadow(sh scene.Shadow) {
	if f.phase != phasePainted {
		panic("testFrame: InsertShadow called outside paint phase")
	}
	f.shadows = append(f.shadows, sh)
}

func (f *testFrame) InsertPath(p scene.Path[geometry.ScaledPixels]) {
	if f.phase != phasePainted {
		panic("testFrame: InsertPath called outside paint phase")
	}
	f.paths = append(f.paths, p)
}

func (f *testFrame) InsertUnderline(u scene.Underline) {
	if f.phase != phasePainted {
		panic("testFrame: InsertUnderline called outside paint phase")
	}
	f.underlines = append(f.underlines, u)
}

func (f *testFrame) InsertMonochromeSprite(sp scene.MonochromeSprite) {
	if f.phase != phasePainted {
		panic("testFrame: InsertMonochromeSprite called outside paint phase")
	}
	f.monoSprites = append(f.monoSprites, sp)
}

func (f *testFrame) InsertPolychromeSprite(sp scene.PolychromeSprite) {
	if f.phase != phasePainted {
		panic("testFrame: InsertPolychromeSprite called outside paint phase")
	}
	f.polySprites = append(f.polySprites, sp)
}

func (f *testFrame) ShapeLine(str string, runs []text.StyleRun) (text.ShapedLine, error) {
	if f.textSys != nil {
		return f.textSys.ShapeLine(str, runs)
	}
	return text.ShapedLine{}, nil
}

func (f *testFrame) RasteriseGlyph(face text.Face, gid text.GlyphID, size geometry.Pixels, subpixel text.SubpixelOffset) (scene.AtlasTile, geometry.Bounds[geometry.DevicePixels], bool) {
	if f.phase != phasePainted {
		panic("testFrame: RasteriseGlyph called outside paint phase")
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

func (f *testFrame) ScaleFactor() float32 {
	return f.scaleFactor
}

func (f *testFrame) RemSize() geometry.Pixels {
	return f.remSize
}

func (f *testFrame) solve(root layout.NodeID, availableWidth, availableHeight float32) {
	avail := layout.Size[layout.AvailableSpace]{
		Width:  layout.Definite(availableWidth),
		Height: layout.Definite(availableHeight),
	}
	measureFn := func(inputs layout.LayoutInput, id layout.NodeID, ctx any, style *layout.Style) layout.LayoutOutput {
		cb, ok := f.measureCallbacks[id]
		if !ok || cb == nil {
			return layout.ComputeLeafLayout(inputs, style, nil)
		}
		leafMeasure := func(known layout.Size[layout.OptF32], avail layout.Size[layout.AvailableSpace]) layout.Size[float32] {
			contentSize := cb(known, avail)
			return layout.Size[float32]{
				Width:  float32(contentSize.Width),
				Height: float32(contentSize.Height),
			}
		}
		return layout.ComputeLeafLayout(inputs, style, leafMeasure)
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

func (f *testFrame) populateBounds(parentID layout.NodeID, parentOrigin geometry.Point[geometry.Pixels]) {
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

func (f *testFrame) dispatchPointer(event platform.PointerEvent) {
	pt := geometry.Point[geometry.Pixels]{
		X: event.Position.X.ToPixels(f.scaleFactor),
		Y: event.Position.Y.ToPixels(f.scaleFactor),
	}

	var hitID element.HitRegionID
	var nodeID input.DispatchNodeID = -1
	for i := len(f.hitRegions) - 1; i >= 0; i-- {
		if f.hitRegions[i].bounds.Contains(pt) {
			hitID = f.hitRegions[i].id
			nodeID = f.hitRegions[i].nodeID
			break
		}
	}

	if event.Phase == platform.PointerDown {
		f.downHitRegion = hitID
		f.downNodeID = nodeID
	} else if event.Phase == platform.PointerUp {
		if hitID != 0 && hitID == f.downHitRegion {
			clickEvt := element.ClickEvent{
				Position:  pt,
				Button:    element.MouseButton(event.Button),
				Modifiers: element.Modifiers(event.Modifiers),
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
		f.downNodeID = -1
	}

	if nodeID >= 0 {
		f.dispatchTree.DispatchPointer(event, nodeID)
	}
}

func TestButtonLifecycleAndScene(t *testing.T) {
	frame := newTestFrame()
	btn := NewButton("Save Changes")

	rootID := btn.RequestLayout(frame)
	frame.solve(rootID, 200, 100)

	btnBounds := frame.LayoutBounds(rootID)
	if btnBounds.Size.Width <= 0 || btnBounds.Size.Height <= 0 {
		t.Fatalf("expected positive solved button dimensions, got %v", btnBounds.Size)
	}

	frame.phase = phasePrepainted
	btn.Prepaint(frame, btnBounds)

	if len(frame.hitRegions) == 0 {
		t.Fatalf("expected hit region registered during prepaint")
	}

	frame.phase = phasePainted
	btn.Paint(frame, btnBounds)

	if len(frame.quads) != 1 {
		t.Fatalf("expected exactly 1 background quad, got %d", len(frame.quads))
	}

	q := frame.quads[0]
	if q.Background != defaultButtonBg {
		t.Errorf("expected default background %v, got %v", defaultButtonBg, q.Background)
	}
	if q.BorderColour != defaultButtonBorder {
		t.Errorf("expected default border colour %v, got %v", defaultButtonBorder, q.BorderColour)
	}
	expectedRadius := geometry.ScaledPixels(4.0 * frame.scaleFactor)
	if q.CornerRadii.TopLeft != expectedRadius {
		t.Errorf("expected corner radius %v, got %v", expectedRadius, q.CornerRadii.TopLeft)
	}

	if len(frame.monoSprites) == 0 {
		t.Fatalf("expected monochrome text sprites emitted into scene")
	}
}

func TestButtonClickDispatch(t *testing.T) {
	frame := newTestFrame()
	clicked := false

	btn := NewButton("Click Me").
		OnClick(func(event element.ClickEvent) bool {
			clicked = true
			return true
		})

	rootID := btn.RequestLayout(frame)
	frame.solve(rootID, 200, 100)

	bounds := frame.LayoutBounds(rootID)
	frame.phase = phasePrepainted
	btn.Prepaint(frame, bounds)

	centerPt := geometry.Point[geometry.DevicePixels]{
		X: geometry.DevicePixels((bounds.Origin.X + bounds.Size.Width/2) * geometry.Pixels(frame.scaleFactor)),
		Y: geometry.DevicePixels((bounds.Origin.Y + bounds.Size.Height/2) * geometry.Pixels(frame.scaleFactor)),
	}

	frame.dispatchPointer(platform.PointerEvent{
		Position: centerPt,
		Phase:    platform.PointerDown,
		Button:   platform.PointerLeft,
	})

	if clicked {
		t.Fatalf("click handler fired prematurely on PointerDown")
	}

	frame.dispatchPointer(platform.PointerEvent{
		Position: centerPt,
		Phase:    platform.PointerUp,
		Button:   platform.PointerLeft,
	})

	if !clicked {
		t.Fatalf("expected click handler to fire on PointerUp")
	}
}

func TestButtonHoverStyling(t *testing.T) {
	frame := newTestFrame()
	btn := NewButton("Hover Test")

	rootID := btn.RequestLayout(frame)
	frame.solve(rootID, 200, 100)

	bounds := frame.LayoutBounds(rootID)
	frame.phase = phasePrepainted
	btn.Prepaint(frame, bounds)

	hitID := frame.hitRegions[0].id
	frame.setHovered(hitID, true)

	frame.phase = phasePainted
	btn.Paint(frame, bounds)

	if len(frame.quads) != 1 {
		t.Fatalf("expected 1 quad, got %d", len(frame.quads))
	}
	if frame.quads[0].Background != defaultButtonHoverBg {
		t.Errorf("expected hover background %v, got %v", defaultButtonHoverBg, frame.quads[0].Background)
	}
}

func TestButtonActiveStyling(t *testing.T) {
	frame := newTestFrame()
	btn := NewButton("Active Test")

	rootID := btn.RequestLayout(frame)
	frame.solve(rootID, 200, 100)

	bounds := frame.LayoutBounds(rootID)
	frame.phase = phasePrepainted
	btn.Prepaint(frame, bounds)

	hitID := frame.hitRegions[0].id
	frame.setActive(hitID, true)

	frame.phase = phasePainted
	btn.Paint(frame, bounds)

	if len(frame.quads) != 1 {
		t.Fatalf("expected 1 quad, got %d", len(frame.quads))
	}
	if frame.quads[0].Background != defaultButtonActiveBg {
		t.Errorf("expected active background %v, got %v", defaultButtonActiveBg, frame.quads[0].Background)
	}
}

func TestButtonFocusStyling(t *testing.T) {
	frame := newTestFrame()
	focusID := input.NewFocusID()
	btn := NewButton("Focus Test").TrackFocus(focusID)

	rootID := btn.RequestLayout(frame)
	frame.solve(rootID, 200, 100)

	bounds := frame.LayoutBounds(rootID)
	frame.phase = phasePrepainted
	btn.Prepaint(frame, bounds)

	frame.setFocused(focusID, true)

	frame.phase = phasePainted
	btn.Paint(frame, bounds)

	if len(frame.quads) != 1 {
		t.Fatalf("expected 1 quad, got %d", len(frame.quads))
	}
	q := frame.quads[0]
	if q.BorderColour != defaultButtonFocus {
		t.Errorf("expected focus border colour %v, got %v", defaultButtonFocus, q.BorderColour)
	}
	expectedBorderWidth := geometry.ScaledPixels(2.0 * frame.scaleFactor)
	if q.BorderWidths.Top != expectedBorderWidth {
		t.Errorf("expected focus border width %v, got %v", expectedBorderWidth, q.BorderWidths.Top)
	}
}

func TestButtonDisabled(t *testing.T) {
	frame := newTestFrame()
	clicked := false

	btn := NewButton("Disabled Button").
		Disabled(true).
		OnClick(func(event element.ClickEvent) bool {
			clicked = true
			return true
		})

	rootID := btn.RequestLayout(frame)
	frame.solve(rootID, 200, 100)

	bounds := frame.LayoutBounds(rootID)
	frame.phase = phasePrepainted
	btn.Prepaint(frame, bounds)

	centerPt := geometry.Point[geometry.DevicePixels]{
		X: geometry.DevicePixels((bounds.Origin.X + bounds.Size.Width/2) * geometry.Pixels(frame.scaleFactor)),
		Y: geometry.DevicePixels((bounds.Origin.Y + bounds.Size.Height/2) * geometry.Pixels(frame.scaleFactor)),
	}

	frame.dispatchPointer(platform.PointerEvent{
		Position: centerPt,
		Phase:    platform.PointerDown,
		Button:   platform.PointerLeft,
	})
	frame.dispatchPointer(platform.PointerEvent{
		Position: centerPt,
		Phase:    platform.PointerUp,
		Button:   platform.PointerLeft,
	})

	if clicked {
		t.Fatalf("disabled button must not fire click events")
	}
}

func TestButtonEmptyLabel(t *testing.T) {
	frame := newTestFrame()
	btn := NewButton("")

	rootID := btn.RequestLayout(frame)
	frame.solve(rootID, 100, 50)

	bounds := frame.LayoutBounds(rootID)
	frame.phase = phasePrepainted
	btn.Prepaint(frame, bounds)

	frame.phase = phasePainted
	btn.Paint(frame, bounds)

	if len(frame.quads) != 1 {
		t.Fatalf("expected 1 quad for empty button, got %d", len(frame.quads))
	}
	if len(frame.monoSprites) != 0 {
		t.Fatalf("expected 0 text sprites for empty button, got %d", len(frame.monoSprites))
	}
}

func TestButtonCustomRefinement(t *testing.T) {
	frame := newTestFrame()
	customBg := colour.Rgba{R: 0.9, G: 0.1, B: 0.1, A: 1.0}
	customHover := colour.Rgba{R: 1.0, G: 0.2, B: 0.2, A: 1.0}

	var customRefine style.Refinement
	customRefine.SetBackground(customBg)

	btn := NewButton("Custom").
		Refine(customRefine).
		Hover(func(r *style.Refinement) {
			r.SetBackground(customHover)
		})

	rootID := btn.RequestLayout(frame)
	frame.solve(rootID, 150, 60)

	bounds := frame.LayoutBounds(rootID)
	frame.phase = phasePrepainted
	btn.Prepaint(frame, bounds)

	// Normal paint
	frame.phase = phasePainted
	btn.Paint(frame, bounds)
	if len(frame.quads) != 1 || frame.quads[0].Background != customBg {
		t.Fatalf("expected custom base background %v, got %v", customBg, frame.quads[0].Background)
	}

	// Hover frame pass with fresh element
	hoverFrame := newTestFrame()
	hoverBtn := NewButton("Custom").
		Refine(customRefine).
		Hover(func(r *style.Refinement) {
			r.SetBackground(customHover)
		})

	hoverRootID := hoverBtn.RequestLayout(hoverFrame)
	hoverFrame.solve(hoverRootID, 150, 60)
	hoverBounds := hoverFrame.LayoutBounds(hoverRootID)

	hoverFrame.phase = phasePrepainted
	hoverBtn.Prepaint(hoverFrame, hoverBounds)

	hoverFrame.setHovered(hoverFrame.hitRegions[0].id, true)
	hoverFrame.phase = phasePainted
	hoverBtn.Paint(hoverFrame, hoverBounds)

	if len(hoverFrame.quads) != 1 || hoverFrame.quads[0].Background != customHover {
		t.Fatalf("expected custom hover background %v, got %v", customHover, hoverFrame.quads[0].Background)
	}
}
