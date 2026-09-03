package ui

import (
	"unicode/utf8"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/style"
)

// Default text field theme colours.
var (
	defaultTextFieldBg          = colour.Rgba{R: 0.16, G: 0.18, B: 0.22, A: 1.0}
	defaultTextFieldBorder      = colour.Rgba{R: 0.28, G: 0.32, B: 0.38, A: 1.0}
	defaultTextFieldText        = colour.Rgba{R: 0.95, G: 0.95, B: 0.95, A: 1.0}
	defaultTextFieldPlaceholder = colour.Rgba{R: 0.50, G: 0.53, B: 0.58, A: 1.0}
	defaultTextFieldCaret       = colour.Rgba{R: 0.30, G: 0.55, B: 0.90, A: 1.0}
	defaultTextFieldSelection   = colour.Rgba{R: 0.20, G: 0.35, B: 0.60, A: 0.6}

	defaultTextFieldPaddingX   = geometry.Pixels(8)
	defaultTextFieldPaddingY   = geometry.Pixels(4)
	defaultTextFieldLineHeight = geometry.Pixels(20)
)

// TextFieldState holds retained text, caret, selection and focus state across frames.
type TextFieldState struct {
	text            string
	cursor          int
	selectionAnchor int
	focusID         input.FocusID
	lastLayout      element.TextLayout
	isDragging      bool
}

// NewTextFieldState constructs a new initialised TextFieldState.
func NewTextFieldState(initialText string) TextFieldState {
	return TextFieldState{
		text:            initialText,
		cursor:          len(initialText),
		selectionAnchor: len(initialText),
		focusID:         input.NewFocusID(),
	}
}

// Text returns the current text content.
func (s *TextFieldState) Text() string {
	return s.text
}

// SetText sets the text content, clamping cursor and selection to valid bounds.
func (s *TextFieldState) SetText(text string) {
	s.text = text
	if s.cursor > len(text) {
		s.cursor = len(text)
	}
	if s.selectionAnchor > len(text) {
		s.selectionAnchor = len(text)
	}
}

// Cursor returns the current caret byte offset.
func (s *TextFieldState) Cursor() int {
	return s.cursor
}

// SetCursor sets the caret byte offset and collapses any active selection.
func (s *TextFieldState) SetCursor(cursor int) {
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(s.text) {
		cursor = len(s.text)
	}
	s.cursor = cursor
	s.selectionAnchor = cursor
}

// Selection returns the start and end byte offsets of the active selection (start <= end).
func (s *TextFieldState) Selection() (start, end int) {
	if s.selectionAnchor < s.cursor {
		return s.selectionAnchor, s.cursor
	}
	return s.cursor, s.selectionAnchor
}

// HasSelection reports whether there is an active non-empty selection.
func (s *TextFieldState) HasSelection() bool {
	return s.selectionAnchor != s.cursor
}

// SelectAll selects the entire text content.
func (s *TextFieldState) SelectAll() {
	s.selectionAnchor = 0
	s.cursor = len(s.text)
}

// ClearSelection collapses the selection to the current caret position.
func (s *TextFieldState) ClearSelection() {
	s.selectionAnchor = s.cursor
}

// FocusID returns the focus identifier for this text field.
func (s *TextFieldState) FocusID() input.FocusID {
	return s.focusID
}

// SetFocusID updates the focus identifier for this text field.
func (s *TextFieldState) SetFocusID(id input.FocusID) {
	s.focusID = id
}

// Layout returns the shaped line layout recorded from the most recent frame pass.
func (s *TextFieldState) Layout() element.TextLayout {
	return s.lastLayout
}

// InsertText inserts text at the caret, replacing any active selection.
func (s *TextFieldState) InsertText(str string) {
	if str == "" {
		return
	}
	start, end := s.Selection()
	if start != end {
		s.text = s.text[:start] + str + s.text[end:]
		s.cursor = start + len(str)
	} else {
		s.text = s.text[:s.cursor] + str + s.text[s.cursor:]
		s.cursor += len(str)
	}
	s.selectionAnchor = s.cursor
}

// DeleteBackward removes the preceding rune (Backspace), or the selected range.
func (s *TextFieldState) DeleteBackward() {
	start, end := s.Selection()
	if start != end {
		s.text = s.text[:start] + s.text[end:]
		s.cursor = start
		s.selectionAnchor = start
		return
	}
	if s.cursor == 0 {
		return
	}
	_, prevRuneSize := utf8.DecodeLastRuneInString(s.text[:s.cursor])
	s.text = s.text[:s.cursor-prevRuneSize] + s.text[s.cursor:]
	s.cursor -= prevRuneSize
	s.selectionAnchor = s.cursor
}

// DeleteForward removes the following rune (Delete), or the selected range.
func (s *TextFieldState) DeleteForward() {
	start, end := s.Selection()
	if start != end {
		s.text = s.text[:start] + s.text[end:]
		s.cursor = start
		s.selectionAnchor = start
		return
	}
	if s.cursor >= len(s.text) {
		return
	}
	_, nextRuneSize := utf8.DecodeRuneInString(s.text[s.cursor:])
	s.text = s.text[:s.cursor] + s.text[s.cursor+nextRuneSize:]
	s.selectionAnchor = s.cursor
}

// MoveLeft moves the caret one rune to the left, optionally extending the selection.
func (s *TextFieldState) MoveLeft(selectRange bool) {
	if !selectRange && s.HasSelection() {
		start, _ := s.Selection()
		s.cursor = start
		s.selectionAnchor = start
		return
	}
	if s.cursor > 0 {
		_, size := utf8.DecodeLastRuneInString(s.text[:s.cursor])
		s.cursor -= size
	}
	if !selectRange {
		s.selectionAnchor = s.cursor
	}
}

// MoveRight moves the caret one rune to the right, optionally extending the selection.
func (s *TextFieldState) MoveRight(selectRange bool) {
	if !selectRange && s.HasSelection() {
		_, end := s.Selection()
		s.cursor = end
		s.selectionAnchor = end
		return
	}
	if s.cursor < len(s.text) {
		_, size := utf8.DecodeRuneInString(s.text[s.cursor:])
		s.cursor += size
	}
	if !selectRange {
		s.selectionAnchor = s.cursor
	}
}

// MoveToStart moves the caret to the start of the text, optionally extending the selection.
func (s *TextFieldState) MoveToStart(selectRange bool) {
	s.cursor = 0
	if !selectRange {
		s.selectionAnchor = 0
	}
}

// MoveToEnd moves the caret to the end of the text, optionally extending the selection.
func (s *TextFieldState) MoveToEnd(selectRange bool) {
	s.cursor = len(s.text)
	if !selectRange {
		s.selectionAnchor = s.cursor
	}
}

// TextField is an interactive single-line text input widget displaying text,
// a caret, and selection highlights.
type TextField struct {
	app         *app.App
	state       app.Entity[TextFieldState]
	placeholder string
	disabled    bool
	paddingX    geometry.Pixels
	paddingY    geometry.Pixels
	lineHeight  geometry.Pixels
	refinement  style.Refinement

	// Ephemeral element tree constructed for lifecycle execution
	container   *element.Div
	textEl      *element.Text
	caretEl     *element.Div
	selectionEl *element.Div
	bounds      geometry.Bounds[geometry.Pixels]
}

// Ensure TextField implements element.Element.
var _ element.Element = (*TextField)(nil)

// NewTextField constructs a new TextField bound to the given App and retained TextFieldState entity.
func NewTextField(a *app.App, state app.Entity[TextFieldState]) *TextField {
	return &TextField{
		app:        a,
		state:      state,
		paddingX:   defaultTextFieldPaddingX,
		paddingY:   defaultTextFieldPaddingY,
		lineHeight: defaultTextFieldLineHeight,
	}
}

// Placeholder sets placeholder text displayed when the text field is empty.
func (t *TextField) Placeholder(placeholder string) *TextField {
	t.placeholder = placeholder
	return t
}

// Disabled sets whether the text field ignores interactions.
func (t *TextField) Disabled(disabled bool) *TextField {
	t.disabled = disabled
	return t
}

// LineHeight sets the line height in logical pixels.
func (t *TextField) LineHeight(height geometry.Pixels) *TextField {
	if height > 0 {
		t.lineHeight = height
	}
	return t
}

// Padding sets the horizontal and vertical content padding in logical pixels.
func (t *TextField) Padding(x, y geometry.Pixels) *TextField {
	t.paddingX = x
	t.paddingY = y
	return t
}

// Refine applies custom style overrides onto the text field container.
func (t *TextField) Refine(r style.Refinement) *TextField {
	t.refinement.MergeFrom(&r)
	return t
}

// buildTree constructs the ephemeral element tree containing container, selection quad,
// text element, and caret div.
func (t *TextField) buildTree() {
	var txt string
	var cursor int
	var selStart, selEnd int
	var hasSelection bool
	var lastLayout element.TextLayout

	if t.app != nil {
		st := t.state.Read(t.app)
		txt = st.text
		cursor = st.cursor
		selStart, selEnd = st.Selection()
		hasSelection = st.HasSelection()
		lastLayout = st.lastLayout
	}

	t.container = element.NewDiv().
		Relative().
		Flex().
		FlexRow().
		AlignItems(style.AlignItemsCentre).
		PaddingLeft(style.Px(t.paddingX)).
		PaddingRight(style.Px(t.paddingX)).
		PaddingTop(style.Px(t.paddingY)).
		PaddingBottom(style.Px(t.paddingY)).
		MinHeight(style.Px(t.lineHeight + t.paddingY*2)).
		Rounded(geometry.Pixels(4)).
		Border(geometry.Pixels(1)).
		Bg(defaultTextFieldBg)

	t.container.Refine(t.refinement)

	// Selection quad (rendered as a Div behind text)
	if hasSelection {
		startX := lastLayout.XForIndex(selStart)
		endX := lastLayout.XForIndex(selEnd)
		selWidth := endX - startX
		if selWidth < 0 {
			selWidth = -selWidth
			startX = endX
		}
		t.selectionEl = element.NewDiv().
			Absolute().
			InsetLeft(style.Px(t.paddingX + startX)).
			InsetTop(style.Px(t.paddingY)).
			Width(style.Px(selWidth)).
			Height(style.Px(t.lineHeight)).
			Bg(defaultTextFieldSelection)
		t.container.Child(t.selectionEl)
	}

	// Text element
	if txt == "" && t.placeholder != "" {
		t.textEl = element.NewText(t.placeholder).TextColour(defaultTextFieldPlaceholder)
	} else {
		t.textEl = element.NewText(txt).TextColour(defaultTextFieldText)
	}
	t.textEl.LineHeight(t.lineHeight)
	t.container.Child(t.textEl)

	// Caret (rendered as a Div in front of text)
	if !t.disabled {
		caretX := lastLayout.XForIndex(cursor)
		t.caretEl = element.NewDiv().
			Absolute().
			InsetLeft(style.Px(t.paddingX + caretX)).
			InsetTop(style.Px(t.paddingY)).
			Width(style.Px(2)).
			Height(style.Px(t.lineHeight)).
			Bg(defaultTextFieldCaret)
		t.container.Child(t.caretEl)
	}
}

// RequestLayout builds the ephemeral element tree and requests layout through Frame.
func (t *TextField) RequestLayout(f element.Frame) element.NodeID {
	t.buildTree()
	return t.container.RequestLayout(f)
}

// Prepaint commits solved bounds, registers hit regions and attaches dispatch node listeners.
func (t *TextField) Prepaint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {
	t.bounds = bounds

	var focusID input.FocusID
	if t.app != nil {
		st := t.state.Read(t.app)
		focusID = st.focusID
	}

	node := element.DispatchNode{
		FocusID:          focusID,
		TabStop:          !t.disabled,
		Cursor:           style.CursorText,
		TextListeners:    []input.TextEventHandler{t.textHandler()},
		KeyListeners:     []input.KeyEventHandler{t.keyHandler()},
		PointerListeners: []input.PointerEventHandler{t.pointerHandler(f)},
		ClickListeners:   []func(element.ClickEvent) bool{t.clickHandler()},
	}
	if t.disabled {
		node.Cursor = style.CursorNotAllowed
		node.TextListeners = nil
		node.KeyListeners = nil
		node.PointerListeners = nil
		node.ClickListeners = nil
	}

	nodeID := f.PushDispatchNode(node)
	f.RegisterHitRegion(bounds, nodeID)

	t.container.Prepaint(f, bounds)
	f.PopDispatchNode()
}

// Paint draws the container, selection quad, text glyphs, and caret div into the frame scene,
// and records the shaped line layout into TextFieldState for subsequent event queries.
func (t *TextField) Paint(f element.Frame, bounds geometry.Bounds[geometry.Pixels]) {
	if t.app != nil && t.textEl != nil {
		layout := t.textEl.Layout()
		t.state.Update(t.app, func(st *TextFieldState, cx *app.Context[TextFieldState]) {
			st.lastLayout = layout
		})
	}
	t.container.Paint(f, bounds)
}

func (t *TextField) textHandler() input.TextEventHandler {
	return func(event input.TextEvent) bool {
		if t.disabled || t.app == nil {
			return false
		}
		t.state.Update(t.app, func(st *TextFieldState, cx *app.Context[TextFieldState]) {
			st.InsertText(event.Text)
			cx.Notify()
		})
		return true
	}
}

func (t *TextField) keyHandler() input.KeyEventHandler {
	return func(event input.KeyEvent, phase input.DispatchPhase) bool {
		if t.disabled || t.app == nil {
			return false
		}
		if phase != input.Bubble {
			return false
		}
		// event.Phase == 1 is KeyUp; we only handle KeyDown and KeyRepeat
		if event.Phase == 1 {
			return false
		}
		ks := input.KeystrokeFromEvent(event)
		handled := false
		t.state.Update(t.app, func(st *TextFieldState, cx *app.Context[TextFieldState]) {
			switch ks.String() {
			case "left":
				st.MoveLeft(false)
				handled = true
			case "shift-left":
				st.MoveLeft(true)
				handled = true
			case "right":
				st.MoveRight(false)
				handled = true
			case "shift-right":
				st.MoveRight(true)
				handled = true
			case "home":
				st.MoveToStart(false)
				handled = true
			case "shift-home":
				st.MoveToStart(true)
				handled = true
			case "end":
				st.MoveToEnd(false)
				handled = true
			case "shift-end":
				st.MoveToEnd(true)
				handled = true
			case "backspace":
				st.DeleteBackward()
				handled = true
			case "delete":
				st.DeleteForward()
				handled = true
			case "ctrl-a", "super-a":
				st.SelectAll()
				handled = true
			}
			if handled {
				cx.Notify()
			}
		})
		return handled
	}
}

func (t *TextField) pointerHandler(f element.Frame) input.PointerEventHandler {
	return func(event input.PointerEvent, phase input.DispatchPhase) bool {
		if t.disabled || t.app == nil {
			return false
		}
		if phase != input.Bubble {
			return false
		}
		scale := f.ScaleFactor()
		if scale <= 0 {
			scale = 1.0
		}
		ptrLogicalX := geometry.Pixels(float32(event.Position.X) / scale)
		localTextX := ptrLogicalX - t.bounds.Origin.X - t.paddingX

		// event.Phase: Move = 0, Down = 1, Up = 2
		switch event.Phase {
		case 1: // PointerDown
			// ButtonLeft is 1
			if event.Button == 1 {
				t.state.Update(t.app, func(st *TextFieldState, cx *app.Context[TextFieldState]) {
					idx := st.lastLayout.ClosestIndexForX(localTextX)
					// Modifiers: Shift is 1
					if event.Modifiers&1 != 0 {
						st.cursor = idx
					} else {
						st.SetCursor(idx)
					}
					st.isDragging = true
					cx.Notify()
				})
				return true
			}
		case 0: // PointerMove
			handled := false
			t.state.Update(t.app, func(st *TextFieldState, cx *app.Context[TextFieldState]) {
				if st.isDragging {
					idx := st.lastLayout.ClosestIndexForX(localTextX)
					st.cursor = idx
					handled = true
					cx.Notify()
				}
			})
			return handled
		case 2: // PointerUp
			handled := false
			t.state.Update(t.app, func(st *TextFieldState, cx *app.Context[TextFieldState]) {
				if st.isDragging {
					st.isDragging = false
					handled = true
					cx.Notify()
				}
			})
			return handled
		}
		return false
	}
}

func (t *TextField) clickHandler() func(element.ClickEvent) bool {
	return func(event element.ClickEvent) bool {
		if t.disabled || t.app == nil {
			return false
		}
		localTextX := event.LocalPosition.X - t.paddingX
		t.state.Update(t.app, func(st *TextFieldState, cx *app.Context[TextFieldState]) {
			idx := st.lastLayout.ClosestIndexForX(localTextX)
			if event.Modifiers.Has(element.ModShift) {
				st.cursor = idx
			} else {
				st.SetCursor(idx)
			}
			cx.Notify()
		})
		return true
	}
}
