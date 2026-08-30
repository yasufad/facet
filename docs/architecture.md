# Architecture

Status: implementation. Five packages are built — geometry, colour, app,
layout, text — and the layering test enforces their imports. The remaining
packages (scene, platform, render, style, input, element, window, ui) are
not yet started.

## Overview

Facet is a GUI framework for Go. You write Go, you get a native desktop application.

The design follows GPUI, the framework behind the Zed editor: application state lives
in an entity map rather than a pointer graph, invalidation is precise rather than
diff-based, and the element tree is rebuilt every frame from retained state. Those
ideas are the port. The Rust is not.

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
    Linux     GTK window    cgo bridge

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

A view repaints when the entities it holds change, and at no other time.

## The frame

    1. flush effects      drain notifications, run observers, mark views dirty
    2. request layout     walk dirty views, build elements, compute style
    3. layout             flexbox solve, producing bounds for every node
    4. prepaint           hit-test regions, scroll offsets, focus geometry
    5. paint              emit primitives into the scene
    6. present            hand the scene to the renderer

Steps 2 and 5 rebuild rather than mutate. An element is a cheap value describing what
should be on screen, discarded after paint. State that survives the frame lives in an
entity.

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
