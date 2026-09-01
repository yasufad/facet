# build: nothing has ever been compiled by anything but a person

This one is not a package, which breaks the shape of every other file here. It gets a
prompt anyway because it is the highest-leverage item in `docs/audit.md` and the reason
most of the rest of that document exists: every package passes its own tests, and the
application does not run.

There is no `.github/`. Nothing has ever built this repository except whoever happened to
be sitting in front of it, on whatever operating system they had.

## What that has cost so far

The layering test has been failing on `main` and the only reason anyone knows is that a
prompt says so. `examples/button` demonstrates the framework's central idiom and panics
on the first click; it cannot ever have been run. `go build ./...` and `go test ./...`,
the two commands `AGENTS.md` names, do not pass on a Linux or macOS checkout.

Three of the four blocking defects in the audit are integration failures between packages
that individually work and individually pass. That is what nobody is checking.

## 1. tools/compile_shaders

`main.go` uses `syscall.NewLazyDLL` and `syscall.SyscallN`, which exist on Windows only,
with no build constraint. It is genuinely a Windows tool — it drives the HLSL compiler —
so the right fix is `//go:build windows` on it and nothing else.

`go build ./...`, `go vet ./...` and `go test ./...` all skip a package whose files are
entirely excluded by constraints, with no error. I checked; a stub for the other
platforms is not needed and would be worse.

The examples are a different case and belong to `platform`. Its prompt covers them.

## 2. Continuous integration

A workflow that runs on every push and every pull request, over the three targets:

    go build -o bin/ ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)
    go test ./...
    go test -tags facet_debug ./...

`gofmt -l` must print nothing, so it needs to fail the job when it prints something —
`gofmt -l` exits zero either way, which is the trap in every version of this check ever
written. Scope it through `go list` rather than running `gofmt -l .`, which walks into
`_upstream/` and reports every upstream file.

Cross-compilation is a stated invariant rather than a convenience, so build the other two
targets from each host as well, with `CGO_ENABLED=0`. A change that breaks
cross-compilation should fail on the machine that made it.

The `facet_debug` run matters more here than the default one. It carries the clip-stack
balance assertion, the context threading check and the renderer's pixel readback, and
`AGENTS.md` is explicit that skipping it means those tests never execute. On a runner
without a GPU the readback tests will not run; say so in the workflow rather than letting
a green tick imply they did.

Do not add a linter. There is none configured and choosing one is a decision, not a
housekeeping task.

## 3. The test that would have caught all of it

CI only helps if something is testing the seams. Nothing is.

The test to have: build a view with real entity state, drive a full frame through
`window.Draw`, dispatch a synthetic pointer down and up through `window.DispatchEvent`,
and assert both that the entity changed and that the next frame reflects it. Against the
fake platform and fake renderer that `window_test.go` already has, so it needs no GPU.

That one test fails today on three separate defects: the handler cannot reach its entity,
`ReadEntity` hands back a copy, and hit regions ignore clipping.

It is not yours to write — it lives in `window` and it cannot exist until `element` lands
the listener seam. Your job is to make sure CI is standing when it does, and to say in the
workflow what it is for. When `window` reports, come back and check the test is in the
run.

## Where the files go

`.github/workflows/` is a shared file in the sense `AGENTS.md` means: nobody's package
owns it and several people depend on it. Keep it to one workflow and no scripts. A CI
setup that needs its own directory of helpers is one nobody will read when it goes red.

## Done when

A push to a branch runs all five commands on Windows, Linux and macOS, and the run is red
today — the layering failure is real and CI should say so rather than being configured
around it.

`go build ./...` succeeds on a Linux checkout once `platform` lands its side.

The workflow says, in a comment, which assertions do not run on a GPU-less runner.
