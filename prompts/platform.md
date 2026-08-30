# Assignment: platform

Build `platform`: the interface, and the Windows backend behind it.

This is the largest package in Facet and the one everything visible waits on. It is
also the one where a mistake is most expensive, because `render`, `window` and
`input` are all written against its interface. Read `docs/packages.md` and the
Rendering section of `docs/architecture.md` before starting.

macOS and Linux are separate assignments. Do not write them. Do design the interface
so they fit — say in a comment where you are aware a platform will differ.

## Stop after the interface

`platform.Platform` is a layer boundary. `AGENTS.md` requires a change to one to be
planned before it is written, and this is the first writing of it.

So: propose the interface first, in a single commit that adds the types and the
method signatures with doc comments and no implementation. Then stop and say so. It
gets reviewed before the backend is built, because three packages inherit whatever
shape it has and the cost of changing it later is theirs, not yours.

What the interface has to cover:

    application lifecycle      start, run the loop, quit, activation
    windows                    create, size, position, title, close, scale factor
    main-thread dispatch       run a closure on the platform thread
    displays                   enumeration, bounds, scale factor, the active one
    input                      pointer, wheel, key, modifiers, focus, IME
    cursor                     shape, visibility
    clipboard                  read and write text
    menus, tray, dialogs, notifications

Two constraints shape it more than the rest:

**A native handle, never a graphics API.** `platform` hands out an `HWND`,
`NSWindow*` or `GtkWidget*` and stops. `render` takes that handle and owns the
device, the swapchain and the shaders. `platform` must not import `render` and must
not know D3D exists. That is what keeps a second graphics backend a package rather
than a rewrite.

**Input is a stream, not callbacks into the framework.** `platform` reads the native
event stream and surfaces events; `input` and `window` decide what they mean.
Nothing above `platform` should have to know that Windows reports wheel deltas in
multiples of 120.

## Vendor, do not write

`docs/sources.md` says what comes from Wails and how. Two pieces come across nearly
untouched and should be vendored into `third_party/` before you write anything:

- `v3/pkg/w32` — Win32 bindings, around thirteen thousand lines, standalone
- `v3/pkg/application/mainthread_windows.go` — main-thread dispatch. It posts to a
  hidden window rather than the thread queue, because a modal inner loop swallows
  thread-queued messages. That is a bug they hit in v2 and fixed with a citation in
  the comment. Vendor it rather than rediscovering it.

Wails is MIT. Every vendored file keeps an attribution header naming the upstream
file, and Wails goes into `NOTICE` with text copied from its `LICENSE`, not from
memory. Two attributions in this repository have already been wrong.

The window code itself is surgery, not a copy: `webview_window_windows.go` is three
thousand lines with window and webview concerns in one type. Take the window, leave
the webview. Facet creates no WebView2.

Sync the checkouts with `go run ./tools/upstream` if `_upstream/` is not there.

## Then the Windows backend

An `HWND` with a message loop we own, a client area nothing else draws into, input
delivered from `WM_*` messages, and the shell pieces above. It must run on this
machine: a window that opens, resizes, reports its scale factor, and reports input.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

The layering test passes: `platform` imports `geometry`, `colour` and `third_party`,
nothing else of ours. A window opens on Windows and reports input. The interface is
documented well enough that whoever writes the macOS backend does not need to ask
you what a method means.

Conventional commits, one file per commit, staged by path — `NOTICE` is shared and
other agents are working in this tree.

## What this package is allowed that others are not

`cgo` and `unsafe`, for platform calls. It is the only package with that permission,
which is the reason the boundary is drawn here. Everything above it is ordinary Go,
and that stays true only if the unsafe parts do not leak upward through the
interface.
