package window

import (
	"cmp"
	"fmt"
	"slices"
	"sync"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/render"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/text"
)

type staticView struct {
	el element.Element
}

func (s staticView) Render(_ *app.App) element.Element {
	return s.el
}

func (s staticView) Observe(_ *app.App, _ func(*app.App) bool) app.Subscription {
	return app.Subscription{}
}

type fnView struct {
	fn func() element.Element
}

func (f fnView) Render(_ *app.App) element.Element {
	if f.fn == nil {
		return nil
	}
	return f.fn()
}

func (f fnView) Observe(_ *app.App, _ func(*app.App) bool) app.Subscription {
	return app.Subscription{}
}

type glyphCacheKey struct {
	face     text.Face
	gid      text.GlyphID
	sizeBits uint32
	subpixel text.SubpixelOffset
}

type cachedGlyph struct {
	tile   scene.AtlasTile
	bounds geometry.Bounds[geometry.DevicePixels]
	ok     bool
}

// Window owns a platform window, layout engine, and graphics renderer, and
// coordinates the seven-step frame loop from layout through paint and presentation.
type Window struct {
	platform       platform.Platform
	platformWindow platform.Window
	renderer       render.Renderer
	app            *app.App

	layoutTree *layout.TaffyTree
	textSystem *text.System
	textAtlas  *text.Atlas

	rendered *frame
	next     *frame
	rootView element.AnyView
	rootSub  app.Subscription

	scaleFactor float32
	size        geometry.Size[geometry.Pixels]
	remSize     geometry.Pixels

	dirty        bool
	needsPresent bool
	phase        drawPhase

	pointerPos       geometry.Point[geometry.Pixels]
	pointerDown      bool
	hoveredHitRegion element.HitRegionID
	hoveredNodeID    input.DispatchNodeID
	activeHitRegion  element.HitRegionID

	focusTree         *input.FocusTree
	keymap            *input.Keymap
	nextHitRegionID   element.HitRegionID
	nodeBounds        map[layout.NodeID]geometry.Bounds[geometry.Pixels]
	measureCallbacks  map[layout.NodeID]element.MeasureFunc
	glyphTileCache    map[glyphCacheKey]cachedGlyph
	textStyleStack    []style.TextStyle
	currentCursor     platform.Cursor
	clipDepth         int
	prepaintClipStack []geometry.Bounds[geometry.Pixels]

	// captureActive routes PointerMove and PointerUp to captureNodeID
	// regardless of what is now under the pointer, from pointer-down against
	// a hit region until the matching pointer-up. captureBounds is refreshed
	// every frame (see resolveCapturedHitTest) so IsActive tracks the
	// captured element even if it moves or resizes mid-drag.
	captureActive bool
	captureNodeID input.DispatchNodeID
	captureBounds geometry.Bounds[geometry.Pixels]

	mu             sync.Mutex
	frameScheduled bool
	focused        bool
	closed         bool
}

// New creates and initialises a new window bound to the platform and application context.
func New(p platform.Platform, a *app.App, opts WindowOptions) (*Window, error) {
	opts = opts.withDefaults()

	platOpts := platform.WindowOptions{
		Title:       opts.Title,
		Size:        opts.Size,
		Position:    opts.Position,
		MinSize:     opts.MinSize,
		MaxSize:     opts.MaxSize,
		Background:  opts.Background,
		Resizable:   opts.Resizable,
		Decorated:   opts.Decorated,
		Transparent: opts.Transparent,
		AlwaysOnTop: opts.AlwaysOnTop,
		Visible:     opts.Visible,
	}

	pw, err := p.NewWindow(platOpts)
	if err != nil {
		return nil, fmt.Errorf("create platform window: %w", err)
	}

	sf := pw.ScaleFactor()
	if sf <= 0 {
		sf = 1.0
	}

	devSize := geometry.SizeToDevicePixels(opts.Size, sf)
	r, err := newDefaultRenderer(pw.NativeSurface(), devSize, render.Options{VSync: opts.VSync})
	if err != nil {
		pw.Close()
		return nil, fmt.Errorf("create renderer: %w", err)
	}

	w := NewWithRenderer(pw, r, a, opts)
	w.platform = p
	w.scaleFactor = sf

	pw.SetEventHandler(w.DispatchEvent)

	// Wire foreground executor progress back into platform thread dispatch.
	if a != nil && a.Foreground() != nil {
		go func() {
			for range a.Foreground().Pending() {
				p.Dispatch(func() {
					a.Foreground().Drain()
					w.ScheduleFrame()
				})
			}
		}()
	}

	return w, nil
}

// NewWithRenderer constructs a Window configured with an explicit platform window and renderer.
func NewWithRenderer(pw platform.Window, r render.Renderer, a *app.App, opts WindowOptions) *Window {
	opts = opts.withDefaults()

	sf := float32(1.0)
	if pw != nil {
		sf = pw.ScaleFactor()
		if sf <= 0 {
			sf = 1.0
		}
	}

	txtSys, _ := text.NewSystem()
	atlas := text.NewAtlas(sf)
	keymap := input.NewKeymap()
	focusTree := input.NewFocusTree()

	w := &Window{
		platformWindow:   pw,
		renderer:         r,
		app:              a,
		layoutTree:       layout.NewTaffyTree(),
		textSystem:       txtSys,
		textAtlas:        atlas,
		keymap:           keymap,
		focusTree:        focusTree,
		rendered:         newFrame(keymap, focusTree),
		next:             newFrame(keymap, focusTree),
		scaleFactor:      sf,
		size:             opts.Size,
		remSize:          opts.RemSize,
		dirty:            true,
		nodeBounds:       make(map[layout.NodeID]geometry.Bounds[geometry.Pixels]),
		measureCallbacks: make(map[layout.NodeID]element.MeasureFunc),
		glyphTileCache:   make(map[glyphCacheKey]cachedGlyph),
		textStyleStack:   []style.TextStyle{style.DefaultTextStyle()},
	}

	if pw != nil {
		pw.SetEventHandler(w.DispatchEvent)
	}

	return w
}

// SetRoot configures a static root element for the window.
func (w *Window) SetRoot(el element.Element) {
	w.SetRootView(staticView{el: el})
}

// SetRootFn configures a root element generator function for the window.
func (w *Window) SetRootFn(fn func() element.Element) {
	w.SetRootView(fnView{fn: fn})
}

// SetRootView sets the root renderable view for the window and attaches an
// observer so entity mutations trigger redraws automatically.
func (w *Window) SetRootView(view element.AnyView) {
	w.rootSub.Close()
	w.rootSub = app.Subscription{}
	w.rootView = view
	if w.app != nil && view != nil {
		w.rootSub = view.Observe(w.app, func(_ *app.App) bool {
			w.dirty = true
			w.ScheduleFrame()
			return true
		})
	}
	w.dirty = true
	w.ScheduleFrame()
}

// ScheduleFrame marks the window as requiring a redraw and schedules a draw turn
// on the UI goroutine.
func (w *Window) ScheduleFrame() {
	w.mu.Lock()
	if w.frameScheduled {
		w.mu.Unlock()
		return
	}
	w.frameScheduled = true
	w.mu.Unlock()

	if w.platform != nil {
		w.platform.Dispatch(func() {
			w.mu.Lock()
			w.frameScheduled = false
			w.mu.Unlock()
			if w.dirty {
				w.Draw()
			}
		})
	}
}

// Invalidate marks the window dirty and requests a redraw.
func (w *Window) Invalidate() {
	w.dirty = true
	w.ScheduleFrame()
}

// Size returns the current logical dimensions of the client area.
func (w *Window) Size() geometry.Size[geometry.Pixels] {
	return w.size
}

// Draw executes the seven frame steps in strict order from layout through paint and presentation.
//
// A root that renders nil still runs the full cycle with an empty frame,
// rather than aborting midway: the scene, tab order, clip depth and
// needsPresent all end the call in the state a completed frame leaves them in,
// and the empty frame is swapped in and presented so the window actually
// shows nothing rather than the previous frame's stale content.
func (w *Window) Draw() {
	if w.closed || w.rootView == nil {
		return
	}
	if w.phase != phaseNone {
		panic(fmt.Sprintf("window: re-entrant Draw call (already in phase %v)", w.phase))
	}

	w.dirty = false

	// 1. Flush effects: drain app notifications and observe dirty views.
	if w.app != nil {
		w.app.Flush()
	}

	// 2. Request layout: evaluate root view and construct Taffy layout nodes.
	w.textStyleStack = w.textStyleStack[:1]
	w.textStyleStack[0] = style.DefaultTextStyle()
	w.clipDepth = 0
	w.prepaintClipStack = w.prepaintClipStack[:0]
	w.phase = phaseLayout
	el := w.rootView.Render(w.app)

	if el != nil {
		rootLayoutID := el.RequestLayout(w)

		// 3. Layout: solve flexbox layout and derive window-relative node bounds.
		w.phase = phaseLayoutSolve
		w.computeLayout(rootLayoutID)

		// 4. Prepaint: elements commit bounds, push dispatch nodes, and register hit regions.
		w.phase = phasePrepaint
		rootBounds := w.LayoutBounds(rootLayoutID)
		el.Prepaint(w, rootBounds)
		w.checkPrepaintClipStackEmpty()
		w.next.tabOrder = sortTabStops(w.next.tabStops)

		// 5. Hit test (intra-frame): resolve pointer against regions registered in next.
		w.phase = phaseHitTest
		w.resolveNextHitTest()

		// 6. Paint: elements evaluate hover/active/focus styles and emit primitives into next.scene.
		w.phase = phasePaint
		el.Paint(w, rootBounds)
		w.checkClipStackEmpty()
	} else {
		clear(w.nodeBounds)
		w.phase = phaseHitTest
		w.resolveNextHitTest()
		w.phase = phasePaint
	}

	// 7. Present: finish scene, swap frames, reset next, and submit to GPU.
	// Clean up focus if the focused element is no longer rendered in the tree.
	if currFocus, hasFocus := w.focusTree.Focused(); hasFocus && currFocus != 0 {
		if !w.next.focusIDs[currFocus] {
			w.focusTree.Blur()
		}
	}

	w.next.scene.Finish()
	w.rendered, w.next = w.next, w.rendered
	w.next.clear()
	w.layoutTree = layout.NewTaffyTree()
	w.phase = phaseNone
	w.needsPresent = true

	if w.renderer != nil && w.needsPresent {
		devSize := geometry.SizeToDevicePixels(w.size, w.scaleFactor)
		if w.renderer.Size() != devSize {
			_ = w.renderer.Resize(devSize)
		}
		_ = w.renderer.Draw(w.rendered.scene)
		_ = w.renderer.Present()
		w.needsPresent = false
	}
}

// computeLayout executes the flexbox algorithm and maps relative node offsets
// to absolute window coordinates.
func (w *Window) computeLayout(rootID layout.NodeID) {
	clear(w.nodeBounds)
	avail := layout.Size[layout.AvailableSpace]{
		Width:  layout.Definite(float32(w.size.Width)),
		Height: layout.Definite(float32(w.size.Height)),
	}

	w.phase = phaseLayoutSolve
	measureFn := func(inputs layout.LayoutInput, id layout.NodeID, ctx any, style *layout.Style) layout.LayoutOutput {
		cb, ok := w.measureCallbacks[id]
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

	w.layoutTree.ComputeLayoutWithMeasure(rootID, avail, measureFn)
	clear(w.measureCallbacks)

	rootLayout := w.layoutTree.Layout(rootID)
	rootSize := geometry.Size[geometry.Pixels]{
		Width:  geometry.Pixels(rootLayout.Size.Width),
		Height: geometry.Pixels(rootLayout.Size.Height),
	}
	rootOrigin := geometry.Point[geometry.Pixels]{X: 0, Y: 0}
	w.nodeBounds[rootID] = geometry.Bounds[geometry.Pixels]{
		Origin: rootOrigin,
		Size:   rootSize,
	}

	w.populateNodeBounds(rootID, rootOrigin)
}

func (w *Window) populateNodeBounds(parentID layout.NodeID, parentOrigin geometry.Point[geometry.Pixels]) {
	children := w.layoutTree.Children(parentID)
	for _, childID := range children {
		childLayout := w.layoutTree.Layout(childID)
		childOrigin := parentOrigin.Add(geometry.Point[geometry.Pixels]{
			X: geometry.Pixels(childLayout.Location.X),
			Y: geometry.Pixels(childLayout.Location.Y),
		})
		childSize := geometry.Size[geometry.Pixels]{
			Width:  geometry.Pixels(childLayout.Size.Width),
			Height: geometry.Pixels(childLayout.Size.Height),
		}
		w.nodeBounds[childID] = geometry.Bounds[geometry.Pixels]{
			Origin: childOrigin,
			Size:   childSize,
		}
		w.populateNodeBounds(childID, childOrigin)
	}
}

// resolveNextHitTest executes Step 5 by resolving the pointer against next.hitRegions.
//
// While a pointer capture is active, the pointer is not free to hover or
// activate whatever now sits under it: only the captured node is eligible,
// found by nodeID rather than by point since hit region ids are minted fresh
// every frame and the id from the frame the capture began in never appears in
// next.hitRegions again. Dispatch node ids are stable across frames for an
// unchanged tree shape (see DispatchTree.PushNode), which is what makes the
// lookup meaningful.
func (w *Window) resolveNextHitTest() {
	if w.captureActive {
		w.resolveCapturedHitTest()
		return
	}

	hr, ok := hitTest(w.next.hitRegions, w.pointerPos)
	var cursor style.CursorStyle
	if ok {
		w.hoveredHitRegion = hr.id
		w.hoveredNodeID = hr.nodeID
		if w.pointerDown {
			w.activeHitRegion = hr.id
		} else {
			w.activeHitRegion = 0
		}
		cursor = hr.cursor
	} else {
		w.hoveredHitRegion = 0
		w.hoveredNodeID = 0
		w.activeHitRegion = 0
		cursor = style.CursorDefault
	}

	w.setCursor(cursor)
}

// resolveCapturedHitTest resolves hover, active and cursor state for a
// captured drag: the captured node's hit region for this frame (found by
// dispatch node id, not by point), with active and hover true only while the
// pointer remains inside that region's bounds.
func (w *Window) resolveCapturedHitTest() {
	w.hoveredHitRegion = 0
	w.hoveredNodeID = 0
	w.activeHitRegion = 0

	var cursor style.CursorStyle
	for _, hr := range w.next.hitRegions {
		if hr.nodeID != w.captureNodeID {
			continue
		}
		w.captureBounds = hr.bounds
		if hr.bounds.Contains(w.pointerPos) {
			w.hoveredHitRegion = hr.id
			w.hoveredNodeID = hr.nodeID
			w.activeHitRegion = hr.id
			cursor = hr.cursor
		}
		break
	}

	w.setCursor(cursor)
}

func (w *Window) setCursor(cursor style.CursorStyle) {
	platCursor := cursorStyleToPlatform(cursor)
	if platCursor != w.currentCursor {
		w.currentCursor = platCursor
		if w.platform != nil {
			w.platform.SetCursor(platCursor)
		}
	}
}

// DispatchEvent routes arriving platform input and lifecycle events against the
// on-screen rendered frame.
func (w *Window) DispatchEvent(event platform.Event) {
	switch e := event.(type) {
	case platform.ResizeEvent:
		w.size = e.Size
		w.dirty = true
		w.ScheduleFrame()

	case platform.ScaleChangedEvent:
		w.scaleFactor = e.ScaleFactor
		if w.textAtlas != nil {
			w.textAtlas.ScaleFactor = e.ScaleFactor
			w.textAtlas.Clear()
		}
		if w.glyphTileCache != nil {
			clear(w.glyphTileCache)
		}
		if w.renderer != nil {
			w.renderer.ClearAtlas(scene.TextureMonochrome)
			w.renderer.ClearAtlas(scene.TexturePolychrome)
		}
		w.dirty = true
		w.ScheduleFrame()

	case platform.PointerEvent:
		w.pointerPos = geometry.Point[geometry.Pixels]{
			X: e.Position.X.ToPixels(w.scaleFactor),
			Y: e.Position.Y.ToPixels(w.scaleFactor),
		}

		var nodeID input.DispatchNodeID = -1

		switch {
		case e.Phase == platform.PointerDown:
			// A fresh press always re-hit-tests against where the pointer
			// actually is, even if a stale capture from an unmatched up
			// somehow survived.
			hr, ok := hitTest(w.rendered.hitRegions, w.pointerPos)
			w.pointerDown = true

			if ok {
				nodeID = hr.nodeID

				// Pointer-down on a focusable element moves focus to it;
				// pointer-down on anything else leaves focus alone. Only the
				// empty background (no hit at all) blurs unconditionally.
				if hr.focusID != 0 {
					w.RequestFocus(hr.focusID)
				}

				w.captureActive = true
				w.captureNodeID = hr.nodeID
				w.captureBounds = hr.bounds
			} else {
				w.captureActive = false
				w.focusTree.Blur()
			}

		case w.captureActive:
			// Route to the node that was pressed, regardless of what is now
			// under the pointer, including outside the window entirely.
			nodeID = w.captureNodeID
			inBounds := w.captureBounds.Contains(w.pointerPos)

			if e.Phase == platform.PointerUp {
				w.pointerDown = false
				if inBounds {
					localPos := geometry.NewPoint[geometry.Pixels](
						w.pointerPos.X-w.captureBounds.Origin.X,
						w.pointerPos.Y-w.captureBounds.Origin.Y,
					)
					clickEvt := element.ClickEvent{
						Position:      w.pointerPos,
						LocalPosition: localPos,
						Button:        element.MouseButton(e.Button),
						Modifiers:     element.Modifiers(e.Modifiers),
					}
					if listeners, ok := w.rendered.clickListeners[nodeID]; ok {
						for _, l := range listeners {
							if l(clickEvt) {
								break
							}
						}
					}
				}
				w.captureActive = false
			}

		default:
			hr, ok := hitTest(w.rendered.hitRegions, w.pointerPos)
			if ok {
				nodeID = hr.nodeID
			}
			if e.Phase == platform.PointerUp {
				w.pointerDown = false
			}
		}

		if nodeID >= 0 {
			w.rendered.dispatchTree.DispatchPointer(e, nodeID)
		}

		w.dirty = true
		w.ScheduleFrame()

	case platform.KeyEvent:
		res := w.rendered.dispatchTree.DispatchKey(e)
		if !res.Handled && e.Phase == platform.KeyDown && e.Code == platform.KeyTab {
			if e.Modifiers.Has(platform.Shift) {
				w.FocusPrev()
			} else {
				w.FocusNext()
			}
		}
		w.dirty = true
		w.ScheduleFrame()

	case platform.WheelEvent:
		if hr, ok := hitTest(w.rendered.hitRegions, w.pointerPos); ok {
			w.rendered.dispatchTree.DispatchWheel(e, hr.nodeID)
		}
		w.dirty = true
		w.ScheduleFrame()

	case platform.TextEvent:
		w.rendered.dispatchTree.DispatchText(e)
		w.dirty = true
		w.ScheduleFrame()

	case platform.FocusEvent:
		w.focused = e.Focused
		w.dirty = true
		w.ScheduleFrame()
	}
}

// FocusNext moves keyboard focus to the next element in tab order, wrapping around.
func (w *Window) FocusNext() {
	tabOrder := w.rendered.tabOrder
	if len(tabOrder) == 0 {
		return
	}
	curr, ok := w.focusTree.Focused()
	if !ok || curr == 0 {
		w.RequestFocus(tabOrder[0])
		return
	}
	idx := slices.Index(tabOrder, curr)
	if idx < 0 {
		w.RequestFocus(tabOrder[0])
		return
	}
	nextIdx := (idx + 1) % len(tabOrder)
	w.RequestFocus(tabOrder[nextIdx])
}

// FocusPrev moves keyboard focus to the previous element in tab order, wrapping around.
func (w *Window) FocusPrev() {
	tabOrder := w.rendered.tabOrder
	if len(tabOrder) == 0 {
		return
	}
	curr, ok := w.focusTree.Focused()
	if !ok || curr == 0 {
		w.RequestFocus(tabOrder[len(tabOrder)-1])
		return
	}
	idx := slices.Index(tabOrder, curr)
	if idx < 0 {
		w.RequestFocus(tabOrder[len(tabOrder)-1])
		return
	}
	prevIdx := (idx - 1 + len(tabOrder)) % len(tabOrder)
	w.RequestFocus(tabOrder[prevIdx])
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

// Renderer returns the graphics renderer attached to the window.
func (w *Window) Renderer() render.Renderer {
	return w.renderer
}

// Close releases platform window and renderer resources.
func (w *Window) Close() error {
	w.closed = true
	w.rootSub.Close()
	w.rootSub = app.Subscription{}
	if w.renderer != nil {
		_ = w.renderer.Close()
	}
	if w.platformWindow != nil {
		w.platformWindow.Close()
	}
	return nil
}
