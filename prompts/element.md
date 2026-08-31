# element: clipping works, its test does not

Tab order and clipping are both in, and the plumbing is right. `window`'s
`TestWindowPushClipPrimitiveMask` asserts the child quad's `ContentMask.Bounds`, which
is the assertion that matters, and it fails when I disable the push in `Div.Paint`.
`elementtest.Frame` gained `PushClip`, `PopClip`, `SimulateTab` and `TabOrder`, so the
scroll view can be tested from `ui` without reaching below `element`.

One thing to fix, and it is the same shape as last round.

## `TestDivOverflowClipping` cannot fail

I disabled the clip push in `Div.Paint` entirely. `window`'s test failed.
`element`'s passed.

    // Clip stack must be empty after Paint finishes (push/pop balanced)
    if len(frame.clips) != 0 {

That is a balance check, and an empty stack is trivially balanced. The test passes
whether `Div` clips or not, which is the one thing its name claims to verify.

It needs to assert that a clip was pushed, with the parent's bounds, and popped. The
fake frame records `clips`; record the pushes as events rather than only current depth,
then assert the sequence: one push carrying `rootBounds`, one pop, empty at the end.
Keep the balance check — it is worth having — but it is not the test.

Then disable the push again yourself and confirm both packages fail.

This is the second consecutive round where the package that owns a behaviour had a
test that survived the behaviour being removed, and the consumer caught it instead.
Last time it was hover text colour; `ui` now asserts that. The pattern is worth naming
in your own review before you report: for each thing you added, ask what single line
you would delete to break it, and whether your test notices.

## Also worth a look

`TestDivTabStopAndTabIndex` — check it the same way. If tab participation stopped
being recorded, would it fail? Tab order has three behaviours: explicit index ordering,
opting out, and wrapping. Each needs an assertion that knows the answer, not a
non-empty order.

## Done when

`TestDivOverflowClipping` fails when `Div` stops pushing the clip, and you have
checked that it does.

The tab order tests fail when participation, ordering or wrapping is broken.

Then retire this: `docs/packages.md` already has the clip contract and overflow
behaviour, so check tab order collection is recorded there too, set the README row,
and delete.
