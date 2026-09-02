# app: the concurrency paths nothing exercises

Last round is closed. `OnRelease` fires again, the swallow is a panic pinned by a test
that fails when reverted, and the executor terminates. I checked the panic fix and found
something better than you claimed for it: reverting the assertion to `value.(T)` no longer
compiles, because `onRelease` takes `*T` directly now. The bug that took `OnRelease` out
for a round is a compile error rather than a runtime one. Say that on the function if it
is not already there — it is the kind of property that gets lost in the next refactor.

Two of the three items below are defects by inspection with no test that can see them, and
the third is a decision I have been deferring and should stop deferring.

## 1. AsyncApp.Update after App.Close blocks for ever

`run` enqueues a closure and blocks on a result channel. The shutdown check that returns
"application has been shut down" is *inside* that closure, so it only runs if something
drains a queue nothing is draining any more.

The error path the doc comment promises is unreachable. Check before the enqueue.

## 2. rc.shutdown is read without its mutex

`async.go:51` and `:83` read `rc.shutdown` from background goroutines while `markShutdown`
writes it under `r.mu`. A race by inspection, on the one type documented as crossing await
points.

No test exercises concurrent shutdown, and item 1 is why: the obvious test deadlocks
before the read ever happens. Fix 1 first, then write the test that would have caught
both, and run it under `-race`. A race the race detector cannot reach because the code
deadlocks first is worse than one it can.

## 3. Decide the ownership model

I have deferred this twice and said both times it would be decided soon. Deferring it
again costs more than deciding it, so this round it gets an answer.

`Entity[T]` is reference counted by hand. Go has no destructor, so a missed `Release`
leaks the entity and every observer registered against it with no diagnostic, and a double
`Release` panics somewhere unrelated to the mistake. `Context.Observe` spends four
refcount operations per observer per notification and carries a comment explaining which
handles are borrows.

What changed since I last deferred it: `element.Listener` landed, and it does an `Upgrade`
and a deferred `Release` **per event**. Manual pairing is no longer confined to setup code
holding a few long-lived handles; it is on the pointer-move path. I also removed that
`defer` in review and no test in the tree noticed.

So the question is live and it is not mine to answer from the outside. **Spike it and
report before changing anything.** Specifically:

Does `weak.Pointer[T]` plus `runtime.AddCleanup` actually express what the entity map
wants — the map owns the value, handles are ordinary reachable references, an entity drops
when nothing can reach it? The parts I want measured rather than argued: whether a cleanup
can run `OnRelease` at all, given it runs on a separate goroutine and every exported entry
point here asserts the UI goroutine; whether the lease survives, since leasing takes the
value out of the map and a weak reference to something temporarily unreachable is exactly
the case to get wrong; and what a notification costs afterwards against the four refcount
operations it costs now.

If it does not work, that is a good answer and I want the reason written down, because it
will be asked again. If it does, it is a large change and it lands on its own, with
`element` and `ui` told first.

Do not start the migration inside this round. The spike is the deliverable.

## 4. Small, and it belongs with 1 and 2

`NewApp` binds the UI goroutine to whichever goroutine constructs it, and nothing says so.
Every `AsyncApp` operation reaches `app.update` and its goroutine check, which only
succeeds if the foreground queue is drained on that same goroutine. `examples/button` is
correct by accident of ordering. Build an `App` anywhere else and every background update
panics from inside the platform dispatcher, naming neither mistake.

State the invariant on `NewApp`. Then decide whether it can be checked rather than only
documented — a constructor that records its goroutine and a `Drain` that panics when
called from another one would catch it at the point of the error instead of three frames
later.

## Done when

    go test ./app
    go test -tags facet_debug ./app
    go test -race ./app

all pass, with a concurrent-shutdown test that reaches the read in item 2 rather than
deadlocking before it, and that fails under `-race` if the mutex is removed.

The spike is reported, with numbers, before anything in item 3 is written.
