# Architecture

Status: implementation. Every layer described here exists and runs on Windows. A
program can open a window and draw a styled element tree containing text, verified by
reading the swapchain back. What is missing is a widget library and the macOS and
Linux backends.

For what is finished rather than what is designed, read the table in `README.md` and
`prompts/`, where an assignment is retired once its package is done. "What is
deferred" below lists the decisions taken to postpone something, as distinct from
things nobody has thought about yet.

## Overview

Facet is a GUI framework for Go. You write Go, you get a native desktop application.

The design follows GPUI, the framework behind the Zed editor: application state lives
in an entity map rather than a pointer graph, repaints are driven by notification
rather than by diffing, and the element tree is rebuilt every frame from retained
state. Those ideas are the port. The Rust is not.

Invalidation is not yet precise — the root view re-renders every frame. See "What is
deferred".

GPUI supplies the model; Wails v3 supplies the per-OS shell. `docs/sources.md` sets
out which layer draws on which.

## Layers

Each layer depends only on those below it. Seams marked *interface* are swap points.

    ui/            widget library — buttons, lists, text fields
    ─────────────────────────────────────────────────────────────
    element/       element tree, three-phase lifecycle
    style/         style properties and the fluent builder
    layout/        flexbox engine
    text/          font loading, shaping, line breaking, glyph raster
    ─────────────────────────────────────────────────────────────
    window/        frame loop, focus, hit testing, input dispatch
    app/           entity map, contexts, effect queue
    ─────────────────────────────────────────────────────────────
    scene/         paint primitives — the renderer's input language
    input/         keymaps, actions, focus tree, dispatch
    render/        Renderer                                    *interface*
    platform/      Platform, Window, Display, Clipboard        *interface*
    ─────────────────────────────────────────────────────────────
    geometry/      Pixels, Point, Size, Bounds, Edges
    colour/        Rgba, Hsla

`geometry` and `colour` depend on nothing, including each other. Everything above
`app` is single-threaded.

## Scene

The renderer's entire input language is a flat, draw-ordered list of instanced
primitives:

    Quad             rectangle, per-corner radius, per-edge border
    Shadow           blurred rounded rectangle
    MonochromeSprite glyphs, from the glyph atlas
    PolychromeSprite images and emoji
    Path             filled bezier
    Underline        straight and wavy

Six primitives are enough to draw a text editor, so they are enough for a widget
library. A short list keeps a second renderer backend cheap.

## What using Facet requires

Go, and nothing else. `go get`, `go build`, an application. No C compiler, no Xcode
command line tools, no GTK development headers, no shader toolchain, no CLI of ours
to install first.

That is a constraint on us, not a hope. It follows from the premise: a framework
whose pitch is pure Go in, native application out, that then asks for a C toolchain
at the last step, has not delivered the premise.

**No cgo.** `CGO_ENABLED=0` builds everywhere, and cross-compiling from one machine
to all three targets keeps working. Windows needs nothing special — Win32 and the
COM vtables behind Direct3D are reachable through `syscall` and
`golang.org/x/sys/windows`, which is how the Wails bindings we vendor already do it.
macOS and Linux go through `github.com/ebitengine/purego`, which calls
`objc_msgSend` and `dlopen`ed C without a C compiler; Ebitengine ships that way.

This one has a real chance of hurting. Cocoa expects the main thread and an
`NSApplication` run loop, and struct-returning Objective-C messages have awkward
calling conventions. If a platform genuinely cannot be reached without cgo, that is
a decision to raise and record here — not a build tag to slip in quietly, and not a
reason to ship a worse macOS.

**Shaders ship as bytecode.** HLSL, MSL and SPIR-V are compiled by us, ahead of
time, and embedded with `go:embed`. A user's build never invokes `fxc`, `dxc`,
`metal` or `glslc`, because requiring any of them would put a platform SDK back in
the dependency list through the side door. Compiling shaders is a job for our
tooling and CI, and the bytecode is checked in.

What a user's machine still has to provide is a GPU driver. Direct3D 11 and Metal
are always present. Vulkan on Linux usually is but is not guaranteed, which is the
argument for an OpenGL fallback there rather than a second first-class backend.

## Rendering

Go owns every layer above the pixel. `render.Renderer` consumes a `scene.Scene` and
draws it onto a GPU surface bound to a native window. No webview stands between the
framework and the screen.

Wails supplies the window and the shell around it, but as vendored source rather
than as a dependency. Four properties of v3 at the pinned commit decided that:

- `WebviewWindow.NativeWindow()` returns a genuine `HWND`, `NSWindow*` and
  `GtkWindow*`. Binding a swapchain to a Wails window is possible.
- The `Window` interface carries unexported methods, so no type declared outside
  `pkg/application` can satisfy it. A webview-free window cannot be added from the
  outside.
- The webview is constructed unconditionally during window setup. Importing Wails
  means every Facet window carries a WebView2 or WKWebView that is never drawn into.
- Client-area input never reaches Go. Wails surfaces window lifecycle events and
  Windows-only non-client mouse events; pointer, wheel and key input belong to the
  webview and arrive as DOM messages. Per-OS input capture is ours to write either
  way.

The first property makes importing Wails tempting; the last three make it a cost
with no return. `platform/` therefore vendors what it needs — see `docs/sources.md`.

### Surface and input

The split is at the native handle. `platform/` owns a window whose client area
belongs to us, the native event loop, main-thread dispatch, and input read from the
native event stream. It hands out a handle and touches no graphics API.

    Windows   HWND          message loop, raw input
    macOS     NSWindow      layer-backed view, event monitors
    Linux     GTK window    purego bindings, event loop

`render/` takes that handle and owns everything API-specific — the device, the
swapchain, the shaders:

    render/d3d11   render/metal   render/vulkan

Keeping the swapchain out of `platform/` is what lets a second graphics backend be a
package rather than a rewrite, and what keeps `platform/` testable without a GPU.

## State and reactivity

State lives in entities. An `Entity[T]` is a handle — an ID into an app-owned map.
Reads and writes go through a context, which is what makes mutation observable.

    ent.Update(cx, func(v *Counter, cx *Context[Counter]) {
        v.count++
        cx.Notify()
    })

`Notify` marks the entity dirty and schedules its observers. Effects queue and flush
at the end of the update cycle, so a burst of mutations produces one frame.

Three ways to react, in increasing order of coupling:

    Observe      run when another entity notifies; the observer is not told why
    Subscribe    receive typed events an entity emits
    Render       a view is an entity whose notification schedules a repaint

A view repaints when its own entity notifies, and at no other time. State a view
reads from another entity does not repaint it on its own: the view observes that
entity and notifies itself, which is explicit and one line. Nothing tracks which
entities a render happened to read.

## The frame

    1. flush effects      drain notifications, run observers, mark views dirty
    2. request layout     walk dirty views, build elements, compute style
    3. layout             flexbox solve, producing bounds for every node
    4. prepaint           hit-test regions, scroll offsets, focus geometry
    5. hit test           resolve the pointer against the regions just registered
    6. paint              emit primitives into the scene
    7. present            hand the scene to the renderer

Steps 2 and 6 rebuild rather than mutate. An element is a cheap value describing what
should be on screen, discarded after paint. State that survives the frame lives in an
entity.

Step 5 is what makes hover work without giving elements an identity that survives the
frame. A region registered during prepaint is resolved against the pointer before
paint runs, so an element asking "am I hovered" during paint is asking about the
regions this frame registered, not last frame's. Nothing has to be remembered between
frames and no element-keyed map is needed.

A hover style that changes layout still lags one frame, because layout ran at step 3
before any region existed. A hover style that only changes what is painted does not
lag at all.

## Threading

The UI runs on one goroutine and contexts do not leave it. That is what keeps the
entity map lock-free and the effect queue ordered. Background work returns to the
foreground to touch state:

    cx.Background(func(ctx context.Context) (Result, error) { ... }).
        Then(cx, func(r Result, cx *Context[T]) { ... })

Using a context from another goroutine panics.

## Text

`go-text/typesetting` handles font loading and matching, script and bidi
segmentation, and shaping, in pure Go. `text/` wraps it and exposes shaped lines and
glyph runs; no layer above `text/` knows the dependency exists.

Rasterising outlines into the glyph atlas uses `golang.org/x/image/vector`, which
computes analytic area coverage — all 256 levels — with SIMD paths on amd64 and
arm64. A custom scanline rasteriser was tried first and measured against it. The
custom one supersampled at 4×4 per pixel, giving at most 17 distinct coverage
values before scaling to a byte; at 10 to 14 pixels stem edges landed on those
steps and came out unevenly weighted. `x/image/vector` has no such ceiling.

The timings, averaged over six Latin glyphs at each size:

    10 px   custom 9.99 µs   x/image/vector 7.19 µs
    12 px   custom 12.70 µs  x/image/vector 7.02 µs
    14 px   custom 15.41 µs  x/image/vector 7.31 µs
    16 px   custom 19.03 µs  x/image/vector 8.01 µs

`x/image/vector` is faster at every size, and the gap widens as the glyph grows
because the custom rasteriser's cost scales with the supersample grid while the
analytic rasteriser's does not.

GPU compute, as GPUI does, was not considered: it belongs to `render/`, which
sits above `text/`. The text package produces data; it does not draw.

The rasteriser handles the four OpenType outline operations (move, line, quadratic,
cubic) and nothing else. Bitmap, SVG and COLR glyphs fall back to their embedded
outline when typesetting provides one; glyphs with no outline at all (whitespace)
produce an empty mask. Glyph bounds go through `geometry.BoundsToDevicePixels`
with the display's real scale factor, so atlas tiles agree with the geometry
around them.

## Layout

A port of Taffy's flexbox solver, algorithm for algorithm, including its
browser-derived test suite. Grid is out of scope.

## What is deferred

Each of these was decided rather than overlooked. The reasoning is in the package's
entry in `docs/packages.md` or in its `doc.go`; this list exists so nobody has to
find that out by reading the whole tree.

**Precise invalidation.** The root view re-renders every frame. `app` tracks nothing
about which entities a render read, so `window` observes the root view's entity and
nothing else, and a view that reads another entity observes it explicitly. GPUI
tracks accessed entities and invalidates per view. Doing the same costs a per-frame
accessed set and subscription churn, and the case for paying it should come from `ui`
showing what real applications need.

**The element arena.** Every element is a separate heap allocation: 584 bytes and
about 420 ns for a `Div`, so a thousand elements a frame is roughly 640 KB of garbage
and 38 MB/s at 60 fps. That arrives as GC pauses inside frames rather than as a
slower average. GPUI allocates elements from a per-frame arena and resets it at frame
end. `window` owns the frame boundary and so would own the arena; `element.NewDiv`
takes no arguments, so the arena has to be reachable without one.

**Text beyond one line.** `element.Text` shapes and paints a single line with one
style run. Wrapping, bidi, selection and multiple runs are not implemented. The
`text` package already handles bidi and script segmentation, so this is element work
rather than text work.

**`layout.MeasureFunction` returns a `LayoutOutput`.** That is Taffy's shape,
faithfully ported, and it forces callers to do the leaf arithmetic themselves, which
is why `ComputeLeafLayout` is exported. Wrapping it in something size-in, size-out is
a decision we are allowed to take if it keeps costing exports.

**macOS and Linux.** `platform` and `render` are Windows only. Both are designed as a
backend per operating system and per graphics API, so each is a new subpackage rather
than a change to the interface. No cgo, so Cocoa and GTK go through purego, and
`docs/architecture.md` above says what happens if that turns out to be impossible.
