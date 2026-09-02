# input: the aliases are right and there are three more names

Reopened one round after you retired it, and both reasons are mine rather than yours.

What you landed is correct. The four aliases are aliases, the doc comment says why, the
reasoning about the wheel delta distinction is exactly right, and `main` at your last
commit builds, vets, formats and tests clean on all three platforms. I checked it in a
detached worktree rather than in the shared tree, because three other agents have
uncommitted work here and my first run reported their failures as yours.

## What I got wrong

I scoped the aliases from your four handler *signatures*. I should have read the call
site that was blocked. Here it is:

```go
s.viewport.OnScrollWheel(func(event platform.WheelEvent, phase input.DispatchPhase) bool {
    if phase != input.Bubble {
        return false
    }
    var deltaY geometry.Pixels
    if event.Delta.Unit == platform.ScrollPixels {
```

Two `platform` references, not one. `WheelEvent` is fixed. `ScrollPixels` is not, and no
alias you added could have covered it, because it is a constant rather than a type.

The second error made the first invisible. I wrote "`go test ./internal/layering` is green
with `ui` importing `input` and not `platform`" as your done-condition, which is an
outcome you cannot produce from inside your own package. So the one check that would have
caught the gap read as someone else's unfinished work, and the prompt retired over it. You
flagged the discrepancy in your report rather than passing over it, which is why this cost
a round and not a package.

## What to add

The scroll unit vocabulary, so a handler body can read a wheel delta:

    type ScrollUnit = platform.ScrollUnit

    const (
        ScrollPixels = platform.ScrollPixels
        ScrollLines  = platform.ScrollLines
    )

Constants cannot be aliased, so those two are declarations — but they are typed constants
of an aliased type, so `input.ScrollPixels` and `platform.ScrollPixels` remain the same
value of the same type and comparison across the boundary keeps working. Confirm that
rather than taking it from me; it is the whole reason this is safe.

`ScrollDelta` is the open question and I am not deciding it for you. `ui` reaches it only
through `event.Delta`, so nothing names it today and an alias for it would have no caller.
Add it if you find a caller, leave it if you do not, and say which you did.

The test to write is the one that was missing: a file in `input` that names only `input`
types and constructs and inspects a wheel event end to end. If it compiles, `ui` can drop
`platform`. That is the assertion, and it belongs here rather than in `ui`, because it is
your vocabulary that either covers the case or does not.

## The rule this round establishes

Alias what a caller above has to *write*, not what your own signatures happen to name.
The test is not "does my API mention `platform`" but "can someone above me write a working
handler without importing it".

Apply it once more before reporting: `element/interactivity_test.go` imports `platform` to
build a `KeyEvent`, so there is a second consumer of this vocabulary and it may want names
you have not added. Read what it constructs. Anything it has to reach into `platform` for
is in scope for this round — do not fix that file, it is `element`'s, but do make sure it
would not have to.

## Done when

A test inside `input`, importing no `platform`, constructs a wheel event, reads
`Delta.Unit` against `ScrollPixels` and `ScrollLines`, and asserts on both.

`docs/packages.md`'s `input` entry records the rule above alongside the alias reasoning
already there — the vocabulary is what a caller must write.

Then retire this prompt again, and this time the README row goes with it: `input` moves off
`in progress` in the status table. That is step one of retiring and it was missed, which is
mine too for not saying it in the prompt when `AGENTS.md` already says it.

`ui` going green is `ui`'s landing, not yours. It is no longer your done-condition.
