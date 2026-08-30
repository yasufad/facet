# Assignment: app

The `app` package is implemented and committed. Three review fixes landed; one was
right and two introduced new defects. This round undoes the damage and finishes the
job properly.

## State

`c1aa1dd` narrowed the threading check, `37a2a80` made dispatch order deterministic,
`f0045c5` made `Subscription` closable inline.

`f0045c5` is correct — leave it alone.

The design underneath is still sound. Keep the lease mechanism, the
inert-insert-then-activate subscriber model, and `doc.go`. Do not redesign.

## Defect 1 — the threading invariant is no longer enforced

`Notify`, `Observe`, `Subscribe`, `Emit`, `Defer` and `OnRelease` are exported
methods on `App`. `c1aa1dd` removed `checkUI` from all six on the grounds that they
are "accessor methods called only from within an update". That is not true of
anything exported: any caller can reach them from any goroutine. Reproduction:

    func TestBackgroundDeferIsCaught(t *testing.T) {
        app := NewApp()
        defer app.Close()

        caught := make(chan any, 1)
        go func() {
            defer func() { caught <- recover() }()
            app.Defer(func(*App) {})
        }()
        if r := <-caught; r == nil {
            t.Error("App.Defer from a background goroutine did not panic")
        }
    }

It does not panic. The effect queue is mutated off the UI goroutine with no
synchronisation.

**This part is not a trade-off.** Every exported entry point that touches entity
state, the effect queue or the subscriber sets enforces the invariant. Restore that
first, before doing anything about cost. A slow framework is a complaint; silent
cross-goroutine corruption is a heisenbug in somebody's application months later.

Write the test above, and one like it for each of the six. Their absence is why this
regression passed — the three surviving wrong-goroutine tests only cover the paths
that kept the check.

## Defect 2 — the cost problem is still there

12.3ms per thousand update-and-notify cycles is roughly two 6µs checks per update.
Reducing the number of calls was mitigation, not a fix, and the per-check cost is
untouched.

Solve it as its own problem, with the invariant restored. The routes are a cheaper
identity mechanism, or compiling the expensive check out behind a build tag and
saying so plainly.

One candidate worth considering, not an instruction: a **generation counter**.
`Context[T]` records the update generation it was created in; accessors compare it
against the App's current generation with an integer compare, which is about a
nanosecond. That catches the realistic mistake — a context stored and used after its
update has ended — at every accessor, cheaply. Keep the full goroutine check at the
boundaries, where a few microseconds a few times per frame is affordable. Two
mechanisms, each placed where it is cheap. If you can see a hole in that, say so and
propose something better.

Whatever you land, measure it and put the number in the commit message, and make
sure the comment above `goroutineID` and the Threading section of `doc.go` describe
what is actually true.

## Defect 3 — dispatch now allocates

The order requirement from the last round is right and stays. The implementation is
not: `retain` builds and sorts an id slice on every dispatch.

    BenchmarkRetainDispatch-14   1358210   881.6 ns/op   120 B/op   3 allocs/op

Three allocations per notification, on the per-frame path. `sort.Slice` also goes
through `reflectlite.Swapper`, and `AGENTS.md` rules out reflection on a per-frame
path.

Sorting at dispatch time is the wrong shape. GPUI's `SubscriberSet` uses a
`BTreeMap`, so it is inherently ordered and never sorts when dispatching. Hold
subscribers in registration order — an ordered slice, or insertion in order — so
dispatch walks them directly. Dispatch should allocate nothing.

Add a benchmark asserting zero allocations, so this cannot come back quietly.

## Done when

    go build -o bin/ ./...
    go test ./app/
    go test ./internal/layering
    go vet ./app/
    gofmt -l $(go list -f '{{.Dir}}' ./...)

Every exported entry point that touches state panics when called off the UI
goroutine, with a test each. Dispatch is in registration order and allocates
nothing, with a benchmark pinning it. The threading check's real cost is measured,
stated in a commit message, and described accurately in the code.

Each fix is its own conventional commit whose body says what was wrong.

`layout` has failing tests and unformatted files. That is another agent's work in
progress — do not touch it, and do not let it stop you.

## How this went wrong, so it does not repeat

Both bad fixes were verified against the symptom rather than the invariant. The
benchmark got faster and the order test passed, so the work looked done. Neither
change was checked against the property it was supposed to preserve.

When you change how a guarantee is enforced, write the test that fails if the
guarantee is gone, before changing the enforcement.

## Constraints that have not changed

The UI runs on one goroutine and a context used from another panics. The entity map
stays free of mutexes: it is single-goroutine by design, and adding one to make it
safe from elsewhere would remove the reason the rest of the design works. `app`
knows nothing about drawing — no geometry, no colours, no elements.
