// Package input provides keymaps, actions, the focus tree, and event dispatch
// from raw platform events to target handlers.
//
// The input layer sits between the low-level platform event stream and the
// higher-level window and element layers. It imports only geometry and
// platform.
//
// # Architecture
//
// Input dispatch in Facet is built around four primary concepts:
//
//   - Actions: Named, serialisable intents (such as "editor::MoveUp") that
//     keybindings resolve to and that menus and command palettes invoke. Actions
//     are comparable Go values.
//   - Keymaps: Collections of keybindings mapping key chords to actions,
//     evaluated against a context stack with deterministic precedence.
//   - Focus Tree: Tracks which node has keyboard focus, its ancestor hierarchy,
//     and the path from the root to the focused node.
//   - Dispatch: Walks the focus path or hit-test path, evaluating keybindings
//     from the innermost context outward, routing actions and bubbling raw
//     pointer/wheel events through capture and bubble phases.
//
// # Context-Scoped Resolution
//
// Keybinding precedence is resolved deterministically using the following
// rules:
//
//  1. Context Depth: Bindings matching a deeper (more specific) context in the
//     active context stack take precedence over shallower contexts. Global
//     bindings with no context predicate match at the deepest active level.
//  2. Load Order: When multiple bindings match at the same context depth,
//     bindings added later (such as user overrides) take precedence.
//  3. Action Suppression: A NoAction binding suppresses out-ranked bindings for
//     the same chord. An Unbind binding specifically suppresses bindings for a
//     named action.
//
// Every dispatch decision can be inspected via DispatchExplanation to answer
// "why did this keystroke do that?".
package input
