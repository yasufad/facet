package window

import (
	"fmt"
	"math"

	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/text"
)

type drawPhase uint8

const (
	phaseNone drawPhase = iota
	phaseLayout
	phaseLayoutSolve
	phasePrepaint
	phaseHitTest
	phasePaint
)

type tabStopEntry struct {
	focusID  input.FocusID
	tabIndex int
	order    int
}

// frame holds the complete scene, hit regions, and input dispatch hierarchy for
// a single frame.
type frame struct {
	scene          *scene.Scene
	hitRegions     []hitRegion
	dispatchTree   *input.DispatchTree
	clickListeners map[input.DispatchNodeID][]func(element.ClickEvent) bool
	nodeCursors    map[input.DispatchNodeID]style.CursorStyle
	nodeFocusIDs   map[input.DispatchNodeID]input.FocusID
	focusIDs       map[input.FocusID]bool
	tabStops       []tabStopEntry
	tabOrder       []input.FocusID
}

func newFrame(keymap *input.Keymap, focusTree *input.FocusTree) *frame {
	return &frame{
		scene:          scene.New(),
		dispatchTree:   input.NewDispatchTree(keymap, focusTree),
		clickListeners: make(map[input.DispatchNodeID][]func(element.ClickEvent) bool),
		nodeCursors:    make(map[input.DispatchNodeID]style.CursorStyle),
		nodeFocusIDs:   make(map[input.DispatchNodeID]input.FocusID),
		focusIDs:       make(map[input.FocusID]bool),
	}
}

func (f *frame) clear() {
	f.scene.Clear()
	f.hitRegions = f.hitRegions[:0]
	f.dispatchTree.Clear()
	clear(f.clickListeners)
	clear(f.nodeCursors)
	clear(f.nodeFocusIDs)
	clear(f.focusIDs)
	f.tabStops = f.tabStops[:0]
	f.tabOrder = f.tabOrder[:0]
}

// Ensure *Window implements element.Frame.
var _ element.Frame = (*Window)(nil)

// RequestLayout requests a layout node in the layout engine for the current frame.
func (w *Window) RequestLayout(s layout.Style, children []layout.NodeID) layout.NodeID {
	if w.phase != phaseLayout {
		panic(fmt.Sprintf("window: RequestLayout called in phase %v (expected phaseLayout)", w.phase))
	}
	var id layout.NodeID
	if len(children) == 0 {
		id = w.layoutTree.NewLeaf(s)
	} else {
		id = w.layoutTree.NewWithChildren(s, children)
	}
	return id
}

// RequestMeasuredLayout requests a leaf layout node with a content measurement callback.
func (w *Window) RequestMeasuredLayout(s layout.Style, measure element.MeasureFunc) layout.NodeID {
	if w.phase != phaseLayout {
		panic(fmt.Sprintf("window: RequestMeasuredLayout called in phase %v (expected phaseLayout)", w.phase))
	}
	id := w.layoutTree.NewLeaf(s)
	if measure != nil {
		w.measureCallbacks[id] = measure
	}
	return id
}

// LayoutBounds returns the solved absolute bounds in logical pixels for node id.
func (w *Window) LayoutBounds(id layout.NodeID) geometry.Bounds[geometry.Pixels] {
	if w.phase != phasePrepaint && w.phase != phasePaint {
		panic(fmt.Sprintf("window: LayoutBounds called in phase %v (expected phasePrepaint or phasePaint)", w.phase))
	}
	return w.nodeBounds[id]
}

// PushDispatchNode registers an input dispatch node during prepaint.
func (w *Window) PushDispatchNode(node element.DispatchNode) input.DispatchNodeID {
	if w.phase != phasePrepaint {
		panic(fmt.Sprintf("window: PushDispatchNode called in phase %v (expected phasePrepaint)", w.phase))
	}
	nodeID := w.next.dispatchTree.PushNode()
	if node.KeyContext != nil {
		w.next.dispatchTree.SetContext(*node.KeyContext)
	}
	if node.FocusID != 0 {
		w.next.dispatchTree.SetFocusID(node.FocusID)
		w.next.focusIDs[node.FocusID] = true
		w.next.nodeFocusIDs[nodeID] = node.FocusID
		if node.TabStop || node.TabIndex != 0 {
			w.next.tabStops = append(w.next.tabStops, tabStopEntry{
				focusID:  node.FocusID,
				tabIndex: node.TabIndex,
				order:    len(w.next.tabStops),
			})
		}
	}
	if node.Cursor != style.CursorDefault {
		w.next.nodeCursors[nodeID] = node.Cursor
	}
	for _, ab := range node.ActionBindings {
		w.next.dispatchTree.OnAction(ab.ActionName, ab.Handler)
	}
	for _, kl := range node.KeyListeners {
		w.next.dispatchTree.OnKeyEvent(kl)
	}
	for _, pl := range node.PointerListeners {
		w.next.dispatchTree.OnPointerEvent(pl)
	}
	for _, wl := range node.WheelListeners {
		w.next.dispatchTree.OnWheelEvent(wl)
	}
	for _, tl := range node.TextListeners {
		w.next.dispatchTree.OnTextEvent(tl)
	}
	if len(node.ClickListeners) > 0 {
		w.next.clickListeners[nodeID] = append([]func(element.ClickEvent) bool(nil), node.ClickListeners...)
	}
	return nodeID
}

// PopDispatchNode closes the active dispatch node on the stack.
func (w *Window) PopDispatchNode() {
	if w.phase != phasePrepaint {
		panic(fmt.Sprintf("window: PopDispatchNode called in phase %v (expected phasePrepaint)", w.phase))
	}
	w.next.dispatchTree.PopNode()
}

// RegisterHitRegion commits a hit region into the in-flight frame, clipped to
// the prepaint clip mask in force at the call site.
func (w *Window) RegisterHitRegion(bounds geometry.Bounds[geometry.Pixels], nodeID input.DispatchNodeID) element.HitRegionID {
	if w.phase != phasePrepaint {
		panic(fmt.Sprintf("window: RegisterHitRegion called in phase %v (expected phasePrepaint)", w.phase))
	}
	if len(w.prepaintClipStack) > 0 {
		bounds = bounds.Intersect(w.prepaintClipStack[len(w.prepaintClipStack)-1])
	}
	w.nextHitRegionID++
	id := w.nextHitRegionID
	cursor := w.next.nodeCursors[nodeID]
	focusID := w.next.nodeFocusIDs[nodeID]
	w.next.hitRegions = append(w.next.hitRegions, hitRegion{
		id:      id,
		bounds:  bounds,
		nodeID:  nodeID,
		focusID: focusID,
		cursor:  cursor,
	})
	return id
}

// IsHovered reports whether id is hovered by the pointer in the current frame.
func (w *Window) IsHovered(id element.HitRegionID) bool {
	if w.phase != phasePaint {
		panic(fmt.Sprintf("window: IsHovered called in phase %v (expected phasePaint)", w.phase))
	}
	return id != 0 && id == w.hoveredHitRegion
}

// IsActive reports whether id is currently pressed by the pointer.
func (w *Window) IsActive(id element.HitRegionID) bool {
	if w.phase != phasePaint {
		panic(fmt.Sprintf("window: IsActive called in phase %v (expected phasePaint)", w.phase))
	}
	return id != 0 && id == w.activeHitRegion
}

// IsFocused reports whether focusID currently holds focus.
func (w *Window) IsFocused(id input.FocusID) bool {
	if w.phase != phasePaint {
		panic(fmt.Sprintf("window: IsFocused called in phase %v (expected phasePaint)", w.phase))
	}
	if id == 0 {
		return false
	}
	focused, ok := w.focusTree.Focused()
	if !ok {
		return false
	}
	return focused == id || w.focusTree.Contains(id, focused)
}

// RequestFocus moves keyboard focus to the node identified by id.
func (w *Window) RequestFocus(id input.FocusID) {
	if w.phase == phaseLayoutSolve {
		panic("window: RequestFocus called in phaseLayoutSolve")
	}
	if id == 0 {
		w.focusTree.Blur()
	} else {
		w.focusTree.Focus(id)
	}
	w.dirty = true
	w.ScheduleFrame()
}

// PushClip pushes a content clip mask onto the clip stack for the current
// phase. Valid during prepaint, where it confines the bounds later intersected
// into RegisterHitRegion, and during paint, where it also confines painted
// primitives via the scene clip stack. The two stacks are independent: the
// scene takes no primitives during prepaint, so a prepaint push does not touch
// it.
func (w *Window) PushClip(bounds geometry.Bounds[geometry.Pixels]) {
	switch w.phase {
	case phasePrepaint:
		if len(w.prepaintClipStack) > 0 {
			bounds = bounds.Intersect(w.prepaintClipStack[len(w.prepaintClipStack)-1])
		}
		w.prepaintClipStack = append(w.prepaintClipStack, bounds)
	case phasePaint:
		scale := w.scaleFactor
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
		w.clipDepth++
		w.next.scene.PushClip(scene.ContentMask[geometry.ScaledPixels]{
			Bounds: scaledBounds,
		})
	default:
		panic(fmt.Sprintf("window: PushClip called in phase %v (expected phasePrepaint or phasePaint)", w.phase))
	}
}

// PopClip pops the top content clip mask from the clip stack for the current phase.
func (w *Window) PopClip() {
	switch w.phase {
	case phasePrepaint:
		if len(w.prepaintClipStack) > 0 {
			w.prepaintClipStack = w.prepaintClipStack[:len(w.prepaintClipStack)-1]
		}
	case phasePaint:
		if w.clipDepth > 0 {
			w.clipDepth--
		}
		w.next.scene.PopClip()
	default:
		panic(fmt.Sprintf("window: PopClip called in phase %v (expected phasePrepaint or phasePaint)", w.phase))
	}
}

// InsertQuad adds a quad primitive to the in-flight frame scene.
func (w *Window) InsertQuad(q scene.Quad) {
	if w.phase != phasePaint {
		panic(fmt.Sprintf("window: InsertQuad called in phase %v (expected phasePaint)", w.phase))
	}
	w.next.scene.InsertQuad(q)
}

// InsertShadow adds a shadow primitive to the in-flight frame scene.
func (w *Window) InsertShadow(sh scene.Shadow) {
	if w.phase != phasePaint {
		panic(fmt.Sprintf("window: InsertShadow called in phase %v (expected phasePaint)", w.phase))
	}
	w.next.scene.InsertShadow(sh)
}

// InsertPath adds a vector path primitive to the in-flight frame scene.
func (w *Window) InsertPath(p scene.Path[geometry.ScaledPixels]) {
	if w.phase != phasePaint {
		panic(fmt.Sprintf("window: InsertPath called in phase %v (expected phasePaint)", w.phase))
	}
	w.next.scene.InsertPath(p)
}

// InsertUnderline adds an underline primitive to the in-flight frame scene.
func (w *Window) InsertUnderline(u scene.Underline) {
	if w.phase != phasePaint {
		panic(fmt.Sprintf("window: InsertUnderline called in phase %v (expected phasePaint)", w.phase))
	}
	w.next.scene.InsertUnderline(u)
}

// InsertMonochromeSprite adds a monochrome sprite primitive to the in-flight frame scene.
func (w *Window) InsertMonochromeSprite(sp scene.MonochromeSprite) {
	if w.phase != phasePaint {
		panic(fmt.Sprintf("window: InsertMonochromeSprite called in phase %v (expected phasePaint)", w.phase))
	}
	w.next.scene.InsertMonochromeSprite(sp)
}

// InsertPolychromeSprite adds a polychrome sprite primitive to the in-flight frame scene.
func (w *Window) InsertPolychromeSprite(sp scene.PolychromeSprite) {
	if w.phase != phasePaint {
		panic(fmt.Sprintf("window: InsertPolychromeSprite called in phase %v (expected phasePaint)", w.phase))
	}
	w.next.scene.InsertPolychromeSprite(sp)
}

// ShapeLine shapes a single line of text with the window's text system.
func (w *Window) ShapeLine(str string, runs []text.StyleRun) (text.ShapedLine, error) {
	if w.phase != phaseLayoutSolve && w.phase != phasePaint {
		panic(fmt.Sprintf("window: ShapeLine called in phase %v (expected phaseLayoutSolve or phasePaint)", w.phase))
	}
	if w.textSystem == nil {
		return text.ShapedLine{}, nil
	}
	return w.textSystem.ShapeLine(str, runs)
}

// RasteriseGlyph returns the atlas tile and device-pixel bounding box relative to
// the pen position for the specified glyph, rasterising and uploading on miss.
func (w *Window) RasteriseGlyph(face text.Face, gid text.GlyphID, size geometry.Pixels, subpixel text.SubpixelOffset) (scene.AtlasTile, geometry.Bounds[geometry.DevicePixels], bool) {
	if w.phase != phasePaint {
		panic(fmt.Sprintf("window: RasteriseGlyph called in phase %v (expected phasePaint)", w.phase))
	}
	if w.textAtlas == nil || w.renderer == nil {
		return scene.AtlasTile{}, geometry.Bounds[geometry.DevicePixels]{}, false
	}

	key := glyphCacheKey{
		face:     face,
		gid:      gid,
		sizeBits: math.Float32bits(float32(size)),
		subpixel: subpixel,
	}
	if cached, ok := w.glyphTileCache[key]; ok {
		return cached.tile, cached.bounds, cached.ok
	}

	entry := w.textAtlas.Entry(face, gid, size, subpixel)
	if entry.Mask.Width <= 0 || entry.Mask.Height <= 0 || len(entry.Mask.Coverage) == 0 {
		w.glyphTileCache[key] = cachedGlyph{bounds: entry.Bounds, ok: false}
		return scene.AtlasTile{}, entry.Bounds, false
	}

	tileSize := geometry.NewSize[geometry.DevicePixels](
		geometry.DevicePixels(entry.Mask.Width),
		geometry.DevicePixels(entry.Mask.Height),
	)
	tile, err := w.renderer.Upload(scene.TextureMonochrome, tileSize, entry.Mask.Coverage)
	if err != nil {
		w.glyphTileCache[key] = cachedGlyph{bounds: entry.Bounds, ok: false}
		return scene.AtlasTile{}, entry.Bounds, false
	}

	res := cachedGlyph{
		tile:   tile,
		bounds: entry.Bounds,
		ok:     true,
	}
	w.glyphTileCache[key] = res
	return res.tile, res.bounds, true
}

// ScaleFactor returns the display scale factor for physical device pixel snapping.
func (w *Window) ScaleFactor() float32 {
	return w.scaleFactor
}

// RemSize returns the root font size in logical pixels.
func (w *Window) RemSize() geometry.Pixels {
	return w.remSize
}

// PushTextStyle pushes a text style refinement onto the inherited text style stack.
func (w *Window) PushTextStyle(refinement style.Refinement) {
	current := w.TextStyle()
	var s style.Style
	s.Text = current
	s.Refine(refinement)
	w.textStyleStack = append(w.textStyleStack, s.Text)
}

// PopTextStyle pops the top text style refinement from the stack.
func (w *Window) PopTextStyle() {
	if len(w.textStyleStack) > 1 {
		w.textStyleStack = w.textStyleStack[:len(w.textStyleStack)-1]
	}
}

// TextStyle returns the current inherited text style computed by layering all
// pushed refinements from root to the current element.
func (w *Window) TextStyle() style.TextStyle {
	if len(w.textStyleStack) == 0 {
		return style.DefaultTextStyle()
	}
	return w.textStyleStack[len(w.textStyleStack)-1]
}
