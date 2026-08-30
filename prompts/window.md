# Assignment: window

Build `window`: the frame loop. It owns a platform window, drives layout through
paint, presents the scene, and routes input into the dispatch tree.

This is the package where everything below finally runs. `platform` produces events,
`app` produces notifications, `layout` solves, `text` shapes, `scene` collects,
`render` draws — and none of them know about each other. `window` is the only place
that sees all of it, so every seam in `docs/architecture.md` is closed here or not at
all.

## Prerequisites

`element` must be merged first. `window` implements `element.Frame`, and elements
call into that interface to request layout, register hit regions and paint. Writing
`window` against an imagined `Frame` produces two halves that do not meet.

The interface itself is decided jointly and belongs to `element` — see "The Frame
interface" below for what `window` is able to supply. If you find `Frame` cannot be
implemented as declared, say so rather than widening it here.

Everything in "Decisions to make first" can be worked out before `element` lands,
and should be, because it is the substance of this package.

## Read first

1. `AGENTS.md` — conventions, commit style, GB English
2. `docs/architecture.md` — "The frame", "State and reactivity", "Threading"
3. `docs/packages.md` — the `window`, `element`, `app`, `platform`, `render` entries
4. `_upstream/gpui/crates/gpui/src/window.rs` — 7,700 lines; read `Window`,
   `Frame`, `draw`, `draw_roots`, `present_if_needed` and `dispatch_event`, and
   skip the rest until you need it

Sync the checkouts with `go run ./tools/upstream` if `_upstream/` is not there.

## Stop after the frame loop

Not the whole package. One window, one root view, the six steps running end to end,
producing a quad on screen through `render` — and nothing else. No focus listeners,
no tooltips, no deferred draws, no IME, no cursor styles.

Then stop and say so.

That discipline has caught real defects twice on `platform`. It was skipped once, on
`render`, and produced two thousand lines built on a call to the wrong function.

## Two frames, not one

The structure that matters, and the one that is expensive to retrofit: a window holds
**two** frames — the one on screen and the one being built.

    rendered   what the user is looking at; hit regions, dispatch tree, focus
    next       what paint is filling in

Paint writes into `next`. Everything that answers a question about the *current*
state of the screen — where the pointer is, which hit region it is over, which node
has focus, what a keystroke dispatches against — reads `rendered`. At the end of a
frame the two swap and the new `next` is cleared.

One mutable frame is the obvious shape and it is wrong. Input arrives between
frames, and it has to resolve against geometry that exists. Halfway through paint a
single frame holds a partial element tree: hit regions for the elements painted so
far and none for the rest. A pointer event landing there hits nothing, or hits the
wrong thing, and the bug is a click that works except when it does not.

Decide what lives on a frame and what lives on the window. GPUI's `Frame` carries
the scene, hit regions, the dispatch tree, mouse listeners, focus and element state;
the window carries the platform handle, the renderer, the layout engine, scale
factor, pointer position and the dirty set. That split is a reasonable starting
point, but state it in `doc.go` as your decision rather than inheriting it silently.

## What window owns

**The frame loop** — the six steps in `docs/architecture.md`, in order, with no
phase reaching backwards.

**The layout engine.** A `layout.TaffyTree` lives here, is cleared each frame, and
is reached through `Frame`. `element` never holds one.

**Hit testing.** Regions are registered during prepaint and resolved against the
rendered frame. A hit test walks back to front, and the answer is a path, not a
single node — input dispatch needs the ancestors.

**Input routing.** `platform.Event` in, `input.DispatchTree` walked, handlers
invoked. `input` already owns precedence, focus and the explanation; `window`
supplies the hit path and the context stack, and re-implements none of it.

**Presentation.** Scene to `render.Renderer`, then present.

**The wiring nobody else can do**, because `window` is the only package that imports
both sides:

    app.ForegroundExecutor  ->  platform.Dispatch    background work returns to the UI goroutine
    text coverage masks     ->  render.Upload        glyph rasters reach the GPU atlas
    platform.Event          ->  input.DispatchTree   raw events become actions
    app notification        ->  a dirty view         state changes become frames

The first is called out explicitly in `platform`'s doc comment. The second is why
`docs/packages.md` says neither `text` nor `render` imports the other.

## Decisions to make first

Write the answers in `doc.go`. Each is cheap now and expensive after the loop exists.

**What schedules a frame.** `app`'s effects flush and mark views dirty; `platform`
delivers a paint or resize event. Those are two independent sources and they have to
produce one frame, not two, and not zero. Say who calls draw, what happens when a
notification arrives mid-frame, and what stops a frame that only redraws because the
last frame redrew.

**Whether a frame that changed nothing presents.** GPUI keeps a `needs_present`
flag, because presenting with vsync blocks for up to a frame interval and doing that
for an unchanged scene burns the main thread and the GPU for nothing. Decide, and
say what the idle cost of an open Facet window is.

**Where per-element state that survives the frame lives.** `docs/packages.md` says
elements are values discarded after paint and anything that outlives the frame
belongs in an entity. GPUI also keeps an element-state map on the frame, keyed by
element ID, for things like scroll offsets that are not worth an entity. Those two
answers conflict. Pick one and say which. If it is the map, say what evicts an entry
when the element stops being rendered — a map keyed by element ID with no eviction
is a leak that grows with every list the user scrolls.

**The order of resize.** `platform` reports a resize in logical pixels; `render`
sizes its swapchain in device pixels; the scale factor can change in the same
gesture when a window is dragged between displays. Say the order — swapchain, then
relayout, then paint, or otherwise — and what happens to a frame in flight.
`examples/quad` resizes the renderer and redraws from the event handler, which is
enough for one quad and is not a model for this.

**What a scale-factor change invalidates.** Glyph masks are rasterised at a
particular scale and cached by `text`; atlas tiles are allocated in device pixels. A
window moving from a 1.0 display to a 1.5 one invalidates both. Say what is dropped
and who drops it.

## The Frame interface

`element` declares it; `window` implements it. Elements never import `window` —
that would be a cycle, and the interface is what breaks it.

What `window` can supply, from what exists today:

    request layout         a node in the layout tree, with a style and children
    computed bounds        after the solve, for a node
    register a hit region  during prepaint, returning a handle
    paint primitives       the six in `scene`
    shape text             through `text`, at the window's scale factor
    scale factor           for anything that has to snap to device pixels

Keep it narrow. Every method on `Frame` is something `element` may do at any point in
any phase, and the phase ordering invariant is enforced here or not at all.

## Invariants

Implements `element.Frame`. The only package that sees both a `platform.Window` and
a `render.Renderer`.

The six frame steps execute here, in order. No phase reaches backwards: prepaint may
not request layout, paint may not register hit regions.

Everything runs on the UI goroutine. `window` never touches `app` state from
elsewhere; that is what the foreground executor is for.

Imports are the whole stack below it and nothing else — `element` does not import
`window`.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

The layering test passes. A test drives a frame with a stub renderer and asserts the
scene it produced — the primitives, in order, with their bounds — rather than
asserting that drawing returned no error. A test shows that input arriving between
two frames resolves against the frame on screen. A test shows that a notification
during a flush produces one frame and not two.

An example puts a styled element on screen through the real loop, replacing the
hand-wired `examples/quad`.

Conventional commits, one file per commit, staged by path.

## Habits from earlier rounds

When a struct of options exists, at least one test passes it empty. Two defects in
`platform` survived their suites because every test configured the field it was
about to check.

Assertions should know the answer. A test that a size "is positive" passed while the
size was wrong by a window frame. If a contract is written in a doc comment, the
test checks the contract.

Compiling is not evidence and not-erroring is not evidence. Three wrong D3D11 vtable
indices reached reviewed, fully green code; a pixel readback found them. The
equivalent here is asserting the scene's contents, because a frame loop that
produces an empty scene errors nowhere.
