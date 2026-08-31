# window: reopened for the measure seam

`element` is building a `Text` element and it needs two things from `window` that
do not exist. The design is agreed; `prompts/element.md` has the reasoning. Your job
is the implementation and one phase rule that is yours to decide.

## What element needs

**`RequestMeasuredLayout(style layout.Style, measure element.MeasureFunc) layout.NodeID`.**
Register the callback against the node ID, create the leaf in `w.layoutTree`, and
call `ComputeLayoutWithMeasure` instead of `ComputeLayout` at
`window/window.go:314`, dispatching by node to the registered callbacks. Clear the
callbacks after layout. `layout.ComputeLeafLayout` is being exported for this, because
`MeasureFunction` returns a `LayoutOutput` and the leaf arithmetic — padding, border,
box-sizing, min and max clamping — is not something to reimplement here.

**`RasteriseGlyph(face, gid, size, subpixel) (scene.AtlasTile, bounds, bool)`.** This
is the `text` to `render` wiring that `docs/packages.md` says is yours and nobody
else's: ask `text.Atlas` for the coverage mask, upload it through
`render.Renderer.Upload`, cache the tile, and hand back the reference. A scale change
already drops both caches; make sure the tile cache is one of them.

## The phase rule you own

`ShapeLine` currently panics outside `phasePaint` (`window/frame.go:211`). The measure
callback runs during `phaseLayoutSolve` and shaping is how text measures itself, so
that rule has to widen.

Widen it precisely: `ShapeLine` is legal in `phaseLayoutSolve` and `phasePaint`, and
**nothing else on `Frame` is legal during `phaseLayoutSolve`**. The callback may
measure and do nothing else. That stops an element requesting layout re-entrantly,
registering a hit region, or painting from inside the solver, which are all things the
layout engine cannot survive.

Add a test that a `Frame` method other than `ShapeLine` panics when called from inside
a measure callback. The phase machinery already exists; this is the case it was built
for and the only one that runs inside another package's call stack.

Phase rules for the new methods: `RequestMeasuredLayout` in `phaseLayout` only.
`RasteriseGlyph` in `phasePaint` only, since it uploads to the GPU.

## Done when

A `Text` element sizes itself through the solver and its glyphs reach the back buffer,
proven by a `facet_debug` pixel test alongside the existing one: coverage inside a
glyph, background outside it.

A test shows a non-`ShapeLine` `Frame` call from inside a measure callback panicking.

`docs/packages.md` gains the two methods and the `phaseLayoutSolve` rule.

Then retire this again: guarantees into `docs/packages.md`, README row, delete.

## Worth carrying

The pixel test you wrote is the standard for this package now. Text is the next thing
that cannot be verified any other way, because a glyph that shapes, uploads and lands
one pixel off looks identical to a working one in every assertion except a readback.
