package elementtest

import (
	"testing"

	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/style"
)

// TestDivPrepaintClipExcludesHitRegion is element's own
// TestDivPrepaintClipExcludesHitRegion, run again through this package's
// exported Frame double rather than element's internal one. It exists
// because fixing element's internal fake left this double still enforcing
// PushClip/PopClip's old paint-only rule, which panicked the first time ui
// prepainted an interactive Div through it — "elementtest: PushClip called
// outside paint phase" — since nothing here drove a clipping Div through
// Prepaint to catch it first.
func TestDivPrepaintClipExcludesHitRegion(t *testing.T) {
	frame := NewFrame()

	// A button positioned entirely outside the 100x100 viewport its
	// overflow-hidden parent clips to, the way a scrolled-out list row sits
	// outside a ScrollView's viewport.
	child := element.NewDiv().
		Width(style.Px(200)).
		Height(style.Px(200)).
		OnClick(func(event element.ClickEvent) bool { return true })

	parent := element.NewDiv().
		Width(style.Px(100)).
		Height(style.Px(100)).
		OverflowHidden().
		Child(child)

	rootID := parent.RequestLayout(frame)

	childNodes := frame.ChildrenOf(rootID)
	if len(childNodes) != 1 {
		t.Fatalf("expected 1 child layout node, got %d", len(childNodes))
	}

	// Bypass Frame's Taffy solve so the child's bounds can be placed
	// independently of its parent's, the way absolute positioning or a
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
	frame.bounds[childNodes[0]] = childBounds

	frame.SetPhase(PhasePrepaint)
	parent.Prepaint(frame, parentBounds)

	regions := frame.HitRegions()
	if len(regions) != 1 {
		t.Fatalf("expected 1 hit region, got %d", len(regions))
	}
	got := regions[0].Bounds
	if got.Size.Width > 0 || got.Size.Height > 0 {
		t.Fatalf("expected the hit region clipped out of the overflow-hidden parent to be empty, got %v", got)
	}

	pt := geometry.NewPoint[geometry.Pixels](200, 200)
	if got.Contains(pt) {
		t.Fatalf("expected a pointer at %v, inside the clipped-out child, to miss hit region %v", pt, got)
	}
}
