package ui

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/style"
)

// Default button theme colours.
var (
	defaultButtonBg       = colour.Rgba{R: 0.22, G: 0.25, B: 0.30, A: 1.0}
	defaultButtonHoverBg  = colour.Rgba{R: 0.28, G: 0.32, B: 0.38, A: 1.0}
	defaultButtonActiveBg = colour.Rgba{R: 0.18, G: 0.20, B: 0.24, A: 1.0}
	defaultButtonBorder   = colour.Rgba{R: 0.35, G: 0.40, B: 0.48, A: 1.0}
	defaultButtonText     = colour.Rgba{R: 0.95, G: 0.95, B: 0.95, A: 1.0}
	defaultButtonFocus    = colour.Rgba{R: 0.30, G: 0.55, B: 0.90, A: 1.0}
)

// Button is an interactive button widget displaying a single-line label
// with hover, active, and focus pseudo-state styling.
type Button struct {
	label    string
	disabled bool
	focusID  input.FocusID
	onClick  func(event element.ClickEvent) bool

	// User-configured style refinements
	refinement  style.Refinement
	hoverStyle  *style.Refinement
	activeStyle *style.Refinement
	focusStyle  *style.Refinement

	// Ephemeral element tree constructed for lifecycle execution
	div  *element.Div
	text *element.Text
}

// Ensure Button implements element.Element.
var _ element.Element = (*Button)(nil)

// NewButton constructs a new Button with the specified label text.
func NewButton(label string) *Button {
	return &Button{
		label: label,
	}
}

// Label updates the button's display text.
func (b *Button) Label(label string) *Button {
	b.label = label
	return b
}

// Disabled sets whether the button ignores interactions and renders with disabled styling.
func (b *Button) Disabled(disabled bool) *Button {
	b.disabled = disabled
	return b
}

// OnClick registers a callback invoked when the button is clicked.
func (b *Button) OnClick(handler func(event element.ClickEvent) bool) *Button {
	b.onClick = handler
	return b
}

// TrackFocus registers a focus identifier on this button for keyboard navigation and focus styling.
func (b *Button) TrackFocus(id input.FocusID) *Button {
	b.focusID = id
	return b
}

// Refine applies custom style overrides onto the button container.
func (b *Button) Refine(r style.Refinement) *Button {
	b.refinement.MergeFrom(&r)
	return b
}

// Hover configures style overrides applied when the button is hovered.
func (b *Button) Hover(f func(r *style.Refinement)) *Button {
	if b.hoverStyle == nil {
		b.hoverStyle = &style.Refinement{}
	}
	f(b.hoverStyle)
	return b
}

// Active configures style overrides applied when the button is actively pressed.
func (b *Button) Active(f func(r *style.Refinement)) *Button {
	if b.activeStyle == nil {
		b.activeStyle = &style.Refinement{}
	}
	f(b.activeStyle)
	return b
}

// Focus configures style overrides applied when the button holds keyboard focus.
func (b *Button) Focus(f func(r *style.Refinement)) *Button {
	if b.focusStyle == nil {
		b.focusStyle = &style.Refinement{}
	}
	f(b.focusStyle)
	return b
}

// buildTree constructs the internal element tree (*element.Div and *element.Text)
// with default styling, user refinements, and interaction handlers.
func (b *Button) buildTree() {
	b.div = element.NewDiv().
		Flex().
		AlignItems(style.AlignItemsCentre).
		JustifyContent(style.AlignContentCentre).
		PaddingX(style.Px(12)).
		PaddingY(style.Px(6)).
		Rounded(geometry.Pixels(4)).
		Border(geometry.Pixels(1)).
		BorderColour(defaultButtonBorder).
		Bg(defaultButtonBg).
		Cursor(style.CursorPointer)

	// Default hover styling: lighter background
	b.div.Hover(func(r *style.Refinement) {
		r.SetBackground(defaultButtonHoverBg)
	})

	// Default active styling: pressed background
	b.div.Active(func(r *style.Refinement) {
		r.SetBackground(defaultButtonActiveBg)
	})

	// Default focus styling: distinct border ring
	b.div.Focus(func(r *style.Refinement) {
		r.SetBorderColour(defaultButtonFocus)
		r.SetBorderWidth(geometry.Pixels(2))
	})

	// Apply user overrides on top of defaults
	if b.hoverStyle != nil {
		b.div.Hover(func(r *style.Refinement) {
			r.MergeFrom(b.hoverStyle)
		})
	}
	if b.activeStyle != nil {
		b.div.Active(func(r *style.Refinement) {
			r.MergeFrom(b.activeStyle)
		})
	}
	if b.focusStyle != nil {
		b.div.Focus(func(r *style.Refinement) {
			r.MergeFrom(b.focusStyle)
		})
	}
	b.div.Refine(b.refinement)

	if b.focusID != 0 {
		b.div.TrackFocus(b.focusID)
	}

	if b.disabled {
		b.div.Cursor(style.CursorNotAllowed)
		b.div.Opacity(0.5)
	} else if b.onClick != nil {
		b.div.OnClick(b.onClick)
	}

	if b.label != "" {
		b.text = element.NewText(b.label).
			TextColour(defaultButtonText)
		b.div.Child(b.text)
	}
}

// RequestLayout builds the ephemeral element tree, calculates flexbox layout
// nodes through Frame, and returns the root NodeID.
func (b *Button) RequestLayout(f element.Frame) element.NodeID {
	b.buildTree()
	return b.div.RequestLayout(f)
}

// Prepaint commits solved bounds, registers hit regions, and prepares dispatch nodes.
func (b *Button) Prepaint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {
	b.div.Prepaint(f, bounds)
}

// Paint draws the button background, borders, and text glyphs into the frame scene.
func (b *Button) Paint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {
	b.div.Paint(f, bounds)
}
