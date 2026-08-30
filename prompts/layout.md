# layout: reopened — the engine cannot be driven from outside

This package was retired as finished. The solver is finished: Taffy came across
algorithm for algorithm, the ported fixtures pass, and none of that is in question
here. What is missing is the boundary. Every caller `layout` will ever have sits
outside it, and from outside, the flexbox engine cannot be told to do flexbox.

Nothing about the algorithm changes. This is a visibility and naming pass.

## Response to the plan

The shape is right: external test first, export what it needed, unexport the solver
internals, and `TestStyleAllNonDefaultFields` is a better idea than anything in the
original prompt — a field that cannot be given a non-default value from outside is
dead, and that test says so permanently. Four things before you write it.

**`Center` must be `Centre`, and this is the last cheap moment.** `AGENTS.md` names
`centre` as one of its examples, and the public API is filling up with the American
spelling:

    layout/alignment.go     AlignItemsCenter, AlignItemsSafeCenter,
                            AlignContentCenter, AlignContentSafeCenter
    layout/style_enums.go   TextAlignCenter
    style/enums.go          AlignItemsCenter, AlignContentCenter, TextAlignCenter

Your plan proposes exporting `TextAlignCenter` as well. Rename all of them to
`Centre`. `style` mirrors the same names and converts into them, so tell that agent
rather than leaving the two halves inconsistent — the packages are both in flight,
nothing outside the repo depends on either, and after `element` and `ui` reference
these it stops being a two-file change.

**Rename `availableSpace`, do not alias it.** `type AvailableSpace = availableSpace`
compiles and makes the calls possible, but the declared parameter type stays
`availableSpace`, so `go doc` keeps printing

    func (t *TaffyTree) ComputeLayout(root NodeID, avail Size[availableSpace])

and the API still reads as unusable to the next person who looks at it. Rename the
type and change the signatures.

**Decide about the alignment values while you are in there.**
`AlignItemsCentre` and its neighbours are exported package-level `var`s, because they
are structs and cannot be constants. That means `layout.AlignItemsCentre =
layout.AlignItemsStart` compiles and silently corrupts every later use. Returning
them from functions closes it; keeping them as vars with a comment saying why is also
an answer. It is in scope because it is the same question your plan is asking about
everything else on the surface.

**Do not derive the expected numbers from our own solver.** Root 400×100, children at
0 and 110 — get those from Taffy's fixtures or a browser, not from running our code
and recording what it printed. An assertion that matches the implementation by
construction tests nothing.

One thing not to change: `Layout.Order` shows as `u32` in `go doc`, but `u32` is an
alias for `uint32`, so a caller can use it. It is only cosmetic.

## The evidence

**`ComputeLayout` cannot be called.**

    func (t *TaffyTree) ComputeLayout(root NodeID, avail Size[availableSpace])
    func (t *TaffyTree) ComputeLayoutWithMeasure(root NodeID, avail Size[availableSpace], measure MeasureFunction)

`availableSpace` is unexported (`layout/available_space.go:7`) and its constructors —
`definiteAvail`, `minContent`, `maxContent` — are all lower-case. An outside caller
can neither name the type nor obtain a value of it, so both entry points are dead
API. `window` found this while planning its frame loop.

**`Style` cannot be filled in.** It is exported, and nine of its fields have
unexported types with no exported constants:

    Display        display          displayBlock, displayFlex, ...
    BoxSizing      boxSizing        boxSizingBorderBox, ...
    Direction      direction        directionLtr, ...
    Position       position         positionRelative, ...
    Overflow       Point[overflow]  overflowVisible, ...
    TextAlign      textAlign        textAlignAuto, ...
    Contain        contain          containNone, ...

`FlexDirection` and `FlexWrap` are the exceptions — `FlexRow` and `FlexNoWrap` in
`flex.go` are exported constants of unexported types, which works and is the pattern
the rest should follow.

The consequence is worth stating plainly: `Style.Display` defaults to `displayBlock`,
and `displayFlex` is unreachable from outside the package. A caller can build a tree
and can never make it a flex container.

## Why this happened, so the fix does not repeat it

The port mirrored Taffy's module privacy. Rust's `pub(crate)` and Go's lower-case are
not the same boundary: Taffy's enums are `pub` within a crate whose consumers are
external, and ours became package-private in a package whose consumers are all
external. Faithful to the source, wrong for the language.

Every test in the package is `package layout`, so every test could see everything,
and the suite went green over an API no caller can use.

## The fix

**Write the caller first.** Before changing any visibility, add an external test
package — `package layout_test`, in `layout/`, importing `github.com/yasufad/facet/layout`.
Build a flex row with two children, solve it, and assert the resulting bounds.

That test will not compile today, and what it cannot say is the list of what to
export. It is also the permanent guard: an external test package sees exactly what a
real caller sees, so this class of defect cannot come back.

**Then export what that test needed, and nothing more.** Follow the `FlexRow`
pattern — exported constants of unexported types — wherever the caller only needs to
name values, so the type stays closed and the set of valid values stays fixed. Export
the type itself only where a caller must declare a variable or a parameter of it.

`AvailableSpace` needs both: the type, because `Size[AvailableSpace]` appears in
`ComputeLayout`'s signature, and constructors — a definite length, min-content,
max-content.

**Check the surface in the other direction too.** `CacheGet`, `CacheStore`,
`ComputeChildLayout`, `SetUnroundedLayout` and their neighbours are solver internals.
If they are exported only because the `LayoutTree` and `FlexboxTree` interfaces
require them, say so in a comment; if nothing requires them, unexport them. A public
API that omits `displayFlex` while exposing the layout cache is the wrong shape in
both directions.

## Naming

GB English, Go idiom. `displayFlex` becomes `DisplayFlex`, `overflowHidden` becomes
`OverflowHidden`. Taffy's names are Rust's; ours are ours.

`style` will have its own `DisplayFlex` and convert into this one. Two constants of
the same name in two packages is correct and expected — `style` owns the vocabulary
users write, `layout` owns the vocabulary the solver reads, and the conversion
between them is `style`'s job. Do not try to share a type across that seam.

## Who is waiting

`style` converts into `Dimension`, `LengthPercentage` and `LengthPercentageAuto` —
those are already exported and fine — and into the enums, which are not.

`window` owns the `TaffyTree`, clears it each frame, and calls `ComputeLayout`.

Both are blocked on this in the sense that neither can finish against the API as it
stands. Neither should work around it, and `window`'s agent has been told to leave it
to you rather than exporting things from the outside.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

An external `package layout_test` builds a flex row with two children, solves it, and
asserts the children's positions and sizes — real numbers it knows the answer to, not
that the sizes are positive.

`ComputeLayout` is callable from that test. Every `Style` field can be given a
non-default value from it.

The ported fixture suite still passes unchanged. If a rename touches it, the rename
is what changed, not the expectations.

Conventional commits, one file per commit, staged by path.

## Worth carrying

An exported struct with unexported field types is not exported. Neither is a method
whose signature names a type the caller cannot spell. Both compile, both pass tests
written from inside the package, and both are invisible until someone tries to be the
caller.

The general form: a package's public API is whatever its tests exercise. Where every
test is internal, the public API is untested by construction, and green means nothing
about whether the package can be used.
