> **Read before starting.** You are picking up one package in a framework several agents
> build in parallel. `AGENTS.md` loads automatically and is the standard you are held to —
> read it, and read your package's entry in `docs/packages.md`, before you touch code. This
> file is only the work outstanding; it does not restate what those two say.
>
> The four things that cause the most damage here, in order:
>
> 1. **Commit one file per commit, by path:** `git commit -m "..." -- path/to/file.go`.
>    Never `git add -A`, never `git add .`, never `git commit -a`. The git index is shared
>    with other agents — staging then committing lets another agent's file land in your
>    commit. Three agents have had that happen. Check `git show --name-only` after every
>    commit; it should name exactly the file its message describes. The exception is a
>    change that is not true in pieces (a rename, or code plus the comment stating its
>    guarantee) — say so in the message when you use it.
> 2. **Stay in your package.** If your change needs another package's exported API to move,
>    stop and report it. Do not edit across the boundary unless this file explicitly says
>    you may.
> 3. **Verify at HEAD, not in the working tree.** Other agents have uncommitted files here,
>    so `go build ./...` in this checkout reports their half-finished work as yours. Use
>    `git worktree add --detach <scratch>/check HEAD`, run the commands there, remove it.
> 4. **Break your fix and confirm the test fails.** A test that cannot fail is not a test,
>    and every package here has passed its own tests while getting something wrong. Watch
>    for the break silently not applying — check `git diff --stat` before believing a green
>    run, and for an unused variable turning your break into a build error rather than a
>    failure.
>
> Report back when done rather than expanding scope. Do not edit `docs/`, `README.md`,
> `AGENTS.md` or `prompts/` — those belong to the reviewing agent, who writes up what your
> package guarantees once it lands.

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
