package element

import (
	"testing"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
)

func TestOneElementThroughThreePhases(t *testing.T) {
	frame := newFakeFrame()
	red := colour.Rgba{R: 1, G: 0, B: 0, A: 1}
	green := colour.Rgba{R: 0, G: 1, B: 0, A: 1}
	blue := colour.Rgba{R: 0, G: 0, B: 1, A: 1}

	child1 := NewDiv().Bg(green)
	child2 := NewDiv().Bg(blue)
	root := NewDiv().
		Flex().
		Bg(red).
		Children(child1, child2)

	// Phase 1: Request Layout
	frame.phase = phaseLayoutRequested
	rootID := root.RequestLayout(frame)

	if len(frame.nodes) != 3 {
		t.Fatalf("expected 3 layout nodes requested, got %d", len(frame.nodes))
	}
	if len(frame.children[rootID]) != 2 {
		t.Fatalf("expected root to have 2 children in layout tree, got %d", len(frame.children[rootID]))
	}

	// Layout solve (e.g. 200x100 logical pixels)
	frame.solve(rootID, 200, 100)
	rootBounds := frame.LayoutBounds(rootID)
	if rootBounds.Size.Width != 200 || rootBounds.Size.Height != 100 {
		t.Fatalf("expected root bounds 200x100, got %v", rootBounds)
	}

	// Phase 2: Prepaint
	frame.phase = phasePrepainted
	root.Prepaint(frame, rootBounds)

	// Phase 3: Paint
	frame.phase = phasePainted
	root.Paint(frame, rootBounds)

	// Assert scene primitives
	// ScaleFactor is 2.0, so 200x100 logical becomes 400x200 scaled.
	if len(frame.quads) != 3 {
		t.Fatalf("expected 3 quads emitted into scene, got %d", len(frame.quads))
	}

	// Quad 0: Parent background (Red)
	q0 := frame.quads[0]
	if q0.Background != red {
		t.Errorf("quad 0 background = %v, want %v", q0.Background, red)
	}
	if q0.Bounds.Origin.X != 0 || q0.Bounds.Origin.Y != 0 {
		t.Errorf("quad 0 origin = %v, want (0, 0)", q0.Bounds.Origin)
	}
	if q0.Bounds.Size.Width != 400 || q0.Bounds.Size.Height != 200 {
		t.Errorf("quad 0 size = %v, want (400, 200)", q0.Bounds.Size)
	}

	// Quad 1: Child 1 (Green) - left half (0 to 100 logical -> 0 to 200 scaled)
	q1 := frame.quads[1]
	if q1.Background != green {
		t.Errorf("quad 1 background = %v, want %v", q1.Background, green)
	}
	if q1.Bounds.Origin.X != 0 || q1.Bounds.Origin.Y != 0 {
		t.Errorf("quad 1 origin = %v, want (0, 0)", q1.Bounds.Origin)
	}
	if q1.Bounds.Size.Width != 200 || q1.Bounds.Size.Height != 200 {
		t.Errorf("quad 1 size = %v, want (200, 200)", q1.Bounds.Size)
	}

	// Quad 2: Child 2 (Blue) - right half (100 to 200 logical -> 200 to 400 scaled)
	q2 := frame.quads[2]
	if q2.Background != blue {
		t.Errorf("quad 2 background = %v, want %v", q2.Background, blue)
	}
	if q2.Bounds.Origin.X != 200 || q2.Bounds.Origin.Y != 0 {
		t.Errorf("quad 2 origin = %v, want (200, 0)", q2.Bounds.Origin)
	}
	if q2.Bounds.Size.Width != 200 || q2.Bounds.Size.Height != 200 {
		t.Errorf("quad 2 size = %v, want (200, 200)", q2.Bounds.Size)
	}
}

func TestPhaseOrderingEnforcement(t *testing.T) {
	frame := newFakeFrame()

	t.Run("prepaint before layout panics", func(t *testing.T) {
		d := NewDiv()
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected Prepaint before RequestLayout to panic")
			}
		}()
		d.Prepaint(frame, geometry.Bounds[geometry.Pixels]{})
	})

	t.Run("paint before prepaint panics", func(t *testing.T) {
		d := NewDiv()
		frame.phase = phaseLayoutRequested
		d.RequestLayout(frame)
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected Paint before Prepaint to panic")
			}
		}()
		d.Paint(frame, geometry.Bounds[geometry.Pixels]{})
	})

	t.Run("request layout called twice panics", func(t *testing.T) {
		d := NewDiv()
		frame.phase = phaseLayoutRequested
		d.RequestLayout(frame)
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("expected second RequestLayout to panic")
			}
		}()
		d.RequestLayout(frame)
	})
}

func TestEmptyDiv(t *testing.T) {
	frame := newFakeFrame()
	empty := NewDiv()

	frame.phase = phaseLayoutRequested
	id := empty.RequestLayout(frame)

	frame.solve(id, 100, 50)
	bounds := frame.LayoutBounds(id)

	frame.phase = phasePrepainted
	empty.Prepaint(frame, bounds)

	frame.phase = phasePainted
	empty.Paint(frame, bounds)

	// An empty unstyled Div has alpha 0 background, so no quad should be inserted.
	if len(frame.quads) != 0 {
		t.Errorf("expected 0 quads for empty div, got %d", len(frame.quads))
	}
}

type testView struct {
	label string
}

func (v *testView) Render(cx *app.Context[testView]) Element {
	return NewDiv().Flex()
}

func TestViewRenderBridge(t *testing.T) {
	a := app.NewApp()
	ent := app.New(a, func(cx *app.Context[testView]) testView {
		return testView{label: "hello"}
	})

	view := NewView(ent)
	el := view.Render(a)
	if el == nil {
		t.Fatalf("expected non-nil element from View.Render")
	}
	if _, ok := el.(*Div); !ok {
		t.Errorf("expected *Div element, got %T", el)
	}
}
