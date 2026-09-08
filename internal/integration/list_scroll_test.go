package integration

import (
	"testing"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/platform/platformtest"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/ui"
	"github.com/yasufad/facet/window"
)

// TestListScrollRendersVisibleItemsInWindow is the integration test the
// elementtest-based list tests structurally cannot be: it drives the list
// through a real window.Window frame loop — the real Taffy layout solver,
// the real paint phase, the real scene emission — and asserts on what
// actually rendered, not on what the list requested.
//
// elementtest records what an element asks for; it does not lay out a real
// tree against a real frame loop, and it never paints. So the four list
// tests in ui/list_test.go prove the list requests the right layout — not
// that it appears. This test closes that gap.
//
// The list is scrolled to a non-zero offset where the visible set is neither
// the first nor the last items (offset 600 with 30px items → item 20 at the
// top, items 20–24 visible). Each item's background red component encodes its
// index (R = i / count), so the quad the renderer received identifies which
// item is on screen. A list that declares its extent correctly and paints
// the wrong window of items passes every elementtest but fails here: the
// quad at the viewport top would carry item 0's colour, not item 20's.
//
// This test caught a real defect on its first run: the item wrapper was a
// flex row (the default), so items had zero width and produced no quads.
// elementtest never saw this because it checks which items were built, not
// their solved bounds. The wrapper is now FlexCol, and the items fill the
// viewport width.
func TestListScrollRendersVisibleItemsInWindow(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	const (
		winW       = 200
		winH       = 150
		count      = 100
		itemHeight = geometry.Pixels(30)
		scrollTo   = geometry.Pixels(600) // 600 / 30 = item 20 at the top
	)

	size := geometry.NewSize[geometry.Pixels](winW, winH)
	pw := platformtest.NewWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := window.NewWithRenderer(pw, r, a, window.WindowOptions{Size: size})

	state := app.New(a, func(cx *app.Context[ui.ListState]) ui.ListState {
		return ui.NewListState()
	})
	defer state.Release()

	// Each item's red component encodes its index: R = i / count.
	// Item 0 → R=0.00, item 20 → R=0.20, item 99 → R=0.99.
	// This makes the rendered quad's colour identify which item is on screen.
	list := ui.NewList(a, state).
		Count(count).
		ItemHeight(itemHeight).
		Render(func(i int) element.Element {
			return element.NewDiv().
				Height(style.Px(itemHeight)).
				Bg(colour.Rgba{R: float32(i) / float32(count), A: 1.0})
		})

	// The list's viewport uses WFull/HFull (percentage sizes), which need a
	// definite parent to resolve against. The root element has no parent,
	// so we wrap the list in a Div with explicit dimensions. SetRootFn
	// creates a fresh root each frame — elements are ephemeral, and reusing
	// a static root panics on the second frame because its phase is still
	// set from the first.
	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Width(style.Px(winW)).
			Height(style.Px(winH)).
			Child(list)
	})

	// Frame 1: bootstrap. The list has no recorded viewport height, so it
	// builds no items. Paint records the viewport height (150) and content
	// height (3000) into the entity for frame 2.
	w.Draw()

	// Scroll to the middle of the list. 600 / 30 = 20, so item 20 should
	// be at the top of the viewport and items 20–24 should be visible —
	// neither the first nor the last items.
	state.Update(a, func(st *ui.ListState, cx *app.Context[ui.ListState]) {
		st.SetOffset(scrollTo)
	})

	// Frame 2: the list reads the recorded viewport height and the scrolled
	// offset, builds items 20–24, and the real frame loop lays them out and
	// paints them into the scene the stub renderer captures.
	w.Draw()

	// Assert on what actually rendered. The stub renderer captured the
	// scene's quads — each quad carries solved bounds (in scaled pixels,
	// which equal logical pixels at scale 1.0) and the item's background
	// colour. After scrolling to offset 600, item 20 should be at the
	// viewport top (y = 0) with R = 20/100 = 0.20.
	//
	// A list that computes the visible window from the wrong offset —
	// always building item 0, or ignoring the scroll offset, or applying
	// it to the wrong side of the spacer — renders a different item at the
	// viewport top, and this assertion catches it.
	if len(r.quads) == 0 {
		t.Fatalf("no quads rendered after scroll; the frame produced an empty scene")
	}

	var topItem *scene.Quad
	for i := range r.quads {
		q := &r.quads[i]
		if q.Background.A < 0.5 {
			continue // skip transparent quads (divs with no background)
		}
		// The first visible item sits at the viewport top (y = 0). A
		// tolerance of 1px absorbs float32 rounding in the layout solver.
		if q.Bounds.Origin.Y > -1.0 && q.Bounds.Origin.Y < 1.0 {
			topItem = q
			break
		}
	}
	if topItem == nil {
		t.Fatalf("no item quad rendered at viewport top (y≈0) after scrolling to offset %v; the list did not place any visible item at the top of the viewport", scrollTo)
	}

	expectedR := float32(20) / float32(count)
	if topItem.Background.R < expectedR-0.01 || topItem.Background.R > expectedR+0.01 {
		t.Fatalf("expected item at viewport top to be index 20 (R=%.3f), got R=%.3f — the list rendered the wrong window of items after scroll",
			expectedR, topItem.Background.R)
	}

	// Sanity: item 0 (R=0) should NOT be at the viewport top. A list that
	// ignores the offset and always builds from index 0 would put item 0
	// here; this makes that failure mode explicit.
	if topItem.Background.R < 0.01 {
		t.Fatalf("item at viewport top has R≈0 (item 0); the list did not scroll — item 0 should not be at the viewport top after scrolling to offset %v", scrollTo)
	}
}
