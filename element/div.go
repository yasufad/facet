package element

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
)

type drawPhase uint8

const (
	phaseInitial drawPhase = iota
	phaseLayoutRequested
	phasePrepainted
	phasePainted
)

// Div is the general-purpose container element in Facet.
//
// It lays out and paints its children according to its flexbox style properties
// and emits background and border primitives into the frame scene.
type Div struct {
	refinement style.Refinement
	children   []Element

	// Per-phase state mutated across the lifecycle.
	layoutID       layout.NodeID
	childLayoutIDs []layout.NodeID
	bounds         geometry.Bounds[geometry.Pixels]
	phase          drawPhase
}

// NewDiv creates a new, unstyled Div container element.
func NewDiv() *Div {
	return &Div{}
}

// Flex sets the display layout strategy to flexbox.
func (d *Div) Flex() *Div {
	d.refinement.SetDisplay(style.DisplayFlex)
	return d
}

// Bg sets the background colour of the div.
func (d *Div) Bg(c colour.Rgba) *Div {
	d.refinement.SetBackground(c)
	return d
}

// BgHsla sets the background colour from an HSLA value.
func (d *Div) BgHsla(c colour.Hsla) *Div {
	d.refinement.SetBackgroundHsla(c)
	return d
}

// Opacity sets the opacity of the div in [0, 1].
func (d *Div) Opacity(o float32) *Div {
	d.refinement.SetOpacity(o)
	return d
}

// FlexGrow sets the flex grow factor of the div.
func (d *Div) FlexGrow(grow float32) *Div {
	d.refinement.SetFlexGrow(grow)
	return d
}

// Child adds a single child element to the div. Nil children are ignored.
func (d *Div) Child(child Element) *Div {
	if child != nil {
		d.children = append(d.children, child)
	}
	return d
}

// Children adds multiple child elements to the div. Nil children are ignored.
func (d *Div) Children(children ...Element) *Div {
	for _, child := range children {
		if child != nil {
			d.children = append(d.children, child)
		}
	}
	return d
}

// RequestLayout requests layout for all children, converts the resolved style
// to layout inputs, and adds this element to the layout tree.
func (d *Div) RequestLayout(f Frame) layout.NodeID {
	if d.phase != phaseInitial {
		panic("element: RequestLayout called out of order or multiple times")
	}
	d.phase = phaseLayoutRequested

	d.childLayoutIDs = d.childLayoutIDs[:0]
	for _, child := range d.children {
		childID := child.RequestLayout(f)
		d.childLayoutIDs = append(d.childLayoutIDs, childID)
	}

	st := style.Default()
	st.Refine(d.refinement)

	rem := f.RemSize()
	layoutStyle := st.ToLayout(rem)
	d.layoutID = f.RequestLayout(layoutStyle, d.childLayoutIDs)
	return d.layoutID
}

// Prepaint commits computed bounds and prepaints children.
func (d *Div) Prepaint(f Frame, bounds geometry.Bounds[geometry.Pixels]) {
	if d.phase != phaseLayoutRequested {
		panic("element: Prepaint called before RequestLayout or out of order")
	}
	d.phase = phasePrepainted
	d.bounds = bounds

	st := style.Default()
	st.Refine(d.refinement)
	if st.Display == style.DisplayNone {
		return
	}

	for i, child := range d.children {
		childLayoutID := d.childLayoutIDs[i]
		childBounds := f.LayoutBounds(childLayoutID)
		child.Prepaint(f, childBounds)
	}
}

// Paint draws the background and border quad if visible and paints children.
func (d *Div) Paint(f Frame, bounds geometry.Bounds[geometry.Pixels]) {
	if d.phase != phasePrepainted {
		panic("element: Paint called before Prepaint or out of order")
	}
	d.phase = phasePainted
	d.bounds = bounds

	st := style.Default()
	st.Refine(d.refinement)
	if st.Display == style.DisplayNone {
		return
	}

	if st.Background.A > 0 || st.BorderColour.A > 0 {
		scale := f.ScaleFactor()
		scaledBounds := geometry.Bounds[geometry.ScaledPixels]{
			Origin: geometry.Point[geometry.ScaledPixels]{
				X: bounds.Origin.X.Scale(scale),
				Y: bounds.Origin.Y.Scale(scale),
			},
			Size: geometry.Size[geometry.ScaledPixels]{
				Width:  bounds.Size.Width.Scale(scale),
				Height: bounds.Size.Height.Scale(scale),
			},
		}
		q := scene.Quad{
			Bounds:       scaledBounds,
			Background:   st.Background,
			BorderColour: st.BorderColour,
			CornerRadii: geometry.Corners[geometry.ScaledPixels]{
				TopLeft:     st.CornerRadii.TopLeft.Scale(scale),
				TopRight:    st.CornerRadii.TopRight.Scale(scale),
				BottomRight: st.CornerRadii.BottomRight.Scale(scale),
				BottomLeft:  st.CornerRadii.BottomLeft.Scale(scale),
			},
			BorderWidths: geometry.Edges[geometry.ScaledPixels]{
				Top:    st.BorderWidths.Top.Scale(scale),
				Right:  st.BorderWidths.Right.Scale(scale),
				Bottom: st.BorderWidths.Bottom.Scale(scale),
				Left:   st.BorderWidths.Left.Scale(scale),
			},
			BorderStyle: scene.BorderStyle(st.BorderStyle),
		}
		f.InsertQuad(q)
	}

	for i, child := range d.children {
		childLayoutID := d.childLayoutIDs[i]
		childBounds := f.LayoutBounds(childLayoutID)
		child.Paint(f, childBounds)
	}
}
