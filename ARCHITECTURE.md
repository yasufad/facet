# Architecture

## What this is

A GUI framework for Go. You write Go, you get a native desktop application. No HTML,
no CSS, no JavaScript in the programming model — not as a philosophical position, but
because a framework whose abstractions leak the web platform inherits the web
platform's constraints forever.

The design follows GPUI, the framework behind the Zed editor. GPUI got several things
right that are worth taking wholesale: state lives in an entity map rather than in a
pointer graph, invalidation is precise instead of diff-based, and the element tree is
rebuilt every frame from retained state rather than mutated in place. Those ideas are
the port. The Rust is not.

## Layering

Each layer depends only on the ones below it. The seams marked *interface* are the ones
we expect to swap; everything else is free to be concrete.

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
    color/         Rgba, Hsla

`geometry` and `color` have no dependencies at all, including on each other. Everything
above `app` is single-threaded by construction.

## How pixels happen

Wails v3 supplies the shell: a native window, its lifecycle, menus, tray, system
dialogs, native input events, and application packaging. That is a genuinely large
amount of per-OS work we do not want to own.

It does not supply the rendering model. Inside the Wails window we drive a GPU surface
directly. Go builds a `scene.Scene` — a flat list of instanced primitives, sorted into
draw order — and the renderer uploads it. The webview hosting that surface runs a fixed
shader set and a frame loop; it never sees application structure, only buffers. From
the framework's side it is a GPU driver that happens to be reachable over IPC.

The scene is deliberately narrow. Everything the framework can draw reduces to:

    Quad             rectangle, per-corner radius, per-edge border
    Shadow           blurred rounded rectangle
    MonochromeSprite glyphs, from the glyph atlas
    PolychromeSprite images and emoji
    Path             filled bezier, for anything the above cannot express
    Underline        straight and wavy

Six primitives cover a text editor, so they cover a widget library. Keeping the list
short is what makes a second renderer backend cheap, and a second backend is how we
stay honest about the abstraction.

### The risk, stated plainly

Shipping a scene over an IPC boundary every frame is the part of this design most
likely to hurt. Mitigations, in the order we will need them: instanced primitives so
payload scales with distinct draw calls rather than pixels; a binary frame format
rather than JSON; damage tracking so an unchanged region costs nothing; and the entity
system's precise invalidation, which tells us exactly which views are dirty instead of
making us guess. If those are not enough, the `render.Renderer` seam is where a native
surface replaces the webview without the layers above noticing.

## State and reactivity

Application state lives in entities. An `Entity[T]` is a handle — an ID into an
app-owned map — not a pointer. You cannot read or mutate the value without going
through a context, and the context is the thing that knows a mutation happened.

    ent.Update(cx, func(v *Counter, cx *Context[Counter]) {
        v.count++
        cx.Notify()
    })

`Notify` marks the entity dirty and schedules its observers. Nothing repaints
immediately; effects queue and flush at the end of the update cycle, so a burst of
mutations produces one frame, not one frame each. This is the whole reason for the
handle indirection: it gives the framework a place to stand between the write and the
consequences of the write.

Three ways to react, in increasing order of coupling:

- `Observe` — run when another entity notifies. The observer does not learn why.
- `Subscribe` — receive typed events an entity explicitly emits.
- `Render` — a view is an entity whose notification schedules a repaint.

Views hold entity handles, so a view repaints when its state changes and at no other
time. There is no virtual tree diff, because there is nothing to diff against: we
already know what changed.

## The frame

    1. flush effects      drain notifications, run observers, mark views dirty
    2. request layout     walk dirty views, build elements, compute style
    3. layout             flexbox solve, producing bounds for every node
    4. prepaint           hit-test regions, scroll offsets, focus geometry
    5. paint              emit primitives into the scene
    6. present            hand the scene to the renderer

Steps 2 and 5 rebuild rather than mutate. An element is a cheap value describing what
should be on screen; it is discarded after paint. State that must survive the frame
lives in an entity, which is the distinction the whole design rests on.

## Threading

The UI runs on one goroutine, and contexts are not safe to move off it. This is a
constraint, not an oversight — it is what makes the entity map lock-free and the effect
queue ordered. Background work goes through the executor and returns to the foreground
to touch state:

    cx.Background(func(ctx context.Context) (Result, error) { ... }).
        Then(cx, func(r Result, cx *Context[T]) { ... })

Attempting to use a context from the wrong goroutine panics with a clear message rather
than corrupting state quietly.

## What is deliberately absent

No CSS cascade — styles are set on elements, and inheritance is explicit and limited to
text properties. No stylesheet language. No template language. No code generation. No
reflection in the hot path. If a feature would require a Go developer to learn a second
notation to use this framework, it needs a strong argument.

## Status

Early. The layers below `window` are being built first, since everything above them is
straightforward once the foundations are correct.
