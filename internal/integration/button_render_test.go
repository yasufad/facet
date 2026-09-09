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

// TestButtonRendersAtPositiveSizeInWindow drives a Button through a real
// window.Window frame loop and asserts on the quad the renderer received —
// not on what the button requested.
//
// The existing TestButtonClickInWindowMutatesEntityAndRendersNextFrame
// covers the button's interaction path through a real window, but checks
// only that quads exist (len >= 2). It does not check the button's solved
// size. A button that requests perfect layout but resolves to zero width
// in a real frame loop would still emit a container quad and pass that
// test, because the button's own quad would be skipped for empty bounds.
//
// This test identifies the button's background quad by a distinctive
// custom colour applied to the base, hover, and active states — the
// platformtest window's default mouse position is inside the button's
// bounds, so the hover state is active. It asserts the quad has positive
// width and height. A zero-width wrapper defect — the button's div
// resolving to zero width because its content doesn't stretch or its
// size doesn't solve — would skip the quad entirely, and this test
// would fail with "button quad not found".
func TestButtonRendersAtPositiveSizeInWindow(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	const (
		winW = 200
		winH = 100
	)

	size := geometry.NewSize[geometry.Pixels](winW, winH)
	pw := platformtest.NewWindow(size, 1.0)
	r := newStubRenderer(geometry.SizeToDevicePixels(size, 1.0))
	w := window.NewWithRenderer(pw, r, a, window.WindowOptions{Size: size})

	// Distinctive background applied to base, hover, and active states.
	// The platformtest window's default mouse position is inside the
	// button's bounds, so the hover state is active and overrides the
	// base background. Setting the same colour on all states keeps the
	// quad identifiable regardless of interaction state.
	btnBg := colour.Rgba{R: 0.3, G: 0.6, B: 0.9, A: 1.0}
	var ref style.Refinement
	ref.SetBackground(btnBg)
	btn := ui.NewButton("OK").
		Refine(ref).
		Hover(func(r *style.Refinement) {
			r.SetBackground(btnBg)
		}).
		Active(func(r *style.Refinement) {
			r.SetBackground(btnBg)
		})

	// SetRootFn creates a fresh root each frame — elements are ephemeral,
	// and reusing a static root panics on the second frame because its
	// phase is still set from the first.
	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Width(style.Px(winW)).
			Height(style.Px(winH)).
			Child(btn)
	})

	w.Draw()

	var btnQuad *scene.Quad
	for i := range r.quads {
		q := &r.quads[i]
		if q.Background == btnBg {
			btnQuad = q
			break
		}
	}
	if btnQuad == nil {
		t.Fatalf("button background quad not found in rendered scene; the button did not paint a quad (quad count: %d)", len(r.quads))
	}

	// The zero-width wrapper defect: a container whose children don't
	// stretch on the cross axis, or whose size doesn't solve, renders at
	// zero width and the quad is skipped. The button's width comes from
	// its content (label + padding), so it should be positive. This
	// assertion would fail if the button resolved to zero width.
	if btnQuad.Bounds.Size.Width <= 0 {
		t.Fatalf("button quad has zero width (%v); the button did not resolve to a positive width in a real frame loop",
			btnQuad.Bounds.Size.Width)
	}
	if btnQuad.Bounds.Size.Height <= 0 {
		t.Fatalf("button quad has zero height (%v); the button did not resolve to a positive height in a real frame loop",
			btnQuad.Bounds.Size.Height)
	}
}
