# ui: answer one question before writing the button

The scope is right and it stops in the right place. Three things, and the first is a
decision the plan takes silently at the one moment it is cheap to take deliberately.

## 1 — Fifteen forwarding methods per widget does not scale, and this is widget one

The plan gives `Button` its own `Bg`, `BgHsla`, `TextColour`, `TextColourHsla`,
`Border`, `BorderColour`, `CornerRadius`, `Padding`, `PaddingX`, `PaddingY`, `Hover`,
`Active`, `Focus` and `Refine`, each forwarding to the refinement.

`Div` has 129 builder methods. A widget that re-exposes even a tenth of them has
picked a number, and the next widget picks a different one, and adding a style
property means visiting every widget in the library. That is the shape `AGENTS.md`
warns about: a feature that requires editing a list somewhere else.

Four ways out, and you should pick one and say why in `doc.go`:

- **Expose the refinement.** `func (b *Button) Style() *style.Refinement`, and the
  caller mutates it. Zero duplication, no fluency.
- **Embed `*element.Div`.** Its builder methods promote, so `NewButton("Go").Bg(c)`
  compiles — but returns `*Div`, so `Button`-specific methods have to come first in
  the chain. Zero duplication, fluency that degrades in one direction.
- **Forward deliberately, and few.** Decide that a widget exposes only what is part
  of *being that widget*, and everything else goes through `Style()`. Then the list
  is short by rule rather than by accident, and the rule is writable down.
- **Something better.** GPUI solves this with a `Styled` trait and a macro; Go has
  neither, and that is worth saying rather than pretending the shapes match.

I lean towards the second or third. What I want is the reasoning recorded, because
whatever you choose, widget two follows it and widget twenty cannot change it.

## 2 — Three ways to construct one button

`NewButton(label)`, `NewButtonWithOptions(opts)`, and fluent setters is two too many.
`ButtonOptions` earns its place only if there is something that cannot be set after
construction, and for a button there is not. Drop it, keep `NewButton` and the
setters, and make sure `NewButton("")` is sound — which your empty-value test already
plans to check.

## 3 — The third hand-written fake frame is a finding, not a chore

`element` has one, `window` has one, and you are about to write a third. Ours are in
`_test.go` files, so nothing outside can use them, and `AGENTS.md` forbids
cross-package fixtures for good reasons.

But notice what that means for anyone outside this repository writing a widget: they
cannot test an element at all without reimplementing a `Frame` from scratch, and
`Frame` is now over twenty methods. That is a real framework gap and exactly the kind
of thing this milestone exists to surface.

Do not fix it from here. Write your fake, and put it at the top of the gap report
with what it cost you and what would have helped — most likely an exported test
double from `element`. That decision is mine and I would rather take it with your
report in hand than guess now.

## The gap report is the deliverable

Everything you reach for that is missing, awkward, or surprising, with the package
that owns it. Include the things you worked around successfully — those are the ones
nobody else will ever notice.

`TrackFocus(input.FocusID)` is already on that list as far as I am concerned: making
the caller mint and hold a focus identifier is more ceremony than GPUI's focus handle,
and if you agree, say so rather than absorbing it.

## On the manual check

Running the example to see hover and press respond is worth doing and is not
evidence. The static render is assertable and your `fakeFrame` tests cover it; keep
the eyeballing for the interactive half, and say in the report which half was which.

## Otherwise proceed as planned

The lifecycle, the default styling, the click dispatch and pseudo-state tests, and the
empty-options probe are all right. So is stopping after one widget.
