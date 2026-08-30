package input

import (
	"fmt"
	"slices"
	"strings"
)

// KeyBinding binds a sequence of keystrokes to an action, optionally scoped
// by a context predicate.
type KeyBinding struct {
	Keystrokes       []Keystroke
	Action           Action
	ContextPredicate ContextPredicate // nil if global
	Meta             any              // user metadata
}

// NewKeyBinding parses the key chord and context string into a KeyBinding.
// Context may be empty for a global binding.
func NewKeyBinding(chord string, action Action, context string) (KeyBinding, error) {
	if action == nil {
		return KeyBinding{}, fmt.Errorf("new keybinding: action cannot be nil")
	}

	seq, err := ParseKeySequence(chord)
	if err != nil {
		return KeyBinding{}, fmt.Errorf("new keybinding %q: %w", chord, err)
	}

	var pred ContextPredicate
	if strings.TrimSpace(context) != "" {
		p, err := ParseContextPredicate(context)
		if err != nil {
			return KeyBinding{}, fmt.Errorf("new keybinding context %q: %w", context, err)
		}
		pred = p
	}

	return KeyBinding{
		Keystrokes:       seq,
		Action:           action,
		ContextPredicate: pred,
	}, nil
}

// MustKeyBinding is like NewKeyBinding but panics if parsing fails.
func MustKeyBinding(chord string, action Action, context string) KeyBinding {
	b, err := NewKeyBinding(chord, action, context)
	if err != nil {
		panic(err)
	}
	return b
}

// MatchesKeystrokes checks if the given typed keystrokes match this binding.
// It returns matches=true if typed is a prefix of or equal to this binding's
// keystrokes. isPending is true if typed is a proper prefix.
func (b KeyBinding) MatchesKeystrokes(typed []Keystroke) (matches bool, isPending bool) {
	if len(typed) > len(b.Keystrokes) || len(typed) == 0 {
		return false, false
	}

	for i := 0; i < len(typed); i++ {
		if !typed[i].Matches(b.Keystrokes[i]) {
			return false, false
		}
	}

	return true, len(b.Keystrokes) > len(typed)
}

// KeystrokesString returns the space-separated string representation of the
// keystrokes.
func (b KeyBinding) KeystrokesString() string {
	var parts []string
	for _, k := range b.Keystrokes {
		parts = append(parts, k.String())
	}
	return strings.Join(parts, " ")
}

// Keymap maintains a collection of keybindings and resolves typed inputs
// against active context stacks according to deterministic precedence rules.
type Keymap struct {
	bindings []KeyBinding
	version  uint64
}

// NewKeymap creates a Keymap initialised with the given bindings.
func NewKeymap(bindings ...KeyBinding) *Keymap {
	km := &Keymap{}
	km.AddBindings(bindings...)
	return km
}

// Version returns the current keymap revision counter, incremented whenever
// bindings are added or cleared.
func (km *Keymap) Version() uint64 {
	return km.version
}

// Bindings returns a copy of all bindings in the keymap in registration order.
func (km *Keymap) Bindings() []KeyBinding {
	if len(km.bindings) == 0 {
		return nil
	}
	res := make([]KeyBinding, len(km.bindings))
	copy(res, km.bindings)
	return res
}

// AddBindings appends bindings to the keymap.
func (km *Keymap) AddBindings(bindings ...KeyBinding) {
	if len(bindings) == 0 {
		return
	}
	km.bindings = append(km.bindings, bindings...)
	km.version++
}

// Clear removes all bindings from the keymap.
func (km *Keymap) Clear() {
	km.bindings = nil
	km.version++
}

type matchedBinding struct {
	depth   int
	index   int
	binding KeyBinding
}

// BindingsForInput evaluates the typed keystrokes against the keymap in the
// given context stack.
//
// Precedence resolution follows:
//  1. Context Depth: Bindings matching a deeper context in the context stack
//     take precedence over shallower contexts. Global bindings match at the
//     deepest active depth (len(contextStack)).
//  2. Load Order: Bindings added later take precedence at equal depth.
//  3. NoAction Suppression: A NoAction binding suppresses equal or
//     lower-precedence bindings on the same chord.
//  4. Unbind Suppression: An Unbind binding specifically suppresses bindings
//     for the named target action.
//
// It returns all winning bindings in precedence order, and hasPending=true if
// there are longer multi-key bindings that could match if more keystrokes arrive.
func (km *Keymap) BindingsForInput(input []Keystroke, contextStack []KeyContext) (matches []KeyBinding, hasPending bool) {
	if len(input) == 0 {
		return nil, false
	}

	var exactMatches []matchedBinding
	var pendingMatches []matchedBinding

	for i := len(km.bindings) - 1; i >= 0; i-- {
		b := km.bindings[i]
		depth, ok := km.bindingEnabled(b, contextStack)
		if !ok {
			continue
		}

		match, isPending := b.MatchesKeystrokes(input)
		if !match {
			continue
		}

		if !isPending {
			exactMatches = append(exactMatches, matchedBinding{
				depth:   depth,
				index:   i,
				binding: b,
			})
		} else {
			pendingMatches = append(pendingMatches, matchedBinding{
				depth:   depth,
				index:   i,
				binding: b,
			})
		}
	}

	slices.SortStableFunc(exactMatches, compareMatchedBindings)

	var activeBindings []KeyBinding
	noActionDepth := -1
	var unboundActions []string

	for _, m := range exactMatches {
		if IsNoAction(m.binding.Action) {
			if noActionDepth == -1 || m.depth > noActionDepth {
				noActionDepth = m.depth
			}
			continue
		}

		if noActionDepth != -1 && m.depth <= noActionDepth {
			continue
		}

		if IsUnbind(m.binding.Action) {
			unbound := m.binding.Action.(Unbind).TargetAction
			if !slices.Contains(unboundActions, unbound) {
				unboundActions = append(unboundActions, unbound)
			}
			continue
		}

		if slices.Contains(unboundActions, m.binding.Action.ActionName()) {
			continue
		}

		activeBindings = append(activeBindings, m.binding)
	}

	// Filter pending matches for suppression
	activePendingCount := 0
	for _, p := range pendingMatches {
		if IsNoAction(p.binding.Action) || IsUnbind(p.binding.Action) {
			continue
		}
		activePendingCount++
	}

	return activeBindings, activePendingCount > 0
}

// BindingsForAction returns all bindings for the given action in the current
// context stack, in registration order. Only bindings that are not shadowed by
// higher-precedence bindings are included.
func (km *Keymap) BindingsForAction(action Action, contextStack []KeyContext) []KeyBinding {
	if action == nil {
		return nil
	}

	var res []KeyBinding
	for _, b := range km.bindings {
		if !ActionsEqual(b.Action, action) {
			continue
		}

		// Verify this binding is the winning match for its keystrokes in this context
		matches, _ := km.BindingsForInput(b.Keystrokes, contextStack)
		if len(matches) > 0 && ActionsEqual(matches[0].Action, action) {
			res = append(res, b)
		}
	}
	return res
}

// HighestPrecedenceBindingForAction returns the highest-precedence binding for
// action in the given context stack, or false if none is active.
func (km *Keymap) HighestPrecedenceBindingForAction(action Action, contextStack []KeyContext) (KeyBinding, bool) {
	bindings := km.BindingsForAction(action, contextStack)
	if len(bindings) == 0 {
		return KeyBinding{}, false
	}
	// The last matching binding is the highest precedence (latest registration)
	return bindings[len(bindings)-1], true
}

// PossibleNextBindingsForInput returns all multi-key bindings that could follow
// the currently typed input sequence in the given context stack.
func (km *Keymap) PossibleNextBindingsForInput(input []Keystroke, contextStack []KeyContext) []KeyBinding {
	var next []matchedBinding
	for i := len(km.bindings) - 1; i >= 0; i-- {
		b := km.bindings[i]
		depth, ok := km.bindingEnabled(b, contextStack)
		if !ok {
			continue
		}
		match, isPending := b.MatchesKeystrokes(input)
		if match && isPending && !IsNoAction(b.Action) && !IsUnbind(b.Action) {
			next = append(next, matchedBinding{
				depth:   depth,
				index:   i,
				binding: b,
			})
		}
	}

	slices.SortStableFunc(next, compareMatchedBindings)

	res := make([]KeyBinding, len(next))
	for i, m := range next {
		res[i] = m.binding
	}
	return res
}

func compareMatchedBindings(a, b matchedBinding) int {
	if a.depth != b.depth {
		if a.depth > b.depth {
			return -1
		}
		return 1
	}
	if a.index > b.index {
		return -1
	}
	if a.index < b.index {
		return 1
	}
	return 0
}

func (km *Keymap) bindingEnabled(b KeyBinding, contextStack []KeyContext) (int, bool) {
	if b.ContextPredicate == nil {
		return len(contextStack), true
	}
	return b.ContextPredicate.Match(contextStack)
}
