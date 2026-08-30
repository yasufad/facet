# style: second review

Four of the six are closed. `Background` is `Rgba` in both types and `SetBackground`
costs 2.41 ns against a 2.08 ns control, so the conversion is genuinely gone. The
value-receiver builders are gone and the mutators read well. `doc.go` answers
compound granularity, slice immutability and the `Default()` asymmetry, and records
the numbers behind the receiver decision.

The per-word guard works. I disabled the `hi` block and `TestHighWordProperty` failed,
which is what a test is for.

Three things left, then the property list.

## 1 — The Merge half of that test cannot fail

Look again at what happened when I disabled the guard: the `Refine` assertion fired
and the `Merge` assertion did not.

`Merge` early-returns `return other` when the receiver's mask is empty
(`style/refinement.go:31`), and the test merges into an empty `r1` — so it never
reaches the guarded copy block. It passes via the early return whether the fix is
there or not. With a non-empty receiver it fails as it should:

    Merge dropped high-word value: testHigh = 0, want 0.25

One line fixes it: set something on `r1` before merging. Then re-break the `hi` guard
and confirm *both* halves fail before putting it back.

This is the trap the earlier note was about, in its exact form: half a test proving
the fix, half proving nothing, and both green. When a function has an early return,
a test aimed at the main path has to get past it first.

## 2 — testHigh is scaffolding, and it ships

`propTestHigh` at bit 64, a `testHigh` field on both `Refinement` and `Style`, and a
`SetTestHigh` mutator live in `mask.go`, `refinement.go` and `style.go`. It is a
style property that is not a style property, and it reserves bit 64 for nothing.

It cannot move into a `_test.go` file — a test file cannot add a struct field — but
it does not need to exist. Bit indices are arbitrary: **give a real property the high
word.** Put `flexGrow` at 64 and the high-word path is covered by a genuine property
with genuine tests, permanently, with no fake field and no reserved bit. Then delete
`testHigh` entirely.

The fake property was the right way to reproduce the bug and the wrong thing to keep.

## 3 — Merge still has the pattern the builders just lost

    BenchmarkRefinementMerge      56.74 ns
    BenchmarkStyleRefineNonEmpty  29.22 ns

`Merge` takes a value receiver, does `out := r`, and returns a value: copy in, mutate,
copy out. That is the same shape that cost 19 ns per builder call, on the same
48-byte struct that per-edge compound properties will grow to several hundred bytes.
`Refine` avoids it by taking `*Style`.

Settle it now and the same way: `func (r *Refinement) MergeFrom(other *Refinement)`,
measured against what it replaces. Leaving `Merge` by value while the builders went
pointer is an inconsistency that reads as an oversight later, and it gets more
expensive with every property added.

## Two smaller things

`SetBgHsla` should be `SetBackgroundHsla`, to match `SetBackground`.

The `doc.go` line "sequence of 4 mutators 17.95 ns" is mostly the per-iteration
zeroing of a 48-byte struct and the sink store, not the mutators — four calls at
~2 ns each is ~8 ns. Either measure the four calls against an addressable receiver
hoisted out of the loop, or say what the number includes. A recorded benchmark is
read later by someone who will not re-derive it.

## Then the property list

With the model settled, the rest is mechanical: display, position, inset, size and
min/max, margin, padding, flex direction, wrap, grow, shrink, basis, alignment and
justification, gap, background, border colour and widths, corner radii, shadows,
opacity, overflow, text colour, font family, size, weight, style and line height.

Plus the conversion into `layout`'s `Dimension`, `LengthPercentage` and
`LengthPercentageAuto` — this package is the only place those two vocabularies meet.

Per-edge bits, as `doc.go` now says, which means the property list crosses bit 64 and
item 1's guard stops being theoretical.

Two things you should know while writing it, both decided since your last round:

- The fluent chain lives on `*Div` in `element`, which another agent is building now.
  Keep the mutators complete and orthogonal — one per settable property — and do not
  add a chain here.
- `element` and `ui` may now import `input`, so a widget can declare a click. It does
  not change anything in `style`, but it is why the layering table moved.

## Done when

The Merge high-word assertion fails when the `hi` guard is disabled, and you have
checked that it does.

`testHigh` is gone and a real property occupies the high word.

`Merge` does not copy the struct in and out, with the measurement that decided it in
`doc.go`.

Then the full property list, and this prompt is retired.
