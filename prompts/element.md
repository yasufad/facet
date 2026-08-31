# element: builder milestone reviewed — one naming trap, then interactivity

Coverage is complete. I diffed every `Set*` on `style.Refinement` against `*Div`'s
methods: all eighty-one are reachable, the eight that do not match by name being the
deliberate short forms — `Bg`, `Border*`, `Cursor`. Nothing is stranded.

The allocation result is the one that matters and it holds:

    unstyled        3942 ns   6896 B   15 allocs
    fully styled    4103 ns   6896 B   15 allocs

Styling every element with a dozen properties adds no allocations at all and about
3% of construction time. The mutators-on-a-pointer decision is now paid off twice
over.

## 1 — `Hidden()` does not mean what anyone will expect

    func (d *Div) None() *Div     { SetDisplay(DisplayNone) }
    func (d *Div) Hidden() *Div   { SetVisibility(VisibilityHidden) }

GPUI's `hidden()` sets `display: none` (`styled.rs:59`), and so does Tailwind's
`hidden`, which is where GPUI took its vocabulary. Ours sets visibility.

So `div().Hidden()` leaves the element occupying its full space in the layout, which
is the opposite of what the person writing it wanted, and nothing tells them. It is a
silent behavioural difference in the API users touch most, and the kind that gets
discovered as "why is there a gap here" three weeks later.

Follow the convention:

    Hidden()      display: none
    Invisible()   visibility: hidden

`None()` can stay as an explicit alias or go; say which. Where we take a vocabulary
from somewhere, we take it whole — deviating on one word costs more than inventing a
different scheme would have.

Worth a general check while you are in there: any other method whose name matches
GPUI's or Tailwind's but whose behaviour does not.

## 2 — The benchmark numbers in `doc.go` do not reproduce

`doc.go` records ~5.0 µs unstyled and ~5.5 µs fully styled. I measure 3.9 µs and
4.1 µs on the same machine, twice each. Nothing is wrong with the code — the box was
busier when you ran it — but a recorded absolute that is 25% off is worse than no
number, because the next person treats a change to 5.0 µs as a regression.

Record what survives the machine: allocations per node, which are exact and
meaningful, and the styling overhead as a proportion rather than in nanoseconds. Keep
one absolute if you like, and say what it was measured on.

## Then interactivity

The four answers from the previous round still stand, and none of them has been
addressed yet because the milestone split deferred them:

- `OnClick` cannot name `platform`. Declare a `ClickEvent` in `element`, in
  `geometry` units — a click is synthesised from down and up on the same target, so
  it is ours to define.
- `Frame` takes `PushDispatchNode(DispatchNode) input.DispatchNodeID` and
  `PopDispatchNode()`, not nine methods sharing an implied active node.
- `Hover`, `Focus`, `InFocus` and `Active` need `IsHovered`, `IsActive` and
  `IsFocused` on `Frame` to be evaluated at all, and the answers come from the
  **rendered** frame, so a hover style is always one frame behind. Say that in
  `doc.go`. Agree the three names with `window`'s agent before either of you writes
  them.
- Corner radii: decide whether percentages are supported, and be consistent about
  why rather than by which type came to hand.

`Occlude()` was a good call. Keep it.

## Done when

`Hidden()` removes the element from layout, `Invisible()` hides it in place.

`doc.go` records allocations per node and the styling overhead as a proportion.

Then interactivity, with the four above settled first rather than discovered during.
