# element: the measure seam is right, two things block it

The design is sound and better researched than I expected. `RequestMeasuredLayout`
with Taffy's own vocabulary is the honest signature — GPUI passes the same
`Size<Option<Pixels>>` and `Size<AvailableSpace>` through `request_measured_layout`,
because flattening them loses what the solver needs to know. `RasteriseGlyph` keeps
`element` clear of `render` while letting `window` do the wiring. Per-glyph sprites,
the pixel test and the caching benchmark are all the right calls.

Two things stop it working as written, and two more to settle.

## 1 — ShapeLine panics in the phase your measure callback runs in

    window/frame.go:211
    if w.phase != phasePaint {
        panic("window: ShapeLine called in phase %v (expected phasePaint)")
    }

The measure callback runs inside the solver, during `phaseLayoutSolve`. Your
`RequestLayout` closure calls `f.ShapeLine`. It will panic on the first frame.

This is the seam I warned about from the wrong side: I said the callback must not
call back into `Frame`, and in fact it *has* to, because shaping is how text
measures itself. So the rule needs deciding rather than obeying.

Decide it this way and say so in `doc.go`: `ShapeLine` is legal during
`phaseLayoutSolve` and `phasePaint`, and nothing else on `Frame` is legal during
`phaseLayoutSolve`. That keeps the phase discipline meaningful — the callback may
measure and may do nothing else, so it cannot request layout re-entrantly, register a
hit region, or paint from inside the solver.

`window` owns that check, so agree the exact rule with its agent before either of you
writes it.

## 2 — The text/atlas.go change is a separate matter and needs proving

    Fix Atlas.rasterise logical origin Y ... so that Bounds.Origin.Y represents
    the negative offset from baseline to top of glyph mask.

`text` is retired, this is a behaviour change to it, and it is described as a fix
inside a plan about something else. Two problems with that shape: the sign of a glyph
origin is exactly the sort of thing that looks obviously wrong in both directions, and
nothing today exercises it, so "the tests still pass" will not be evidence either way.

Raise it on its own. Write a test in `text` that fails against the current sign and
states what the correct value is and why — a known glyph, a known face, a number you
can derive from the font metrics rather than from our output. If it is a real bug, it
gets its own prompt and its own commit, and the text element is built on the corrected
behaviour rather than around it.

If it turns out you only need a different convention at the call site, that is not a
`text` change at all.

## 3 — The measure cache has to be keyed by width

"Caches the result" is not enough. Flexbox may measure the same node several times
with different available widths, and it will call you again on a later frame with the
same string. Key the cache by the text, the resolved text style and the available
width, and let the benchmark show the second call at the same width is cheap while a
different width is not. A cache that returns a stale width is a layout bug that only
appears when a window is resized.

## 4 — layout's exports, and one note for the future

`OptF32` is fine. `ComputeLeafLayout` is the awkward one: the boundary pass on
`layout` deliberately unexported the solver internals, and this asks for one back.

It is justified — `layout.MeasureFunction` returns a `LayoutOutput`, so any caller
supplying one has to do the leaf sizing arithmetic, and that arithmetic is padding,
border, box-sizing and min/max clamping which nobody should reimplement. Export it,
and record in `docs/packages.md` that it is exported deliberately and why, so the next
boundary audit does not quietly remove it.

Worth noting for later, not now: the reason this is awkward is that
`MeasureFunction` returns a `LayoutOutput` rather than a size. Taffy's shape, faithfully
ported. If it keeps costing us, changing it is a decision we are allowed to take.

## Phase rules for the new methods

`RequestMeasuredLayout` in `phaseLayout` only, like `RequestLayout`.

`RasteriseGlyph` in `phasePaint` only. It can upload to the GPU, so it mutates
renderer state; that is fine during paint and nowhere else.

## Done when

The measure callback shapes without panicking, and a test proves a `Text` element
gets its width from shaping rather than from a style.

A `facet_debug` pixel test in `window` reads back glyph coverage and background.

The benchmark shows the cache working, keyed by width.

`docs/packages.md` records the two new `Frame` methods, the `phaseLayoutSolve` rule,
and why `ComputeLeafLayout` is exported.

## Worth carrying

You found that `Frame` needed two methods rather than the one I proposed, and you
found the second by following the glyph all the way to the GPU rather than stopping at
the shaping. That is the right depth. The `ShapeLine` phase rule is the same kind of
thing one step further on, and it is checkable in advance: a call in a plan is worth
grepping for the constraint it will meet.
