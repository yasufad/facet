# window: focus and the cursor work. One ordering note, then retire.

Both seams are joined and all three tests can fail. I broke the pointer-down focus
call and `TestPointerDownFocusAndStyling` and `TestFocusDroppedWhenElementLeavesTree`
both failed; I removed the cursor comparison so `SetCursor` fires unconditionally and
`TestCursorTransitionsAndDeduplication` failed. Those are real tests.

Dropping focus at present when the focused ID is not in the new frame is the right
answer and the right place for it — a focus ID surviving its element is the failure
mode that produces keystrokes going nowhere, and it would have taken weeks to find.

## The ordering, which you half got right

You implemented `RequestFocus` on `*Window` before `element` declared it on `Frame`.
That is the rule, followed deliberately, and it is the first time.

But the same change carried `DispatchNode.Cursor`, a field `element` owns and `window`
reads, and that one runs the other way. Two commits do not build:

    2278735  window\frame.go:111:10: node.Cursor undefined
    f375e85  window\frame.go:111:10: node.Cursor undefined

The rule as written only covered interface methods. I have extended it: data flows the
opposite way to behaviour, so a field the lower package reads has to be declared
first, and a change containing both directions is two commits rather than one. The
question to ask is which side is waiting on the other **per name**, not per change.

Nothing to fix in the code. Read the amended entry in `AGENTS.md` before the next one.

## Then retire it

`docs/packages.md` already has the focus lifecycle and the cursor deduplication. Check
it also records:

- `RequestFocus` is on `Frame`, and is illegal inside a measure callback, since that
  is now a third rule about `phaseLayoutSolve` and they should be in one place.
- Pointer down focuses the hit region's focus ID and blurs when the background is
  clicked. That is a behaviour a widget author will rely on and will not find in
  `input`, which owns the focus tree but not this policy.

Then the README row and delete this file.

## What is not yours

Tab order. `docs/packages.md` assigns it to `element` because it needs tree order, and
`element` is retired pending exactly this. Say in your report that pointer focus works
and keyboard focus movement does not exist, so it gets picked up rather than assumed.

## Worth carrying

Both of these were seams designed on both sides and never joined, and both were found
by the first widget rather than by any test in the packages that owned them. `ui`
existing is now doing more for correctness than another round of unit tests would.
That is an argument for building the next few widgets sooner rather than deepening
what is here.
