# element: a text element, and the Frame gap it exposes

Facet draws boxes. It cannot display a word. `text` shapes and rasterises, `scene`
has `MonochromeSprite`, `render` has pixel tests proving glyphs reach the screen, and
`Frame` declares `ShapeLine` — and nothing calls any of it, because no element
renders text.

That is the gap between "the frame loop works" and "you can write an application".
Every widget in `ui` needs a label, so this comes first.

## Read first

1. `docs/packages.md` — the `element`, `text`, `layout` and `window` entries
2. `_upstream/gpui/crates/gpui/src/elements/text.rs` — `Text` and `StyledText`
3. `text`'s own API: `ShapedLine`, `ShapedRun`, `Glyph`, `RasterMask`, `StyleRun`

## The problem to solve first: text has to measure itself

A `div` gets its size from flexbox. A run of text gets its size from shaping, which
means layout has to ask the element how big it is, at a width the solver proposes.
That is what Taffy's measure function is for, and the seam does not exist yet:

- `layout` has `MeasureFunction` and `TaffyTree.ComputeLayoutWithMeasure`. Both are
  exported and callable.
- `window` calls plain `ComputeLayout` (`window/window.go:314`), so no measure
  function is ever supplied.
- `Frame` has `RequestLayout(style, children)` and no way to register a leaf whose
  size is computed by a callback.

So a text element cannot be sized by the layout engine as things stand. Work out the
seam before writing the element, and expect it to change `Frame`.

The shape I would start from, and argue with if it is wrong:

    RequestMeasuredLayout(style layout.Style, measure func(known, available geometry.Size[...]) geometry.Size[...]) layout.NodeID

`window` collects those callbacks against their node IDs and passes one
`layout.MeasureFunction` into `ComputeLayoutWithMeasure` that dispatches by node.

`Frame` is a layer boundary and `window` is retired, so this is a change to agree
before either of you writes it, not one to land and mention. Say what you need and
why, and I will settle it.

Two things to get right in the design, because they are what makes text layout hard:

**Measurement runs inside the solver**, possibly several times for one node as
flexbox tries widths. Shaping is expensive, so the result has to be cached by the
text and the width it was measured at. `text` already caches shaped output by run.

**The measure callback runs during step 3**, when the element is mid-layout. Whatever
it captures must still be valid, and it must not call back into `Frame` — that would
be a phase violation from inside the layout engine.

## Then the element

A `Text` element that takes a string, resolves the text properties already on
`style.Refinement` — colour, family, size, weight, style, line height, alignment,
white space, overflow — shapes through `Frame.ShapeLine`, and paints
`MonochromeSprite` per glyph.

Glyph masks come from `text` and the atlas tile from `render.Upload`, wired through
`window`. Check what `Frame` exposes for that today; if the sprite cannot be built
without something `Frame` does not offer, that is the same conversation as above.

Start with one line, left-aligned, single style run. Not wrapping, not bidi, not
selection. Stop there and say so.

## Done when

A `facet_debug` test in `window` renders a `Text` element and reads back pixels that
prove glyphs landed: coverage inside a glyph, background outside it. The stack has a
pixel test now and text is the thing most worth adding to it, because "text renders"
is not observable any other way.

A benchmark shows what measuring costs when the solver asks twice for the same
string at the same width, so the cache is visible.

`docs/packages.md` records the measure seam, since it is a `Frame` change and
outlives this prompt.

## Worth carrying

`Frame` gained `ShapeLine` in the interactivity round on the assumption text would
need it. It turned out to be necessary but not sufficient, and nobody noticed for two
milestones because nothing exercised it. An interface method with no caller is a
guess; this one was half right.
