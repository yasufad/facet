# window: focus and the cursor, from the first widget

`ui` built a button and found two things that are yours. Both are the difference
between a widget that renders and a widget that behaves.

## 1 — Clicking a button does not focus it

Three halves of this are missing and they are in three packages:

- `DispatchEvent` does not move focus on pointer down. Clicking anything focuses
  nothing.
- `Frame` has `IsFocused(input.FocusID)` and no way to *request* focus, so an element
  cannot take it even deliberately.
- Nothing moves focus by keyboard. Tab and Shift-Tab do nothing.

`input` already owns the focus tree and the dispatch precedence; this is about
connecting it, which is your job by the same argument as every other seam in this
package.

Start with pointer focus and `Frame.RequestFocus(input.FocusID)`, which together make
`:focus` styling real in a running window rather than only in a synthetic test. Tab
order is a separate question and a harder one — `docs/packages.md` says tab order
belongs to `element`, which knows tree order, so do not invent it here. Raise it when
the first two work.

Follow the interface ordering rule: implement `RequestFocus` on `*Window` first, then
`element` adds it to `Frame`. Declaring first stops this package compiling.

Decide and record what focus does when the focused element is not rendered in the next
frame — a button that is focused and then scrolled out of the tree. Dropping focus to
nothing is a defensible answer; leaving a focus ID that no longer corresponds to
anything is not.

## 2 — The cursor never reaches the operating system

`Div.Cursor(style.CursorPointer)` sets a property nothing reads. Step 5 already
resolves which hit region is under the pointer, so the cursor for that region is
known at exactly the point the hit test finishes. Take it to
`platform.Window.SetCursor`.

Set it once per frame when it changes, not on every mouse move: a cursor set on every
motion event is a visible flicker on Windows and a wasted syscall everywhere.

A disabled button asks for `CursorNotAllowed`, so this is already exercised by the
example if you want to see it working.

## Done when

Clicking a button focuses it and its focus styling appears in a running window, not
only under a test frame.

`Frame.RequestFocus` exists here before it exists on the interface.

The cursor changes over a button and over a disabled button, and does not change
while the pointer is still.

`docs/packages.md` records what happens to focus when the focused element leaves the
tree.

## Worth carrying

Both of these are seams that were designed, built on both sides, and never joined.
`Cursor` has been settable since the style property list landed and readable since hit
testing landed, and in between nobody asked what read it. When a package exposes
something nothing consumes, that is worth noticing before the widget library finds it.
