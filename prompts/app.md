# app: the entity map hands out copies

Three defects, all in this package, none of which change a single exported signature.
That is why this runs now, in parallel with everything else: nothing above you moves.

`docs/audit.md` has the full reasoning. The parts that are yours are below.

## 1. Store pointers, not values

`entityMap.values` is `map[entityID]any` holding a `T`. Every path in and out copies:

```go
func ReadEntity[T any](app *App, handle Entity[T]) *T {
    v := app.entities.read(handle.id)   // any
    t, ok := v.(T)                      // copies out of the interface box
    return &t                           // address of the local
}
```

The returned `*T` points at a stack local. Writing through it is discarded silently. I
probed it: write 42, read back 1, no panic, no warning. `Context.OnRelease` has the same
shape and so does the `endLease` write-back.

**Store `*T`.** `New` boxes the value once at construction; nothing boxes again.
`ReadEntity` returns the stored pointer and aliases the real thing. `UpdateEntity` hands
`f` the stored pointer instead of the address of a copy.

Keep the lease. `lease` still takes the pointer out of the map so a re-entrant update
finds an empty slot and panics, and `endLease` still puts it back — the guard is the
reason the mechanism exists and it does not change. What goes is the copying, in both
directions.

Measured before the change: one allocation per `UpdateEntity`, from `endLease` re-boxing
the value. After, it should be zero.

No exported signature moves. `ReadEntity` still returns `*T`; it just starts telling the
truth.

Write the probe as a test and keep it. Then break the fix and confirm the test fails — a
test that cannot fail is not a test, and this is a defect that survived five test files.

### What this does to the defect `element` is fixing

Worth knowing, because the two are in flight together and I only noticed by probing it.

Today a click handler that captures the leased `v` and writes to it after the update has
ended writes into a stack local that `endLease` has already superseded, so the write is
lost. Once the map stores `*T`, the captured pointer is the stored value, and that same
wrong code starts landing its write. I ran both leases side by side to be sure: value
lease loses it, pointer lease applies it.

The mutation half of the broken pattern therefore goes quiet under you. What still makes
it loud is `cx.Notify()` tripping the generation check — and a handler that mutates
without notifying was already silent and stays silent, so nothing gets worse than it is.

This does not change what you build. It changes why part 2 is bounded the way it is:
keeping the generation compare in release builds is now holding up correctness as well as
paying for itself on performance. Do not let it follow `checkUI` behind the tag, and say
in the doc comment that it is the check that survives, so the next person to look at the
two of them together knows they are not the same kind of guard.

## 2. Take the goroutine check out of release builds

`app`'s doc says the 6 µs check is affordable because it runs "at update boundaries" and
"a frame opens a few update boundaries". That sentence is wrong and it was mine. Every
`UpdateEntity` is an update boundary. Every `Notify` is an exported method. Every
observer firing opens another `UpdateEntity` of its own.

The measurement, from your own benchmark:

    BenchmarkFrameSimulatesUpdateNotify   4.87 ms / 1000 entities, 3745 allocs
    BenchmarkCheckUI                      3.16 µs
    BenchmarkCheckGeneration              1.50 ns

Two `checkUI` calls per entity update is most of the 4.87 ms. At 1000 entities that is
roughly 29% of a 60 Hz frame spent parsing `runtime.Stack` output.

`checkGeneration` already has the shape: `context_check.go` for release,
`context_debug.go` for `facet_debug`. Do the same to `checkUI` — an empty body behind
`!facet_debug`, the real check behind `facet_debug`. The generation compare stays in
release and still catches a context used after its update ended; the goroutine check
becomes a debug-build guarantee.

Then rewrite the paragraph in `goroutine.go` that explains the three mechanisms, because
it currently describes a placement the code no longer has. A comment stating a guarantee
is part of that guarantee.

Re-run the benchmark and put the new number in the commit body.

## 3. The foreground executor never stops

```go
func (fg *ForegroundExecutor) stop() { close(fg.done) }
```

`Pending()` returns `fg.wake`, which nothing closes. `window.New` starts a goroutine that
ranges over `Pending()` forever; it holds the platform, the app and the window alive for
the life of the process, and `Window.Close` cannot end it.

`stop` also has no `sync.Once`, unlike `BackgroundExecutor.stop` three types down. Calling
`App.Close` twice closes a closed channel and panics.

Close `wake` under a `sync.Once` so the range terminates, and make `stop` idempotent.
Anything that sends on `wake` after shutdown must not panic either — check the send path.

`window` owns the goroutine and will handle its side; you own making termination
possible. Say in the commit that `window` can now range to completion, so whoever picks
that prompt up knows it is there.

## Not in scope, deliberately

Manual reference counting is the other thing `docs/audit.md` raises about this package,
and it is a design question rather than a defect: `Clone` and `Release` have to be paired
by hand in a language with no destructor, and a missed `Release` leaks silently. Go 1.24
added `weak.Pointer[T]` and `runtime.AddCleanup`, which would let the collector own the
lifetime instead.

I am not deciding that this round. Changing it after `ui` has thirty widgets is much
worse than changing it now, so it will be decided soon — but not while three other
packages are mid-flight against the current handle semantics. If you hit something while
doing the above that changes the argument either way, say so.

## Done when

    go test ./app
    go test -tags facet_debug ./app

both pass, with a test that fails if the pointer storage is reverted.

`BenchmarkFrameSimulatesUpdateNotify` and `TestUpdateAllocs`-style probes show zero
allocations per `UpdateEntity` and the release-build cost of `checkUI` gone.

`goroutine.go`'s explanation matches where the checks now are.

Then report, with the before and after numbers.
