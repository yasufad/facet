# Assignment: platform

A window opens on Windows and reports input. That is the first thing in this project
you can actually see, and it works — the smoke test posts a real `WM_MOUSEMOVE` and
asserts it arrives as a `PointerEvent`, which is testing the message path rather
than the type system.

All four interface corrections landed correctly: `ScrollDelta` carries its unit,
`ScrollPhase` carries the gesture, window geometry is in logical `Pixels`,
`NativeSurface` is separate from `NativeHandle`, and the display-change paths are
down to two with both documented. The vendoring preserved the Winc and W32 Authors
copyrights as well as Wails', which is more care than the licence strictly required.

Three things to fix. The first is a crash.

## 1 — The window can be freed while Windows still holds a pointer to it

    // window_windows.go:145
    w32.SetWindowLongPtr(hwnd, w32.GWLP_USERDATA, uintptr(unsafe.Pointer(w)))

    // window_windows.go:43
    ptr := w32.GetWindowLongPtr(hwnd, w32.GWLP_USERDATA)
    return (*windowsWindow)(unsafe.Pointer(ptr))

A Go pointer is stored in Win32 memory, and `windowsPlatform` holds no window
references — there is no slice or map of windows anywhere. So once a caller drops
its `Window` handle, which is entirely legal (call `Show()`, keep nothing), the only
thing pointing at that `windowsWindow` is a `uintptr` the garbage collector cannot
see. It gets collected, and the next `WM_*` message dereferences freed memory inside
the wndproc.

Be precise about the mechanism, because it changes how you look for it: Go does not
relocate heap objects, so the address stays valid. This is a liveness bug, not a
relocation bug — the object is freed, not moved. It fires under GC pressure and
presents as random corruption, which is the worst kind to chase.

Keep a `map[w32.HWND]*windowsWindow` on the platform behind the mutex it already
has, and look the window up by `HWND` in the wndproc. The Go pointer stays visible
to the collector, no Go pointer crosses into OS memory, and the unsafe conversion
disappears rather than being justified.

Delete the entry when the window is destroyed, or you have swapped a crash for a
leak.

## 2 — go vet was red, so nobody read it

The report said `go vet ./...` exits 0 with expected warnings. It exits 1, with
nineteen findings, and one of them was defect 1 — pointing at line 47 exactly.

The check has been changed in `AGENTS.md` to `go vet -unsafeptr=false ./...`, which
is green. The analyser cannot be scoped to skip `third_party`: `vet` analyses
dependencies, so excluding those packages from the list still reports them.
Seventeen findings from vendored bindings drowned two of ours.

The trade is that `unsafeptr` no longer runs at all, so it is now on you: every
`unsafe.Pointer` conversion in our own code carries a comment saying why it is
sound. There are two, and after fixing defect 1 there will be one —
`window_windows.go:212`, where `lParam` points at an OS-owned `RECT`. That one is
legitimate and needs a sentence saying so.

Run `go vet ./...` unfiltered by hand after touching this package and read what it
says about `platform/`. Ignore the `third_party/` lines; they are not ours.

## 3 — Half the vendoring changes are unrecorded

`ole32.go` does this properly: it says `RegisterDragDrop` and `RevokeDragDrop` were
removed and why. Copy that pattern to the rest.

Unrecorded right now:

    constants.go     gained MK_* and WHEEL_DELTA
    user32.go        gained ShowCursor
    idroptarget.go   skipped entirely, needs the webview2 COM bridge

Write `third_party/README` recording what was vendored from where, at which pinned
commit, what was skipped, and what was changed. `AGENTS.md` asks third_party to make
the next upstream bump a merge rather than an excavation, and right now those three
divergences are invisible to whoever does it.

`layout/testdata/flex/README` is the pattern to follow — it does the same job for
the ported fixtures.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

No Go pointer is stored in OS memory. The remaining `unsafe.Pointer` conversion
carries its justification. `third_party/README` records the vendoring, the skips and
the modifications.

A test that exercises the lifetime would be worth having: create a window, drop
every Go reference to it, force `runtime.GC()`, then pump a message and confirm the
handler still fires. That fails today and passes after the fix.

Conventional commits, one file per commit, staged by path.

## Still true

`platform.Platform` is a layer boundary. It changed once under review, which is how
it should change; from here a change is planned and raised, not made in passing.

No cgo. `CGO_ENABLED=0` builds on every target and keeps building. Windows reaches
Win32 through `syscall`; macOS and Linux will go through `purego` when their turn
comes. `unsafe` is permitted here and only here — and after defect 1 is fixed, only
for memory the OS owns, never for our own objects.

macOS and Linux remain separate assignments.
