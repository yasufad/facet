# Assignment: app

Implement the `app` package in Facet: the entity map, contexts, the effect queue and
the executors. This is the reactive core, and it is the package everything else
assumes is correct. Take the time.

## Read first

1. `AGENTS.md` — conventions, commit style, GB English
2. `docs/packages.md` — the `app` entry
3. `docs/architecture.md` — the state, frame and threading sections
4. `_upstream/gpui/crates/gpui/docs/contexts.md` — written by GPUI's authors on
   exactly this, and the single most useful thing to read
5. `_upstream/gpui/crates/gpui/src/app.rs` and `src/app/entity_map.rs`
6. `_upstream/gpui/crates/scheduler` — the executor

Run `go run ./tools/upstream` if `_upstream/` is not there.

## Build

**Entities.** `Entity[T]` is a handle — an identifier into a map the app owns — not
a pointer. Reads and writes go through a context. Handles are cheap to copy and
compare. A weak handle that does not keep its value alive is needed too, because
observers must not resurrect what they watch.

**Contexts.** `App` for the global surface, `Context[T]` for working on one entity,
and an async context that survives across an await point. Get the split right: what
each may reach, and which of them can exist at the same time. `contexts.md` is
explicit about this.

**Reactivity.** `Notify` marks an entity dirty. `Observe` runs when another entity
notifies. `Subscribe` receives typed events an entity emits. Effects queue during an
update and flush once at its end, so a burst of mutations produces one frame rather
than one each. Work out the flush order and what happens when an effect causes
another.

**Executors.** A foreground executor bound to the UI goroutine and a background pool,
with `Task[T]` for a result that arrives later and a way to return to the foreground
to touch state.

## Decisions already made

The UI runs on one goroutine. A context used from another panics with a message
naming the mistake rather than corrupting state quietly. That check is not optional
and must be cheap enough to leave on in release builds.

The entity map is not lock-free by accident — it is single-threaded by design. Do not
add a mutex to make it safe from other goroutines; that would remove the reason the
rest of the design works.

`app` knows nothing about drawing. No geometry, no colours, no elements. If a
concept here needs one of those, it is in the wrong package.

Reference counting decides when an entity is dropped. Cycles between entities holding
strong handles will leak — decide what to do about that and write it down.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test ./internal/layering
    gofmt -l $(go list -f '{{.Dir}}' ./...)

`doc.go` states what the package owns and its invariants. This package earns real
tests: effect ordering, observer and subscriber lifetimes, notification during a
flush, dropping an entity that others observe, and the cross-goroutine panic.

## Out of scope

Views, rendering, windows. A view is an entity whose notification schedules a
repaint, but the repaint belongs to `window` and the render method belongs to
`element`. `app` provides the mechanism and stops there.
