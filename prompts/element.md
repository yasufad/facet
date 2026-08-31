# element: done, once you close it out

The correction landed properly. I verified rather than took it on trust:

- `nextHitRegionID` is gone from the element. The one left in `fakeFrame` is that
  double's own ID allocator, which is right.
- `interactivity_test.go` touches no unexported element field anywhere. The hover
  test builds a fresh `NewDiv()` per frame, exactly as a real `Render` would.
- I replaced the `f.IsHovered(...)` call in `Paint` with `false` and the test failed:
  `hovered: expected blue background from hover style, got {1 0 0 1}`. It can fail,
  which is the whole point.
- The phase rule fires. I called `IsHovered` during prepaint and it panicked.

`doc.go` gets the distinction right, including the subtle half: a hover style that
changes layout is not merely delayed, it is unreachable from inside the element, and
the way to have one is for the application to observe hover, put it in an entity, and
let the next frame's `RequestLayout` read it. That is a real frame of lag and the
sentence says so.

The package is complete for its scope: the interface, the three phases, `div`, the
full builder, interactivity, and the view bridge.

## Retiring it

Two steps, and the deletion is the second. The `docs/packages.md` entry is missing
everything decided since it was written. Add:

**The `Frame` contract.** It is a layer boundary and changes by explicit decision.
`PushDispatchNode(DispatchNode)` and `PopDispatchNode()` hand a node over whole, so
there is no implied "active node" to get wrong. `IsHovered`, `IsActive` and
`IsFocused` are valid during paint only.

**Hit regions are per-frame.** A region registered in prepaint is resolved by
`window` at step 5 and queried in paint of the same frame. Elements keep no identity
across frames and there is no element-keyed map. Say that plainly, because the
obvious implementation carries an ID forward, and that cannot work when the element
is rebuilt every frame.

**`ClickEvent` is ours.** A click is synthesised from down and up on the same target,
so `element` declares it in `geometry` units rather than naming a `platform` type.

**What an element costs.** 584 bytes per `Div`, one allocation, about 420 ns. Styling
adds no allocations. `NewDiv()` takes no arguments, which is what a future arena in
`window` has to work around, and that constraint should outlive this prompt.

Then set the `element` row in the README table, and delete this file.

## What comes after, for whoever picks it up

Nothing in `element` blocks `window` any more. `window` is the last package with real
design in it, and `ui` follows it.
