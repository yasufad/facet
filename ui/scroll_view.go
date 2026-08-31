package ui

import (
	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/style"
)

// Default line height for line-based mouse wheel delta scrolling.
const defaultScrollLineHeight = geometry.Pixels(20)

// ScrollState holds retained vertical scrolling state across frames.
type ScrollState struct {
	offset         geometry.Pixels
	viewportHeight geometry.Pixels
	contentHeight  geometry.Pixels
}

// NewScrollState constructs a new initialised ScrollState with zero offset.
func NewScrollState() ScrollState {
	return ScrollState{}
}

// Offset returns the current vertical scroll offset in logical pixels.
func (s *ScrollState) Offset() geometry.Pixels {
	return s.offset
}

// MaxOffset returns the maximum valid vertical scroll offset based on recorded
// viewport and content dimensions.
func (s *ScrollState) MaxOffset() geometry.Pixels {
	if s.contentHeight <= s.viewportHeight {
		return 0
	}
	return s.contentHeight - s.viewportHeight
}

// SetOffset sets the vertical scroll offset in logical pixels, clamped between
// 0 and MaxOffset.
func (s *ScrollState) SetOffset(offset geometry.Pixels) {
	if offset < 0 {
		offset = 0
	}
	maxOffset := s.MaxOffset()
	if s.viewportHeight > 0 && s.contentHeight > 0 && offset > maxOffset {
		offset = maxOffset
	}
	s.offset = offset
}

// ScrollBy adjusts the vertical scroll offset by delta in logical pixels,
// clamping to valid scroll boundaries.
func (s *ScrollState) ScrollBy(delta geometry.Pixels) {
	s.SetOffset(s.offset + delta)
}

// UpdateMetrics updates the recorded viewport and content dimensions from the
// active frame pass and re-clamps the current scroll offset.
func (s *ScrollState) UpdateMetrics(viewportHeight, contentHeight geometry.Pixels) {
	s.viewportHeight = viewportHeight
	s.contentHeight = contentHeight
	s.SetOffset(s.offset)
}

type contentWrapper struct {
	inner    element.Element
	onLayout func(id element.NodeID)
}

func (w *contentWrapper) RequestLayout(f element.Frame) element.NodeID {
	id := w.inner.RequestLayout(f)
	if w.onLayout != nil {
		w.onLayout(id)
	}
	return id
}

func (w *contentWrapper) Prepaint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {
	w.inner.Prepaint(f, bounds)
}

func (w *contentWrapper) Paint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {
	w.inner.Paint(f, bounds)
}

// ScrollView is a container element providing vertical clipping and scroll wheel
// interaction over an inner content element.
type ScrollView struct {
	app        *app.App
	state      app.Entity[ScrollState]
	child      element.Element
	lineHeight geometry.Pixels
	refinement style.Refinement

	// Ephemeral element tree constructed for lifecycle execution
	viewport        *element.Div
	content         *element.Div
	contentLayoutID element.NodeID
}

// Ensure ScrollView implements element.Element.
var _ element.Element = (*ScrollView)(nil)

// NewScrollView constructs a new vertical ScrollView bound to the given App and
// retained ScrollState entity.
func NewScrollView(a *app.App, state app.Entity[ScrollState]) *ScrollView {
	return &ScrollView{
		app:        a,
		state:      state,
		lineHeight: defaultScrollLineHeight,
	}
}

// Child sets the content element contained within the scrollable viewport.
func (s *ScrollView) Child(child element.Element) *ScrollView {
	s.child = child
	return s
}

// LineHeight sets the scroll distance in logical pixels applied per notch of a
// line-based mouse wheel event.
func (s *ScrollView) LineHeight(height geometry.Pixels) *ScrollView {
	if height > 0 {
		s.lineHeight = height
	}
	return s
}

// Refine applies custom style overrides onto the scroll view viewport container.
func (s *ScrollView) Refine(r style.Refinement) *ScrollView {
	s.refinement.MergeFrom(&r)
	return s
}

// buildTree constructs the ephemeral viewport and content containers.
func (s *ScrollView) buildTree() {
	var offset geometry.Pixels
	if s.app != nil {
		st := s.state.Read(s.app)
		offset = st.Offset()
	}

	s.content = element.NewDiv().
		Flex().
		FlexCol().
		WFull().
		FlexShrink(0).
		Relative().
		InsetTop(style.Px(-offset))

	if s.child != nil {
		s.content.Child(s.child)
	}

	wrapper := &contentWrapper{
		inner: s.content,
		onLayout: func(id element.NodeID) {
			s.contentLayoutID = id
		},
	}

	s.viewport = element.NewDiv().
		Flex().
		FlexCol().
		WFull().
		HFull().
		Relative().
		OverflowScroll().
		Child(wrapper)

	s.viewport.Refine(s.refinement)

	s.viewport.OnScrollWheel(func(event platform.WheelEvent, phase input.DispatchPhase) bool {
		if phase != input.Bubble {
			return false
		}
		var deltaY geometry.Pixels
		if event.Delta.Unit == platform.ScrollPixels {
			deltaY = geometry.Pixels(event.Delta.DeltaY)
		} else {
			lh := s.lineHeight
			if lh <= 0 {
				lh = defaultScrollLineHeight
			}
			deltaY = geometry.Pixels(event.Delta.DeltaY) * lh
		}

		if s.app != nil {
			s.state.Update(s.app, func(st *ScrollState, cx *app.Context[ScrollState]) {
				st.ScrollBy(deltaY)
				cx.Notify()
			})
		}
		return true
	})
}

// RequestLayout builds the ephemeral viewport tree and requests layout through Frame.
func (s *ScrollView) RequestLayout(f element.Frame) element.NodeID {
	s.buildTree()
	return s.viewport.RequestLayout(f)
}

// Prepaint commits solved viewport bounds and registers hit and scroll listeners.
func (s *ScrollView) Prepaint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {
	s.viewport.Prepaint(f, bounds)
}

// Paint draws viewport background and children clipped to the viewport bounds,
// and records layout metrics into ScrollState for scroll clamping.
func (s *ScrollView) Paint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {
	if s.app != nil && s.contentLayoutID != (element.NodeID{}) {
		contentBounds := f.LayoutBounds(s.contentLayoutID)
		s.state.Update(s.app, func(st *ScrollState, cx *app.Context[ScrollState]) {
			st.UpdateMetrics(bounds.Size.Height, contentBounds.Size.Height)
		})
	}
	s.viewport.Paint(f, bounds)
}
