# app: OnRelease stopped firing, and main is red

Both parts of the round landed and the pointer storage is right. One thing is broken on
`main` and it is broken silently, which is the failure mode this round existed to remove.

## 1. OnRelease never fires

`go test ./app` fails at HEAD, on both tags:

    --- FAIL: TestOnReleaseFiresOnDrop
        entity_test.go:108: OnRelease callback did not fire when the entity was dropped
    --- FAIL: TestOnReleaseCascadesHandleDrop
        entity_test.go:139: owned entity should have been dropped when its owner was dropped

The map now stores `*T`. `ReadEntity` and `UpdateEntity` were both updated to assert
`v.(*T)`. This was not:

```go
return c.app.onRelease(AnyEntity{id: id}, func(value any, app *App) {
    t, ok := value.(T)
    if !ok {
        return
    }
    onRelease(&t, app)
})
```

`value` is a `*T`, the assertion fails, and `if !ok { return }` swallows it. No panic, no
log, no diagnostic — the callback simply never runs.

Fix the assertion, and then fix the swallow, because the swallow is the real defect. A
failed type assertion here means the entity map holds something other than what this
entity's type says it holds, which is a programmer error with no recovery — exactly the
case `AGENTS.md` says to panic on. `ReadEntity` and `UpdateEntity` already panic with a
message naming the id and the type. This should read the same way. Had it done so, the
two tests would have failed with a message pointing straight at the cause instead of at a
callback that did not run.

Grep the package for every remaining assertion on a value out of the map before you call
this done. Two of three were updated; the one that was not is the one with no panic on the
failure path, which is not a coincidence — the compiler could not help, and neither could
the code.

## 2. One commit contained two rounds

`8cec6a5` is titled "make the goroutine-identity check a facet_debug-only guarantee". Its
diff also contains the whole of part 1 — `insert(id, &value)`, both assertion changes, and
the new doc comments about aliasing.

I tried to bisect the `OnRelease` failure and got an incoherent answer, because the commit
that introduced the storage change does not say it did. `AGENTS.md` asks for one file per
commit so that diffs stay reviewable; the deeper reason is that a commit whose subject
does not describe its diff cannot be bisected, and this is the first time that has
actually cost something here.

Nothing to undo — the history stands. Land the fix above as its own commit that says what
it does.

## What is right

The pointer storage is correct and the doc comments you added around it are the best
prose in the package. Both of these say something a caller could not otherwise know:

    // The returned pointer aliases the value stored in the entity map, so a write
    // through it is visible to every other reader.

    // The pointer f receives is the one stored in the entity map, not the address
    // of a copy: a handler that captures it and writes later ... writes to the real
    // entity.

The `checkUI` split is right too, and the rewritten explanation in `goroutine.go` is
honest about the gap it opens rather than glossing it — including the sentence saying a
release build now has no protection against a live context used from the wrong goroutine.
That paragraph is what the prompt meant by prose being part of the guarantee.

One consequence to keep in view, since your part 1 note now makes it live: a handler that
captures the leased pointer and writes to it after the update ends now *lands* its write
instead of losing it. `element` has landed `Listener`, so there is a correct way to write
that handler, but the incorrect way is quieter than it was. `Context.checkGeneration`
staying in release is what still makes it loud, and it must not follow `checkUI` behind
the tag.

## 3. Then the executor

Unchanged from last round and not yet done. `ForegroundExecutor.stop` closes `done`, not
`wake`, so `window.New`'s goroutine ranging over `Pending()` never ends and holds the
platform, the app and the window alive for the process lifetime. `stop` also lacks the
`sync.Once` its background counterpart has, so a second `App.Close` panics on a closed
channel.

Close `wake` under a `sync.Once`, make `stop` idempotent, and check the send path does not
panic after shutdown. Say in the commit that `window` can now range to completion —
`window` has landed its round and will pick that up.

## Done when

    go test ./app
    go test -tags facet_debug ./app

both pass. Break the assertion fix and confirm the two `OnRelease` tests fail — and check
they fail with a message that names the cause, not one that says a callback did not run.

The executor tests close a window's foreground queue and observe the range return.
