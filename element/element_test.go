package element

import (
	"testing"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/style"
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

func TestFluentBuilderToLayout(t *testing.T) {
	frame := newFakeFrame()
	frame.remSize = 16

	d := NewDiv().
		Flex().
		Absolute().
		Visible().
		Overflow(style.OverflowScroll).
		ScrollbarWidth(12).
		AllowConcurrentScroll(true).
		RestrictScrollToAxis(true).
		InsetTop(style.Px(10)).
		InsetRight(style.Px(20)).
		InsetBottom(style.Px(30)).
		InsetLeft(style.Px(40)).
		MarginTop(style.Px(5)).
		MarginRight(style.Px(6)).
		MarginBottom(style.Px(7)).
		MarginLeft(style.Px(8)).
		PaddingTop(style.Px(1)).
		PaddingRight(style.Px(2)).
		PaddingBottom(style.Px(3)).
		PaddingLeft(style.Px(4)).
		BorderTop(1).
		BorderRight(2).
		BorderBottom(3).
		BorderLeft(4).
		Width(style.Px(100)).
		Height(style.Px(200)).
		MinWidth(style.Px(50)).
		MinHeight(style.Px(60)).
		MaxWidth(style.Px(300)).
		MaxHeight(style.Px(400)).
		AspectRatio(16.0 / 9.0).
		GapRow(style.Px(15)).
		GapCol(style.Px(25)).
		AlignItems(style.AlignItemsCentre).
		AlignSelf(style.AlignItemsFlexEnd).
		AlignContent(style.AlignContentSpaceBetween).
		JustifyContent(style.AlignContentCentre).
		FlexDirection(style.FlexDirectionRow).
		FlexWrap(style.FlexWrapWrap).
		FlexBasis(style.Px(80)).
		FlexGrow(1.5).
		FlexShrink(0.5).
		TextAlign(style.TextAlignCentre)

	frame.phase = phaseLayoutRequested
	id := d.RequestLayout(frame)

	st, ok := frame.styles[id]
	if !ok {
		t.Fatalf("style not found for layout node %v", id)
	}

	if st.Display != layout.DisplayFlex {
		t.Errorf("Display = %v, want DisplayFlex", st.Display)
	}
	if st.Position != layout.PositionAbsolute {
		t.Errorf("Position = %v, want PositionAbsolute", st.Position)
	}
	if st.Overflow.X != layout.OverflowScroll || st.Overflow.Y != layout.OverflowScroll {
		t.Errorf("Overflow = %v, want OverflowScroll", st.Overflow)
	}
	if st.ScrollbarWidth != 12 {
		t.Errorf("ScrollbarWidth = %v, want 12", st.ScrollbarWidth)
	}
	if st.AspectRatio == nil || *st.AspectRatio != float32(16.0/9.0) {
		t.Errorf("AspectRatio = %v, want %v", st.AspectRatio, float32(16.0/9.0))
	}
	if st.AlignItems == nil || st.AlignItems.Keyword != layout.AlignItemsCentre.Keyword {
		t.Errorf("AlignItems = %v, want AlignItemsCentre", st.AlignItems)
	}
	if st.AlignSelf == nil || st.AlignSelf.Keyword != layout.AlignItemsFlexEnd.Keyword {
		t.Errorf("AlignSelf = %v, want AlignItemsFlexEnd", st.AlignSelf)
	}
	if st.AlignContent == nil || st.AlignContent.Keyword != layout.AlignContentSpaceBetween.Keyword {
		t.Errorf("AlignContent = %v, want AlignContentSpaceBetween", st.AlignContent)
	}
	if st.JustifyContent == nil || st.JustifyContent.Keyword != layout.AlignContentCentre.Keyword {
		t.Errorf("JustifyContent = %v, want AlignContentCentre", st.JustifyContent)
	}
	if st.FlexWrap != layout.FlexWrapWrap {
		t.Errorf("FlexWrap = %v, want FlexWrapWrap", st.FlexWrap)
	}
	if st.FlexGrow != 1.5 {
		t.Errorf("FlexGrow = %v, want 1.5", st.FlexGrow)
	}
	if st.FlexShrink != 0.5 {
		t.Errorf("FlexShrink = %v, want 0.5", st.FlexShrink)
	}
	if st.TextAlign != layout.TextAlignCentre {
		t.Errorf("TextAlign = %v, want TextAlignCentre", st.TextAlign)
	}
}

func TestHiddenVsInvisible(t *testing.T) {
	frame := newFakeFrame()

	hiddenDiv := NewDiv().Hidden()
	frame.phase = phaseLayoutRequested
	hiddenID := hiddenDiv.RequestLayout(frame)
	hiddenStyle := frame.styles[hiddenID]
	if hiddenStyle.Display != layout.DisplayNone {
		t.Errorf("Hidden() Display = %v, want DisplayNone", hiddenStyle.Display)
	}

	invisibleDiv := NewDiv().Invisible()
	invisibleID := invisibleDiv.RequestLayout(frame)
	invisibleStyle := frame.styles[invisibleID]
	if invisibleStyle.Display != layout.DisplayFlex {
		t.Errorf("Invisible() Display = %v, want DisplayFlex (default)", invisibleStyle.Display)
	}
	// Invisible preserves layout but sets visibility: hidden on refinement
	st := style.Default()
	st.Refine(invisibleDiv.refinement)
	if st.Visibility != style.VisibilityHidden {
		t.Errorf("Invisible() Visibility = %v, want VisibilityHidden", st.Visibility)
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

// TestDivVisibilityHiddenPaintsNothingButKeepsLayout confirms Visibility's
// first real consumer: a hidden div and its children emit no primitives, but
// layout is unaffected — its sibling's bounds do not change, unlike
// Display: none.
func TestDivVisibilityHiddenPaintsNothingButKeepsLayout(t *testing.T) {
	frame := newFakeFrame()
	red := colour.Rgba{R: 1, G: 0, B: 0, A: 1}
	blue := colour.Rgba{R: 0, G: 0, B: 1, A: 1}

	hiddenChild := NewDiv().Bg(red)
	hidden := NewDiv().
		Width(style.Px(50)).
		Height(style.Px(50)).
		Bg(red).
		Visibility(style.VisibilityHidden).
		Child(hiddenChild)
	sibling := NewDiv().Width(style.Px(50)).Height(style.Px(50)).Bg(blue)

	frame.phase = phaseLayoutRequested
	rootID := NewDiv().Flex().Children(hidden, sibling).RequestLayout(frame)
	frame.solve(rootID, 100, 50)

	frame.phase = phasePrepainted
	hidden.Prepaint(frame, frame.LayoutBounds(hidden.layoutID))
	sibling.Prepaint(frame, frame.LayoutBounds(sibling.layoutID))

	frame.phase = phasePainted
	hidden.Paint(frame, frame.LayoutBounds(hidden.layoutID))
	sibling.Paint(frame, frame.LayoutBounds(sibling.layoutID))

	if len(frame.quads) != 1 {
		t.Fatalf("expected 1 quad (the visible sibling only), got %d", len(frame.quads))
	}
	if frame.quads[0].Background != blue {
		t.Fatalf("expected the surviving quad to be the sibling's blue, got %v", frame.quads[0].Background)
	}

	if sibWidth := frame.LayoutBounds(sibling.layoutID).Size.Width; sibWidth != 50 {
		t.Fatalf("expected the sibling's width unaffected by the hidden div's visibility, got %v", sibWidth)
	}
}

// TestDivBoxShadowDropPaintsBeforeBackground gives scene.Shadow its first
// producer in the project's history. render's readback territory is where
// pixel correctness gets proven; this is the insert-level half — it asserts
// a drop shadow reaches the scene, dilated by blur+spread and translated by
// its offset per scene.Shadow's documented contract, and painted before the
// element's own background quad so the fill draws on top of it.
func TestDivBoxShadowDropPaintsBeforeBackground(t *testing.T) {
	frame := newFakeFrame()
	black := colour.Rgba{A: 1}
	red := colour.Rgba{R: 1, A: 1}

	d := NewDiv().
		Width(style.Px(100)).
		Height(style.Px(100)).
		Bg(red).
		BoxShadow([]style.BoxShadow{style.Shadow(2, 4, 6, 1, black)})

	frame.phase = phaseLayoutRequested
	rootID := d.RequestLayout(frame)
	frame.solve(rootID, 100, 100)
	rootBounds := frame.LayoutBounds(rootID)

	frame.phase = phasePrepainted
	d.Prepaint(frame, rootBounds)
	frame.phase = phasePainted
	d.Paint(frame, rootBounds)

	if len(frame.shadows) != 1 {
		t.Fatalf("expected 1 shadow, got %d", len(frame.shadows))
	}
	if len(frame.quads) != 1 {
		t.Fatalf("expected 1 quad, got %d", len(frame.quads))
	}

	sh := frame.shadows[0]
	if sh.Inset {
		t.Fatal("expected a non-inset (drop) shadow")
	}
	if sh.Colour != black {
		t.Fatalf("shadow colour = %v, want %v", sh.Colour, black)
	}

	// scale is 2.0: offset (2,4) and dilation by blur+spread (7) both scale.
	wantBounds := geometry.NewBounds(
		geometry.NewPoint[geometry.ScaledPixels](2*2-7*2, 4*2-7*2),
		geometry.NewSize[geometry.ScaledPixels](100*2+14*2, 100*2+14*2),
	)
	if sh.Bounds != wantBounds {
		t.Fatalf("shadow bounds = %v, want %v", sh.Bounds, wantBounds)
	}

	wantElementBounds := geometry.NewBounds(
		geometry.NewPoint[geometry.ScaledPixels](0, 0),
		geometry.NewSize[geometry.ScaledPixels](200, 200),
	)
	if sh.ElementBounds != wantElementBounds {
		t.Fatalf("shadow element bounds = %v, want %v", sh.ElementBounds, wantElementBounds)
	}
}

// TestDivBoxShadowInsetPaintsAfterBackground is the inset counterpart:
// bounds shrink rather than grow, and the shadow is recorded after the
// background quad so it layers on top of the fill.
func TestDivBoxShadowInsetPaintsAfterBackground(t *testing.T) {
	frame := newFakeFrame()
	black := colour.Rgba{A: 1}
	red := colour.Rgba{R: 1, A: 1}

	inset := style.Shadow(0, 0, 4, 2, black)
	inset.Inset = true
	d := NewDiv().
		Width(style.Px(100)).
		Height(style.Px(100)).
		Bg(red).
		BoxShadow([]style.BoxShadow{inset})

	frame.phase = phaseLayoutRequested
	rootID := d.RequestLayout(frame)
	frame.solve(rootID, 100, 100)
	rootBounds := frame.LayoutBounds(rootID)

	frame.phase = phasePrepainted
	d.Prepaint(frame, rootBounds)
	frame.phase = phasePainted
	d.Paint(frame, rootBounds)

	if len(frame.shadows) != 1 {
		t.Fatalf("expected 1 shadow, got %d", len(frame.shadows))
	}
	sh := frame.shadows[0]
	if !sh.Inset {
		t.Fatal("expected an inset shadow")
	}

	// scale is 2.0: bounds shrink by blur+spread (6) on the logical side.
	wantBounds := geometry.NewBounds(
		geometry.NewPoint[geometry.ScaledPixels](6*2, 6*2),
		geometry.NewSize[geometry.ScaledPixels](100*2-12*2, 100*2-12*2),
	)
	if sh.Bounds != wantBounds {
		t.Fatalf("shadow bounds = %v, want %v", sh.Bounds, wantBounds)
	}
}

func TestDivOverflowClipping(t *testing.T) {
	frame := newFakeFrame()
	child := NewDiv().
		Width(style.Px(200)).
		Height(style.Px(200)).
		Bg(colour.Rgba{R: 0, G: 0, B: 1, A: 1})

	parent := NewDiv().
		Width(style.Px(100)).
		Height(style.Px(100)).
		OverflowHidden().
		Child(child)

	frame.phase = phaseLayoutRequested
	rootID := parent.RequestLayout(frame)
	frame.solve(rootID, 100, 100)

	rootBounds := frame.LayoutBounds(rootID)
	frame.phase = phasePrepainted
	parent.Prepaint(frame, rootBounds)

	frame.phase = phasePainted
	parent.Paint(frame, rootBounds)

	// 1. Assert exactly 1 clip was pushed with parent's bounds
	if len(frame.pushedClips) != 1 {
		t.Fatalf("expected exactly 1 clip push, got %d", len(frame.pushedClips))
	}
	if frame.pushedClips[0] != rootBounds {
		t.Fatalf("expected pushed clip bounds %v, got %v", rootBounds, frame.pushedClips[0])
	}

	// 2. Assert PopClip was called
	if frame.popClipCount != 1 {
		t.Fatalf("expected exactly 1 clip pop, got %d", frame.popClipCount)
	}

	// 3. Clip stack must be empty after Paint finishes (push/pop balanced)
	if len(frame.clips) != 0 {
		t.Fatalf("expected clip stack to be empty after paint, got depth %d", len(frame.clips))
	}
}

// TestDivPrepaintClipExcludesHitRegion reproduces the bug docs/audit.md names
// under "Hit testing ignores clipping": a button scrolled out of an
// overflow-hidden container was invisible and still clickable, because hit
// regions were registered against unclipped bounds. Div.Prepaint now pushes
// its clip around its children's Prepaint the same way Paint already does, so
// a hit region registered outside the visible area is intersected down to
// nothing at registration time, not merely painted over later.
func TestDivPrepaintClipExcludesHitRegion(t *testing.T) {
	frame := newFakeFrame()

	// A button positioned entirely outside the 100x100 viewport its
	// overflow-hidden parent clips to, the way a scrolled-out list row sits
	// outside a ScrollView's viewport.
	child := NewDiv().
		Width(style.Px(200)).
		Height(style.Px(200)).
		OnClick(func(event ClickEvent) bool { return true })

	parent := NewDiv().
		Width(style.Px(100)).
		Height(style.Px(100)).
		OverflowHidden().
		Child(child)

	frame.phase = phaseLayoutRequested
	rootID := parent.RequestLayout(frame)

	// Bypass fakeFrame's simplified flex solve so the child's bounds can be
	// placed independently of its parent's, the way absolute positioning or a
	// scroll offset would.
	parentBounds := geometry.NewBounds(
		geometry.NewPoint[geometry.Pixels](0, 0),
		geometry.NewSize[geometry.Pixels](100, 100),
	)
	childBounds := geometry.NewBounds(
		geometry.NewPoint[geometry.Pixels](150, 150),
		geometry.NewSize[geometry.Pixels](200, 200),
	)
	frame.bounds[rootID] = parentBounds
	frame.bounds[child.layoutID] = childBounds

	frame.phase = phasePrepainted
	parent.Prepaint(frame, parentBounds)

	if len(frame.hitRegions) != 1 {
		t.Fatalf("expected 1 hit region, got %d", len(frame.hitRegions))
	}
	got := frame.hitRegions[0].bounds
	if got.Size.Width > 0 || got.Size.Height > 0 {
		t.Fatalf("expected the hit region clipped out of the overflow-hidden parent to be empty, got %v", got)
	}

	// The point where the child visually sits must not register as a hit.
	pt := geometry.NewPoint[geometry.Pixels](200, 200)
	if got.Contains(pt) {
		t.Fatalf("expected a pointer at %v, inside the clipped-out child, to miss hit region %v", pt, got)
	}
}

func TestDivTabStopAndTabIndex(t *testing.T) {
	frame := newFakeFrame()
	focusID1 := input.NewFocusID()
	focusID2 := input.NewFocusID()
	focusID3 := input.NewFocusID()

	d1 := NewDiv().
		TrackFocus(focusID1).
		TabIndex(3)

	d2 := NewDiv().
		TrackFocus(focusID2).
		TabStop(false)

	d3 := NewDiv().
		TrackFocus(focusID3)

	root := NewDiv().Children(d1, d2, d3)

	frame.phase = phaseLayoutRequested
	rootID := root.RequestLayout(frame)
	frame.solve(rootID, 200, 100)

	rootBounds := frame.LayoutBounds(rootID)
	frame.phase = phasePrepainted
	root.Prepaint(frame, rootBounds)

	if len(frame.dispatchNodes) != 3 {
		t.Fatalf("expected 3 dispatch nodes, got %d", len(frame.dispatchNodes))
	}
	n1 := frame.dispatchNodes[0]
	if n1.FocusID != focusID1 || !n1.TabStop || n1.TabIndex != 3 {
		t.Errorf("node 1: got FocusID=%v, TabStop=%v, TabIndex=%v; want %v, true, 3", n1.FocusID, n1.TabStop, n1.TabIndex, focusID1)
	}

	n2 := frame.dispatchNodes[1]
	if n2.FocusID != focusID2 || n2.TabStop {
		t.Errorf("node 2: got FocusID=%v, TabStop=%v; want %v, false", n2.FocusID, n2.TabStop, focusID2)
	}

	n3 := frame.dispatchNodes[2]
	if n3.FocusID != focusID3 || !n3.TabStop || n3.TabIndex != 0 {
		t.Errorf("node 3 (default): got FocusID=%v, TabStop=%v, TabIndex=%v; want %v, true, 0", n3.FocusID, n3.TabStop, n3.TabIndex, focusID3)
	}
}
