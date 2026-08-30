package input

// Action is a named, serialisable intent dispatched in response to
// keybindings, menu selections or command palettes.
//
// Actions must be comparable Go values (such as unit structs or structs with
// comparable fields) so they can be compared with == without reflection.
type Action interface {
	ActionName() string
}

// ActionsEqual reports whether two actions are equal using value equality.
func ActionsEqual(a, b Action) bool {
	return a == b
}

// NoAction is a sentinel action that suppresses keybindings for the associated
// keystrokes if it is the highest-precedence match.
type NoAction struct{}

// ActionName returns "NoAction".
func (NoAction) ActionName() string { return "NoAction" }

// IsNoAction reports whether an action represents a suppressed keybinding.
func IsNoAction(a Action) bool {
	if a == nil {
		return false
	}
	_, ok := a.(NoAction)
	return ok
}

// Unbind is a sentinel action that suppresses bindings for a specific action
// name when they dispatch the named action, regardless of that action's
// context.
type Unbind struct {
	TargetAction string
}

// ActionName returns "Unbind".
func (u Unbind) ActionName() string { return "Unbind" }

// IsUnbind reports whether an action represents an unbind marker.
func IsUnbind(a Action) bool {
	if a == nil {
		return false
	}
	_, ok := a.(Unbind)
	return ok
}
