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
	offset geometry.Pixels
}

// NewScrollState constructs a new initialised ScrollState with zero offset.
func NewScrollState() ScrollState {
	return ScrollState{}
}

// Offset returns the current vertical scroll offset in logical pixels.
func (s *ScrollState) Offset() geometry.Pixels {
	return s.offset
}

// SetOffset sets the vertical scroll offset in logical pixels, clamped to non-negative.
func (s *ScrollState) SetOffset(offset geometry.Pixels) {
	if offset < 0 {
		offset = 0
	}
	s.offset = offset
}

// ScrollBy adjusts the vertical scroll offset by delta in logical pixels.
func (s *ScrollState) ScrollBy(delta geometry.Pixels) {
	s.SetOffset(s.offset + delta)
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
	viewport *element.Div
	content  *element.Div
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
		Relative().
		InsetTop(style.Px(-offset))

	if s.child != nil {
		s.content.Child(s.child)
	}

	s.viewport = element.NewDiv().
		Flex().
		FlexCol().
		Relative().
		OverflowScroll().
		Child(s.content)

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

// Paint draws viewport background and children clipped to the viewport bounds.
func (s *ScrollView) Paint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {
	s.viewport.Paint(f, bounds)
}
