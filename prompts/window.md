# window: the clip stack

`scene` has `PushClip`, `PopClip`, `PushLayer` and `PopLayer`, and has had them since
it was written. `style` has `Overflow` with `Hidden`, `Clip` and `Scroll`. `Frame`
exposes none of it, so an element cannot confine its children to its own bounds.

That means no scroll view, no overflow hidden, and no widget that draws inside a box
it does not overflow. It is the same shape as focus and the cursor: designed on both
sides, never joined. This time it was found before the widget that needs it rather
than by it.

You implement first, then `element` declares on `Frame`. Per the amended entry in
`AGENTS.md`, and note that this change is behaviour only, with no struct field going
the other way.

## What to add

    PushClip(bounds geometry.Bounds[geometry.Pixels])
    PopClip()

Paint phase only. `scene.PushClip` takes a `ContentMask` in scaled pixels and
intersects nested masks already, so most of this is converting units and forwarding.

Two things to decide and record:

**Whether `Frame` exposes layers as well as clips.** `scene.PushLayer` exists for
stacking rather than clipping, and nothing has ever needed it. Do not expose it
speculatively — a `Frame` method with no caller is a guess, and we have had one of
those already. Say in `doc.go` that layers are deliberately not exposed yet.

**What an unbalanced push does.** An element that pushes and does not pop corrupts
every sibling painted afterwards, and the symptom appears somewhere else entirely.
Under `facet_debug`, assert at the end of paint that the clip stack is empty, and
panic naming the element phase if it is not. In a release build the cost is not worth
paying; the debug build is exactly where this belongs.

## Done when

A test drives an element that pushes a clip, paints a child outside the clip bounds,
and asserts the primitive carries the intersected mask. Not that it was inserted.

A `facet_debug` test shows an unbalanced push panicking at end of paint.

`docs/packages.md` records the clip stack as part of the `Frame` contract, that it is
paint-phase only, and that layers are not exposed.

Then retire this again.

## Worth carrying

Three seams now have been built on both sides and left unjoined: the cursor, focus,
and clipping. Each was found by something above trying to use it. When a package
exposes a capability, it is worth asking in the same review which package is supposed
to consume it and whether anything does — an unconsumed capability is not finished
work, it is an unstarted seam that looks finished from below.
