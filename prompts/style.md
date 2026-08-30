# Assignment: style

Build `style`: the style properties, the refinement model that layers them, and the
fluent builder that elements expose.

This is the package users actually touch. Everything below it is machinery they will
never see; this is what writing Facet code feels like. `element`, `window` and `ui`
all wait on it.

Read `docs/packages.md` for the `style` and `layout` entries, and
`_upstream/gpui/crates/gpui/src/style.rs` and `styled.rs` for the model. Sync the
checkouts with `go run ./tools/upstream` if `_upstream/` is not there.

## Stop after the refinement model

Not the whole interface — the property list is long and mostly mechanical. Stop
after the part that is hard to change later: how a style is represented, how two of
them layer, and what the builder returns.

One commit: the style struct, the refinement type, the merge operation, and two or
three representative properties wired through the builder end to end. Then stop and
say so.

That discipline has caught real defects twice on `platform`. It was skipped once, on
`render`, and produced two thousand lines built on a call to the wrong function.

## The refinement model is the whole design

GPUI's `StyleRefinement` exists so styles can be layered: a base style, then a hover
refinement, then a focus refinement, each overriding only the fields it actually
sets. That is what makes this work —

    div().bg(blue).hover(func(s Style) Style { return s.Bg(darkBlue) })

— without every element carrying a complete style struct, and without a cascade.

The mechanism has to distinguish *unset* from *set to the zero value*. A refinement
that sets `opacity: 0` must override the base; one that never mentions opacity must
not. Reinventing this as "a struct with pointer fields" is the obvious move and it
costs an allocation per property per element per frame, on the hottest path in the
framework.

Work out the representation before writing the property list. Options worth
weighing: a parallel bitset of which fields are set, sentinel values per field type,
or something else — but measure the merge, because it runs for every element every
frame. Say in `doc.go` what you chose and why.

## What the package owns

Properties, roughly GPUI's set: display, position, inset, size and min/max, margin,
padding, flex direction, wrap, grow, shrink, basis, alignment and justification,
gap, background, border colour and widths, corner radii, shadows, opacity, overflow,
text colour, font family, size, weight, style and line height.

The fluent builder that sets them. Method per property, returning the builder, so
`div().Flex().Gap(4).Bg(c)` reads as one expression.

Conversion down into `layout`'s own types. `layout` deliberately imports nothing and
carries Taffy's `Dimension`, `LengthPercentage` and `LengthPercentageAuto`; `style`
converts into them. That conversion is this package's job and it is the only place
the two vocabularies meet.

## Invariants

No cascade and no stylesheet. Inheritance is explicit and confined to text
properties — a child does not inherit its parent's background by accident.

Setting a property is a method call resolved at build time, not a lookup later.

`style` imports `geometry`, `colour`, `layout` and `text`, and nothing else of ours.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

The layering test passes. Merging refinements has tests covering the case that
matters: a refinement that sets a field to its zero value overrides the base, and a
refinement that omits the field does not. A benchmark shows what a merge costs and
whether it allocates.

Conventional commits, one file per commit, staged by path.

## Two habits from earlier rounds

When a struct of options or properties exists, at least one test passes it empty.
Two defects in `platform` survived their test suites because every test configured
the fields it was about to check.

Assertions should know the answer. A test that a size "is positive" passed while the
size was wrong by a window frame. If a contract is written in a doc comment, the
test checks the contract.

## Decisions from the plan review

The bitset is the right call and the reasoning for rejecting pointers and sentinels
is sound. Four things to settle before the property list.

**Benchmark the builder chain, not just the merge.** `Merge` and `Refine` are not
where the cost is. Builder methods return `Refinement` by value, so every call in
`div().Flex().Gap(4).Bg(c).Opacity(0.8)` copies the whole struct. With `colour.Rgba`
at 16 bytes, `Edges` at 16 and `Corners` at 16, a fifty-property `Refinement` is
somewhere around 300 to 400 bytes; a six-call chain moves two kilobytes per element,
and a thousand elements a frame makes that megabytes a second of memcpy.

That may well be fine — it is still cheaper than allocating per property, and it is
linear and cache-friendly. But measure the chain a user actually writes, not the
merge, and put the number in `doc.go`. If it is too slow, the lever is the receiver:
a pointer receiver mutating in place keeps the bitset and drops the copying, at the
cost of value semantics in the refinement closure. That choice is much cheaper now
than after fifty properties exist.

**Decide the granularity of compound properties.** Is `Padding` one bit or four?
GPUI keeps per-edge options, which is what lets `hover(s => s.PaddingLeft(4))` work
without disturbing the other three. One bit per compound property is smaller and
simpler and makes that impossible. The same question applies to margin, inset,
border widths and corner radii. Whichever you choose, it decides the mask layout and
the shape of the API, so decide it explicitly and say so in `doc.go`.

**Say what happens to properties that are not plain values.** Box shadows are a
slice and a font family is a string. Both break the zero-allocation claim when set,
and both make `Refinement` non-comparable, which rules out `==` and using it as a
map key. Neither is fatal; both need a stated answer rather than being discovered
when the property list reaches them.

**The zero value of `Style` is invisible.** With `Opacity` defaulting to 1.0, a
`var s Style` has opacity 0 and renders nothing. `AGENTS.md` asks that zero values
be usable where reasonable, and here it is arguably not reasonable — a resolved
style genuinely needs defaults. That is fine, but it has to be explicit: document
that `Default()` is the only valid way to obtain a `Style`, and consider whether
anything should stop a zero one being used by accident. `Refinement`'s zero value,
by contrast, is correct and meaningful — nothing set — and that asymmetry is worth
naming.
