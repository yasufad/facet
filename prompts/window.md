# window: the capture test agrees with the bug it is testing for

Five of six items landed and the work is good. The prepaint clip relaxation is exactly the
shape asked for — a separate stack, `RegisterHitRegion` intersecting against it, and the
`facet_debug` balance assertion extended to cover it. `element` is unblocked and has been
told so. Returning the whole hit region from `hitTest` removed the three redundant scans
cleanly, and `platform` can now take its `SetCursor` half; I have told it the function has
settled.

One test does not test what its name says.

## The capture test cannot fail on half its claim

`TestPointerCaptureRoutesMoveOutsideRegionAndClearsActive`. I dead-coded the entire
capture-aware hit resolution:

```go
func (w *Window) resolveNextHitTest() {
    if false {                      // was: if w.captureActive
        w.resolveCapturedHitTest()
        return
    }
```

The test still passes.

The routing half is real — the `moveEvents` assertion runs through `DispatchEvent`, which
my edit did not touch, and that half would catch a regression. The `ClearsActive` half
cannot. The test drags to (5000, 5000), and at that point the plain `hitTest` path misses
every region too, so `activeHitRegion` becomes 0 and the background reverts either way.
The assertion is satisfied by the fix and by its absence, which is the "check the
assertion knows the answer" case in `AGENTS.md` — it is a correct expectation that
distinguishes nothing.

`resolveCapturedHitTest` is therefore entirely unprotected, and what it prevents is not
obscure:

```go
for _, hr := range w.next.hitRegions {
    if hr.nodeID != w.captureNodeID {
        continue
    }
```

Only the captured node is eligible. Without it, `hitTest` finds whatever is under the
pointer, and since `w.pointerDown` is true during a drag, that region becomes
`activeHitRegion`. Press button A, drag onto button B, and **B lights up as pressed**.
That is the bug, and every toolkit gets it right, which is why the prompt separated
routing from `IsActive` in the first place.

The discriminating test drags onto a *second* hit region rather than into empty space:
two adjacent buttons, press A, move over B, draw, and assert B is not active and A is not
active either. Break `resolveCapturedHitTest` and confirm it fails. The existing test is
worth keeping — it covers routing and the outside-the-window case — but it needs the
sibling that sees the difference.

This is the second time this round a test has been written well and pointed at empty
space. It is the most valuable check we have and the easiest to satisfy accidentally.

## Item 6 is now unblocked

`app` has landed `ForegroundExecutor.stop` closing `wake` under a `sync.Once`, so the pump
goroutine can terminate. Make `Window.Close` end it and prove it with a test that closes a
window and observes the range return.

IME dispatch still waits on `platform` producing `IMECompositionEvent`; that half of its
prompt is live and not yet reported.

## What is right

The comment on `resolveNextHitTest` explains why the captured node is found by `nodeID`
rather than by point — hit region ids are minted fresh every frame and the id from the
capture frame never appears again, while dispatch node ids are stable for an unchanged
tree. That is the non-obvious part of the whole mechanism and it is written down where
someone will hit it.

Refreshing `captureBounds` from the current frame's region is right too, and not what a
first implementation does. A drag that resizes the element it started on keeps working.

Your process note: you found `git commit -m` picking up another agent's staged file, and
you found the fix — `git commit -m "..." -- <path>` commits only the named path whatever
else is staged. Three agents hit this; you are the one who closed it. It is in `AGENTS.md`
now, credited to this round.

## Done when

    go test ./window
    go test -tags facet_debug ./window

pass, with the second-region capture test failing when `resolveCapturedHitTest` is
disabled, and `Window.Close` ending the pump goroutine.
