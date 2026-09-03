> **Read before starting.** You are picking up one package in a framework several agents
> build in parallel. `AGENTS.md` loads automatically and is the standard you are held to —
> read it, and read your package's entry in `docs/packages.md`, before you touch code. This
> file is only the work outstanding; it does not restate what those two say.
>
> The four things that cause the most damage here, in order:
>
> 1. **Commit one file per commit, by path:** `git commit -m "..." -- path/to/file.go`.
>    Never `git add -A`, never `git add .`, never `git commit -a`. The git index is shared
>    with other agents — staging then committing lets another agent's file land in your
>    commit. Three agents have had that happen. Check `git show --name-only` after every
>    commit; it should name exactly the file its message describes. The exception is a
>    change that is not true in pieces (a rename, or code plus the comment stating its
>    guarantee) — say so in the message when you use it.
> 2. **Stay in your package.** If your change needs another package's exported API to move,
>    stop and report it. Do not edit across the boundary unless this file explicitly says
>    you may.
> 3. **Verify at HEAD, not in the working tree.** Other agents have uncommitted files here,
>    so `go build ./...` in this checkout reports their half-finished work as yours. Use
>    `git worktree add --detach <scratch>/check HEAD`, run the commands there, remove it.
> 4. **Break your fix and confirm the test fails.** A test that cannot fail is not a test,
>    and every package here has passed its own tests while getting something wrong. Watch
>    for the break silently not applying — check `git diff --stat` before believing a green
>    run, and for an unused variable turning your break into a build error rather than a
>    failure.
>
> Report back when done rather than expanding scope. Do not edit `docs/`, `README.md`,
> `AGENTS.md` or `prompts/` — those belong to the reviewing agent, who writes up what your
> package guarantees once it lands.

# window: the test the audit asked for, and IME has a producer now

Both findings closed and I verified both by breaking them. Dead-coding
`resolveCapturedHitTest` leaves the old test passing and fails
`TestPointerCaptureDoesNotActivateAnotherRegion` — which is exactly the discrimination
that was missing, demonstrated rather than asserted.

The pump work is better than what I asked for. I asked you to end the goroutine; you made
`Close` block on `pumpDone` so termination is observable rather than hoped for, which is
why the test fails in 0.12s instead of hanging. And the `case _, ok := <-pending` guard is
the thing I went looking for and expected to find missing: after `App.Close` closes `wake`,
a select on a closed channel spins, and you handled it.

## 1. The test docs/audit.md asked for, which nothing has written

`prompts/build.md` has been carrying this and it names `window` as its home. It was
blocked on `element` landing the listener seam. That landed several rounds ago and nothing
picked this up, which is my tracking failure rather than yours.

Nothing in `window/*_test.go` references `element.Listener`. So the framework's central
path — a view whose state changes because someone clicked it — is still not exercised
anywhere above `element`'s own unit tests.

Write it. Build a view with real entity state, give it a `Div` with an
`element.Listener`-wrapped `OnClick`, drive a frame through `Draw`, dispatch a synthetic
pointer down and up through `DispatchEvent`, and assert two things: the entity changed,
and the next frame reflects it. Against the fakes `window_test.go` already has, so it
needs no GPU.

`docs/audit.md` says this one test would have caught three of the four blocking defects it
opens with. Two of those three are fixed now; the value is that the path executes at all,
and that it keeps executing.

Break the listener and confirm it fails.

## 2. IME dispatch — unblocked

`platform` has landed. `window_windows.go` emits `IMECompositionEvent` from
`WM_IME_STARTCOMPOSITION`, `WM_IME_COMPOSITION` and `WM_IME_ENDCOMPOSITION`, with `Cursor`
converted to a rune offset. `DispatchEvent` still has no case for it.

Route it. `input.DispatchText` currently delivers only to the focused node with no capture
or bubble phase, unlike the other three dispatchers — that asymmetry is recorded as an open
question in `prompts/queued.md` and composition is the event that makes it concrete. If
routing IME to the focused node alone is the right contract, say so and I will write it
down; if you find you need the phases, that is a finding and `input` owns the change.

There is no consumer above you yet — `ui`'s text field is being built now. So route it and
prove it arrives with a test; do not invent an API for a caller that does not exist.

## 3. The part of the frame work that does not need element identity

Still outstanding from last round:

```go
nodeBounds        map[layout.NodeID]geometry.Bounds[geometry.Pixels]
measureCallbacks  map[layout.NodeID]element.MeasureFunc
```

`layout.NodeID`s are dense integers within a frame. Slices indexed by id, reset rather
than cleared. This is per-frame allocation on the layout path and it does not depend on
anything I have not decided.

## Still not yet, unchanged

**The layout tree is rebuilt every frame.** `w.layoutTree = layout.NewTaffyTree()` throws
away a faithfully ported per-node cache before it can span a frame. It is the biggest
single performance defect in the tree and it needs node identity that survives a frame,
which needs the element identity decision. `docs/audit.md` now has GPUI's actual mechanism
read from source rather than inferred — a path from the root, state carried between the
two frames you already keep, reachable in prepaint. I have not taken that decision and
taking it badly is worse than waiting.

**Layers.** `scene` has them, `Frame` does not expose them, and they arrive with their
first caller in `ui`, not before.

## Done when

    go test ./window
    go test -tags facet_debug ./window

pass, with the click-to-state test failing when the listener is broken, and an IME
composition arriving at a focused node in a test.

Report on the `DispatchText` phase question either way — it is a contract and it should be
written down rather than left as an accident of which dispatcher was written first.
