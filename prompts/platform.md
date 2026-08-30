# Assignment: platform

The interface is written and reviewed — twelve files, one concept each, every check
clean, `platform` importing only `geometry` and `colour`. The shape is right:
`uintptr` rather than `unsafe.Pointer` across the boundary, no graphics API in
sight, and you stopped for review before building on it, which is why the four
corrections below cost an afternoon instead of a rewrite.

Fix these, then build the Windows backend. macOS and Linux remain separate
assignments; design so they fit, and say in a comment where you know a platform will
differ.

## Correction 1 — WheelEvent discards trackpad precision

    // Delta is normalised to lines ... The platform converts from its native
    // units — Windows multiples of 120, macOS pixel-based deltas

Normalising pixel deltas to lines is lossy in the direction that matters. A mouse
wheel emits discrete notches; a trackpad or a Windows precision touchpad emits
pixel-exact deltas with momentum. They are different inputs, and the consumer has to
know which arrived: pixel deltas apply directly, line deltas multiply by a line
height. GPUI keeps them apart for this reason, in `interactive.rs`:

    pub enum ScrollDelta {
        Pixels(Point<Pixels>),   // exact
        Lines(Point<f32>),       // inexact
    }

Carry the distinction, and carry a scroll phase with it — GPUI's `TouchPhase` is
`Started`, `Moved`, `Ended`, `Cancelled`. Without a phase there is no rubber-banding
and no way to cancel momentum when a finger lands.

Both are unrecoverable once flattened here. Nothing upstairs can reconstruct
precision this layer threw away.

## Correction 2 — window geometry belongs in logical pixels

`SetSize`, `Size`, `SetPosition`, `SetMinSize` and `SetMaxSize` all take
`geometry.Size[DevicePixels]`. GPUI uses `Bounds<Pixels>`.

Ask for an 800×600 window in device pixels on a 2× display and you get one
physically half the size intended, so every caller does scale arithmetic — which is
what typed units exist to prevent. Worse, a minimum size in device pixels changes
meaning when the window moves to a monitor with a different scale factor: the same
constraint becomes a different physical size, and the window can end up violating
it.

Move window geometry to `Pixels` and leave `ScaleFactor()` for anyone who needs
device units. This does not affect the renderer: the swapchain is sized in device
pixels, but that is `render`'s business, downstream of `ScaleFactor()`.

## Correction 3 — NativeHandle is one handle short on macOS

Metal draws into a `CAMetalLayer` on the content view, not into the `NSWindow`. If
`platform` owns the layer-backed view, as `docs/architecture.md` says, then `render`
needs the view. Return the drawing surface rather than the window, or add a second
accessor for it.

Get this right now even though macOS is not your assignment. Otherwise `render`
ends up doing Cocoa work to find its own surface, which is the seam this boundary
exists to prevent.

## Correction 4 — three ways to hear about a display change

`SetDisplayChangeHandler` exists on both `Platform` and `Window`, and there is also
a `ScaleChangeEvent`. Decide which is authoritative and delete or document the
others. A consumer that has to guess which one fires will subscribe to all three and
handle the change twice.

## The interface stays a layer boundary

`platform.Platform` is one of the three contracts `AGENTS.md` names. It changed once,
under review, which is how it should change. From here on a change to it is planned
and raised, not made in passing while implementing a backend — `render`, `window`
and `input` inherit whatever shape it has, and the cost of a late change is theirs.

The two constraints that shaped it still hold, and the corrections above are both
cases of them being applied more carefully rather than new rules:

**A native handle, never a graphics API.** `platform` hands out a handle and stops.
`render` takes it and owns the device, the swapchain and the shaders. `platform`
must not import `render` and must not know D3D exists. Correction 3 is this rule:
the handle has to be the one the graphics API can actually draw into.

**Input is a stream, not callbacks into the framework.** `platform` reads the native
event stream and surfaces typed events; `input` and `window` decide what they mean.
Nothing above should learn that Windows reports wheel deltas in multiples of 120 —
but Correction 1 is the limit of that principle. Hiding the *units* is the job.
Hiding whether the input was precise is destroying information, not abstracting it.

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

## No cgo

`CGO_ENABLED=0` must build on every target, and it must keep building. A user
installs Go and runs `go build`; they do not install a C compiler, Xcode command
line tools or GTK development headers. The requirements section of
`docs/architecture.md` says why that is a constraint rather than a preference.

Windows makes this easy and is the reason to start here: the `w32` bindings you are
vendoring already reach Win32 through `syscall.NewLazyDLL` and
`golang.org/x/sys/windows`, with no C anywhere. Keep it that way.

macOS and Linux, when their turn comes, go through
`github.com/ebitengine/purego` — `objc_msgSend` and `dlopen` without a C compiler.
Ebitengine ships that way, so it is proven, but Cocoa expects the main thread and an
`NSApplication` run loop and struct-returning messages have awkward calling
conventions. Design the interface now so those fit; you do not have to solve them.

If a platform turns out to be genuinely unreachable without cgo, that is a decision
to raise and record, not a build tag to add quietly.

`unsafe` is permitted here, and only here, for platform calls. Everything above is
ordinary Go, and stays that way only if the unsafe parts do not leak upward through
the interface.
