# element: clipping and tab order

Two pieces of work, independent of each other. Tab order can start now; clipping waits
on `window`.

## 1 — Tab order

`docs/packages.md` has assigned this here since before any of it existed, because tab
order needs tree order and `input` only knows the focus hierarchy. A focus parent is
not the previous sibling in layout, which is exactly why this could not live there.

Pointer focus now works: clicking a focusable element focuses it, clicking the
background blurs. What does not exist is any way to move focus from the keyboard, so
a form cannot be filled in without a mouse.

What is needed:

- An element can declare that it takes part in tab order, and in what position.
  Declaration order in the tree is the default and is usually right; an explicit index
  is the escape hatch, and GPUI's `TabStopMap` is worth reading before deciding
  whether we need one.
- Tab moves to the next participant in tree order, Shift-Tab to the previous, and
  both wrap.
- A focusable element that is not rendered this frame is not in the order. `window`
  already drops focus when the focused element leaves the tree; the same fact decides
  membership here.

`input` owns the focus tree and dispatch; do not reimplement either. This is about
producing the ordered list and asking `input` to move focus along it.

Decide where the order is collected. Prepaint walks the tree in order and already
registers hit regions and dispatch nodes, so it is the obvious place, but say so in
`doc.go` rather than leaving it implied.

## 2 — Clipping, after window

`Frame` cannot confine children to bounds. `scene` has had `PushClip` since it was
written and `style.Overflow` has had `Hidden`, `Clip` and `Scroll` since the property
list landed. Nothing joins them, so `Div.OverflowHidden()` currently sets a property
that changes nothing.

`window` is implementing `PushClip` and `PopClip` on `*Window` first. When that lands:

- Declare both on `Frame`, paint phase only.
- Make `Div` honour `style.Overflow`: push its bounds as a clip around painting its
  children when overflow is hidden, clipped or scrolling, and pop after.

The push has to be balanced on every path out of `Paint`, including early returns.
`window` will panic under `facet_debug` if the stack is not empty at end of paint,
which is the check that catches this, so make sure your tests run under that tag too.

A test that a child painted outside its parent's bounds carries the intersected mask
is the one that matters. Asserting the child was inserted proves nothing.

## Done when

Tab and Shift-Tab move focus through a tree of focusable elements in tree order, wrap
at both ends, and skip elements not rendered this frame. Tested against
`elementtest.Frame`.

`Div` clips its children when overflow says so, with a test asserting the mask on a
child's primitives rather than their presence.

`docs/packages.md` records how tab order is collected and that `Div` honours overflow.

## Worth carrying

Both of these are capabilities that existed on one side of a boundary for weeks while
the thing above them silently did nothing. `Div.OverflowHidden()` compiles, runs, and
has never had any effect. A setter with no reader is worse than a missing feature,
because nobody thinks to look for it.
