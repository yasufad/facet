# Assignment: app

The `app` package is implemented — the entity map, contexts, effect queue and
executors all exist and their tests pass. This assignment is to land it properly and
fix two defects found in review.

## State

Fourteen files in `app/`, none of them committed. `go build`, `go vet`, `gofmt` and
`go test ./app/` are all clean. The design is sound: keep it. In particular keep the
lease mechanism that makes a re-entrant update panic rather than alias, the
inert-insert-then-activate subscriber model, and `doc.go`, which is good.

Do not redesign. Two things are wrong; fix those.

## Defect 1 — the threading check costs 6.25µs

`goroutineID()` benchmarks at 6253 ns/op:

    func BenchmarkGoroutineID(b *testing.B) {
        for b.Loop() {
            _ = goroutineID()
        }
    }

`checkUI` calls it from thirteen public entry points — every read, update, notify,
observe, subscribe and emit. A frame touching a thousand entities spends six
milliseconds on the check, out of a sixteen millisecond budget.

The invariant is right and stays. The mechanism or the frequency has to change.
Candidates, in the order I would try them:

- Assert at boundaries rather than accessors: `update`, the `App` entry points that
  can be reached from outside an update, and `AsyncApp`. Be careful — an early
  return based on "we are already inside an update" is not sound, because a
  background goroutine can call while the UI goroutine is mid-update.
- Find a cheaper identity. Six microseconds is high even for `runtime.Stack`;
  measure whether the pool or the buffer size is the cost before concluding the
  approach is doomed.
- Gate the full check behind a build tag if it cannot be made cheap, and say so
  plainly rather than leaving a claim that it is free.

Measure first, then choose, and put the number in the commit message.

Whatever you land, the comment above `goroutineID` and the Threading section of
`doc.go` must stop claiming the check is cheap unless it has become cheap. Two
places currently say a few hundred nanoseconds and "cheap enough to leave on in
release builds". Both were written before anyone measured.

## Defect 2 — observer dispatch order is nondeterministic

`subscriberSet.retain` iterates a Go map, so subscribers fire in a different order
each pass. Five distinct orders across 200 runs of a five-observer set:

    abcde:105  bcdea:28  cdeab:16  deabc:27  eabcd:24

GPUI keys its `SubscriberSet` on a `BTreeMap` of `(entity, subscriber_id)`, so
dispatch is in registration order and reproducible. `remove()` has the same problem.

Dispatch must be in registration order. The monotonic `id` already exists; it is
just not being sorted on. Add a test that pins the order, because this is the kind
of regression that reappears quietly.

## Defect 3 — Subscription cannot be closed inline

`Subscription` is returned by value while `Close` and `Detach` have pointer
receivers, so `app.Subscribe(...).Close()` does not compile. Return a pointer, or
move the state behind one.

## Then commit it

Fourteen files, untracked. Conventional commits, one file per commit, as
`AGENTS.md` requires. Order them so the tree builds at every commit — types before
the code that uses them. The commit for a fix explains what was wrong; the rest
need no body.

## Done when

    go build -o bin/ ./...
    go test ./app/
    go test ./internal/layering
    go vet ./app/
    gofmt -l $(go list -f '{{.Dir}}' ./...)

The order test passes. The cost of the threading check is measured, stated in a
commit message, and described accurately wherever the code talks about it. Every
file is committed.

`layout` currently has failing tests and unformatted files. That is another agent's
work in progress, not yours — do not touch it, and do not let it stop you.

## Constraints that have not changed

The UI runs on one goroutine and a context used from another panics. The entity map
stays free of mutexes: it is single-goroutine by design, and adding one to make it
safe from elsewhere would remove the reason the rest of the design works. `app`
knows nothing about drawing — no geometry, no colours, no elements.
