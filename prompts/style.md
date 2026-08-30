# style: review of the first milestone

The refinement semantics are right, which is the part that mattered. I verified them
directly rather than from the tests: `Opacity(0)` overrides a default of 1, a
refinement omitting opacity leaves it alone, a later refinement wins on merge, and
nothing allocates anywhere. The reasoning in `doc.go` for rejecting pointer fields
and sentinels holds up.

Six things before the property list. All of them are cheap now and expensive at fifty
properties.

Your second walkthrough described the same tree as the first — nothing had changed
between them — so if this is the first you are seeing of the list, that is why.

## 1 — Merge and Refine ignore any property in the high word

    style/refinement.go:34    if other.mask.lo != 0 {
    style/style.go:35         if r.mask.lo != 0 {

Both guard the *entire* value-copy block on the low word. A property at bit 64 or
above gets its mask bit carried across by `or()` and its value never copied. I set
bit 64 in a throwaway in-package test and merged:

    Merge skipped the value copy for a high-word property: opacity = 0.9, want 0.25
    Refine skipped the value copy for a high-word property: opacity = 0.9, want 0.25

That is worse than dropping the property: the resolved mask says it was refined while
the value is the base's, so nothing downstream can tell.

It passes today only because all four properties sit at bits 0 to 3. The mask is 128
bits precisely because fifty-odd are coming — and per-edge padding, margin, inset,
border widths and corner radii are twenty bits before anything else.

Keep the optimisation, make it per-word: guard the low-word properties on `lo`, the
high-word ones on `hi`. Write the test first and confirm it fails against what is
there now.

## 2 — Bg converts on every call, and stores the wrong type

    control: store 48 bytes    2.6 ns
    Opacity(0.5)              18.8 ns
    BgHsla(hsla)              19.9 ns
    Bg(rgba)                  75.7 ns

`Bg` calls `c.Hsla()`. That conversion is fifty-six of the seventy-six nanoseconds,
on the call users write most.

It also converts the wrong way. `scene.Quad.Background` is `colour.Rgba`, so a colour
set as Rgba becomes Hsla on the way in and Rgba again at paint — twice per colour per
frame to arrive where it started. GPUI stores Hsla because GPUI's scene consumes
Hsla; ours does not.

Store `Rgba` in `Refinement.background` and `Style.Background` both. `BgHsla` keeps
converting at set time, where it is the uncommon case and paid once.

## 3 — The builder chain is still unmeasured

The three benchmarks measure `Refine` and `Merge`, which run once per element per
frame. Builder methods run once per *property* per element per frame. I measured the
chain:

    Refinement{}.Flex().Bg(c).Opacity(0.8).FlexGrow(1)    108 ns

Four properties, 48-byte struct — several hundred bytes once the list is full. And a
single builder call costs 19 ns against a 2.6 ns copy control, seven times the copy
it performs, which reads like the methods are not inlining: a value receiver
returning 48 bytes is likely past the budget, and it grows.

Add the chain benchmark, find out why 19 ns, and let that decide the receiver. A
pointer receiver mutating in place keeps the bitset and drops both copies, at the
cost of value semantics in the refinement closure. Number and decision into `doc.go`.

## 4, 5, 6 — Three decisions `doc.go` does not answer

**Compound granularity.** Is `Padding` one bit or four? Per-edge is what lets
`hover(s => s.PaddingLeft(4))` leave the other three alone; one bit is smaller and
makes that impossible. Same for margin, inset, border widths, corner radii. Nothing
compound exists yet, so it is still free — and it fixes the mask layout and the API
shape.

**Properties that are not plain values.** Shadows are a slice, font family a string.
Both allocate when set and both make `Refinement` non-comparable, ruling out `==` and
map keys. Neither is fatal; both want an answer now rather than at the property that
hits them.

**`Style{}` is invisible.** `Default()` gives opacity 1, the zero value gives 0 and
renders nothing. That is defensible — a resolved style needs defaults — but say it:
`Default()` is the only valid way to obtain a `Style`. `Refinement`'s zero value is
meaningful by contrast, and that asymmetry is worth a sentence.

## Then the property list

Roughly GPUI's set: display, position, inset, size and min/max, margin, padding, flex
direction, wrap, grow, shrink, basis, alignment and justification, gap, background,
border colour and widths, corner radii, shadows, opacity, overflow, text colour, font
family, size, weight, style and line height. Plus the conversion into `layout`'s
`Dimension`, `LengthPercentage` and `LengthPercentageAuto` — this package is the only
place those two vocabularies meet.

## Done when

A high-word property survives `Merge` and `Refine`, and the test proving it fails
against the current guard. `Bg` stores `Rgba` and costs what `Opacity` costs. The
chain benchmark exists and its number is in `doc.go` with the receiver decision it
drove. The three questions above are answered there too.

## Worth carrying

A guard that is correct for the properties that exist today is not correct. Item 1 is
that exactly: four properties, all low-word, a fully green suite over code that
silently drops the sixty-fifth. It is the same shape as two `platform` defects that
survived their tests because every test configured the field it was about to check.

And benchmark the path users write. A benchmark that misses the hot path is worse
than none — it produces a number that looks like reassurance.
