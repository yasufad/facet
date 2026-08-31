# layout: three exports for the measure seam

`element` is gaining a text element, which sizes itself through the solver rather
than from its style. That needs `ComputeLayoutWithMeasure`, and a caller supplying a
`MeasureFunction` cannot currently do the arithmetic it demands.

**Run this first.** `window` and `element` are both blocked on it. Nothing here
depends on either of them.

## What to export

**`OptF32`**, with constructors and accessors. A measure callback receives
`Size[OptF32]` for the known dimensions, so any caller outside this package has to be
able to name and read one.

**`LeafMeasureFunc`** and **`ComputeLeafLayout`**. `MeasureFunction` returns a
`LayoutOutput`, not a size, so every caller supplying one has to compute padding,
border, box-sizing and min and max clamping to build it. That is solver arithmetic
and nobody outside should reimplement it.

Exporting `ComputeLeafLayout` re-opens something the boundary pass deliberately
closed, so record in `docs/packages.md` that it is exported on purpose and why. A
future audit will otherwise remove it and break `window` again.

Extend the external test package rather than only exporting: `layout_test` should
build a tree with a measured leaf, supply a measure function through
`ComputeLayoutWithMeasure`, and assert the solved size. If the callback cannot be
written from outside the package with what you exported, the export list is still
incomplete, and that test is the only thing that will tell you.

## Worth saying out loud

The reason this is awkward is that `MeasureFunction` returns a `LayoutOutput` rather
than a size. That is Taffy's shape, faithfully ported, and faithfulness is this
package's stated invariant. Keep it. But note it in `docs/packages.md` as a known
cost, because if it keeps forcing exports we may decide to wrap it, and the next
person should find the reasoning rather than rediscover it.

## Done when

    go build -o bin/ ./...
    go test ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

`layout_test` solves a tree containing a measured leaf using only exported API.

`docs/packages.md` records why `ComputeLeafLayout` is public.

Then retire this: guarantees into `docs/packages.md` first, then delete.
