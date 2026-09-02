# build: one build tag from compiling everywhere, and still nothing checks it

This is not a package, which breaks the shape of every other file here. It gets a prompt
anyway because it is the one change in `docs/audit.md` that would have prevented most of
that document: every package passes its own tests, and things keep breaking between them.

There is still no `.github/`. Nothing has ever built this repository except whoever
happened to be sitting in front of it, on whatever operating system they had.

## What that has cost, updated

Since this prompt was last written: `element` landed a prepaint clip and left the exported
test double enforcing the old rule, so `go test ./ui` panics and has for two rounds —
found by me running it by hand, not by anything automatic. `app` shipped a type assertion
that stopped `OnRelease` firing on every entity, silently, and it sat on `main` until a
review caught it. The layering test has been red the entire time.

Every one of those is a green package and a broken tree.

## 1. tools/compile_shaders

`main.go` uses `syscall.NewLazyDLL` and `syscall.SyscallN`, which exist on Windows only,
with no build constraint.

**This is now the only thing stopping `go build ./...` on Linux and macOS.** `platform`
landed `platform_other.go`, so the examples and `platform` itself cross-compile clean —
I verified both targets. One `//go:build windows` on this file and the whole tree builds
on all three for the first time.

It is genuinely a Windows tool: it drives the HLSL compiler. The constraint is the fix.
`go build`, `go vet` and `go test` all skip a package whose files are entirely excluded,
with no error — I checked. A stub for the other platforms is not needed and would be
worse.

Land this first and on its own. It is one line and it changes what every other agent's
`go build ./...` means.

## 2. Continuous integration

A workflow on every push and every pull request, over the three targets:

    go build -o bin/ ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)
    go test ./...
    go test -tags facet_debug ./...

`gofmt -l` exits zero whether or not it prints anything, which is the trap in every
version of this check ever written — it has to fail the job when it prints. Scope it
through `go list` rather than `gofmt -l .`, which walks into `_upstream/` and reports
every upstream file.

Cross-compilation is a stated invariant rather than a convenience, so build the other two
targets from each host as well, with `CGO_ENABLED=0`. A change that breaks
cross-compilation should fail on the machine that made it.

The `facet_debug` run matters more here than the default one. It carries the clip-stack
balance assertions, the context threading check, the atlas generation check and the
renderer's pixel readback, and `AGENTS.md` is explicit that skipping it means those never
execute. Several of those readback tests need a GPU and will not run on a hosted runner —
say so in the workflow rather than letting a green tick imply they did.

Do not add a linter. There is none configured and choosing one is a decision, not
housekeeping.

## 3. The test that would have caught all of it

This is no longer blocked. `element` landed `Listener` and `PhasedListener`, so a handler
can reach its entity, and a real dispatch can be driven end to end.

It is still not yours to write. Two tests are wanted and they live in different places:

`window` owns the one its prompt describes — build a view with state, drive a frame,
dispatch a synthetic pointer down and up through `window.DispatchEvent`, assert the entity
changed and the next frame reflects it, against the fakes `window_test.go` already has.

`internal/integration` owns the widget-level one, because `window` may not import `ui` and
`ui` may not import `window`, so no package can test a `ui.Button` clicked in a real
window — the path `examples/button` demonstrates and the one that has never been run.
`docs/packages.md` records why that directory exists. It goes to whoever holds `ui`.

Your job is to have CI standing when they land, and to say in the workflow what it is for.
When they report, come back and check both are in the run.

## Where the files go

`.github/workflows/` is a shared file in the sense `AGENTS.md` means: nobody's package
owns it and everyone depends on it. One workflow, no scripts. A CI setup that needs its
own directory of helpers is one nobody will read when it goes red.

## Done when

A push runs all five commands on Windows, Linux and macOS.

`go build ./...` succeeds on a Linux checkout — after item 1, it should.

The run is red on `ui` and on `internal/layering`, because both are genuinely broken right
now. Do not configure around either. CI saying so out loud is the entire point, and those
two failures are the first thing it will have ever told anyone.

The workflow says, in a comment, which assertions do not run on a GPU-less runner.
