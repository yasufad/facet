# build: watching, not building

The comment is fixed and reads correctly — it says the readback tests need a real adapter,
that they fail on hosted runners, that the run is red for that reason, and that a skip is
assigned. Nothing about the workflow is outstanding.

`render` has that skip now. `prompts/render.md` carries your spec as you wrote it: the
sentinel, the wrap at `DXGI_ERROR_UNSUPPORTED`, and `errors.Is` in `setupTestWindow`.
Naming the exact `hr` and the exact call site is why it went out without a clarifying
round.

This round is three things to watch and one to check, not to build.

## 1. When render lands the sentinel

Confirm the `facet_debug` step actually goes green on the Linux leg. That is the first
moment this workflow produces a signal anyone can act on, and it is worth watching rather
than assuming — a skip that silently matches every error would also go green, and would be
worse than the red.

Then change the comment again, in that commit, to say the readback assertions skip without
an adapter. Not before.

## 2. When window and ui land their tests

`window` has landed `TestClickMutatesEntityStateAndRendersNextFrame`. Confirm it ran in
CI, not just locally — you established `go test ./...` reaches recursively, so it should,
but "should" is what this whole workflow exists to replace.

`internal/integration` still does not exist. It is assigned to whoever holds `ui`. When it
appears, check the same thing.

## 3. The one thing to check now

`main` does not build:

    ui\button.go:163:9: b.div.Opacity undefined

`element` deleted the setter, `ui` has not caught up, and that has been true for a while.
CI would have said so on the first push after the deletion — except the workflow only runs
on `main` and on pull requests now, and these commits are landing directly.

That is worth a look rather than a change. Does the current trigger actually catch a broken
`main`, given how work reaches it here? If commits land on `main` directly then yes, the
push trigger fires and this would have been caught. If they land some other way, the
scoping I did to save Actions minutes has a hole in it, and I would rather know than
assume. Report what you find; do not widen the triggers without saying why.

## Not yours

`internal/layering` stays red until `ui` swaps one import. Leave it.
