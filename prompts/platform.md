# platform: finish the Windows backend before opening a second one

The `SetCursor` move is verified. Six files in one commit is exactly the stated exception —
each is part of one change and the tree builds at every step. I broke
`w.platformWindow.SetCursor` and `TestCursorTransitionsAndDeduplication` failed, so moving
the recording onto the window stub kept the test exercising the real path rather than
merely compiling against it. That is the failure mode when a test double moves, and you
avoided it.

`docs/packages.md` is written: the cursor split with the reason it falls where it does, and
the IME production including the `WM_CHAR` finding and the UTF-16 conversion. I used your
observation that IMM32 reports `0` rather than a sentinel for an empty composition. You
were right to flag the stale `platform.Platform.SetCursor` in the `window` entry and right
not to edit it; fixed in the same commit.

## macOS is not next, and here is the gate

I said last round it was coming. It still is, and not yet.

The argument for starting now is real: `platform.Platform` and `render.Renderer` each have
exactly one implementation, an interface with one implementation is untested, and finding
out that Metal's command-encoder model does not fit `Renderer` is far cheaper with one
backend written than three. That argument does not expire.

What it is losing to: this framework still does not work end to end on the platform it
already has. `examples/button` has never run. The integration test that drives a click
through a real frame has never existed. Opening a second platform before the first one
demonstrably works multiplies an unproven surface, and every defect found in the last five
rounds was at a seam rather than in a backend.

**The gate is `examples/button` running and `internal/integration` passing.** Both are in
`ui`'s hands now and both are close. When they land, macOS is your next assignment and I
will say so unprompted.

Meanwhile there is work here that any real application needs and cannot proceed without.

## 0. The window state proposal: accepted, with one change

Take the enum as proposed. `WindowState` with `Normal`, `Minimized`, `Maximized`,
`Fullscreen`, plus `State()` and `SetState()`. That is the interface.

Your fullscreen analysis is the answer I wanted and it is right. `IsWindowFullScreen`
comparing `GetWindowRect` to the monitor bounds is a guess wearing a query's clothes — I
read it and it is exactly that. Refusing to adopt a vendored function because it will pass
review and be wrong in production is the judgement this repository has had to learn three
times, and you applied it to code you were handed rather than code you wrote.

So: `Minimized` and `Maximized` read `WS_MINIMIZE`/`WS_MAXIMIZE` and cannot drift.
`Fullscreen` needs stored state. Accepted.

**One addition.** A stored flag can go stale — if the window leaves fullscreen by a route
that did not go through `SetState`, the flag lies, and a lying `State()` is worse than
Wails' heuristic because nothing contradicts it. Report `Fullscreen` only when the stored
flag is set **and** the window still carries the style and bounds we applied when we
entered. The flag is our intent; the geometry corroborates it. That keeps the false
positive impossible — we never report fullscreen we did not cause — and makes the stale
case detectable rather than silent. Clear the flag when they disagree.

**One change: minimizing from fullscreen.** You proposed a strict transition — exit
fullscreen, then minimize, so restore always lands on `Normal`. I want the other one, and
the reason is that Win32 already models this.

`WINDOWPLACEMENT` carries a `Flags` field alongside `ShowCmd` and `RcNormalPosition`, and
Windows uses it to remember that a minimized window was maximized before, so restoring
returns it to maximized rather than normal. The platform's own model is therefore *current
state plus restore target*, not a stack and not a strict transition.

Follow it. `State()` returns `Minimized`; restoring returns to `Fullscreen`. The enum
still never represents a combination — the restore target is a separate stored field, and
you need stored fullscreen bookkeeping regardless, so it costs nothing new.

Verify `WPF_RESTORETOMAXIMIZED`'s exact semantics at the SDK rather than from my
description of it. The constant is not bound in `w32` and I am going from the struct, not
from having read the docs — this is precisely the kind of detail this repository has got
plausibly wrong before.

Your claim that a browser restores to `Normal` from a minimized fullscreen is the part I
am least sure of, in either direction. If you check it and Windows applications really do
land on `Normal`, say so and bring the evidence — matching the platform beats matching my
reasoning about the platform.

`SetState(Fullscreen)` on Windows means borderless at monitor bounds. Scoping to what
Windows can answer and leaving macOS's Spaces distinction alone was right; do not invent
an enum case no backend can implement.

## 1. Window state, implementation

Nothing in the interface can minimise, maximise, restore or go fullscreen. Not missing
from the Windows backend — missing from `platform.Window`.

That is invisible until you notice what `Decorated: false` implies. A custom title bar is
the reason to turn decorations off, and an application that draws its own title bar must
draw its own window controls, and those controls have nothing to call.

This is an interface change, so it is a decision and I am taking part of it: **a window
state that can be read and set, not four booleans.** Four independent setters make
illegal combinations representable and force every caller to sequence them correctly —
restore-then-maximise behaves differently from maximise-then-restore and nothing says so.

What I want from you before the implementation: the state enumeration itself, in Win32
terms, and specifically what it costs to *read* the current one honestly. `IsZoomed` and
`IsIconic` are straightforward; fullscreen is not a Win32 window state at all but a style
and monitor-bounds change that has to be remembered and undone. If reading it back
requires storing what we did, say so, because a state you can set and cannot read is a
different interface and I would rather decide that deliberately than discover it.

Propose the type. Then implement it.

## 2. File dialogs

Six shell methods return "not implemented" and four are silent TODOs, while the README
says `platform` is "Windows", which reads as finished to anyone who has not opened the
file.

Take the dialogs first, and open before save. No open dialog means no "Open File…", which
is the first menu item in the application this framework exists to make possible.
`ShowOpenDialog`'s contract is already written, including the threading note about not
blocking the platform thread — only the implementation is missing.

Tray and notifications wait. Menus wait for item 1, because a custom title bar changes
what a menu bar is.

## 3. While you are in there

`third_party/README` records the IMM32 bindings you added. If the dialog work needs more
of the common-item-dialog COM surface than `w32` carries, the same applies: add it there,
record what and why, and check the licence at the upstream `LICENSE` rather than from
memory. Two attributions in this repository have been wrong and both read plausibly.

## Done when

`platform.Window` can report and set its state, with a test that sets each and reads it
back — including fullscreen, which is the one that will not round-trip for free.

`ShowOpenDialog` opens a real dialog and returns a real path, and does not block the
platform thread while it is up.

`docs/packages.md` gets the window-state contract from me once the shape is settled.

Report the state type before implementing it, and do not start macOS.
