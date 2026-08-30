package input

import (
	"slices"

	"github.com/yasufad/facet/platform"
)

// DispatchPhase identifies whether an event is in the capture phase (traveling
// from root down to target) or the bubble phase (traveling from target up to root).
type DispatchPhase int

const (
	// Capture travels from the root node down toward the target node.
	Capture DispatchPhase = iota

	// Bubble travels from the target node upward toward the root node.
	Bubble
)

// ActionHandler is invoked when a resolved Action reaches a node during dispatch.
// Returning true marks the action handled and halts further propagation.
type ActionHandler func(action Action, phase DispatchPhase) bool

// KeyEventHandler receives raw key events during dispatch. Returning true marks
// the event handled and halts propagation.
type KeyEventHandler func(event platform.KeyEvent, phase DispatchPhase) bool

// PointerEventHandler receives pointer movement and button events. Returning true
// marks the event handled and halts bubbling.
type PointerEventHandler func(event platform.PointerEvent, phase DispatchPhase) bool

// WheelEventHandler receives scroll-wheel and trackpad events. Returning true
// marks the event handled and halts bubbling.
type WheelEventHandler func(event platform.WheelEvent, phase DispatchPhase) bool

// TextEventHandler receives finalized text input directed at the focused node.
// Returning true marks the text input handled.
type TextEventHandler func(event platform.TextEvent) bool

// DispatchNodeID identifies a node within the DispatchTree.
type DispatchNodeID int

type actionListener struct {
	actionName string
	handler    ActionHandler
}

type dispatchNode struct {
	parent           DispatchNodeID
	focusID          FocusID
	context          *KeyContext
	actionListeners  []actionListener
	keyListeners     []KeyEventHandler
	pointerListeners []PointerEventHandler
	wheelListeners   []WheelEventHandler
	textListeners    []TextEventHandler
}

// DispatchResult reports the outcome of a key event dispatch.
type DispatchResult struct {
	ActionDispatched Action
	Handled          bool
	Pending          bool
	ReplayedKeys     []Keystroke
}

// DispatchTree manages the hierarchy of dispatch nodes, maps raw platform events
// through keymaps and focus paths, and delivers actions and input events to
// registered handlers.
type DispatchTree struct {
	keymap       *Keymap
	focusTree    *FocusTree
	nodes        []dispatchNode
	nodeStack    []DispatchNodeID
	focusNodeMap map[FocusID]DispatchNodeID
	pendingKeys  []Keystroke
}

// NewDispatchTree constructs a DispatchTree wired to the given Keymap and FocusTree.
func NewDispatchTree(keymap *Keymap, focusTree *FocusTree) *DispatchTree {
	if keymap == nil {
		keymap = NewKeymap()
	}
	if focusTree == nil {
		focusTree = NewFocusTree()
	}
	return &DispatchTree{
		keymap:       keymap,
		focusTree:    focusTree,
		focusNodeMap: make(map[FocusID]DispatchNodeID),
	}
}

// Clear resets the dispatch tree nodes while retaining the keymap, focus tree,
// and pending keystroke state.
func (d *DispatchTree) Clear() {
	d.nodes = d.nodes[:0]
	d.nodeStack = d.nodeStack[:0]
	clear(d.focusNodeMap)
}

// PushNode creates a new dispatch node as a child of the currently active node
// (or as a root node if the stack is empty) and pushes it onto the node stack.
func (d *DispatchTree) PushNode() DispatchNodeID {
	parent := DispatchNodeID(-1)
	if len(d.nodeStack) > 0 {
		parent = d.nodeStack[len(d.nodeStack)-1]
	}

	id := DispatchNodeID(len(d.nodes))
	d.nodes = append(d.nodes, dispatchNode{
		parent: parent,
	})
	d.nodeStack = append(d.nodeStack, id)
	return id
}

// PopNode pops the top dispatch node from the active node stack.
func (d *DispatchTree) PopNode() {
	if len(d.nodeStack) > 0 {
		d.nodeStack = d.nodeStack[:len(d.nodeStack)-1]
	}
}

// SetContext associates a KeyContext with the currently active node on the stack.
func (d *DispatchTree) SetContext(ctx KeyContext) {
	if len(d.nodeStack) == 0 {
		return
	}
	id := d.nodeStack[len(d.nodeStack)-1]
	d.nodes[id].context = &ctx
}

// SetFocusID associates a FocusID with the currently active node on the stack.
func (d *DispatchTree) SetFocusID(id FocusID) {
	if len(d.nodeStack) == 0 || id == 0 {
		return
	}
	nodeID := d.nodeStack[len(d.nodeStack)-1]
	d.nodes[nodeID].focusID = id
	d.focusNodeMap[id] = nodeID
}

// OnAction registers an action listener on the currently active node. If actionName
// is non-empty, handler only receives actions whose ActionName matches. If
// actionName is empty, handler receives all actions.
func (d *DispatchTree) OnAction(actionName string, handler ActionHandler) {
	if len(d.nodeStack) == 0 || handler == nil {
		return
	}
	id := d.nodeStack[len(d.nodeStack)-1]
	d.nodes[id].actionListeners = append(d.nodes[id].actionListeners, actionListener{
		actionName: actionName,
		handler:    handler,
	})
}

// OnKeyEvent registers a raw key event listener on the active node.
func (d *DispatchTree) OnKeyEvent(handler KeyEventHandler) {
	if len(d.nodeStack) == 0 || handler == nil {
		return
	}
	id := d.nodeStack[len(d.nodeStack)-1]
	d.nodes[id].keyListeners = append(d.nodes[id].keyListeners, handler)
}

// OnPointerEvent registers a pointer event listener on the active node.
func (d *DispatchTree) OnPointerEvent(handler PointerEventHandler) {
	if len(d.nodeStack) == 0 || handler == nil {
		return
	}
	id := d.nodeStack[len(d.nodeStack)-1]
	d.nodes[id].pointerListeners = append(d.nodes[id].pointerListeners, handler)
}

// OnWheelEvent registers a scroll-wheel event listener on the active node.
func (d *DispatchTree) OnWheelEvent(handler WheelEventHandler) {
	if len(d.nodeStack) == 0 || handler == nil {
		return
	}
	id := d.nodeStack[len(d.nodeStack)-1]
	d.nodes[id].wheelListeners = append(d.nodes[id].wheelListeners, handler)
}

// OnTextEvent registers a text event listener on the active node.
func (d *DispatchTree) OnTextEvent(handler TextEventHandler) {
	if len(d.nodeStack) == 0 || handler == nil {
		return
	}
	id := d.nodeStack[len(d.nodeStack)-1]
	d.nodes[id].textListeners = append(d.nodes[id].textListeners, handler)
}

// PendingKeystrokes returns a copy of the in-progress multi-key sequence.
func (d *DispatchTree) PendingKeystrokes() []Keystroke {
	if len(d.pendingKeys) == 0 {
		return nil
	}
	res := make([]Keystroke, len(d.pendingKeys))
	copy(res, d.pendingKeys)
	return res
}

// FlushPending clears any in-progress multi-key chord sequence and returns the
// buffered keystrokes.
func (d *DispatchTree) FlushPending() []Keystroke {
	keys := d.pendingKeys
	d.pendingKeys = nil
	return keys
}

// DispatchKey dispatches a raw platform.KeyEvent through the focus path.
//
// If the keystroke matches an action binding in the active context stack, the
// action is delivered to action handlers on the focus path during Capture
// (root to leaf) and Bubble (leaf to root) phases.
//
// If no action is bound, raw key listeners are notified.
func (d *DispatchTree) DispatchKey(event platform.KeyEvent) DispatchResult {
	if event.Phase != platform.KeyDown && event.Phase != platform.KeyRepeat {
		// Non-down events trigger raw key listeners only
		handled := d.dispatchRawKey(event)
		return DispatchResult{Handled: handled}
	}

	ks := KeystrokeFromEvent(event)
	targetNode := d.resolveTargetNode()
	path := d.nodePath(targetNode)
	contexts := d.contextStackForPath(path)

	d.pendingKeys = append(d.pendingKeys, ks)
	matches, hasPending := d.keymap.BindingsForInput(d.pendingKeys, contexts)

	if hasPending {
		return DispatchResult{Pending: true}
	}

	if len(matches) > 0 {
		d.pendingKeys = nil
		winning := matches[0]
		handled := d.dispatchAction(winning.Action, path)
		return DispatchResult{
			ActionDispatched: winning.Action,
			Handled:          handled,
		}
	}

	// No match with current sequence
	if len(d.pendingKeys) == 1 {
		d.pendingKeys = nil
		handled := d.dispatchRawKeyOnPath(event, path)
		return DispatchResult{Handled: handled}
	}

	// Multi-key prefix failed. Replay current key independently.
	replayed := d.pendingKeys[:len(d.pendingKeys)-1]
	d.pendingKeys = []Keystroke{ks}
	matches, hasPending = d.keymap.BindingsForInput(d.pendingKeys, contexts)

	if hasPending {
		return DispatchResult{
			Pending:      true,
			ReplayedKeys: replayed,
		}
	}

	if len(matches) > 0 {
		d.pendingKeys = nil
		winning := matches[0]
		handled := d.dispatchAction(winning.Action, path)
		return DispatchResult{
			ActionDispatched: winning.Action,
			Handled:          handled,
			ReplayedKeys:     replayed,
		}
	}

	d.pendingKeys = nil
	handled := d.dispatchRawKeyOnPath(event, path)
	return DispatchResult{
		Handled:      handled,
		ReplayedKeys: replayed,
	}
}

// DispatchPointer dispatches a pointer event to targetNode, bubbling along
// the ancestor path from target up to root.
func (d *DispatchTree) DispatchPointer(event platform.PointerEvent, targetNode DispatchNodeID) bool {
	path := d.nodePath(targetNode)
	if len(path) == 0 {
		return false
	}

	// Capture phase: root -> target
	for i := 0; i < len(path); i++ {
		node := &d.nodes[path[i]]
		for _, l := range node.pointerListeners {
			if l(event, Capture) {
				return true
			}
		}
	}

	// Bubble phase: target -> root
	for i := len(path) - 1; i >= 0; i-- {
		node := &d.nodes[path[i]]
		for _, l := range node.pointerListeners {
			if l(event, Bubble) {
				return true
			}
		}
	}

	return false
}

// DispatchWheel dispatches a scroll wheel event to targetNode, bubbling along
// the ancestor path.
func (d *DispatchTree) DispatchWheel(event platform.WheelEvent, targetNode DispatchNodeID) bool {
	path := d.nodePath(targetNode)
	if len(path) == 0 {
		return false
	}

	// Capture phase
	for i := 0; i < len(path); i++ {
		node := &d.nodes[path[i]]
		for _, l := range node.wheelListeners {
			if l(event, Capture) {
				return true
			}
		}
	}

	// Bubble phase
	for i := len(path) - 1; i >= 0; i-- {
		node := &d.nodes[path[i]]
		for _, l := range node.wheelListeners {
			if l(event, Bubble) {
				return true
			}
		}
	}

	return false
}

// DispatchText delivers finalized text to text listeners on the focused node.
func (d *DispatchTree) DispatchText(event platform.TextEvent) bool {
	targetNode := d.resolveTargetNode()
	if targetNode < 0 || int(targetNode) >= len(d.nodes) {
		return false
	}

	node := &d.nodes[targetNode]
	for _, l := range node.textListeners {
		if l(event) {
			return true
		}
	}
	return false
}

// ExplainKey builds a full diagnostic DispatchExplanation for event in the
// current tree and focus state.
func (d *DispatchTree) ExplainKey(event platform.KeyEvent) DispatchExplanation {
	ks := KeystrokeFromEvent(event)
	targetNode := d.resolveTargetNode()
	path := d.nodePath(targetNode)
	contexts := d.contextStackForPath(path)

	cands := d.keymap.Explain([]Keystroke{ks}, contexts)
	var winning *KeyBinding
	for i := range cands {
		if cands[i].Winner {
			winning = &cands[i].Binding
			break
		}
	}

	handlerFound := false
	if winning != nil {
		name := winning.Action.ActionName()
		for _, nodeID := range path {
			for _, l := range d.nodes[nodeID].actionListeners {
				if l.actionName == "" || l.actionName == name {
					handlerFound = true
					break
				}
			}
			if handlerFound {
				break
			}
		}
	}

	return DispatchExplanation{
		Event:          event,
		Keystroke:      ks,
		ContextStack:   contexts,
		Candidates:     cands,
		WinningBinding: winning,
		TargetNode:     targetNode,
		HandlerFound:   handlerFound,
	}
}

func (d *DispatchTree) dispatchAction(action Action, path []DispatchNodeID) bool {
	name := action.ActionName()

	// Capture phase: root -> target
	for i := 0; i < len(path); i++ {
		node := &d.nodes[path[i]]
		for _, l := range node.actionListeners {
			if l.actionName == "" || l.actionName == name {
				if l.handler(action, Capture) {
					return true
				}
			}
		}
	}

	// Bubble phase: target -> root
	for i := len(path) - 1; i >= 0; i-- {
		node := &d.nodes[path[i]]
		for _, l := range node.actionListeners {
			if l.actionName == "" || l.actionName == name {
				if l.handler(action, Bubble) {
					return true
				}
			}
		}
	}

	return false
}

func (d *DispatchTree) dispatchRawKey(event platform.KeyEvent) bool {
	targetNode := d.resolveTargetNode()
	path := d.nodePath(targetNode)
	return d.dispatchRawKeyOnPath(event, path)
}

func (d *DispatchTree) dispatchRawKeyOnPath(event platform.KeyEvent, path []DispatchNodeID) bool {
	// Capture phase
	for i := 0; i < len(path); i++ {
		node := &d.nodes[path[i]]
		for _, l := range node.keyListeners {
			if l(event, Capture) {
				return true
			}
		}
	}

	// Bubble phase
	for i := len(path) - 1; i >= 0; i-- {
		node := &d.nodes[path[i]]
		for _, l := range node.keyListeners {
			if l(event, Bubble) {
				return true
			}
		}
	}

	return false
}

func (d *DispatchTree) resolveTargetNode() DispatchNodeID {
	if len(d.nodes) == 0 {
		return -1
	}

	if focused, ok := d.focusTree.Focused(); ok {
		if nodeID, exists := d.focusNodeMap[focused]; exists {
			return nodeID
		}
	}

	// Default to deepest leaf in tree if nothing focused, or root node (0)
	if len(d.nodes) > 0 {
		return 0
	}
	return -1
}

func (d *DispatchTree) nodePath(target DispatchNodeID) []DispatchNodeID {
	if target < 0 || int(target) >= len(d.nodes) {
		return nil
	}

	var path []DispatchNodeID
	curr := target
	visited := make(map[DispatchNodeID]bool)

	for curr >= 0 && int(curr) < len(d.nodes) && !visited[curr] {
		visited[curr] = true
		path = append(path, curr)
		curr = d.nodes[curr].parent
	}

	slices.Reverse(path)
	return path
}

func (d *DispatchTree) contextStackForPath(path []DispatchNodeID) []KeyContext {
	var stack []KeyContext
	for _, nodeID := range path {
		if d.nodes[nodeID].context != nil {
			stack = append(stack, *d.nodes[nodeID].context)
		}
	}
	return stack
}
