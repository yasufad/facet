# style: response to the plan

Items 1, 2, 4 and 6 are right as planned — go ahead. Item 3 needs redirecting and
item 5 needs one more rule. Details below, then what done looks like.

## Approved as planned

**1. Per-word guards.** Exactly right, including writing the high-word test first and
confirming it fails against the current code.

**2. `Rgba` in `Style.Background` and `Refinement.background`, `BgHsla` converting at
set time.** Right, and for the reason given.

**4. Per-edge bits for compound properties.** Right. Note it makes item 1 load-bearing
rather than theoretical: inset, margin, padding, border widths and corner radii at
four bits each is twenty before anything else, so the property list will cross bit 64.

**6. `Default()` as the only valid constructor, `Refinement{}`'s zero value
meaningful.** Right, including naming the asymmetry.

## 3 — The receiver: my diagnosis was wrong, and so is the fix

**First, correct something I told you.** I wrote that the 19 ns "reads like the
methods are not inlining." They inline. `go build -gcflags=-m ./style` lists
`can inline Refinement.Opacity`, `Flex`, `BgHsla` and `FlexGrow` — everything except
`Bg`, which the `c.Hsla()` call excludes. The plan's reasoning about the inlining
budget rests on a claim of mine that is false. Sorry.

**What the 19 ns actually is.** Three probe methods — field store only, `mask.set`
only, both with a constant shift:

    control (plain 48-byte copy)      2.55 ns
    field store only                 20.12 ns
    mask.set only                    19.36 ns
    mask constant shift + field      18.61 ns

All the same. Not the mask, not the shift, not the field. It is the
value-receiver-returns-value pattern itself — copy in, mutate, copy out — already
costing ~19 ns at 48 bytes, and it grows with the struct.

So value receivers are the problem. The proposed fix is what fails.

**`Refinement{}.Flex()` does not compile with a pointer receiver.** A composite
literal is not addressable:

    cannot call pointer method Flex on R

That breaks the plan's own benchmark expression, and it breaks
`div().Flex().Gap(4).Bg(c)` — the expression this package exists to make read well.
Returning `*Refinement` also puts an aliasable pointer in every element and escapes
to the heap as soon as one is stored, which is the allocation the bitset was chosen
to avoid.

The "≈2 ns for the entire 4-call chain" is stated as a result before anything was
measured, and cannot have been measured: the expression it names does not build.
Do not put a number in `doc.go` that a benchmark did not produce.

**Where the chain belongs.** GPUI has this same problem in a language where the copy
would be free, and still avoided it. `styled.rs:22`:

    pub trait Styled: Sized {
        fn style(&mut self) -> &mut StyleRefinement;

and every fluent method, `styled.rs:45`:

    fn flex(mut self) -> Self {
        self.style().display = Some(Display::Flex);
        self
    }

The fluent chain is on the **element**. The element is the receiver; the refinement is
reached by mutable reference and never copied.

That is the decision, and it is settled here rather than by you, because it crosses
into `element`:

- `style` exposes **mutators on `*Refinement`** — `SetDisplay`, `SetOpacity`,
  `SetBackground` and so on. No fluent chain on `Refinement`, no value-receiver
  builders.
- `element` puts the chain on `*Div`, which is already addressable:
  `func (d *Div) Flex() *Div { d.style.SetDisplay(DisplayFlex); return d }`.

`div().Flex().Bg(c)` still reads as one expression, nothing is copied, nothing
escapes, and the 19 ns goes away without the API paying for it.

**What to benchmark instead.** Not a chain — there is no chain in this package any
more. Measure a single mutator against the 2.55 ns control, and measure setting four
properties on one addressable `Refinement` in sequence. Those are the numbers for
`doc.go`, alongside the note that the fluent chain lives in `element` and why.

## 5 — Slices need an aliasing rule, not just a comparability note

"Refinements copy slice headers and strings directly" shares the backing array. Two
refinements holding the same `[]BoxShadow`, one of them appended to, is a mutation the
other sees — and with refinements layered per state, they will hold the same slice
routinely.

State the rule in `doc.go` and hold to it: a slice-valued property is immutable once
set. Setting it replaces the slice; nothing appends to one already stored. Non-
comparability is worth recording too, but it is the smaller half.

## Done when

Per-word guards in `Merge` and `Refine`, with a high-word test that fails against the
current single guard.

`Background` is `Rgba` in both types; `Bg` costs what `Opacity` costs.

No value-receiver builders on `Refinement`. Mutators on `*Refinement`, benchmarked
against the 2.55 ns control, with the numbers and the element-owns-the-chain decision
recorded in `doc.go`.

Compound granularity, the slice immutability rule, and the `Default()`/`Style{}`
asymmetry answered in `doc.go`.

Then the property list.

## Worth carrying

A plan that states the result of a measurement it has not taken is not a plan. Item 3
named a number for an expression that does not compile — the compile error and the
real cost were each about a minute's work to find. When a decision is supposed to be
settled by measurement, take the measurement first and let it decide, including when
you are confident you know the answer.

And I made the mirror of that mistake: I offered "not inlining" as the explanation
without checking `-gcflags=-m`, and it sent you at the wrong fix. A hypothesis handed
over as though it were a finding is worse than no hypothesis.
