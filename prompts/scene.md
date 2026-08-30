# Assignment: scene

Implement the `scene` package in Facet. It is the renderer's entire input language:
what `element` paints into and what `render` consumes.

## Unblocked

`geometry` and `colour` have both landed and are reviewed. Nothing is waiting.

    Bounds[Pixels]  Point  Size  Edges  Corners  Axis  Anchor      geometry
    Rgba  Hsla  ParseHex  Blend  Mix  Premultiply                  colour

Two things there are worth knowing before you use them. `BoundsToDevicePixels`
snaps both edges and derives the size, so adjacent rectangles stay adjacent with no
seam — take device-pixel bounds from it rather than rounding origin and size
yourself. And `colour` stores straight alpha; `Premultiply` is an explicit
conversion, so decide deliberately which form a primitive carries and say so in its
doc comment.

## Read first

1. `AGENTS.md` — conventions, commit style, GB English
2. `docs/packages.md` — the `scene` entry
3. `docs/architecture.md` — the Scene section, for why the primitive list is short
4. `_upstream/gpui/crates/gpui/src/scene.rs` — the source of the model

Run `go run ./tools/upstream` if `_upstream/` is not there.

## Build

Six primitives, and no more:

    Quad                bounds, per-corner radius, background, per-edge border
    Shadow              blurred rounded rectangle, with spread and offset
    MonochromeSprite    a glyph, as an atlas tile plus a colour
    PolychromeSprite    an image or emoji, as an atlas tile
    Path                a filled bezier, for what the others cannot express
    Underline           straight and wavy, with thickness and colour

Plus `Scene`, which collects them and yields them in draw order.

Draw order is the substance of this package. Primitives arrive in tree order but
must be drawn in layer order, and the renderer wants them grouped by type so it can
issue one instanced call per group. Work out how a primitive records its depth, how
the scene sorts stably within a layer, and how a caller opens and closes a stacking
context. Read how GPUI does it before inventing an alternative.

Also here: the clip stack. A primitive records the bounds it is clipped to; nesting
intersects.

## Decisions already made

Plain data. No behaviour beyond construction, ordering and batching. A primitive
must not know what produced it.

Adding a seventh primitive touches every renderer backend, so it is a decision to
raise rather than a change to make. If something cannot be drawn with the six, say
so.

Atlas tiles are referenced by identifier and rectangle. `scene` does not own an
atlas, allocate from one, or know how glyphs get into it — that is `text` and
`render`.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test ./internal/layering
    gofmt -l $(go list -f '{{.Dir}}' ./...)

`doc.go` states what the package owns and its invariants. Tests cover ordering:
that stacking contexts nest correctly, that sorting within a layer is stable, and
that batching preserves draw order.

## Out of scope

Rasterisation, GPU resources, shaders, atlas management. This package produces a
description and stops.
