package ui

import (
	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/style"
)

// ListState holds retained virtual-list state across frames: scroll offset,
// viewport height, and total content extent. It is the list's own entity,
// analogous to ScrollState for ScrollView — the viewport belongs here, not
// in anything keyed by element identity.
type ListState struct {
	offset         geometry.Pixels
	viewportHeight geometry.Pixels
	contentHeight  geometry.Pixels
}

// NewListState constructs an initialised ListState with zero offset.
func NewListState() ListState { return ListState{} }

// Offset returns the current vertical scroll offset in logical pixels.
func (s *ListState) Offset() geometry.Pixels { return s.offset }

// MaxOffset returns the maximum valid scroll offset based on recorded
// viewport and content dimensions.
func (s *ListState) MaxOffset() geometry.Pixels {
	if s.contentHeight <= s.viewportHeight {
		return 0
	}
	return s.contentHeight - s.viewportHeight
}

// SetOffset sets the vertical scroll offset, clamped between 0 and MaxOffset.
func (s *ListState) SetOffset(offset geometry.Pixels) {
	if offset < 0 {
		offset = 0
	}
	maxOffset := s.MaxOffset()
	if s.viewportHeight > 0 && s.contentHeight > 0 && offset > maxOffset {
		offset = maxOffset
	}
	s.offset = offset
}

// ScrollBy adjusts the scroll offset by delta, clamping to valid boundaries.
func (s *ListState) ScrollBy(delta geometry.Pixels) {
	s.SetOffset(s.offset + delta)
}

// UpdateMetrics records the solved viewport and content dimensions from the
// active frame pass and re-clamps the current offset.
func (s *ListState) UpdateMetrics(viewportHeight, contentHeight geometry.Pixels) {
	s.viewportHeight = viewportHeight
	s.contentHeight = contentHeight
	s.SetOffset(s.offset)
}

// RenderItem produces an element for the item at the given index. Called only
// for items the list decides to build this frame.
type RenderItem func(index int) element.Element

// List is a virtualised vertical list. It declares its content extent as a
// definite height on its own container and builds only the items visible in
// the current viewport, so a list of thousands of items constructs a handful
// per frame.
//
// The container's height is the declared extent (count × itemHeight), not the
// sum of built children. This is the load-bearing decision: the scroll range
// comes from the declared extent, so it cannot drift from item-height
// estimates the way a spacer-derived extent would. The widget owns the
// container and does not expose it for styling — Refine reaches the item
// wrapper, not the container.
type List struct {
	app        *app.App
	state      app.Entity[ListState]
	count      int
	itemHeight geometry.Pixels
	render     RenderItem
	lineHeight geometry.Pixels
	refinement style.Refinement

	// Ephemeral element tree constructed for lifecycle execution.
	viewport     *element.Div
	container    *element.Div
	containerID  element.NodeID
	containerSet bool
}

// Ensure List implements element.Element.
var _ element.Element = (*List)(nil)

// NewList constructs a virtualised list bound to the given App and retained
// ListState entity.
func NewList(a *app.App, state app.Entity[ListState]) *List {
	return &List{
		app:        a,
		state:      state,
		lineHeight: defaultScrollLineHeight,
	}
}

// Count sets the total number of items in the list.
func (l *List) Count(n int) *List {
	l.count = n
	return l
}

// ItemHeight sets the uniform height of each item in logical pixels.
func (l *List) ItemHeight(h geometry.Pixels) *List {
	l.itemHeight = h
	return l
}

// Render sets the callback that produces an element for the item at a given
// index. Only visible indices trigger the callback each frame.
func (l *List) Render(r RenderItem) *List {
	l.render = r
	return l
}

// LineHeight sets the scroll distance per notch of a line-based mouse wheel.
func (l *List) LineHeight(h geometry.Pixels) *List {
	if h > 0 {
		l.lineHeight = h
	}
	return l
}

// Refine applies style overrides to the item wrapper, not the container.
// The container is owned by the widget: its height, flex-shrink and
// justify-content are load-bearing because it is deliberately taller than
// its children, and exposing them for styling would let a caller break the
// scroll range.
func (l *List) Refine(r style.Refinement) *List {
	l.refinement.MergeFrom(&r)
	return l
}

// buildTree constructs the ephemeral viewport and container. The container
// declares its extent as a definite height; only visible items are built
// inside it, positioned by a single top spacer.
func (l *List) buildTree() {
	var offset, viewportHeight, contentHeight geometry.Pixels
	if l.app != nil {
		st := l.state.Read(l.app)
		offset = st.Offset()
		viewportHeight = st.viewportHeight
		contentHeight = st.contentHeight
	}

	// The declared extent comes from widget configuration. On the first
	// frame the entity has not recorded anything yet, so contentHeight is
	// zero and we compute from count and itemHeight.
	declaredHeight := contentHeight
	if declaredHeight == 0 {
		declaredHeight = geometry.Pixels(l.count) * l.itemHeight
	}

	// Determine the visible index range. On the first frame viewportHeight
	// is zero, so nothing is built — the container's definite height still
	// gives the correct extent, and Paint records the viewport height for
	// the next frame.
	firstVisible := int(offset / l.itemHeight)
	if firstVisible < 0 {
		firstVisible = 0
	}
	lastVisible := firstVisible - 1
	if viewportHeight > 0 && l.itemHeight > 0 {
		endPx := offset + viewportHeight
		lastVisible = int(endPx / l.itemHeight)
		if float32(lastVisible)*float32(l.itemHeight) < float32(endPx) {
			lastVisible++ // ceil: include a partially visible item
		}
		lastVisible-- // convert "first non-visible" to "last visible"
	}
	if lastVisible >= l.count {
		lastVisible = l.count - 1
	}

	// The container: definite height, no shrink, column, packed at top.
	// JustifyContent FlexStart is load-bearing — the container is taller
	// than its children, so any other distribution would spread them.
	// InsetTop shifts the container up by the scroll offset; the viewport
	// clips what falls outside.
	l.container = element.NewDiv().
		Flex().
		FlexCol().
		WFull().
		Height(style.Px(declaredHeight)).
		FlexShrink(0).
		Relative().
		InsetTop(style.Px(-offset)).
		JustifyContent(style.AlignContentFlexStart)

	// Single top spacer positions the first built item at its content
	// position. No bottom spacer — the container's declared height fills
	// the remaining extent, and the empty space is below the last item.
	spacerHeight := geometry.Pixels(firstVisible) * l.itemHeight
	if spacerHeight > 0 {
		l.container.Child(
			element.NewDiv().Height(style.Px(spacerHeight)).FlexShrink(0),
		)
	}

	// Build only the visible items, each in a fixed-height wrapper. The
	// wrapper carries the user's refinement; the container does not.
	for i := firstVisible; i <= lastVisible; i++ {
		wrapper := element.NewDiv().
			Height(style.Px(l.itemHeight)).
			FlexShrink(0).
			WFull().
			Refine(l.refinement)
		if l.render != nil {
			wrapper.Child(l.render(i))
		}
		l.container.Child(wrapper)
	}

	// The wrapper captures the container's layout ID so Paint can read
	// solved bounds without a separate layout node.
	idWrapper := &contentWrapper{
		inner: l.container,
		onLayout: func(id element.NodeID) {
			l.containerID = id
			l.containerSet = true
		},
	}

	// The viewport fills its parent, clips overflow, and handles wheel
	// scrolling.
	l.viewport = element.NewDiv().
		Flex().
		FlexCol().
		WFull().
		HFull().
		Relative().
		OverflowScroll().
		Child(idWrapper)

	l.viewport.OnScrollWheel(func(event input.WheelEvent, phase input.DispatchPhase) bool {
		if phase != input.Bubble {
			return false
		}
		var deltaY geometry.Pixels
		if event.Delta.Unit == input.ScrollPixels {
			deltaY = geometry.Pixels(event.Delta.DeltaY)
		} else {
			lh := l.lineHeight
			if lh <= 0 {
				lh = defaultScrollLineHeight
			}
			deltaY = geometry.Pixels(event.Delta.DeltaY) * lh
		}
		if l.app != nil {
			l.state.Update(l.app, func(st *ListState, cx *app.Context[ListState]) {
				st.ScrollBy(deltaY)
				cx.Notify()
			})
		}
		return true
	})
}

// RequestLayout builds the ephemeral tree and requests layout through Frame.
func (l *List) RequestLayout(f element.Frame) element.NodeID {
	l.buildTree()
	return l.viewport.RequestLayout(f)
}

// Prepaint commits solved bounds and registers hit and scroll listeners.
func (l *List) Prepaint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {
	l.viewport.Prepaint(f, bounds)
}

// Paint draws the viewport and records solved metrics into ListState. The
// container's solved height is the content extent — it equals the declared
// height when the container has one, and falls back to the built children's
// extent when it does not, which is what the scroll range tracks.
func (l *List) Paint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {
	if l.app != nil && l.containerSet {
		containerBounds := f.LayoutBounds(l.containerID)
		l.state.Update(l.app, func(st *ListState, cx *app.Context[ListState]) {
			st.UpdateMetrics(bounds.Size.Height, containerBounds.Size.Height)
		})
	}
	l.viewport.Paint(f, bounds)
}
