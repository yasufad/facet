# element: I gave you the wrong model for hover

Most of this round is right and I will get to it. Start here, because the correction
is mine and it invalidates a design decision I handed you.

## What I told you, and what GPUI actually does

I said the hover answers come from the **rendered** frame, so a hover style is always
one frame behind. That is wrong.

GPUI recomputes the hit test *inside* the frame, between prepaint and paint
(`window.rs:3157`):

    root_element.prepaint(...)          // hitboxes inserted into next_frame
    self.mouse_hit_test = self.next_frame.hit_test(self.mouse_position);
    root_element.paint(...)             // is_hovered() consults it

Hitbox IDs are fresh every frame — a monotonic counter, never reset, a new ID per
insert (`window.rs:4740`). They work because the hit test is recomputed against the
frame just prepainted. An element asks "am I hovered" during paint and gets an answer
about the region *it registered moments earlier in this same frame*.

So there is no cross-frame identity, no carrying an ID forward, and no lag for a
hover style that only changes painting. A hover style that changes *layout* does lag
one frame, because layout ran before any region existed. That distinction is the
whole of it, and I flattened it into "always one frame behind".

`docs/architecture.md` now names the hit test as step 5 of the frame, and
`prompts/window.md` has the obligation to perform it.

## What that means for the code you wrote

`interactivity.hitRegionID` and `nextHitRegionID` carry state from one frame to the
next on a `*Div`. A `Div` does not survive a frame — the user's `Render` calls
`NewDiv()` again — so `hitRegionID` is zero on every real frame, and

    d.interactivity.hitRegionID != 0 && f.IsHovered(...)

is false forever. Hover and active styling never fire in a running application.

`TestHoverPseudoStyleTwoFrameLag` passes because it reaches into the field no caller
can reach:

    // Carry previous frame hit region ID into the element for simulation
    btn2.interactivity.hitRegionID = hitRegionID

That line is the test telling you the design does not work. When a test needs to set
unexported state that nothing in the framework sets, the thing under test cannot
happen in production. I would rather you had stopped there and said so — that
instinct is worth more than the passing test.

The fix follows from the corrected model: keep the ID from *this* frame's prepaint in
the field, read it during paint, and delete `nextHitRegionID` and the carry-forward
entirely. `RequestLayout` must not consult `IsHovered`; there is no region yet.

`IsHovered`, `IsActive` and `IsFocused` stay on `Frame` and stay callable in paint
only. Enforce that with the phase check you already have.

## What is right

`ClickEvent`, `MouseButton` and `Modifiers` in `element`, with `platform` nowhere in
the package. I checked: no production file imports it and the layering test passes.

`PushDispatchNode(DispatchNode)` and `PopDispatchNode()`. Bundling the context, focus
ID and every listener into the struct is exactly the shape asked for, and the
atomic-handoff test earns its place.

`Hidden()` and `Invisible()` now match Tailwind and GPUI.

`Occlude()`, and a test that an occluding box with no handlers still registers a
region.

## Done when

`nextHitRegionID` is gone and the hit region ID is this frame's.

Hover and active are evaluated in paint, never in `RequestLayout`, and a test proves
a hover style reaches the scene **without** any test writing to an unexported field.
If a test cannot be written that way, the design is still wrong and that is the
finding.

`doc.go` says what actually lags: a hover style that changes layout, by one frame,
because layout precedes the hit test. A hover style that changes painting does not.

## Worth carrying

The test that needs a back door is evidence about the design, not an inconvenience to
work around. This one printed its own diagnosis in a comment and was committed
anyway. Read that comment as a failure next time, and say so — I would rather receive
"this cannot work and here is why" than a green suite over a feature that never runs.
