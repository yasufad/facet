# build: the comment says the tests skip, and they do not

Your verification of the trigger and matrix changes closes those out — one run per event,
superseded runs cancelled, cross-compilation still covering `darwin/arm64` from both legs.
Nothing further there.

Your spec for `render` was precise and is assigned. `prompts/render.md` carries it as you
wrote it: the sentinel, the wrap at `DXGI_ERROR_UNSUPPORTED`, `errors.Is` in
`setupTestWindow`, and the downstream `window` case. Naming the exact `hr` and the exact
call site is why it could be assigned without a round of clarification.

## The one thing to fix

`4f0ba7d` says:

    # Pixel readback tests in render/d3d11 and window skip when no
    # hardware adapter is available, so a GPU-less runner proves the
    # non-readback assertions only.

`render.ErrNoAdapter` does not exist. `render/d3d11/d3d11_debug_test.go:54` still calls
`t.Fatalf`. The tests fail exactly as they did before, and every run is still red.

The second half of that sentence is what a GPU-less runner will prove *once the skip
lands*. The first half is not true yet, and together they read as a description of the
current workflow.

This matters more than a stale comment usually does. The permanent red was the problem you
were sent to fix; a reader who takes this comment at face value will conclude the red means
something else and go looking in the wrong place. That is the confusion the change was
meant to end, now written into the file.

Say what is true today: the readback tests need a real adapter, they fail on hosted
runners, the run is red for that reason, and a skip is assigned to `render`. Change it
again in the commit that first sees the skip working — not the commit that expects it to.

## Then

When `render` reports the sentinel, confirm the `facet_debug` step actually goes green on
the Linux leg. That is the moment the workflow starts being readable, and it is worth
watching rather than assuming.

`window`'s click-to-state test and `internal/integration` both still need to appear. You
established that `go test ./...` reaches them recursively, so nothing in the workflow has
to change — but check they ran, once they exist. A test in a package CI never reaches is
worse than no test, because the tick implies it passed.

`internal/layering` should stay red until `ui` swaps one import. Leave it.
