# Assignment: platform

Nearly there. A window opens, input arrives, the lifetime fix holds under a test
with real teeth, `NewWindow` before `Run` works, and `Run` from the wrong goroutine
panics with a legible message. The same-thread contract is back where upstream had
it, and `third_party/README` records the restructuring.

One defect, in the first line of code anyone writes.

## New(Options{}) panics

    New(Options{}) panicked: mainthread: CreateWindowEx failed for hidden window

`Options.Name` becomes the Win32 window class name. Empty, `CreateWindowEx` fails
and the dispatcher panics.

Two rules at once. `AGENTS.md` asks that zero values be usable where reasonable, and
an application name has an obvious default — the executable name, or a constant.
And it asks that panics be reserved for programmer error with no recovery, never for
input; `New` already returns an error and does not use it.

Fix both halves:

- Default `Options.Name` when it is empty, so `New(Options{})` works.
- Have the dispatcher return an error rather than panic when window creation fails.
  A panic is right for `Run` on the wrong goroutine — that is a programmer error
  with no recovery. A failed syscall is not; it is a condition the caller can be
  told about.

## Your tests could not see it

All four pass `Options{Name: "facet-something"}`. Every one configures its way past
the default path, so the case a user hits first is the only case untested.

Add a test that uses the zero value. More usefully, make one test drive the whole
sequence a user actually writes, on one goroutine:

    p, err := New(Options{})
    w, err := p.NewWindow(WindowOptions{Title: "hello", Size: ...})
    w.Show()
    p.Run()

That is the shape of the first program anyone writes against this package, and
nothing currently exercises it end to end.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

`New(Options{})` returns a working platform. A failed syscall inside the dispatcher
returns an error rather than panicking. A test drives New, NewWindow, Show and Run
in that order with zero-value options.

Conventional commits, one file per commit, staged by path.

## Worth carrying

Every test in this package supplied a name, so the default was never exercised. A
suite where each test configures the same field the same way is testing one
configuration many times — and it will be the configuration the author had in mind,
never the one a stranger reaches for.

When a struct has an option, something should use it empty.

## Still true

`platform.Platform` is a layer boundary; a change to it is planned and raised, not
made in passing.

No cgo. `CGO_ENABLED=0` builds on every target. `unsafe` is permitted here and only
here, only for memory the OS owns, and every conversion carries a comment. No Go
pointer goes into OS storage.

When vendored code fights you, work out what it knows before restructuring it. Read
the scars before cutting.

macOS and Linux remain separate assignments.
