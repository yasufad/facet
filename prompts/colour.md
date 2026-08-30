# Assignment: colour

Implement the `colour` package in Facet. It sits at the bottom of the stack beside
`geometry` and depends on nothing, including `geometry`.

## Read first

1. `AGENTS.md` — conventions, commit style, GB English
2. `docs/packages.md` — the `colour` entry
3. `_upstream/gpui/crates/gpui/src/color.rs` — the source of the model

Run `go run ./tools/upstream` if `_upstream/` is not there.

## Build

    Rgba    red, green, blue, alpha as float32 in 0..1
    Hsla    hue, saturation, lightness, alpha as float32 in 0..1

Lossless conversion in both directions. Construction from packed integers — `Rgb`
for `0xRRGGBB` and `Rgba32` for `0xRRGGBBAA` — and parsing of the CSS-style hex
forms `#rgb`, `#rrggbb` and `#rrggbbaa`, since those are what a developer will type.

Operations: `Opacity` to scale alpha, `Blend` for source-over composition, `Mix` for
linear interpolation between two colours, `Lighten` and `Darken` through HSL, and
`IsOpaque` — the renderer uses it to skip blending.

The renderer wants premultiplied components. Provide that conversion explicitly
rather than storing colours premultiplied.

## Decisions already made

The spelling is `colour`, in the package name and in every identifier, comment and
doc string. `Colour`, not `Color`. This is the one place the rule is most tempting
to break, because every reference implementation spells it the other way.

Float32 storage rather than 8-bit channels. Blending and interpolation stay accurate,
and the renderer wants floats anyway.

No named colour table. A palette is a design decision and belongs above this layer.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test ./internal/layering
    gofmt -l $(go list -f '{{.Dir}}' ./...)

`doc.go` states what the package owns and its invariants. Tests cover the round trip
between Rgba and Hsla, hex parsing including the malformed cases, and blending
against known values.

## Out of scope

Gradients, colour spaces beyond sRGB, and theming. Say so if you think one is needed
rather than adding it here.
