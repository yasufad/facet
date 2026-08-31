# ui: scroll view reviewed, both findings answered

The widget is right and the report is again the most valuable part of it. Leaving the
layering test red rather than reaching for `platform` was correct — that is twice now,
and both times the finding was real and got fixed at its source.

`TestScrollViewClipping` asserts the mask on child primitives rather than their
presence, and you checked it fails when `OverflowScroll` is removed. `ScrollState` in
an entity, with ephemeral `ScrollView` values reading it across frames, is exactly
what `docs/packages.md` has described since before any of this existed, and this is
the first time it appears in ordinary code rather than framework code.

Handling `ScrollPixels` and `ScrollLines` separately rather than flattening them to
one delta is the detail I would most expect to be skipped. `platform` preserves that
distinction on purpose and you kept it.

## Finding 1 — decided: `input` names the event types

Not `element.WheelEvent`, and not `platform` in `ui`.

`element.ClickEvent` exists because a click is *synthesised* from down and up on the
same target, which no platform reports. Wheel, key, pointer and text are real events
that `platform` has already normalised across operating systems, so re-declaring them
one layer up is duplication with a chance to lose something. `platform`'s entry says
hiding a platform's units is the job and hiding what it measured is not — and the
pixel-versus-line distinction you just used correctly is exactly what a re-declaration
would put at risk.

`input` owns the vocabulary of input above the OS, and its handler signatures already
name these types. It is adding aliases:

    type WheelEvent = platform.WheelEvent

Then your handler is `func(e input.WheelEvent, phase input.DispatchPhase) bool` and
your import list goes back to what it was. `prompts/input.md` has that work and it
runs first, because you are blocked on it.

## Finding 2 — you can clamp today, and should

You cannot query content height during a wheel event because there is no `Frame`
outside a frame. But you do not need one. During `Paint` you already have both bounds,
from `LayoutBounds` on the viewport node and on the content node. Put the content
height and the viewport height into `ScrollState` there, and let the wheel handler
clamp against them.

That makes the clamp one frame stale, which is invisible: the content cannot change
height between a frame and the wheel event that follows it without a frame in between
to record the new value.

So this is not a framework gap, and I would rather you closed it than shipped a scroll
view that scrolls into empty space. If you implement it and still think the framework
should offer something here, that is a much stronger version of the finding and I will
take it seriously.

## Next: a text field

Scrollbars, horizontal scrolling and virtualisation are all real work and none of them
is next. A text field is, because it is the first widget needing keyboard input, a
caret and selection, and it will find gaps the way the button and the scroll view did.
`element.Text` is single-line with no caret and no selection, so expect the report to
be longer than the widget.

Before you start it, say what you think it needs from below. You have seen enough of
this stack to guess well, and a plan that names the gaps in advance is cheaper than
meeting them one at a time.

## Worth carrying

Two widgets, two rounds, two real defects in packages beneath you, both found because
you refused to work around them. That is the milestone doing its job. Keep leaving the
layering test red when it is telling you something true.
