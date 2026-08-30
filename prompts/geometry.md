# Assignment: geometry

Implement the `geometry` package in Facet. It is the bottom of the stack — every
other package depends on it, and it depends on nothing.

## Read first

1. `AGENTS.md` — conventions, commit style, GB English
2. `docs/packages.md` — the `geometry` entry
3. `_upstream/gpui/crates/gpui/src/geometry.rs` — the source of the model

Run `go run ./tools/upstream` if `_upstream/` is not there.

## Build

Units, as distinct types so they cannot be mixed by accident:

    Pixels          float32, logical pixels, what layout and styling speak
    DevicePixels    int32, physical pixels, what the renderer speaks
    ScaledPixels    float32, logical pixels times the display scale, pre-rounding
    Rems            float32, multiples of the root font size

Conversion between them is explicit and takes the scale factor or rem size as an
argument. There is no implicit widening.

Shapes, generic over a numeric constraint so the same types serve every unit:

    Point[T]    Size[T]    Bounds[T]    Edges[T]    Corners[T]

Bounds needs the usual queries — origin, extent, centre, contains, intersects,
intersection, union, inset, dilate. Edges and Corners need uniform, symmetric and
per-side constructors. `Axis` with `Horizontal` and `Vertical`, and helpers to read
and write the component of a Point or Size along an axis; the layout engine leans on
these to avoid writing every algorithm twice.

## Decisions already made

Values, not pointers. Everything here is small and copied freely; no method takes a
receiver pointer to mutate in place.

Do not define length or dimension types — no `Length`, `Dimension` or
`LengthPercentage`. Those belong to `layout`, which defines its own as part of the
Taffy port, and `style` converts down to them.

`geometry` must not import `colour`. They are siblings, not a stack.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test ./internal/layering
    gofmt -l $(go list -f '{{.Dir}}' ./...)

`doc.go` states what the package owns and its invariants. Tests cover the arithmetic
that is easy to get subtly wrong — bounds intersection, edge insetting, axis
accessors — and not the trivial constructors.

## Out of scope

Anything in another package. If you need something `geometry` should not own, say so
rather than adding it here.
