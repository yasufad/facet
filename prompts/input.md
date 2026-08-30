# Assignment: input

Build `input`: keymaps, actions, the focus tree, and dispatch from a raw platform
event to the handler that should run.

Read `docs/packages.md` for the `input` and `platform` entries, and
`_upstream/gpui/crates/gpui/docs/key_dispatch.md` — written by GPUI's authors about
exactly this problem, and the most useful thing you will read for it. Then
`key_dispatch.rs` and `action.rs`. Sync the checkouts with `go run ./tools/upstream`
if `_upstream/` is not there.

## What it owns

**Actions.** A named, serialisable intent — `editor::MoveUp`, not a closure. Actions
are what keybindings resolve to and what menus and command palettes invoke later, so
they have to be comparable and nameable at runtime, not just callable.

**Keymaps.** Bindings from a key chord to an action, scoped by context. A binding
that applies only inside a text field, and a different one that applies everywhere,
have to coexist and resolve predictably.

**The focus tree.** Which element has focus, what contains it, and the path from the
focused node to the root. Dispatch walks that path.

**Dispatch.** A key event arrives; the focus path is walked from the innermost
context outward; the first binding that matches wins; the action is delivered. The
same path carries bubbling for mouse events that hit-testing has already resolved.

## The part worth getting right

Context-scoped resolution is the whole difficulty. GPUI's `key_dispatch.md` explains
why: the same chord means different things depending on where focus is, bindings are
declared far from where they fire, and the answer has to be predictable enough that
a user can reason about which binding won and why.

Build the answer to "why did this keystroke do that?" in from the start. A dispatch
tree that cannot explain itself is one nobody can debug, and every framework that
skipped this grew a bad approximation later.

## Boundaries

`input` imports `geometry` and `platform`, and nothing else of ours. It does not
import `element` or `window` — they will call into it.

It takes raw events from `platform` and resolves them against a context stack. It
knows about focus, not about geometry beyond hit-test results handed to it: `input`
does not decide what was clicked, it decides what a click on that thing means.

`platform` already carries what you need — typed events with modifiers, layout-
independent key codes, scroll deltas that keep the distinction between a precise
trackpad and a mouse notch, and IME events. Do not re-derive any of it.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

The layering test passes. Tests cover the cases that make this hard rather than the
easy ones: a chord bound in two contexts resolving to the inner one, a chord bound
only in an outer context still firing from within an inner one, an unbound chord
falling through, and a multi-key chord that is a prefix of another.

Conventional commits, one file per commit, staged by path.

## Two habits from earlier rounds

When a struct of options exists, at least one test passes it empty. Two defects in
`platform` survived their suites because every test configured the field it was
about to check.

Assertions should know the answer. A test that a size "is positive" passed while the
size was wrong by a window frame. Where a contract is written down, the test checks
the contract.

## Decisions from the plan review

The plan is sound and these are the corrections. Everything not mentioned stands.

**Drop `Keystroke.KeyChar`.** `platform.KeyEvent` carries no character on purpose —
text arrives as a separate `TextEvent` so key identity and text input stay
decoupled. GPUI's `Keystroke` has `key_char`; ours has no source for it. If you
conclude it is genuinely needed, that is a `platform` interface change and gets
raised, not assumed.

**Resolve `secondary` with a build tag.** Cmd on macOS, Ctrl elsewhere. `platform`
exposes `Super` but never says which OS it is running on, so put the answer in
`secondary_darwin.go` and `secondary_other.go` — the same shape the rest of the tree
uses for platform variance, and no new dependency.

**Defer `ActionRegistry` and JSON deserialisation.** There is no command palette, no
settings system and no external keymap file to load. Built now it will be designed
against imagined requirements. The `Action` interface and lookup by name are enough
for dispatch; add the registry when something asks for it.

**Leave tab navigation out.** Tab order follows document and layout order, which
this package cannot see. A focus hierarchy is not the same thing: a focus parent is
not necessarily the previous sibling in layout. `element` knows tree order and will
own tab order; `input` owns which node has focus. Drop `TabIndex`, `TabStop` and
`NextTabStop`.

**No `reflect.DeepEqual` for action equality.** It puts reflection on the dispatch
path, and it accommodates actions that should not exist. Require actions to be
comparable Go values so `==` works. An action carrying a slice or a map is carrying
state that belongs somewhere else.

Keep exactly as planned: the explanation type designed in from the start rather than
retrofitted, capture and bubble as distinct phases, the precedence rules written
down as an ordered list before the code, and all four hard cases in the test plan.
