package input

import (
	"fmt"
	"slices"
	"strings"

	"github.com/yasufad/facet/platform"
)

// CandidateExplanation explains how a single candidate keybinding was
// evaluated during dispatch.
type CandidateExplanation struct {
	Binding KeyBinding
	Depth   int
	Matched bool
	Winner  bool
	Reason  string
}

// DispatchExplanation provides full diagnostic details explaining why a
// keystroke resolved to a specific action (or fell through) and how it was
// handled.
type DispatchExplanation struct {
	Event          platform.KeyEvent
	Keystroke      Keystroke
	ContextStack   []KeyContext
	Candidates     []CandidateExplanation
	WinningBinding *KeyBinding
	TargetNode     DispatchNodeID
	HandlerFound   bool
	Handled        bool
}

// String returns a human-readable diagnostic report of the dispatch decision.
func (e DispatchExplanation) String() string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "Keystroke: %s (Code: %v, Modifiers: %s)\n",
		e.Keystroke.String(), e.Keystroke.Code, e.Keystroke.Modifiers.String())

	if len(e.ContextStack) == 0 {
		sb.WriteString("Context Stack: (empty)\n")
	} else {
		fmt.Fprintf(&sb, "Context Stack (depth %d):\n", len(e.ContextStack))
		for i, ctx := range e.ContextStack {
			fmt.Fprintf(&sb, "  [%d] %s\n", i, ctx.String())
		}
	}

	if len(e.Candidates) == 0 {
		sb.WriteString("Candidates: (no matching keybindings in keymap)\n")
	} else {
		fmt.Fprintf(&sb, "Candidates (%d evaluated):\n", len(e.Candidates))
		for _, c := range e.Candidates {
			status := "SHADOWED"
			if c.Winner {
				status = "WINNER"
			} else if !c.Matched {
				status = "MISMATCH"
			}
			ctxStr := "global"
			if c.Binding.ContextPredicate != nil {
				ctxStr = c.Binding.ContextPredicate.String()
			}
			fmt.Fprintf(&sb, "  [%s] %s -> %s [%s] (depth %d): %s\n",
				status,
				c.Binding.KeystrokesString(),
				c.Binding.Action.ActionName(),
				ctxStr,
				c.Depth,
				c.Reason,
			)
		}
	}

	sb.WriteString("Result:\n")
	if e.WinningBinding != nil {
		fmt.Fprintf(&sb, "  Winning Action: %s\n", e.WinningBinding.Action.ActionName())
		fmt.Fprintf(&sb, "  Target Node: %d\n", e.TargetNode)
		fmt.Fprintf(&sb, "  Handler Found: %v\n", e.HandlerFound)
		fmt.Fprintf(&sb, "  Handled: %v\n", e.Handled)
	} else {
		sb.WriteString("  No binding resolved (event fell through)\n")
	}

	return sb.String()
}

// Explain evaluates all bindings for the input in contextStack and returns
// detailed explanations for why each candidate succeeded or was rejected.
func (km *Keymap) Explain(input []Keystroke, contextStack []KeyContext) []CandidateExplanation {
	if len(input) == 0 {
		return nil
	}

	type cand struct {
		index   int
		binding KeyBinding
		depth   int
		matched bool
		reason  string
	}

	var cands []cand
	for i, b := range km.bindings {
		match, isPending := b.MatchesKeystrokes(input)
		if !match || isPending {
			continue
		}

		depth, ok := km.bindingEnabled(b, contextStack)
		if !ok {
			cands = append(cands, cand{
				index:   i,
				binding: b,
				depth:   -1,
				matched: false,
				reason:  "context predicate did not match active context stack",
			})
			continue
		}

		cands = append(cands, cand{
			index:   i,
			binding: b,
			depth:   depth,
			matched: true,
		})
	}

	// Sort matching candidates by precedence: deeper context first, then later load order
	var matchingIndices []int
	for i, c := range cands {
		if c.matched {
			matchingIndices = append(matchingIndices, i)
		}
	}

	slices.SortStableFunc(matchingIndices, func(i, j int) int {
		ci := cands[i]
		cj := cands[j]
		if ci.depth != cj.depth {
			if ci.depth > cj.depth {
				return -1
			}
			return 1
		}
		if ci.index > cj.index {
			return -1
		}
		if ci.index < cj.index {
			return 1
		}
		return 0
	})

	noActionDepth := -1
	var unboundActions []string
	winnerFound := false

	for _, idx := range matchingIndices {
		c := &cands[idx]
		if IsNoAction(c.binding.Action) {
			if noActionDepth == -1 || c.depth > noActionDepth {
				noActionDepth = c.depth
			}
			c.reason = "NoAction binding suppresses matching chords"
			continue
		}

		if noActionDepth != -1 && c.depth <= noActionDepth {
			c.reason = fmt.Sprintf("suppressed by NoAction at depth %d", noActionDepth)
			continue
		}

		if IsUnbind(c.binding.Action) {
			unbound := c.binding.Action.(Unbind).TargetAction
			if !slices.Contains(unboundActions, unbound) {
				unboundActions = append(unboundActions, unbound)
			}
			c.reason = fmt.Sprintf("Unbind marker for action %q", unbound)
			continue
		}

		if slices.Contains(unboundActions, c.binding.Action.ActionName()) {
			c.reason = fmt.Sprintf("suppressed by Unbind marker for action %q", c.binding.Action.ActionName())
			continue
		}

		if !winnerFound {
			winnerFound = true
			c.reason = fmt.Sprintf("highest precedence match (depth %d, index %d)", c.depth, c.index)
		} else {
			c.reason = "shadowed by higher-precedence binding"
		}
	}

	exps := make([]CandidateExplanation, len(cands))
	for i, c := range cands {
		isWinner := false
		if len(matchingIndices) > 0 && matchingIndices[0] == i && c.matched && !IsNoAction(c.binding.Action) && !IsUnbind(c.binding.Action) {
			if noActionDepth == -1 || c.depth > noActionDepth {
				if !slices.Contains(unboundActions, c.binding.Action.ActionName()) {
					isWinner = true
				}
			}
		}
		exps[i] = CandidateExplanation{
			Binding: c.binding,
			Depth:   c.depth,
			Matched: c.matched,
			Winner:  isWinner,
			Reason:  c.reason,
		}
	}

	return exps
}
