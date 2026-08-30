# style: one item left — Centre

The collision fix is correct and well tested. I broke it and watched the guard work:

    property index collision: propTextColour and propFlexGrow share bit index 64
    property index collision: propFontFamily and propTextColour share bit index 64
    ... fifteen in all

`TestDistinctPropertyIndices` is the right test — not a test for that one bug, but
one that holds for every property added afterwards. `TestTypographyPropertyIsolation`
is the right companion: set one thing, check something else. Together they close the
pattern that produced the last three defects in this package.

Indices are distinct now — `flexGrow=64 textColour=65 fontSize=69 lineClamp=79`.

## What is left

The prompt asked for `Center` to become `Centre`, and it has not been done:

    style/enums.go:66    AlignItemsCenter
    style/enums.go:89    AlignContentCenter
    style/enums.go:180   TextAlignCenter
    style/enums.go:265   layout.AlignItemsCenter
    style/enums.go:286   layout.AlignContentCenter

`AGENTS.md` names `centre` as one of its four examples, and this covers identifiers,
not only prose. `layout` is renaming its half right now, so the conversion sites in
`enums.go` change with it.

The window for this is closing. `element` exists as of this morning and does not
reference these names yet; once it and `ui` do, a two-file rename becomes a
four-package one.

## On retiring the prompt

This is the second time `style` has been retired with work outstanding. The first
time the invariants never reached `docs/packages.md` and the package had a defect
that zeroed typography; this time the rename was in the prompt when it was deleted.

Retiring is the last step, and the test for it is that the prompt is empty of work —
not that the last thing you were asked about is done. If something in it looks wrong
or unnecessary, say so and leave it; do not close it silently.

`docs/packages.md` now carries what this package guarantees — the mask, per-edge
granularity, the slice rule, `Default()` as the only constructor, and that the fluent
builder is not here. Read that entry before you retire this file again, and add
anything it is still missing.

## Done when

No identifier in `style` spells it `Center`, including the `layout.*` conversion
targets, and the tree builds against `layout`'s renamed half.

Then retire this, and this time it is genuinely finished.
