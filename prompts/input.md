# input: name the event types you already use

`ui` built a scroll view and could not write a wheel handler without importing
`platform`. The layering test is red on `main` because of it.

The cause is here. Four of your handler types name `platform` in their signatures:

    KeyEventHandler     func(event platform.KeyEvent, phase DispatchPhase) bool
    PointerEventHandler func(event platform.PointerEvent, phase DispatchPhase) bool
    TextEventHandler    func(event platform.TextEvent) bool
    WheelEventHandler   func(event platform.WheelEvent, phase DispatchPhase) bool

Anyone above you writing one of those closures has to spell the parameter type, so
they have to import `platform`. `element` and `ui` are both forbidden it, and rightly:
`platform` also hands out windows, the clipboard, menus and main-thread dispatch, and
a widget has no business reaching any of that.

**Run this first.** `ui` is blocked and the tree is red until it lands.

## What to add

Aliases for the event types your own signatures already name:

    type KeyEvent = platform.KeyEvent
    type PointerEvent = platform.PointerEvent
    type TextEvent = platform.TextEvent
    type WheelEvent = platform.WheelEvent

Aliases, not defined types. The same reasoning as `element.NodeID = layout.NodeID`,
which solved this exact problem for `ui` two rounds ago: the types stay identical, so
`window` keeps passing `platform` values straight through with no conversion, and only
the name a caller has to write changes.

Declaring separate structs here would be worse than the disease. `platform`'s entry in
`docs/packages.md` says hiding a platform's units is the job and hiding what it
measured is not — wheel events deliberately keep the distinction between a trackpad's
exact pixel delta and a mouse notch's inexact line delta, and any re-declaration is a
chance to lose it.

Update the handler signatures to use the aliases so `go doc ./input` reads without
`platform` in it. That is cosmetic, but the doc is what someone reads before deciding
whether they can write a handler.

## Why this package rather than element

`element` declared its own `ClickEvent` because a click is *synthesised* — down and up
on the same target, which no platform reports. That reasoning does not extend to
wheel, key, pointer and text, which are real platform events that `platform` has
already normalised across operating systems.

You own the vocabulary of input above the OS. These belong here.

## Done when

    go test ./internal/layering

is green with `ui` importing `input` and not `platform`.

`go doc ./input` shows handler signatures naming `input` types.

`docs/packages.md` records that `input` names the event vocabulary for everything
above it, and why aliases rather than declarations.

Then retire this prompt.
