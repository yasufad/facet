# Assignment: style

Build `style`: the style properties, the refinement model that layers them, and the
fluent builder that elements expose.

This is the package users actually touch. Everything below it is machinery they will
never see; this is what writing Facet code feels like. `element`, `window` and `ui`
all wait on it.

## Where this stands

The first milestone is merged as `e53c5e1`: the mask, `Refinement`, `Style`, `Merge`,
`Refine`, and four properties wired end to end.

**The refinement semantics are right**, which is the part that mattered. Verified
directly: `Opacity(0)` overrides a default of 1, a refinement omitting opacity leaves
it alone, a later refinement wins on merge, and nothing allocates anywhere. The
bitset does what it was chosen to do, and the reasoning in `doc.go` for rejecting
pointer fields and sentinels is sound.

A second walkthrough of that same milestone arrived after the review below was
written. Nothing in the tree had changed between them, so the six items here are all
still outstanding — read them as the current work, not as history.

Do these before continuing to the full property list. Every one of them is cheap now
and expensive once fifty properties exist.

## 1 — Merge and Refine ignore any property in the high word

    style/refinement.go:34    if other.mask.lo != 0 {
    style/style.go:35         if r.mask.lo != 0 {

Both guard the *entire* value-copy block on the low word. A property whose bit lands
at 64 or above has its mask bit carried across by `or()` and its value never copied.

Reproduced with a throwaway in-package test that sets bit 64 and merges:

    Merge skipped the value copy for a high-word property: opacity = 0.9, want 0.25
    Refine skipped the value copy for a high-word property: opacity = 0.9, want 0.25

That is worse than dropping the property, because the resolved mask claims it was
refined while the value is the base's, and nothing downstream can tell.

It passes today only because all four properties sit at bits 0 to 3. The mask is 128
bits precisely because fifty-odd properties are coming; if compound properties go
per-edge, padding, margin, inset, border widths and corner radii are twenty bits
before anything else, and crossing 64 stops being hypothetical.

The optimisation is worth keeping — it wants to be per-word. Guard the low-word
properties on `lo` and the high-word ones on `hi`. Add a test that fails if a
high-word property is skipped, and confirm it fails against the current code before
you fix it.

## 2 — Bg converts a colour on every call, and stores the wrong type

    control: store 48 bytes    2.6 ns
    Opacity(0.5)              18.8 ns
    BgHsla(hsla)              19.9 ns
    Bg(rgba)                  75.7 ns

`Bg` calls `c.Hsla()` and stores `colour.Hsla`. That conversion is fifty-six
nanoseconds of the seventy-six, on the call users write most.

It also converts the wrong way. `scene.Quad.Background` is `colour.Rgba`, so a colour
set as Rgba is converted to Hsla on the way in and back to Rgba at paint — twice per
colour per frame, to arrive where it started. GPUI stores Hsla because GPUI's scene
consumes Hsla; ours does not.

Store `Rgba`, in `Refinement.background` and in `Style.Background` alike. Keep
`BgHsla` and have it convert at set time, where it is the uncommon case and the cost
is paid once.

## 3 — The builder-chain benchmark was asked for and not written

`BenchmarkStyleRefineEmpty`, `BenchmarkStyleRefineNonEmpty` and
`BenchmarkRefinementMerge` measure the operations, not the expression a user writes.
I measured the chain:

    Refinement{}.Flex().Bg(c).Opacity(0.8).FlexGrow(1)    108 ns

That is four properties against a 48-byte struct that will be several hundred bytes
with fifty. `Refine` and `Merge` run once per element per frame; builder methods run
once per property per element per frame, and that is where the work is.

Note also that a single builder call costs 19 ns against a 2.6 ns copy control —
seven times the cost of the copy it performs, which suggests the methods are not
inlining, probably because a value receiver returning a 48-byte struct is past the
budget. That gets worse as the struct grows.

Add the chain benchmark, find out why 19 ns, and let the answer decide the receiver.
A pointer receiver mutating in place keeps the bitset and drops both copies, at the
cost of value semantics in the refinement closure. Put the number and the decision in
`doc.go`.

## 4 — Compound granularity

Is `Padding` one bit or four? GPUI keeps per-edge options, which is what lets
`hover(s => s.PaddingLeft(4))` work without disturbing the other three. One bit per
compound property is smaller and simpler and makes that impossible. The same question
applies to margin, inset, border widths and corner radii.

Nothing implemented so far is compound, so the question is still free. It decides the
mask layout and the shape of the API. Answer it in `doc.go`.

## 5 — Properties that are not plain values

Box shadows are a slice and a font family is a string. Both allocate when set, and
both make `Refinement` non-comparable, which rules out `==` and using it as a map
key. Neither is fatal; both need a stated answer in `doc.go` rather than being
discovered when the property list reaches them.

## 6 — The zero value of Style is invisible

`Default()` gives opacity 1; `Style{}` gives 0, which renders nothing. `AGENTS.md`
asks that zero values be usable where reasonable, and here it is arguably not — a
resolved style genuinely needs defaults. That is fine, but say it: document that
`Default()` is the only valid way to obtain a `Style`. `Refinement`'s zero value, by
contrast, is correct and meaningful — nothing set — and that asymmetry is worth
naming.

## Then the rest of the package

With the model settled, continue to the full property list, roughly GPUI's set:
display, position, inset, size and min/max, margin, padding, flex direction, wrap,
grow, shrink, basis, alignment and justification, gap, background, border colour and
widths, corner radii, shadows, opacity, overflow, text colour, font family, size,
weight, style and line height.

A fluent method per property, returning the builder, so `div().Flex().Gap(4).Bg(c)`
reads as one expression.

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

The layering test passes. A high-word property survives `Merge` and `Refine`, and the
test proving it fails against the current guard. `Bg` stores `Rgba` and costs what
`Opacity` costs. A builder-chain benchmark exists and its number is in `doc.go`,
along with the receiver decision it drove. The three open design questions are
answered in `doc.go`.

Merging still has tests covering the case that matters: a refinement that sets a
field to its zero value overrides the base, and one that omits the field does not.

Conventional commits, one file per commit, staged by path.

## Habits from earlier rounds

When a struct of options or properties exists, at least one test passes it empty. Two
defects in `platform` survived their test suites because every test configured the
fields it was about to check.

Assertions should know the answer. A test that a size "is positive" passed while the
size was wrong by a window frame. If a contract is written in a doc comment, the test
checks the contract.

Benchmark the path users write, not the one that is easy to benchmark. A benchmark
that misses the hot path is worse than none, because it produces a number that looks
like reassurance.

A guard that is correct for the properties that exist today is not correct. Item 1 is
that mistake exactly: four properties, all in the low word, and a fully green suite
over code that silently drops the sixty-fifth.
