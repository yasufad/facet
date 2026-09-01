# platform: a declared event with no producer, and a cursor on the wrong type

Two items, both small, both blocking something above you. Then a note on what comes next,
so it is not a surprise.

## 1. Produce IMECompositionEvent

`IMECompositionEvent` is declared in `event.go` with a careful doc comment about `WM_IME_*`
on Windows and `NSTextInputClient` on macOS. Nothing produces it. `input_windows.go`
handles no IME message and `window.DispatchEvent` has no case for it, so the type reads as
finished to anyone scanning the package and is a plan written in Go.

Wire it on Windows. The three phases the type already declares map onto the composition
messages: composition starts, the composition string changes, composition ends and the
result arrives as a `TextEvent` — which you already produce, so that half is done.

Read the constants and the string-length semantics at the SDK rather than from this
prompt. `ImmGetCompositionStringW` returns a byte count for a UTF-16 buffer and the
cursor attribute is a separate query, which is exactly the kind of detail that is easy to
get plausibly wrong. The repository has been bitten twice by writing something from memory
that read correctly and was not.

`Cursor` is documented as a rune offset into `Text`, not a UTF-16 offset. Convert, and
test the conversion with a composition containing a character outside the basic plane.

Dead keys on European layouts go through the same client, so a French or German keyboard
producing an accented character is a case worth having in the test even though no IME is
involved.

`window` routes this once you produce it; its prompt says so.

## 2. SetCursor belongs to the window

`Platform.SetCursor(shape)` sets "the pointer shape over the active window". With one
window that works. With two it is ambiguous, and the ambiguity is invisible until someone
opens a second window and the wrong one changes shape.

Move it. `Window.SetCursor(shape Cursor)` — the cursor is a property of the window the
pointer is over, and on Windows it is answered per-window anyway.

`SetCursorVisible` stays on `Platform`. Hiding the pointer is application-wide and moving
it would be a change with no case behind it.

This crosses a layer boundary, so the order matters and it is not the same order as the
IME item. Add `SetCursor` to `platform.Window` first, where it satisfies nothing and
breaks nothing. Then remove it from `Platform` and switch `window.resolveNextHitTest` to
the window method — those two are one commit, because `window` is the only caller and
splitting them leaves `main` unbuildable.

Update `Platform`'s doc comment in the same commit. It currently promises a cursor over
the active window, and that sentence stops being true when the method moves.

## What comes next, and why macOS rather than Linux

The next assignment for this package is the Cocoa backend, and it is worth knowing now
that it is coming before Linux.

Not for reach. For design pressure. `platform` and `render` have only ever had to answer
their questions in the Win32 and D3D11 shape, and an interface with one implementation is
an interface nobody has tested. Cocoa forces the main-thread and run-loop questions that
`mainthread_windows.go` currently answers alone, and Metal's pipeline and command-encoder
model differs enough from D3D11's immediate context that `render.Renderer` may need to
change. Finding that out with one backend written is far cheaper than with three.

The first deliverable will be a window that opens, reports a real `NSWindow*`, and
delivers pointer and key events — no renderer, no drawing. That is enough to prove the
purego path and to find out whether `objc_msgSend` with struct returns is reachable
without cgo. If it is not, that is a decision to record in `docs/architecture.md`, which
already says as much, and the sooner it is recorded the less is built on the assumption.

Nothing to do about it this round. Do not start it inside this one.

## Done when

    go test ./platform
    GOOS=windows go build ./...

pass, with a test that drives a composition through the phases and asserts the rune
offsets, and one that covers a character outside the basic plane.

`go doc ./platform` shows `SetCursor` on `Window` and not on `Platform`, and
`docs/packages.md` records why the split falls where it does.

Report before starting anything in the macOS section.
