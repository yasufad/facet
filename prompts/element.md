# element: response to the milestone 2 plan

The builder half is right and mostly mechanical — go. The interactivity half has one
layering violation, one interface problem, and one capability it needs and does not
have. And the two halves should not land together.

The constructor decision is recorded and reasonable, and identifying the five
non-element allocations was worth doing.

## Split the milestone

The full builder is a hundred-odd methods with no design in them. Interactivity is
most of GPUI's 5,200-line `div.rs` and has open questions below. Landing them
together produces a diff nobody can review and buries the interesting decisions in
property setters.

Builder first, on its own. Then interactivity. Say so when the first is done.

## 1 — `OnClick` puts `platform` in `element`

    OnClick(handler func(event platform.PointerEvent) bool)

`element` may not import `platform`, and this is not a hole in the table like `input`
was — the OS layer has no business in the element tree.

Every other listener you propose is fine, because `input.PointerEventHandler` and its
neighbours are named types in `input`; naming them does not require importing
`platform`. Only `OnClick` spells a `platform` type directly.

A click is not a platform event anyway — it is down and up on the same target, which
somebody has to synthesise. GPUI's `Interactivity` does it and hands the listener its
own `ClickEvent`. Do the same: declare a `ClickEvent` in `element`, carrying position
in `geometry` units, the button, and the modifiers. Then `element` owns the concept it
invented and imports nothing new.

## 2 — Nine new `Frame` methods, with a protocol between them

`Frame` goes from twelve methods to twenty-one, and seven of the new ones operate on
an implied "active dispatch node" established by `PushDispatchNode`. That is a
stateful protocol across an interface boundary: unbalanced push and pop is silent
corruption, and `SetFocusID` called with no node open is a question every
implementation has to answer separately.

Hand the node over in one piece instead:

    PushDispatchNode(node DispatchNode) input.DispatchNodeID
    PopDispatchNode()

where `DispatchNode` is a struct in `element` carrying the key context, the focus ID
and the handlers. Two methods instead of nine, the handlers are attached atomically
with the node they belong to, and "which node is active" stops being askable.

`prompts/window.md` says `Frame` is the third interface we commit to after
`Renderer` and `Platform`, and the one users' code sits closest to. Nine methods that
only make sense in a particular order is not the shape to commit to.

## 3 — Nothing can tell you whether the element is hovered

`Hover`, `Focus`, `InFocus` and `Active` all take a refinement closure, and nothing on
`Frame` reports the state that decides whether to apply it. As specified, they can be
stored and never evaluated.

The state lives in `window` — pointer position, the hit test, the focused node — so
`Frame` has to expose it. Roughly:

    IsHovered(HitRegionID) bool
    IsActive(HitRegionID) bool
    IsFocused(input.FocusID) bool

Two things make this more interesting than it looks, and both belong in `doc.go`:

The answers come from the **rendered** frame, not the one being built. That is the
whole point of `window`'s two-frame model — the pointer is over what is on screen,
which is last frame's geometry. Your hit region for this frame does not exist yet when
prepaint asks.

So a hover style is always one frame behind. That is correct and it is what GPUI does,
but it needs saying out loud, because the alternative — laying out, then hit testing,
then re-laying-out with the hover style applied — is a second layout pass per frame
and is how frameworks end up with a hover that flickers.

Agree the three method names with `window`'s agent before either of you writes them.

## 4 — Smaller

`Padding` and `Margin` take `style.Length`; `BorderWidth` and the radii take
`geometry.Pixels`. Border widths cannot be percentages, so that half is right, but
corner radii can be in CSS. Decide whether we support that and be consistent about
why, rather than by which type came to hand.

`Occlude()` is a good inclusion — an element that blocks the pointer without handling
anything is exactly the case people discover late.

## Done when — builder milestone

Every property `style.Refinement` exposes has a builder method, and a test sets one
of each and asserts it reaches `layout.Style` through `ToLayout`.

The tree-construction benchmark is re-run on a styled element, and `doc.go` says what
styling adds to the 420 ns floor.

Then stop, and we will look at interactivity with these four answered.
