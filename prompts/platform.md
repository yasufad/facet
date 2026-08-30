# Assignment: platform

The lifetime fix is right, and the test that guards it has teeth. I removed the map
retention and it failed with exactly the correct diagnosis — "window was collected
while the OS still holds its HWND" — then passed again when restored. A GC test that
can actually fail is rarer than it should be.

`third_party/README` is thorough, and the remaining `unsafe.Pointer` conversion
carries its justification. Both done.

One defect left, and it is in the natural path.

## NewWindow before Run blocks forever

    p, _ := platform.New(Options{})
    w, _ := p.NewWindow(WindowOptions{Title: "hello"})   // blocks forever
    w.Show()
    p.Run()

That is the ordering every user will write, and the first thing anyone tries. It
hangs with no output.

`NewWindow` dispatches onto the platform thread, dispatch posts to the dispatcher's
hidden window, and the hidden window is now created in `Run`. Before `Run`, there is
nothing to post to.

### The cause is a vendored constraint that got removed

Upstream creates the hidden window in `initMainLoop`, under `runtime.LockOSThread()`,
and says why directly above it:

    // initMainLoop must be called with the same OSThread that is used to
    // call runMainLoop() later.

`runMainLoop` then enforces it — `panic("initMainLoop was not called")`, and a second
panic if the thread is wrong.

When `New` and `Run` ended up on different goroutines and deadlocked, the fix moved
window creation into `Run`. That satisfied the smoke test and broke the API for
everyone who creates a window before starting the loop.

The contract was the right one. Honour it instead:

- `New` locks the OS thread and creates the hidden window, as upstream does.
- `Run` must be called from that same goroutine, and panics with a message saying so
  when it is not — copy upstream's guard. A clear panic beats a silent hang, which
  is the whole reason they wrote it.
- Document on `New` that the goroutine which constructs the platform is the one that
  must run it, and that it should be the main goroutine.

Record the change in `third_party/README` like the others — restructuring vendored
code back toward its original shape is still a divergence worth noting.

### Test the ordering

Add a test for the natural sequence: `New`, then `NewWindow`, then `Run` on the same
goroutine, and assert the window exists and receives a message. Add one for the
misuse too — `Run` from a different goroutine should panic with a legible message
rather than hang. Both are cheap and both are the first things a user will do.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

A window can be created before the loop runs. Calling `Run` from the wrong goroutine
panics with a message naming the mistake. `third_party/README` records the
restructuring.

Conventional commits, one file per commit, staged by path.

## Worth carrying

Vendored code carries constraints that were paid for. This one was written in a
comment directly above the function, backed by two panics, and it still got
restructured away when it was inconvenient — and the thing it was preventing came
straight back in a different form.

When vendored code fights you, the first question is what it knows that you do not.
`mainthread_windows.go` exists at all because of a v2 bug where a modal inner loop
swallowed thread-queued messages, and it carries the issue link to prove it. That
file is scar tissue. Read the scars before cutting.

## Still true

`platform.Platform` is a layer boundary; a change to it is planned and raised, not
made in passing.

No cgo. `CGO_ENABLED=0` builds on every target. `unsafe` is permitted here and only
here, only for memory the OS owns, and every conversion carries a comment. No Go
pointer goes into OS storage.

macOS and Linux remain separate assignments.
