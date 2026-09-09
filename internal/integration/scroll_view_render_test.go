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

// TestScrollViewRendersChildAtViewportWidthInWindow drives a ScrollView
// through a real window.Window frame loop and asserts on the child's
// background quad — not on what the scroll view requested.
//
// ScrollView has the same container-and-content shape as List: a viewport
// that clips, a content div that carries the scroll offset, and a child
// inside the content. The five elementtest-based ScrollView tests prove
// the viewport requests the right layout, clips correctly, records the
// right metrics, and handles wheel events. They do not solve a real tree
// and they do not paint. A ScrollView whose content div is in the wrong
// flex direction — so the child doesn't stretch on the cross axis and
// resolves to zero width — would still pass every elementtest, because
// elementtest checks what was requested, not what a real solver produced.
//
// This test uses a child with NO explicit width, so it must stretch to
// fill the content's width. It asserts the child's quad has width equal
// to the viewport's width (200px at scale 1.0), then scrolls to a
// non-zero offset and asserts the child's quad moved by exactly that
// offset. A zero-width wrapper defect would produce width=0 and skip
// the quad; a broken scroll offset would leave the child unmoved.
func TestScrollViewRendersChildAtViewportWidthInWindow(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	const (
		winW     = 200
		winH     = 150
		scrollTo = geometry.Pixels(100)
	)

	size := geometry.NewSize[geometry.Pixels](winW, winH)
	pw := platformtest.NewWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := window.NewWithRenderer(pw, r, a, window.WindowOptions{Size: size})

	state := app.New(a, func(cx *app.Context[ui.ScrollState]) ui.ScrollState {
		return ui.NewScrollState()
	})
	defer state.Release()

	// Child with NO explicit width — it must stretch to fill the
	// content's width. A zero-width wrapper defect (content in the
	// wrong flex direction) would leave it at zero width.
	childBg := colour.Rgba{R: 0.8, G: 0.2, B: 0.2, A: 1.0}

	sv := ui.NewScrollView(a, state)

	// SetRootFn creates a fresh root, child, and scroll view each frame.
	// The child is a Div and cannot be reused — RequestLayout panics if
	// called twice on the same element. The scroll view's state entity
	// is shared, so the scroll offset persists across frames.
	w.SetRootFn(func() element.Element {
		child := element.NewDiv().
			Height(style.Px(500)).
			Bg(childBg)
		sv.Child(child)
		return element.NewDiv().
			Width(style.Px(winW)).
			Height(style.Px(winH)).
			Child(sv)
	})

	// Frame 1: offset = 0. Paint records viewport height (150) and
	// content height (500) into the entity for frame 2.
	w.Draw()

	childQuad := findQuadByBg(r.quads, childBg)
	if childQuad == nil {
		t.Fatalf("child quad not found after frame 1; the scroll view did not paint the child")
	}

	// The child must fill the viewport's width. A zero-width wrapper
	// defect would produce width=0 and skip the quad entirely.
	expectedWidth := geometry.ScaledPixels(winW)
	if childQuad.Bounds.Size.Width < expectedWidth-1.0 || childQuad.Bounds.Size.Width > expectedWidth+1.0 {
		t.Fatalf("expected child width ≈ %v (viewport width) after frame 1, got %v — the child did not stretch to fill the content width",
			expectedWidth, childQuad.Bounds.Size.Width)
	}

	// Scroll to a non-zero offset. 100 is within [0, 350] (max =
	// 500 - 150), so it should not be clamped.
	state.Update(a, func(st *ui.ScrollState, cx *app.Context[ui.ScrollState]) {
		st.SetOffset(scrollTo)
	})

	// Frame 2: the content shifts up by the scroll offset.
	w.Draw()

	scrolledQuad := findQuadByBg(r.quads, childBg)
	if scrolledQuad == nil {
		t.Fatalf("child quad not found after scroll to %v; the scroll view did not paint the child", scrollTo)
	}

	// The child's origin Y should be -100: the content's InsetTop(-offset)
	// shifts it up by the scroll offset. A scroll view that ignores the
	// offset would leave the child at y=0.
	expectedY := geometry.ScaledPixels(-100)
	if scrolledQuad.Bounds.Origin.Y < expectedY-1.0 || scrolledQuad.Bounds.Origin.Y > expectedY+1.0 {
		t.Fatalf("expected child origin Y ≈ %v after scroll to %v, got %v — the content did not shift by the scroll offset",
			expectedY, scrollTo, scrolledQuad.Bounds.Origin.Y)
	}

	// Width should still be viewport width after scrolling.
	if scrolledQuad.Bounds.Size.Width < expectedWidth-1.0 || scrolledQuad.Bounds.Size.Width > expectedWidth+1.0 {
		t.Fatalf("expected child width ≈ %v after scroll, got %v — the child width changed after scrolling",
			expectedWidth, scrolledQuad.Bounds.Size.Width)
	}
}

func findQuadByBg(quads []scene.Quad, bg colour.Rgba) *scene.Quad {
	for i := range quads {
		q := &quads[i]
		if q.Background == bg {
			return q
		}
	}
	return nil
}
