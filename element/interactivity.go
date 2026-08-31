package element

import (
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/style"
)

// Interactivity manages event listeners, action bindings, focus tracking,
// key context, and pseudo-state style overrides for an element.
type Interactivity struct {
	keyContext       *input.KeyContext
	focusID          input.FocusID
	actionBindings   []ActionBinding
	keyListeners     []input.KeyEventHandler
	pointerListeners []input.PointerEventHandler
	wheelListeners   []input.WheelEventHandler
	textListeners    []input.TextEventHandler
	clickListeners   []func(event ClickEvent) bool

	hoverStyle   *style.Refinement
	focusStyle   *style.Refinement
	inFocusStyle *style.Refinement
	activeStyle  *style.Refinement

	occlude     bool
	hitRegionID HitRegionID
	tabStop     bool
	tabStopSet  bool
	tabIndex    int
}

// hasDispatchNode reports whether this interactivity configuration requires
// creating an input dispatch node and registering a hit region during prepaint.
func (it *Interactivity) hasDispatchNode() bool {
	return it.occlude ||
		it.keyContext != nil ||
		it.focusID != 0 ||
		len(it.actionBindings) > 0 ||
		len(it.keyListeners) > 0 ||
		len(it.pointerListeners) > 0 ||
		len(it.wheelListeners) > 0 ||
		len(it.textListeners) > 0 ||
		len(it.clickListeners) > 0 ||
		it.hoverStyle != nil ||
		it.focusStyle != nil ||
		it.inFocusStyle != nil ||
		it.activeStyle != nil
}

// toDispatchNode creates a DispatchNode containing all listeners and bindings
// registered on this Interactivity instance.
func (it *Interactivity) toDispatchNode() DispatchNode {
	tabStop := it.tabStop
	if !it.tabStopSet && it.focusID != 0 {
		tabStop = true
	}
	return DispatchNode{
		KeyContext:       it.keyContext,
		FocusID:          it.focusID,
		TabStop:          tabStop,
		TabIndex:         it.tabIndex,
		ActionBindings:   it.actionBindings,
		KeyListeners:     it.keyListeners,
		PointerListeners: it.pointerListeners,
		WheelListeners:   it.wheelListeners,
		TextListeners:    it.textListeners,
		ClickListeners:   it.clickListeners,
	}
}
