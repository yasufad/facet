package element

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/layout"
	"github.com/yasufad/facet/scene"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/text"
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
// It lays out and paints its children according to its flexbox style properties,
// registers input dispatch nodes and hit regions for interactive listeners,
// and emits background and border primitives into the frame scene.
type Div struct {
	refinement    style.Refinement
	interactivity Interactivity
	children      []Element

	// Per-phase state mutated across the lifecycle.
	layoutID       layout.NodeID
	childLayoutIDs []layout.NodeID
	bounds         geometry.Bounds[geometry.Pixels]
	phase          drawPhase

	// resolvedStyle is refinement resolved against style.Default(), computed
	// once in RequestLayout and reused by Prepaint and Paint rather than
	// rebuilt from the 504-byte Refinement on every phase.
	resolvedStyle style.Style
}

// NewDiv creates a new, unstyled Div container element.
func NewDiv() *Div {
	return &Div{}
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

// Refine applies all explicitly set properties from r onto this element.
func (d *Div) Refine(r style.Refinement) *Div {
	d.refinement.MergeFrom(&r)
	return d
}

// --- Interactivity & Event Handlers ---

// KeyContext sets the keymap context associated with this element.
func (d *Div) KeyContext(ctx input.KeyContext) *Div {
	d.interactivity.keyContext = &ctx
	return d
}

// TrackFocus associates a focus identifier with this element.
func (d *Div) TrackFocus(focusID input.FocusID) *Div {
	d.interactivity.focusID = focusID
	return d
}

// TabStop configures whether this element participates in keyboard tab order navigation.
func (d *Div) TabStop(tabStop bool) *Div {
	d.interactivity.tabStop = tabStop
	d.interactivity.tabStopSet = true
	return d
}

// TabIndex sets an explicit ordering index for keyboard tab navigation.
func (d *Div) TabIndex(index int) *Div {
	d.interactivity.tabIndex = index
	return d
}

// OnAction registers an action listener invoked when a matching action reaches this element.
func (d *Div) OnAction(actionName string, handler input.ActionHandler) *Div {
	d.interactivity.actionBindings = append(d.interactivity.actionBindings, ActionBinding{
		ActionName: actionName,
		Handler:    handler,
	})
	return d
}

// CaptureAction registers an action handler for the capture dispatch phase.
func (d *Div) CaptureAction(actionName string, handler input.ActionHandler) *Div {
	d.interactivity.actionBindings = append(d.interactivity.actionBindings, ActionBinding{
		ActionName: actionName,
		Handler: func(action input.Action, phase input.DispatchPhase) bool {
			if phase == input.Capture {
				return handler(action, phase)
			}
			return false
		},
	})
	return d
}

// OnKeyDown registers a raw key event listener.
func (d *Div) OnKeyDown(handler input.KeyEventHandler) *Div {
	d.interactivity.keyListeners = append(d.interactivity.keyListeners, handler)
	return d
}

// OnKeyUp registers a raw key up event listener.
func (d *Div) OnKeyUp(handler input.KeyEventHandler) *Div {
	d.interactivity.keyListeners = append(d.interactivity.keyListeners, handler)
	return d
}

// OnMouseDown registers a pointer event listener.
func (d *Div) OnMouseDown(handler input.PointerEventHandler) *Div {
	d.interactivity.pointerListeners = append(d.interactivity.pointerListeners, handler)
	return d
}

// OnMouseUp registers a pointer event listener.
func (d *Div) OnMouseUp(handler input.PointerEventHandler) *Div {
	d.interactivity.pointerListeners = append(d.interactivity.pointerListeners, handler)
	return d
}

// OnMouseMove registers a pointer move listener.
func (d *Div) OnMouseMove(handler input.PointerEventHandler) *Div {
	d.interactivity.pointerListeners = append(d.interactivity.pointerListeners, handler)
	return d
}

// OnScrollWheel registers a scroll wheel event listener.
func (d *Div) OnScrollWheel(handler input.WheelEventHandler) *Div {
	d.interactivity.wheelListeners = append(d.interactivity.wheelListeners, handler)
	return d
}

// OnClick registers a callback invoked upon a synthesised click interaction.
func (d *Div) OnClick(handler func(event ClickEvent) bool) *Div {
	d.interactivity.clickListeners = append(d.interactivity.clickListeners, handler)
	return d
}

// Occlude marks this element as blocking pointer interactions for elements behind it.
func (d *Div) Occlude() *Div {
	d.interactivity.occlude = true
	return d
}

// Hover configures style property overrides applied when this element is hovered
// during the paint phase of the current frame.
func (d *Div) Hover(f func(r *style.Refinement)) *Div {
	if d.interactivity.hoverStyle == nil {
		d.interactivity.hoverStyle = &style.Refinement{}
	}
	f(d.interactivity.hoverStyle)
	return d
}

// Active configures style property overrides applied when this element is actively pressed
// during the paint phase of the current frame.
func (d *Div) Active(f func(r *style.Refinement)) *Div {
	if d.interactivity.activeStyle == nil {
		d.interactivity.activeStyle = &style.Refinement{}
	}
	f(d.interactivity.activeStyle)
	return d
}

// Focus configures style property overrides applied when this element holds keyboard focus
// during the paint phase.
func (d *Div) Focus(f func(r *style.Refinement)) *Div {
	if d.interactivity.focusStyle == nil {
		d.interactivity.focusStyle = &style.Refinement{}
	}
	f(d.interactivity.focusStyle)
	return d
}

// InFocus configures style property overrides applied when an ancestor holds keyboard focus
// during the paint phase.
func (d *Div) InFocus(f func(r *style.Refinement)) *Div {
	if d.interactivity.inFocusStyle == nil {
		d.interactivity.inFocusStyle = &style.Refinement{}
	}
	f(d.interactivity.inFocusStyle)
	return d
}

// --- Display & Position ---

// Display sets the layout strategy for children.
func (d *Div) Display(disp style.Display) *Div {
	d.refinement.SetDisplay(disp)
	return d
}

// Flex sets the layout strategy to flexbox.
func (d *Div) Flex() *Div {
	d.refinement.SetDisplay(style.DisplayFlex)
	return d
}

// Block sets the layout strategy to standard block layout.
func (d *Div) Block() *Div {
	d.refinement.SetDisplay(style.DisplayBlock)
	return d
}

// None removes the element from layout calculation entirely.
func (d *Div) None() *Div {
	d.refinement.SetDisplay(style.DisplayNone)
	return d
}

// Hidden removes the element from layout calculation entirely (display: none),
// matching the standard Tailwind and GPUI convention.
func (d *Div) Hidden() *Div {
	d.refinement.SetDisplay(style.DisplayNone)
	return d
}

// Position sets the CSS positioning strategy.
func (d *Div) Position(p style.Position) *Div {
	d.refinement.SetPosition(p)
	return d
}

// Relative sets the positioning strategy to relative.
func (d *Div) Relative() *Div {
	d.refinement.SetPosition(style.PositionRelative)
	return d
}

// Absolute sets the positioning strategy to absolute.
func (d *Div) Absolute() *Div {
	d.refinement.SetPosition(style.PositionAbsolute)
	return d
}

// Visibility sets whether the element is rendered.
func (d *Div) Visibility(v style.Visibility) *Div {
	d.refinement.SetVisibility(v)
	return d
}

// Visible sets the element visibility to visible (visibility: visible).
func (d *Div) Visible() *Div {
	d.refinement.SetVisibility(style.VisibilityVisible)
	return d
}

// Invisible hides the element in place without removing it from layout
// calculation (visibility: hidden).
func (d *Div) Invisible() *Div {
	d.refinement.SetVisibility(style.VisibilityHidden)
	return d
}

// --- Overflow & Scrolling ---

// Overflow sets both horizontal and vertical overflow handling.
func (d *Div) Overflow(o style.Overflow) *Div {
	d.refinement.SetOverflow(o)
	return d
}

// OverflowX sets horizontal overflow handling.
func (d *Div) OverflowX(o style.Overflow) *Div {
	d.refinement.SetOverflowX(o)
	return d
}

// OverflowY sets vertical overflow handling.
func (d *Div) OverflowY(o style.Overflow) *Div {
	d.refinement.SetOverflowY(o)
	return d
}

// OverflowVisible sets overflow handling on both axes to visible.
func (d *Div) OverflowVisible() *Div {
	d.refinement.SetOverflow(style.OverflowVisible)
	return d
}

// OverflowClip sets overflow handling on both axes to clip.
func (d *Div) OverflowClip() *Div {
	d.refinement.SetOverflow(style.OverflowClip)
	return d
}

// OverflowHidden sets overflow handling on both axes to hidden.
func (d *Div) OverflowHidden() *Div {
	d.refinement.SetOverflow(style.OverflowHidden)
	return d
}

// OverflowScroll sets overflow handling on both axes to scroll.
func (d *Div) OverflowScroll() *Div {
	d.refinement.SetOverflow(style.OverflowScroll)
	return d
}

// ScrollbarWidth sets the space reserved for scrollbars.
func (d *Div) ScrollbarWidth(w geometry.Pixels) *Div {
	d.refinement.SetScrollbarWidth(w)
	return d
}

// RestrictScrollToAxis sets whether scroll is locked to the dominant gesture axis.
func (d *Div) RestrictScrollToAxis(restrict bool) *Div {
	d.refinement.SetRestrictScrollToAxis(restrict)
	return d
}

// --- Insets ---

// Inset sets all four inset offsets.
func (d *Div) Inset(l style.Length) *Div {
	d.refinement.SetInset(l)
	return d
}

// InsetTop sets the top inset offset.
func (d *Div) InsetTop(l style.Length) *Div {
	d.refinement.SetInsetTop(l)
	return d
}

// InsetRight sets the right inset offset.
func (d *Div) InsetRight(l style.Length) *Div {
	d.refinement.SetInsetRight(l)
	return d
}

// InsetBottom sets the bottom inset offset.
func (d *Div) InsetBottom(l style.Length) *Div {
	d.refinement.SetInsetBottom(l)
	return d
}

// InsetLeft sets the left inset offset.
func (d *Div) InsetLeft(l style.Length) *Div {
	d.refinement.SetInsetLeft(l)
	return d
}

// InsetX sets both horizontal inset offsets (left and right).
func (d *Div) InsetX(l style.Length) *Div {
	d.refinement.SetInsetLeft(l)
	d.refinement.SetInsetRight(l)
	return d
}

// InsetY sets both vertical inset offsets (top and bottom).
func (d *Div) InsetY(l style.Length) *Div {
	d.refinement.SetInsetTop(l)
	d.refinement.SetInsetBottom(l)
	return d
}

// --- Margin ---

// Margin sets margin on all four sides.
func (d *Div) Margin(l style.Length) *Div {
	d.refinement.SetMargin(l)
	return d
}

// MarginTop sets top margin.
func (d *Div) MarginTop(l style.Length) *Div {
	d.refinement.SetMarginTop(l)
	return d
}

// MarginRight sets right margin.
func (d *Div) MarginRight(l style.Length) *Div {
	d.refinement.SetMarginRight(l)
	return d
}

// MarginBottom sets bottom margin.
func (d *Div) MarginBottom(l style.Length) *Div {
	d.refinement.SetMarginBottom(l)
	return d
}

// MarginLeft sets left margin.
func (d *Div) MarginLeft(l style.Length) *Div {
	d.refinement.SetMarginLeft(l)
	return d
}

// MarginX sets both horizontal margins (left and right).
func (d *Div) MarginX(l style.Length) *Div {
	d.refinement.SetMarginLeft(l)
	d.refinement.SetMarginRight(l)
	return d
}

// MarginY sets both vertical margins (top and bottom).
func (d *Div) MarginY(l style.Length) *Div {
	d.refinement.SetMarginTop(l)
	d.refinement.SetMarginBottom(l)
	return d
}

// --- Padding ---

// Padding sets padding on all four sides.
func (d *Div) Padding(l style.Length) *Div {
	d.refinement.SetPadding(l)
	return d
}

// PaddingTop sets top padding.
func (d *Div) PaddingTop(l style.Length) *Div {
	d.refinement.SetPaddingTop(l)
	return d
}

// PaddingRight sets right padding.
func (d *Div) PaddingRight(l style.Length) *Div {
	d.refinement.SetPaddingRight(l)
	return d
}

// PaddingBottom sets bottom padding.
func (d *Div) PaddingBottom(l style.Length) *Div {
	d.refinement.SetPaddingBottom(l)
	return d
}

// PaddingLeft sets left padding.
func (d *Div) PaddingLeft(l style.Length) *Div {
	d.refinement.SetPaddingLeft(l)
	return d
}

// PaddingX sets both horizontal paddings (left and right).
func (d *Div) PaddingX(l style.Length) *Div {
	d.refinement.SetPaddingLeft(l)
	d.refinement.SetPaddingRight(l)
	return d
}

// PaddingY sets both vertical paddings (top and bottom).
func (d *Div) PaddingY(l style.Length) *Div {
	d.refinement.SetPaddingTop(l)
	d.refinement.SetPaddingBottom(l)
	return d
}

// --- Borders ---

// Border sets border line thickness on all four sides.
func (d *Div) Border(w geometry.Pixels) *Div {
	d.refinement.SetBorderWidth(w)
	return d
}

// BorderTop sets top border width.
func (d *Div) BorderTop(w geometry.Pixels) *Div {
	d.refinement.SetBorderWidthTop(w)
	return d
}

// BorderRight sets right border width.
func (d *Div) BorderRight(w geometry.Pixels) *Div {
	d.refinement.SetBorderWidthRight(w)
	return d
}

// BorderBottom sets bottom border width.
func (d *Div) BorderBottom(w geometry.Pixels) *Div {
	d.refinement.SetBorderWidthBottom(w)
	return d
}

// BorderLeft sets left border width.
func (d *Div) BorderLeft(w geometry.Pixels) *Div {
	d.refinement.SetBorderWidthLeft(w)
	return d
}

// BorderColour sets the border colour.
func (d *Div) BorderColour(c colour.Rgba) *Div {
	d.refinement.SetBorderColour(c)
	return d
}

// BorderColourHsla sets the border colour from an Hsla value.
func (d *Div) BorderColourHsla(c colour.Hsla) *Div {
	d.refinement.SetBorderColourHsla(c)
	return d
}

// BorderStyle sets the border line style.
func (d *Div) BorderStyle(s style.BorderStyle) *Div {
	d.refinement.SetBorderStyle(s)
	return d
}

// --- Corner Radii ---

// CornerRadius sets corner radius on all four corners.
func (d *Div) CornerRadius(radius geometry.Pixels) *Div {
	d.refinement.SetCornerRadius(radius)
	return d
}

// Rounded is an alias for CornerRadius.
func (d *Div) Rounded(radius geometry.Pixels) *Div {
	d.refinement.SetCornerRadius(radius)
	return d
}

// CornerRadiusTopLeft sets top-left corner radius.
func (d *Div) CornerRadiusTopLeft(radius geometry.Pixels) *Div {
	d.refinement.SetCornerRadiusTopLeft(radius)
	return d
}

// CornerRadiusTopRight sets top-right corner radius.
func (d *Div) CornerRadiusTopRight(radius geometry.Pixels) *Div {
	d.refinement.SetCornerRadiusTopRight(radius)
	return d
}

// CornerRadiusBottomRight sets bottom-right corner radius.
func (d *Div) CornerRadiusBottomRight(radius geometry.Pixels) *Div {
	d.refinement.SetCornerRadiusBottomRight(radius)
	return d
}

// CornerRadiusBottomLeft sets bottom-left corner radius.
func (d *Div) CornerRadiusBottomLeft(radius geometry.Pixels) *Div {
	d.refinement.SetCornerRadiusBottomLeft(radius)
	return d
}

// --- Sizing ---

// Size sets both preferred width and height.
func (d *Div) Size(width, height style.Length) *Div {
	d.refinement.SetSize(width, height)
	return d
}

// Width sets the preferred width.
func (d *Div) Width(w style.Length) *Div {
	d.refinement.SetWidth(w)
	return d
}

// Height sets the preferred height.
func (d *Div) Height(h style.Length) *Div {
	d.refinement.SetHeight(h)
	return d
}

// WFull sets preferred width to 100%.
func (d *Div) WFull() *Div {
	d.refinement.SetWidth(style.Percent(1.0))
	return d
}

// HFull sets preferred height to 100%.
func (d *Div) HFull() *Div {
	d.refinement.SetHeight(style.Percent(1.0))
	return d
}

// Full sets preferred width and height to 100%.
func (d *Div) Full() *Div {
	d.refinement.SetSize(style.Percent(1.0), style.Percent(1.0))
	return d
}

// MinSize sets both minimum width and height constraints.
func (d *Div) MinSize(width, height style.Length) *Div {
	d.refinement.SetMinSize(width, height)
	return d
}

// MinWidth sets minimum width constraint.
func (d *Div) MinWidth(w style.Length) *Div {
	d.refinement.SetMinWidth(w)
	return d
}

// MinHeight sets minimum height constraint.
func (d *Div) MinHeight(h style.Length) *Div {
	d.refinement.SetMinHeight(h)
	return d
}

// MaxSize sets both maximum width and height constraints.
func (d *Div) MaxSize(width, height style.Length) *Div {
	d.refinement.SetMaxSize(width, height)
	return d
}

// MaxWidth sets maximum width constraint.
func (d *Div) MaxWidth(w style.Length) *Div {
	d.refinement.SetMaxWidth(w)
	return d
}

// MaxHeight sets maximum height constraint.
func (d *Div) MaxHeight(h style.Length) *Div {
	d.refinement.SetMaxHeight(h)
	return d
}

// AspectRatio sets the preferred width-to-height ratio.
func (d *Div) AspectRatio(ratio float32) *Div {
	d.refinement.SetAspectRatio(ratio)
	return d
}

// --- Gap ---

// Gap sets both row and column gap between flex items.
func (d *Div) Gap(row, col style.Length) *Div {
	d.refinement.SetGap(row, col)
	return d
}

// GapRow sets row gap.
func (d *Div) GapRow(row style.Length) *Div {
	d.refinement.SetGapRow(row)
	return d
}

// GapCol sets column gap.
func (d *Div) GapCol(col style.Length) *Div {
	d.refinement.SetGapCol(col)
	return d
}

// --- Alignment & Distribution ---

// AlignItems sets cross-axis alignment for children.
func (d *Div) AlignItems(a style.AlignItems) *Div {
	d.refinement.SetAlignItems(a)
	return d
}

// AlignSelf sets cross-axis alignment for this item.
func (d *Div) AlignSelf(a style.AlignSelf) *Div {
	d.refinement.SetAlignSelf(a)
	return d
}

// AlignContent sets multi-line content distribution.
func (d *Div) AlignContent(a style.AlignContent) *Div {
	d.refinement.SetAlignContent(a)
	return d
}

// JustifyContent sets main-axis distribution for children.
func (d *Div) JustifyContent(j style.JustifyContent) *Div {
	d.refinement.SetJustifyContent(j)
	return d
}

// --- Flexbox Properties ---

// FlexDirection sets the flexbox main axis direction.
func (d *Div) FlexDirection(dir style.FlexDirection) *Div {
	d.refinement.SetFlexDirection(dir)
	return d
}

// FlexRow sets the flexbox layout direction to row.
func (d *Div) FlexRow() *Div {
	d.refinement.SetFlexDirection(style.FlexDirectionRow)
	return d
}

// FlexCol sets the flexbox layout direction to column.
func (d *Div) FlexCol() *Div {
	d.refinement.SetFlexDirection(style.FlexDirectionColumn)
	return d
}

// FlexWrap sets flex wrap mode.
func (d *Div) FlexWrap(w style.FlexWrap) *Div {
	d.refinement.SetFlexWrap(w)
	return d
}

// FlexBasis sets the initial main-axis size.
func (d *Div) FlexBasis(b style.Length) *Div {
	d.refinement.SetFlexBasis(b)
	return d
}

// FlexGrow sets the flex grow factor.
func (d *Div) FlexGrow(grow float32) *Div {
	d.refinement.SetFlexGrow(grow)
	return d
}

// FlexShrink sets the flex shrink factor.
func (d *Div) FlexShrink(shrink float32) *Div {
	d.refinement.SetFlexShrink(shrink)
	return d
}

// --- Visuals & Cursor ---

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

// BoxShadow sets box shadows on the element.
func (d *Div) BoxShadow(shadows []style.BoxShadow) *Div {
	d.refinement.SetBoxShadow(shadows)
	return d
}

// Cursor sets the pointer cursor style.
func (d *Div) Cursor(c style.CursorStyle) *Div {
	d.refinement.SetMouseCursor(c)
	return d
}

// --- Typography & Text ---

// TextColour sets the text colour.
func (d *Div) TextColour(c colour.Rgba) *Div {
	d.refinement.SetTextColour(c)
	return d
}

// TextColourHsla sets the text colour from an Hsla value.
func (d *Div) TextColourHsla(c colour.Hsla) *Div {
	d.refinement.SetTextColourHsla(c)
	return d
}

// FontFamily sets the font family name.
func (d *Div) FontFamily(family string) *Div {
	d.refinement.SetFontFamily(family)
	return d
}

// FontFeatures sets OpenType font features.
func (d *Div) FontFeatures(features []text.FontFeature) *Div {
	d.refinement.SetFontFeatures(features)
	return d
}

// FontFallbacks sets fallback font families.
func (d *Div) FontFallbacks(fallbacks []string) *Div {
	d.refinement.SetFontFallbacks(fallbacks)
	return d
}

// FontSize sets the font size in logical pixels.
func (d *Div) FontSize(size geometry.Pixels) *Div {
	d.refinement.SetFontSize(size)
	return d
}

// LineHeight sets the line height in logical pixels.
func (d *Div) LineHeight(height geometry.Pixels) *Div {
	d.refinement.SetLineHeight(height)
	return d
}

// FontWeight sets font weight.
func (d *Div) FontWeight(weight text.Weight) *Div {
	d.refinement.SetFontWeight(weight)
	return d
}

// FontStyle sets font style (e.g. italic).
func (d *Div) FontStyle(s text.Style) *Div {
	d.refinement.SetFontStyle(s)
	return d
}

// TextBackgroundColour sets highlight colour behind text.
func (d *Div) TextBackgroundColour(c colour.Rgba) *Div {
	d.refinement.SetTextBackgroundColour(c)
	return d
}

// TextBackgroundColourHsla sets text highlight colour from an Hsla value.
func (d *Div) TextBackgroundColourHsla(c colour.Hsla) *Div {
	d.refinement.SetTextBackgroundColourHsla(c)
	return d
}

// Underline sets underline style.
func (d *Div) Underline(u style.UnderlineStyle) *Div {
	d.refinement.SetUnderline(u)
	return d
}

// ClearUnderline removes underline styling.
func (d *Div) ClearUnderline() *Div {
	d.refinement.ClearUnderline()
	return d
}

// Strikethrough sets strikethrough line style.
func (d *Div) Strikethrough(s style.StrikethroughStyle) *Div {
	d.refinement.SetStrikethrough(s)
	return d
}

// ClearStrikethrough removes strikethrough styling.
func (d *Div) ClearStrikethrough() *Div {
	d.refinement.ClearStrikethrough()
	return d
}

// --- Lifecycle ---

// RequestLayout requests layout for all children, converts the resolved style
// to layout inputs, and adds this element to the layout tree.
func (d *Div) RequestLayout(f Frame) NodeID {
	if d.phase != phaseInitial {
		panic("element: RequestLayout called out of order or multiple times")
	}
	d.phase = phaseLayoutRequested

	f.PushTextStyle(d.refinement)
	d.childLayoutIDs = d.childLayoutIDs[:0]
	for _, child := range d.children {
		childID := child.RequestLayout(f)
		d.childLayoutIDs = append(d.childLayoutIDs, childID)
	}
	f.PopTextStyle()

	st := style.Default()
	st.Refine(d.refinement)
	d.resolvedStyle = st

	rem := f.RemSize()
	layoutStyle := st.ToLayout(rem)
	d.layoutID = f.RequestLayout(layoutStyle, d.childLayoutIDs)
	return d.layoutID
}

// Prepaint commits computed bounds, registers input dispatch nodes and hit regions,
// and prepaints children.
func (d *Div) Prepaint(f Frame, bounds geometry.Bounds[geometry.Pixels]) {
	if d.phase != phaseLayoutRequested {
		panic("element: Prepaint called before RequestLayout or out of order")
	}
	d.phase = phasePrepainted
	d.bounds = bounds

	st := d.resolvedStyle
	if st.Display == style.DisplayNone {
		return
	}

	hasDispatch := d.interactivity.hasDispatchNode() || st.MouseCursor != style.CursorDefault
	if hasDispatch {
		node := d.interactivity.toDispatchNode()
		node.Cursor = st.MouseCursor
		nodeID := f.PushDispatchNode(node)
		d.interactivity.hitRegionID = f.RegisterHitRegion(d.bounds, nodeID)
	}

	clips := st.Overflow.X != style.OverflowVisible || st.Overflow.Y != style.OverflowVisible
	if clips {
		f.PushClip(bounds)
	}

	for i, child := range d.children {
		childLayoutID := d.childLayoutIDs[i]
		childBounds := f.LayoutBounds(childLayoutID)
		child.Prepaint(f, childBounds)
	}

	if clips {
		f.PopClip()
	}

	if hasDispatch {
		f.PopDispatchNode()
	}
}

// Paint draws the background and border quad if visible and paints children.
func (d *Div) Paint(f Frame, bounds geometry.Bounds[geometry.Pixels]) {
	if d.phase != phasePrepainted {
		panic("element: Paint called before Prepaint or out of order")
	}
	d.phase = phasePainted
	d.bounds = bounds

	// Layer pseudo-state refinements onto a copy of the resolved style rather
	// than rebuilding it from the Refinement again; st is a value type, so
	// this assignment already copies it and st.Refine below only mutates the
	// copy.
	st := d.resolvedStyle
	activeRefinement := d.refinement
	if d.interactivity.hoverStyle != nil && d.interactivity.hitRegionID != 0 && f.IsHovered(d.interactivity.hitRegionID) {
		st.Refine(*d.interactivity.hoverStyle)
		activeRefinement.MergeFrom(d.interactivity.hoverStyle)
	}
	if d.interactivity.activeStyle != nil && d.interactivity.hitRegionID != 0 && f.IsActive(d.interactivity.hitRegionID) {
		st.Refine(*d.interactivity.activeStyle)
		activeRefinement.MergeFrom(d.interactivity.activeStyle)
	}
	if d.interactivity.focusStyle != nil && d.interactivity.focusID != 0 && f.IsFocused(d.interactivity.focusID) {
		st.Refine(*d.interactivity.focusStyle)
		activeRefinement.MergeFrom(d.interactivity.focusStyle)
	}

	if st.Display == style.DisplayNone {
		return
	}

	// visibility:hidden keeps the subtree's layout (its siblings do not
	// reflow around it) but paints nothing, for it or its children — the
	// difference from Display: none is exactly that reserved space.
	if st.Visibility == style.VisibilityHidden {
		return
	}

	scale := f.ScaleFactor()

	// Drop shadows paint behind the element, so before its background quad;
	// inset shadows layer on top of the fill, so after it. GPUI's window
	// keeps the same split for the same reason (paint_drop_shadows /
	// paint_inset_shadows).
	for _, sh := range st.BoxShadow {
		if !sh.Inset {
			f.InsertShadow(boxShadowPrimitive(bounds, st.CornerRadii, sh, scale))
		}
	}

	if st.Background.A > 0 || st.BorderColour.A > 0 {
		q := scene.Quad{
			Bounds:       scaleBounds(bounds, scale),
			Background:   st.Background,
			BorderColour: st.BorderColour,
			CornerRadii:  scaleCorners(st.CornerRadii, scale),
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

	for _, sh := range st.BoxShadow {
		if sh.Inset {
			f.InsertShadow(boxShadowPrimitive(bounds, st.CornerRadii, sh, scale))
		}
	}

	clips := st.Overflow.X != style.OverflowVisible || st.Overflow.Y != style.OverflowVisible
	if clips {
		f.PushClip(bounds)
	}

	f.PushTextStyle(activeRefinement)
	for i, child := range d.children {
		childLayoutID := d.childLayoutIDs[i]
		childBounds := f.LayoutBounds(childLayoutID)
		child.Paint(f, childBounds)
	}
	f.PopTextStyle()

	if clips {
		f.PopClip()
	}
}
