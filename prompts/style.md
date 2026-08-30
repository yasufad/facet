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
