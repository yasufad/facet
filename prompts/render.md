# render: let a machine with no GPU say so

Small round, one seam, and it unblocks something that is currently costing money.

CI runs `go test -tags facet_debug ./...` on hosted runners with no graphics adapter.
`render/d3d11`'s readback tests create a real D3D11 device there, fail, and take every run
red with them. A permanently red CI is a smoke alarm with the battery out: nobody reads
it, nobody notices a genuine failure joining the standing ones, and the runs are billed
either way.

The distinction to encode is that **a test that cannot run in an environment is a skip; a
test that ran and got the wrong answer is a failure.** Conflating them is what destroyed
the signal.

## What to add

A sentinel on `render`, because every backend will need it and the callers branch on it:

    var ErrNoAdapter = errors.New("render: no compatible graphics adapter")

Then in `d3d11.New`, wrap it when device creation fails for that reason rather than for a
real one. `DXGI_ERROR_UNSUPPORTED` (0x887A0004) is the documented case, and a hardware
adapter simply not being present is the other. Keep the `hr` in the message — the sentinel
says which class of failure it is, the code says which failure:

    fmt.Errorf("create D3D11 device: %w (hr=0x%08x)", render.ErrNoAdapter, uint32(hr))

Read the failure codes at the SDK rather than from this prompt. `AGENTS.md` says why, and
this package is the one that has already been bitten by a plausible-looking constant.

Then `render/d3d11/d3d11_debug_test.go`'s `setupTestWindow` checks
`errors.Is(err, render.ErrNoAdapter)` and calls `t.Skipf` instead of `t.Fatalf` at line 54.
Only for that sentinel — every other error stays fatal, or you have built a test that
cannot fail, which is the thing this repository checks for hardest.

`window` has the same shape downstream in `window_debug_test.go`; that is `window`'s to
land and its prompt will say so once you report the sentinel exists.

## What this costs, and say it out loud

The readback assertions are the only thing standing between an unused shader and a broken
one — three wrong vtable indices and a batch offset both reached reviewed, green code, and
only pixel readback found them. Skipping on CI means those assertions run on a developer's
machine and nowhere else.

That is the right trade, because failing is not running either, and a red tick nobody reads
proves less than a green one that says what it covered. But it is a real reduction in what
CI guarantees, so record it beside the skip: a line in the test file saying these
assertions require a GPU and are not exercised by continuous integration. Do not let the
skip read as "this is covered".

## Not this round

The atlas free list and the page-occupancy work stay queued. They need a `Renderer`
interface change and an eviction policy that belongs with whoever owns `window`'s glyph
cache, and that decision is not taken. Your measurements from last round — 2 pages, ~69%
average occupancy, 58% on the final page — are recorded in `prompts/queued.md` and are
what that decision will be sized against.

## Done when

`go test -tags facet_debug ./render/d3d11` still runs and passes on a machine with a GPU,
with every assertion it had before.

The same command on a machine without one skips those tests and reports why, rather than
failing.

Break it: make `errors.Is` match every error and confirm a genuine device failure now
skips silently — that is the mistake this change invites, and the test suite should not
survive it.
