package element

import (
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/scene"
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

	tree     *layout.TaffyTree
	nodes    []layout.NodeID
	styles   map[layout.NodeID]layout.Style
	children map[layout.NodeID][]layout.NodeID
	bounds   map[layout.NodeID]geometry.Bounds[geometry.Pixels]

	nextHitRegionID HitRegionID
	hitRegions      []recordedHitRegion

	quads       []scene.Quad
	shadows     []scene.Shadow
	paths       []scene.Path[geometry.ScaledPixels]
	underlines  []scene.Underline
	monoSprites []scene.MonochromeSprite
	polySprites []scene.PolychromeSprite
}

func newFakeFrame() *fakeFrame {
	return &fakeFrame{
		phase:       phaseLayoutRequested,
		scaleFactor: 2.0,
		remSize:     16.0,
		tree:        layout.NewTaffyTree(),
		styles:      make(map[layout.NodeID]layout.Style),
		children:    make(map[layout.NodeID][]layout.NodeID),
		bounds:      make(map[layout.NodeID]geometry.Bounds[geometry.Pixels]),
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

func (f *fakeFrame) LayoutBounds(id layout.NodeID) geometry.Bounds[geometry.Pixels] {
	return f.bounds[id]
}

func (f *fakeFrame) RegisterHitRegion(bounds geometry.Bounds[geometry.Pixels], nodeID input.DispatchNodeID) HitRegionID {
	if f.phase != phasePrepainted {
		panic("fakeFrame: RegisterHitRegion called outside prepaint phase")
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
	return text.ShapedLine{}, nil
}

func (f *fakeFrame) ScaleFactor() float32 {
	return f.scaleFactor
}

func (f *fakeFrame) RemSize() geometry.Pixels {
	return f.remSize
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
