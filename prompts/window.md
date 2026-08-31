# window: the loop works, the reactivity is not connected

The frame loop is right, and I verified the parts that mattered most rather than
reading the walkthrough.

The two hit tests are genuinely distinct. I pointed step 5 at `rendered` instead of
`next` and the hover test failed; I pointed event routing at `next` instead of
`rendered` and the isolation test failed:

    expected hover background {1 0 0 1} in the same frame, got {0 0 1 1}
    expected click event to be dispatched to rendered frame handler

That is the trap the prompt warned about and you avoided it, with tests that can
actually fail. Phase enforcement, the two-frame swap, and the scene assertions are
all real.

Three things, and the first is the one that matters.

## 1 — Notifying an entity does not redraw the window

`docs/architecture.md` says a view repaints when the entities it holds change, and at
no other time. Nothing in `window` observes anything: there is no `Observe`, no
`Subscribe`, and no path from `cx.Notify()` to `Invalidate`. Step 1 calls
`app.Flush()` and discards what it learns.

    Notify on a view entity left the window clean: dirty = false
    Notify produced no repaint: render count still 1

A running application cannot update its own UI. Every redraw has to be asked for by
hand or provoked by a platform event.

This is the fourth line of "the wiring nobody else can do" in this prompt, and it is
the one that makes the reactive core reactive. `app` has `Observe` for exactly this:
the window observes the entities its root view reads, and a notification marks the
window dirty and schedules a frame.

`ScheduleFrame` and its `frameScheduled` flag are already correct — a burst
collapsing to one frame works once something actually calls it.

**And the test that should have caught this cannot.**
`TestFlushNotificationDeduplication` calls `w.Draw()` by hand and then asserts the
view rendered once. One `Draw` produces one render whether or not deduplication
exists, and whether or not notifications reach the window at all. Rewrite it to
count frames that happen *because of* the notifications, with no manual `Draw`.

## 2 — Nothing has ever drawn a pixel through the whole stack

Every window test uses `stubRenderer`. `examples/quad` is said to run through the
real loop, and it compiles, which `docs/packages.md` already says is not evidence for
this layer.

`render/d3d11` has the machinery: `setupTestWindow` in `d3d11_debug_test.go` creates
a real platform window with `Visible: false` and a real renderer, and eight tests
read pixels back through it. Nothing stops `window` doing the same under
`facet_debug`: build a `div` with a known background, run one `Draw`, read the back
buffer, assert the colour at a point inside it and the window background outside it.

That test is the whole point of the project being assembled. Until it exists, we have
eleven packages that each work and no evidence they work together.

## 3 — Every frame re-renders every view

`w.rootView.Render(w.app)` runs unconditionally, so when a frame does happen, the
whole tree is rebuilt regardless of what changed. That is a reasonable first
milestone and it is not what `doc.go` claims: "deduplicated invalidations" describes
`frameScheduled`, not invalidation of views.

Either implement dirty-view tracking or say plainly in `doc.go` that the root view
re-renders every frame and precise invalidation is not implemented yet. The second is
a fine answer; the current text reads as the first.

## Smaller

`w.mu` guards `frameScheduled`. Everything above `app` is single-goroutine, so say in
`doc.go` why this one field needs a mutex — presumably that `ScheduleFrame` is the
one method safe to call from a background goroutine. That is a reasonable exception
and an undocumented one reads as drift.

## Done when

Notifying an entity a view reads schedules exactly one frame, with a test that does
not call `Draw` itself.

A `facet_debug` test drives `window` with a real `d3d11` renderer on a hidden window
and asserts pixels for a styled `div`.

`doc.go` says what is actually true about invalidation, and why `w.mu` exists.

## Worth carrying

You found the two-hit-test distinction and implemented it correctly, which was the
hardest thing in this package. The gap is on the other side: the loop is complete and
nothing arrives to drive it. When a prompt lists four wirings and three are done, the
fourth is not a detail, and a test asserting the outcome of a manual call will never
notice it is missing.
