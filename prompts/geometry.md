# Assignment: geometry

The `Point` device-pixel pair is landed and correct — `a703694`, matching the
existing origin rounding, with a test pinning that a standalone point lands where
the same point converted as a `Bounds` origin lands. That test stays valid after the
change below, because both still round the origin the same way.

This round fixes a real defect that an earlier review of mine claimed did not exist.

## Adjacent rectangles gap or overlap after conversion

`BoundsToDevicePixels` rounds origin and size independently:

    Origin: round(origin * factor)
    Size:   round(size   * factor)

Two rectangles that share an edge in logical pixels need not share one in device
pixels:

    origin 0.40 size 9.20 at scale 1.5  ->  a covers [1,15], b starts at 14

A one-pixel overlap, which double-blends alpha along the seam. The complementary
case gaps, showing the background through as a hairline. Both are the sort of
artefact that gets chased through the renderer for a week before anyone suspects the
geometry.

The fix is to snap both edges and derive the size:

    x0 = round(origin * factor)
    x1 = round((origin + size) * factor)
    size = x1 - x0

The neighbour's `x0` is then the same expression as this rectangle's `x1`, so they
agree exactly, for every origin and every scale factor.

`DeviceBoundsToPixels` needs the same treatment on the way back.

`SizeToDevicePixels` cannot do this — a size with no origin has no edges to snap, so
it stays round-to-nearest. Say so in its doc comment: a size converted alone will
not always match the size of the same rectangle converted as a `Bounds`, and callers
who care about adjacency must convert the bounds rather than the size.

## Test it against adjacency, not against a single rectangle

The test that matters walks a series of touching rectangles and asserts each one's
right edge equals the next one's left edge, in device pixels.

Use non-zero origins. My review passed this package on a probe whose first rectangle
started at 0, where `round(0) = 0` makes independent rounding and edge-snapping
produce identical output. It could not have detected the bug it was written to look
for. Vary origin, size and scale factor — 0.4/9.2 at 1.5 is a known failing case,
and it should be in the test.

## Done when

    go build -o bin/ ./...
    go test ./geometry/
    go test ./internal/layering
    go vet ./geometry/
    gofmt -l $(go list -f '{{.Dir}}' ./...)

Adjacency holds across a range of fractional origins, sizes and scale factors.
`SizeToDevicePixels` documents its limitation. One conventional commit, staged by
path.

`scene` and `text` have been told the fix is coming and not to work around it, so
tell me when it lands and I will unblock them.

## Worth carrying

You checked the code before matching it, found my description of it was wrong, and
said so instead of building on it. That is the behaviour that catches this class of
bug — a review is a claim, not a fact, and a claim about what code does can be
tested in about a minute.
