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

# build: CI exists, and it is telling the truth badly

Both items this prompt was written for have landed. `tools/compile_shaders` carries
`//go:build windows` and the tree cross-compiles; `.github/workflows/ci.yml` exists and
runs the five commands on a matrix. That was the single highest-value change anyone has
made here — the two defects it was meant to catch had both already happened, and both were
found by hand.

What is left is that the workflow, as built, spends money it does not need to and reports
a result nobody can act on.

## 1. I changed the triggers; understand why before touching them

There is an unpushed commit on this branch altering `ci.yml`. It was mine, not yours, and
the reasoning belongs to you now.

`on: push:` was unscoped and sat beside `pull_request:`, so a branch with a PR open ran
the whole matrix twice per commit and every other branch ran it once. Several agents commit
here daily. It is now `push: branches: [main]` plus `pull_request:`, with a `concurrency`
group so superseded runs cancel instead of finishing.

The macOS leg is gone. GitHub bills macOS at 10x and Windows at 2x, so a three-OS matrix
spent about 13 minute-equivalents per wall minute and macOS was over three quarters of it.
There is no Cocoa backend, so that runner executed exactly the platform-independent
packages Linux already covers; the only thing it proved was that `darwin` compiles, and the
Linux leg cross-compiles `GOOS=darwin` at 1x. It returns in the round that adds the
backend, when it starts testing something Linux cannot.

That prompt named no budget, which is why it was built the other way. My omission.

## 2. Every run is red, permanently, and that is the real problem

The workflow runs `go test -tags facet_debug ./...` on runners with no GPU, where
`render/d3d11`'s readback tests create a real D3D11 device and fail. The comment says this
is "left visible on purpose."

I understand the instinct and it is the wrong call. A CI that is always red is a smoke
alarm with the battery out: nobody reads it, nobody notices when a genuine failure joins
the permanent ones, and we are paying for the runs. The point of CI is a signal that
changes when the tree changes.

Fix it so the run is green when the tree is healthy and red when it is not:

Those tests should **skip** when no adapter is present, not fail. A test that cannot run in
an environment is a skip; a test that ran and got the wrong answer is a failure. Conflating
them destroys the signal. The detection lives in `render/d3d11` — it already creates the
device and can report the absence — so that part is not yours.

**Report what you need from `render` and I will assign it.** Do not reach into that package
yourself.

You did report it, precisely and correctly, and it is assigned — `prompts/render.md` now
carries the sentinel, the wrap at `DXGI_ERROR_UNSUPPORTED`, and the `errors.Is` skip,
built from your spec.

**But the comment you committed with it describes work that has not happened.** It reads:

    # Pixel readback tests in render/d3d11 and window skip when no
    # hardware adapter is available

`render.ErrNoAdapter` does not exist and `d3d11_debug_test.go:54` still calls `t.Fatalf`.
Those tests do not skip; they fail, exactly as before, and every run is still red.

That is the failure this repository names most often — a comment stating a guarantee the
code does not make. It is worse here than in most places, because someone reading the
workflow will now believe the red tick means something other than the missing skip, which
is the specific confusion the change was meant to end.

Fix the comment to describe what is true today: the readback tests require a GPU, fail on
hosted runners, and a skip is assigned to `render`. Change it again when that lands, in
the same commit that first sees it work.

## 3. Check the two tests land in the run

Neither exists yet and both are assigned:

`window` owns the click-to-state test — a view with real entity state, a frame driven
through `Draw`, a synthetic pointer down and up through `DispatchEvent`, asserting the
entity changed and the next frame reflects it.

`internal/integration` owns the widget-level one, because `window` may not import `ui` and
`ui` may not import `window`. That directory does not exist yet either.

`go test ./...` picks both up automatically. Your job is to confirm they actually ran once
they land — a test in a package CI never reaches is worse than no test, because the tick
implies it passed.

## Done when

A push to `main` and a pull request each produce exactly one run.

The run is green when the tree is healthy. Right now `internal/layering` genuinely fails
and should stay failing until `ui` lands its one-line fix — that is a real signal and the
first thing this repository's CI will ever have told anyone. Do not configure around it.

The GPU-dependent assertions skip rather than fail, once `render` gives you the detection.
