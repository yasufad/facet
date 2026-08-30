# style: reinstated — every typography property shares one mask bit

The prompt was retired at `a9782cc`. It is back, because the property list has a
defect that destroys user data on the main path, and the suite is green over it.

Everything from the second review is genuinely closed: `MergeFrom` takes a pointer,
`testHigh` is gone, `SetBackgroundHsla` matches `SetBackground`, and the mutators are
excellent — 1.3 ns each against a 24 ns copy control. That control number is the
vindication of the receiver decision, incidentally: `Refinement` is 504 bytes now, so
value-receiver builders would have cost more per call than the whole four-mutator
sequence costs today.

## The defect

    const (
        propDisplay uint8 = iota
        ...
        propMouseCursor

        propFlexGrow uint8 = 64
        propTextColour
        propFontFamily
        ...
        propLineClamp
    )

In a Go const block an identifier with no expression repeats **the previous
expression**, not the previous value. After `propFlexGrow uint8 = 64` that expression
is the literal `64` — not `iota` — so every constant after it is also 64:

    flexGrow=64 textColour=64 fontFamily=64 fontSize=64 lineHeight=64 lineClamp=64

All sixteen high-word properties are the same bit. Setting any one of them marks all
sixteen as set, so `Refine` and `MergeFrom` then copy all sixteen values across —
fifteen of which are zero.

It corrupts real data, not just the mask:

    r.SetFontSize(12)   // only font size
    s.Refine(r)

    refining FontSize clobbered Text.Colour: got {0 0 0 0}
    refining FontSize clobbered Text.FontWeight: got 0

Set a font size on a hover refinement and the element loses its text colour, weight,
line height, underline and the rest.

## The fix

`iota` is the index within its const block, so `64 + iota` in the same block gives
114, not 64. Put the high-word properties in **their own const block**, where `iota`
restarts:

    const (
        propFlexGrow uint8 = 64 + iota
        propTextColour
        ...
    )

Then add the test that would have caught it — not a test for this one bug, but one
that holds for every property added afterwards: **assert that all property indices
are distinct**. A table of every `prop*` constant, checked for duplicates, fails the
moment two share a bit for any reason. Sixty-seven constants maintained by hand will
collide again otherwise, and the next collision will be two properties nobody thought
to test together.

Confirm it fails against the current code before you fix it.

## Why the suite missed it

`TestHighWordProperty` exercises `propFlexGrow` alone, and a single property cannot
reveal a collision. `TestTypographyRefinement` passes because setting and reading one
typography property still works — the value written is the value read; what is lost
is everything else.

Every test set the properties it was about to check. That is the same shape as the
two `platform` defects and the earlier vacuous `Merge` assertion, and it is worth
naming as the pattern it is: a test that configures exactly what it inspects can only
find bugs in that one field's own code path. Interaction between properties needs a
test that sets one thing and checks something else.

## Also

`docs/packages.md` still describes `style` as owning "the fluent builder that elements
expose". That has been false since the receiver decision — the chain lives on `*Div`
and this package exposes mutators. I have corrected the entry; the point for you is
that retiring a prompt means moving what the package guarantees into
`docs/packages.md` first. None of the decisions in `doc.go` — the bitset, per-edge
granularity, slice immutability, `Default()` as the only constructor — reached it, and
`doc.go` is not where the next agent looks.

## Also — `Center` must be `Centre`

`AGENTS.md` names `centre` as one of its examples, and `style/enums.go` has
`AlignItemsCenter`, `AlignContentCenter` and `TextAlignCenter`. `layout` has the same
names on its side of the conversion and is renaming them now, so rename yours in the
same round or the two halves stop matching.

Nothing outside the repo depends on either package yet. Once `element` and `ui`
reference these names it stops being a two-file change.

## Numbers worth keeping

    control copy (504 bytes)      24.1 ns
    SetOpacity                     1.6 ns
    four mutators, hoisted         5.4 ns
    Style.Refine                 116.3 ns
    MergeFrom                     98.2 ns
    Style.ToLayout               190.2 ns

`Refine`, `MergeFrom` and `ToLayout` run once per element per frame, so together they
are roughly 400 ns per element — 400 µs for a thousand elements, against a 16 ms
budget. That is fine, and it is the first real floor anyone has measured for a Facet
frame. Keep those three numbers in `doc.go` as the frame budget baseline, and say
that is what they are.

## Done when

The high-word properties have distinct indices.

A test asserts every property index is distinct, and it fails against the current
const block.

A test sets one typography property and asserts the others survive.

Then the prompt retires again — after the invariants move to `docs/packages.md`.
