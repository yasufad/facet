# Assignment: platform

`SetSize` is fixed and correct — sizes round-trip exactly and the window stays where
it was. `AdjustWindowRectExForDpi` was the right call over the non-DPI version, and
the w32 addition is recorded in `third_party/README`.

The creation path is not fixed. It behaves exactly as it did before:

    create 640x480 -> 625.3x442.7   (the same 14.7x37.3 frame delta)
    setsize 690x530 -> 690.0x530.0  correct
    pos {200 150} -> {200 150}      correct

## Why the new test passes anyway

`TestNewWindowHonoursClientSize` sets `Decorated: true`. With the zero value it
fails:

    zero-value             decorated=false -> 625.3x442.7  (want 640x480)
    decorated              decorated=true  -> 640.0x480.0
    undecorated explicit   decorated=false -> 625.3x442.7  (want 640x480)

This is the same failure as last round, one field along. Then it was
`Options{Name:}` — every test supplied one, so the default was the only untested
case. Now it is `WindowOptions{Decorated:}`. A test that configures a field cannot
tell you what happens when nobody does, and nobody is the common case.

## Defect 1 — Decorated: false does not make an undecorated window

    Decorated=false  style=0x04c00000  WS_CAPTION=true  WS_THICKFRAME=false
    Decorated=true   style=0x04cb0000  WS_CAPTION=true  WS_THICKFRAME=false

Both have a caption. The flag only toggles the system menu and the minimise and
maximise boxes. An undecorated window wants `WS_POPUP` with no `WS_CAPTION` and no
`WS_THICKFRAME`; that is a different window, not the same one with fewer buttons.

Decide what `Decorated: false` means and make the style match it. If a borderless
window is out of scope for now, say so on the field and reject the option rather
than silently producing a decorated window.

## Defect 2 — the size adjustment reads the option, not the style

This is the one that actually caused the wrong size, and the more important fix.
The frame adjustment is applied conditionally on `opts.Decorated`, while
`CreateWindowEx` is called with a style computed separately. When the two disagree —
as they do right now — the client area is short by exactly one frame.

Compute the adjustment from the same style value you pass to `CreateWindowEx`:

    style, exStyle := stylesFor(opts)
    rect := clientRect(opts.Size, scale)
    w32.AdjustWindowRectExForDpi(&rect, style, false, exStyle, dpi)
    hwnd := w32.CreateWindowEx(exStyle, class, title, style, ...)

One variable, used twice. Then the adjustment cannot disagree with the window,
whatever `Decorated` ends up meaning, and fixing defect 1 cannot silently break the
sizes again.

## Test the zero value, not your configuration

Every geometry test sets `Decorated: true`, `Resizable: true`, `Visible: false`.
Add the same cases with `WindowOptions{Title: ..., Size: ...}` and nothing else.
Both current cases fail that way today.

More generally: when a struct of options exists, at least one test should pass it
empty. That rule would have caught this round and the last one.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

A window created with a 640x480 client size reports 640x480 whether or not
`Decorated` is set. The style used for the frame adjustment is the style the window
is created with. `Decorated: false` either produces a genuinely undecorated window
or is documented and rejected as unsupported.

Conventional commits, one file per commit, staged by path.

## Worth carrying

Two rounds, two defects, one cause: the test configured the field, so the default
went unexercised. It is worth being suspicious of any test that fills in a struct
literal — the fields you thought to set are the ones you already had in mind, and
the bug lives in the ones you did not.

The deeper fix is structural rather than procedural. Defect 2 exists because the
same fact — the window's style — is computed in two places. Derive it once and use
it twice, and a whole category of disagreement stops being possible.

## Still true

`platform.Platform` is a layer boundary; a change to it is planned and raised, not
made in passing.

No cgo. `CGO_ENABLED=0` builds on every target. `unsafe` is permitted here and only
here, only for memory the OS owns, and every conversion carries a comment. No Go
pointer goes into OS storage.

When vendored code fights you, work out what it knows before restructuring it.

macOS and Linux remain separate assignments.
