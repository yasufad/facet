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

// TestTextFieldRendersAtPositiveSizeInWindow drives a TextField through a
// real window.Window frame loop and asserts on the container background
// quad the renderer received.
//
// The six elementtest-based TextField tests prove the container requests
// the right layout, registers the right listeners, and emits the right
// quads into the elementtest frame. They do not solve a real tree through
// a real frame loop. A TextField whose container resolves to zero width
// in a real frame loop would still pass every elementtest, because
// elementtest checks what was inserted, not what a real solver produced.
//
// This test identifies the container's background quad by a distinctive
// custom colour and asserts it has positive width and height. The caret
// quad's position depends on text metrics, which element is rewriting this
// round; the container quad's size does not — it comes from the container's
// own padding and the text's intrinsic width, which the layout solver
// resolves during layout, not paint.
func TestTextFieldRendersAtPositiveSizeInWindow(t *testing.T) {
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

	state := app.New(a, func(cx *app.Context[ui.TextFieldState]) ui.TextFieldState {
		return ui.NewTextFieldState("Hello")
	})
	defer state.Release()

	// Distinctive background so the container's quad can be identified
	// unambiguously — the root has no background, and the caret quad
	// carries a different colour.
	tfBg := colour.Rgba{R: 0.1, G: 0.2, B: 0.3, A: 1.0}
	var ref style.Refinement
	ref.SetBackground(tfBg)
	tf := ui.NewTextField(a, state).Refine(ref)

	w.SetRootFn(func() element.Element {
		return element.NewDiv().
			Width(style.Px(winW)).
			Height(style.Px(winH)).
			Child(tf)
	})

	w.Draw()

	var tfQuad *scene.Quad
	for i := range r.quads {
		q := &r.quads[i]
		if q.Background == tfBg {
			tfQuad = q
			break
		}
	}
	if tfQuad == nil {
		t.Fatalf("text field container quad not found in rendered scene; the container did not paint a quad")
	}

	// The container's width comes from its content (text intrinsic width
	// + padding) and its height from MinHeight. Both should be positive.
	// A zero-width wrapper defect would skip the quad entirely.
	if tfQuad.Bounds.Size.Width <= 0 {
		t.Fatalf("text field container quad has zero width (%v); the container did not resolve to a positive width in a real frame loop",
			tfQuad.Bounds.Size.Width)
	}
	if tfQuad.Bounds.Size.Height <= 0 {
		t.Fatalf("text field container quad has zero height (%v); the container did not resolve to a positive height in a real frame loop",
			tfQuad.Bounds.Size.Height)
	}
}
