# Assignment: text

Implement the `text` package in Facet: font loading and matching, segmentation,
shaping, line breaking, and glyph rasterisation. This is the deepest pit in any GUI
framework, so the boundary matters as much as the implementation.

## Read first

1. `AGENTS.md` — conventions, commit style, GB English
2. `docs/packages.md` — the `text` entry
3. `docs/architecture.md` — the Text section, including the open question
4. `_upstream/gpui/crates/gpui/src/text_system.rs` and the `text_system/` directory
5. The `github.com/go-text/typesetting` documentation

Run `go run ./tools/upstream` if `_upstream/` is not there.

## Build

**Font loading and matching.** Enumerate system fonts, load from bytes, and resolve
a family, weight, style and stretch to a face. Fallback when a face lacks a glyph —
the request is for text, not for a font, and something must draw it.

**Shaping.** Segment by script and bidi level, shape each run, and cache the result.
Cache by run rather than by string: the same word in the same font at the same size
appears constantly, and re-shaping it is the easiest performance mistake to make.

**Line layout.** Wrap to a width, break at the right opportunities, and report the
metrics a caller needs — ascent, descent, line height, and the mapping between byte
offsets and x positions in both directions. Text editing and hit testing both need
that mapping, and it is fiddly enough to deserve its own tests.

**Rasterisation.** Turn outlines into coverage bitmaps and place them in a glyph
atlas keyed by face, size, subpixel offset and any transform.

## Decisions already made

`go-text/typesetting` is the dependency, and the only third-party import permitted
in this package. Add it to `go.mod` and say so in the commit.

Nothing above `text` knows that dependency exists. Expose shaped lines and glyph
runs in our own types. If typesetting turns out to be the wrong choice later, the
blast radius must be this package.

## Still open, and yours to settle

Rasterisation is not decided. `typesetting` stops at outlines. The candidates are
`golang.org/x/image/vector`, rasterising on the GPU in a compute pass as GPUI does,
or writing a scanline rasteriser. Try one, measure it, and write down what you found
and what you chose — that goes into `docs/architecture.md` and closes the last open
item there.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test ./internal/layering
    gofmt -l $(go list -f '{{.Dir}}' ./...)

`doc.go` states what the package owns and its invariants. Tests cover offset-to-
position mapping in both directions, wrapping at the awkward boundaries, and
fallback when a face lacks a glyph. Include something non-Latin and something
right-to-left; a text stack that only works in English is not finished.

## Out of scope

Text editing, selection models, input methods, and anything that draws. This package
produces shaped, measured, rasterised text and stops.
