# window: the clip stack is done, close it out

Everything asked for is in and verified. I made `Div.PushClip` a no-op and
`TestWindowPushClipPrimitiveMask` failed on the mask itself:

    expected child quad content mask {{0 0} {200 200}}, got {{0 0} {0 0}}

That is the assertion that matters — the primitive carries the intersected mask, not
merely that it was inserted.

`TestUnbalancedClipStackPanicsUnderDebug` is a real integration test on a real window
rather than a unit stub, which is the thing most likely to have been faked and was
not.

`docs/packages.md` has the clip contract, the paint-phase rule alongside the other
paint-only methods, and the note that layers are deliberately not exposed.

## Retire it

Nothing is outstanding. Set the `window` row in the README to `done` and delete this
file.

One line for the report, so it is not assumed: pointer focus and tab order both work
now, and `ui` is unblocked for a scroll view. Wheel events reach `DispatchEvent`
already, so the scroll offset itself is the widget's problem rather than yours.

## Worth carrying

`PushLayer` still has no consumer and you were right not to expose it. If a scroll
view or an overlay turns out to need stacking rather than clipping, that is the moment
to add it, and it will arrive as a request from above rather than a guess from here.
