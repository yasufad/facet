# Architecture

Status: planning. Sections marked *open* are not decided.

## Overview

A GUI framework for Go. You write Go, you get a native desktop application.

The design follows GPUI, the framework behind the Zed editor: application state lives
in an entity map rather than a pointer graph, invalidation is precise rather than
diff-based, and the element tree is rebuilt every frame from retained state. Those
ideas are the port. The Rust is not.

Two upstream codebases sit alongside this one as references, cloned into `../`:

    ../gpui      Zed's GUI framework — the conceptual source
    ../wails     Wails v3 — the platform layer we borrow from

GPUI supplies the model. Wails supplies working, per-OS window management, event
loops, menus, tray, dialogs and packaging, in Go, which is code we take rather than
write.

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

## Rendering — *open*

Settled: Go owns every layer above the pixel. The framework emits a `scene.Scene`;
something turns it into pixels inside a native window.

Undecided: what that something is, and how much of Wails comes with it.

    a) Wails window, WebGPU surface   webview runs a fixed shader set and a frame
                                      loop, consuming instance buffers over IPC.
                                      Per-frame IPC cost is the open risk.

    b) Wails platform layer, own      take Wails' per-OS window and event-loop code,
       surface                        attach D3D/Metal/GL directly instead of a
                                      webview. No IPC. More CGO to own.

Both keep `render.Renderer` as the seam, so this decision does not reach the layers
above `scene`.

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

## Text — *open*

Shaping, font fallback and glyph rasterisation. `go-text/typesetting` is a pure Go
candidate; the alternative is binding HarfBuzz.

## Layout — *open*

Flexbox. Either a port of Taffy or a direct implementation against the spec subset
GPUI actually uses.
