# Assignment: platform

`New(Options{})` works, the dispatcher returns an error instead of panicking on a
failed syscall, and the first program a user writes runs end to end — window up,
handle and surface non-zero, scale factor read from the display, visible on screen.
That was the right fix and the right test.

Running that program surfaced two more, both in window geometry, both invisible to
the suite because nothing asserts what the sizes mean.

## 1 — NewWindow does not honour the client-size contract

`WindowOptions.Size` is documented as the client area: "The client area excludes the
title bar and borders; the full window is larger." It is passed to `CreateWindowEx`
as the outer size instead.

    asked 640x480  got 625.3x442.7  delta 14.7x37.3
    asked 800x600  got 785.3x562.7  delta 14.7x37.3
    asked 300x200  got 285.3x162.7  delta 14.7x37.3

The delta is constant because it is the frame — at scale 1.5, 22×56 device pixels of
border and title bar.

`SetSize` already does this correctly, at `window_windows.go:400`: build a `RECT` for
the wanted client size, call `AdjustWindowRectEx` with the window's styles, and use
the adjusted outer size. The creation path at line 123 skips it, so the same
documented contract behaves two different ways depending on which function you
called.

Use `AdjustWindowRectExForDpi` where it is available — the non-DPI-aware version
assumes the primary display's scale, which is wrong the moment a window opens on a
secondary monitor with a different factor.

## 2 — SetSize teleports the window

    position before SetSize {200 150}, after {-0.67 -0.67}

`MoveWindow(w.hwnd, -1, -1, ...)` sets position as well as size, and -1 is not a
sentinel meaning "leave it" — it is a coordinate. Resizing a window moves it to the
top-left corner of the screen.

Use `SetWindowPos` with `SWP_NOMOVE | SWP_NOZORDER`, which says what is meant. The
size arithmetic in that function is right; only the placement is wrong.

## Why the suite missed both

Five tests, all passing, none asserting what a size means. `TestWindowOpensAndReportsInput`
checks `Size()` is positive — which 625.3 is.

Add assertions with content:

- Create a window with a known client size and assert `Size()` returns it, at
  whatever scale factor the test machine reports. Both the 640×480 and 300×200 cases
  above fail today.
- Set a position, call `SetSize`, assert the position is unchanged.
- Round-trip: `SetSize(s)` then `Size()` returns `s`.

"Is positive" and "is non-zero" are the assertions you write when you do not yet
know what the right answer is. Once the contract is written down — and this one is,
in the doc comment — the test should check the contract.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

A window created with a 640×480 client size reports 640×480. `SetSize` leaves the
window where it was. Tests assert both rather than checking for positive numbers.

Conventional commits, one file per commit, staged by path.

## Worth carrying

Both defects are in code that compiles, passes review, and has tests. What neither
had was a test that knew what the answer should be. `Size()` returning 625.3 is
indistinguishable from correct unless something asserts 640.

The doc comment on `WindowOptions.Size` already stated the contract precisely. It
was written, then not implemented, and nothing noticed because nothing checked. A
contract in a comment with no test behind it is a wish.

## Still true

`platform.Platform` is a layer boundary; a change to it is planned and raised, not
made in passing.

No cgo. `CGO_ENABLED=0` builds on every target. `unsafe` is permitted here and only
here, only for memory the OS owns, and every conversion carries a comment. No Go
pointer goes into OS storage.

When vendored code fights you, work out what it knows before restructuring it.

macOS and Linux remain separate assignments.
