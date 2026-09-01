# window: input goes to the wrong place, and sometimes nowhere

Five things, all in `DispatchEvent` and the hit-region slice, plus two that arrive when
the packages below you land. `docs/audit.md` has the reasoning; the parts that are yours
are here.

Nothing in this round changes the `Frame` interface. That is deliberate — see "Not yet".

## 1. Pointer capture

`prompts/ui.md` reported this and it is the widest gap in the input layer. Press inside
an element, drag outside it, and the moves go to whatever is now under the pointer.
Text selection, sliders, resize handles, scrollbar thumbs, drag and drop and window-edge
resizing all need the same thing, so one fix unblocks four widgets rather than half of
one.

Capture automatically on pointer-down against the hit region under the pointer, release on
pointer-up. No `Frame` method: an element that has been pressed is the element that should
receive the drag, and asking it to opt in adds a call every widget would make.

Keep two things apart, because conflating them is the obvious mistake:

Routing. While captured, `PointerMove` and `PointerUp` go to the captured dispatch node
regardless of where the pointer is, including outside the window.

`IsActive`. True only while the pointer is inside the captured region's bounds, so a
button un-presses when you drag off it and presses again when you come back. That is what
every toolkit does and what users expect.

Click synthesis already requires the up to land on the same region as the down, which is
the correct rule and does not change.

## 2. Hit regions ignore the clip stack

```go
for i := len(regions) - 1; i >= 0; i-- {
    if regions[i].bounds.Contains(pt) { return ... }
}
```

No mask. A button scrolled out of a `ScrollView` is invisible and clickable, and so is
anything inside an overflow-hidden container. Any list, panel or editor built on top of
this is built on a lie.

The awkward part is phase. `PushClip` is paint-only; `RegisterHitRegion` is prepaint-only,
so at registration time there is no clip in force to intersect with.

Relax `PushClip` and `PopClip` to be legal in prepaint as well as paint, and intersect
each registered region with the mask on top of the stack. Keep the prepaint stack separate
from the scene's rather than pushing prepaint clips into `next.scene` — the scene takes
no primitives during prepaint, and a second stack is easier to reason about than one that
has to come out balanced across two phases.

You accept the relaxation; `element` then pushes and pops around children in
`Div.Prepaint` exactly as it already does in `Div.Paint`, and updates the phase rules in
its `Frame` doc comment. Implementer first, declarer second — that is the ordering rule
and it has broken `main` before.

Extend the `facet_debug` balance assertion to cover the prepaint stack too. An element
that pushes without popping in prepaint should panic there, not corrupt hit testing three
frames later.

## 3. Clicking a button blurs the text field

```go
if targetFocusID != 0 { w.RequestFocus(targetFocusID) } else { w.focusTree.Blur() }
```

A button registers a hit region and does not track focus, so pressing one blurs whatever
had focus. Clicking a toolbar should not clear the editor's caret.

The rule to implement: pointer-down on a focusable element moves focus to it; pointer-down
on anything else leaves focus alone. Clicking the empty background still blurs, because
that is a deliberate gesture and `docs/packages.md` already records it.

That leaves a gap worth naming rather than solving now: an element that wants to take
focus away without being focusable itself has no way to say so. If you find you need it,
report it — do not invent a flag.

## 4. Four scans where one would do

`resolveNextHitTest` calls `hitTest`, which walks the slice, then walks it again to find
the cursor of the region it just found. `DispatchEvent` walks it once for the hit and
again for the focus id, and a third time for the bounds. Every pointer move pays all of
it, and the slice is one entry per interactive element on screen.

Return the region, not its id. The lookups then disappear rather than being optimised.

The linear scan itself can stay this round. It is O(n) per event against a slice that is
a few hundred entries in any UI we can currently build, and replacing it with a spatial
index before the frame is incremental would be tuning the wrong thing.

## 5. Draw leaves state behind on the empty path

```go
el := w.rootView.Render(w.app)
if el == nil {
    w.phase = phaseNone
    return
}
```

The scene, the tab order, the clip depth and `needsPresent` keep whatever they had.
`Draw` also has no re-entrancy guard, and `RequestFocus` is legal during paint and calls
`ScheduleFrame`. Make the early return leave the window in the same state a completed
frame would, and make a re-entrant `Draw` a panic rather than a corrupted frame.

## 6. Two that wait on the packages below you

**The pump goroutine.** `window.New` starts a goroutine ranging over
`app.Foreground().Pending()` that nothing can stop, so a closed window and its whole
object graph stay reachable for the life of the process. `prompts/app.md` makes the
channel close on shutdown. Once that lands, make `Window.Close` end the goroutine and
prove it with a test that closes a window and observes the range return.

**IME dispatch.** `platform.IMECompositionEvent` exists and `DispatchEvent` has no case
for it. `prompts/platform.md` makes the Windows backend produce it. Route it when it
arrives; until then there is nothing to route.

## Not yet

**The layout tree is rebuilt every frame.** `w.layoutTree = layout.NewTaffyTree()` at the
end of `Draw` throws away a faithfully ported per-node cache before it can span a frame,
so a hover colour change re-solves the whole flexbox tree. It is the biggest single
performance defect in the tree and it is not this round, because keeping the tree needs
node identity that survives a frame, and element identity is a decision I have not taken
yet. Taking it badly is worse than waiting one round.

Do the part that does not depend on it: `nodeBounds` and `measureCallbacks` are maps keyed
by `layout.NodeID`, and the ids are dense integers within a frame. Slices indexed by id,
reset rather than cleared.

**Layers.** `scene` has `PushLayer` and `PopLayer` and they work; `Frame` does not expose
them, so there are no menus, tooltips or popups. Exposing them is right and it comes with
its first caller in `ui`, not before. `AGENTS.md`: an interface method with no caller is a
guess, and we have had two.

## Done when

    go test ./window
    go test -tags facet_debug ./window

pass, with tests that each fail if their own fix is reverted:

- press inside a region, move the pointer outside it, and assert the move reached the
  captured node and `IsActive` went false
- register a region inside a clip that excludes the pointer, and assert the hit misses
- focus a field, click a button with no focus id, and assert focus did not move

The first two are the ones that have never been testable before, so they are the ones
worth writing carefully.

Report before `element` changes `Div.Prepaint`, so the phase relaxation lands under you
first.
