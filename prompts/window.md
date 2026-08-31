# Assignment: window

Build `window`: the frame loop. It owns a platform window, drives layout through
paint, presents the scene, and routes input into the dispatch tree.

Everything below is finished. `platform` produces events, `app` produces
notifications, `layout` solves, `text` shapes, `scene` collects, `render` draws,
`input` dispatches, and `element` builds the tree. None of them know about each
other. `window` is the only package that sees all of it, so every seam in
`docs/architecture.md` is closed here or not at all.

This is the last package with real design in it, and the one that finally puts a
Facet application on screen.

## Read first

1. `AGENTS.md` — conventions, commit style, GB English
2. `docs/architecture.md` — "The frame", "State and reactivity", "Threading"
3. `docs/packages.md` — the `window`, `element`, `app`, `platform`, `render` entries.
   The `element` entry is long and all of it applies to you.
4. `_upstream/gpui/crates/gpui/src/window.rs` — 7,700 lines. Read `Window`, `Frame`,
   `draw`, `draw_roots`, `present_if_needed` and `dispatch_event`. Skip the rest.

Sync the checkouts with `go run ./tools/upstream` if `_upstream/` is not there.

## Stop after the frame loop

One window, one root view, the steps running end to end, a `div` with a background
reaching the screen through `render`. Nothing else. No tooltips, no deferred draws,
no IME, no cursor styles, no arena yet.

Then stop and say so.

That discipline has caught real defects twice on `platform`. It was skipped once, on
`render`, and produced two thousand lines built on a call to the wrong function.

## What you implement

`element.Frame`, in full. It is already declared and it is a layer boundary:

    RequestLayout(style layout.Style, children []layout.NodeID) layout.NodeID
    LayoutBounds(id layout.NodeID) geometry.Bounds[geometry.Pixels]

    PushDispatchNode(node DispatchNode) input.DispatchNodeID
    PopDispatchNode()
    RegisterHitRegion(bounds, nodeID) HitRegionID

    IsHovered(HitRegionID) bool
    IsActive(HitRegionID) bool
    IsFocused(input.FocusID) bool

    InsertQuad / InsertShadow / InsertPath / InsertUnderline
    InsertMonochromeSprite / InsertPolychromeSprite

    ShapeLine(...) / ScaleFactor() / RemSize()

If something on it cannot be implemented as declared, say so rather than changing it
from here. `element` is finished and its prompt is retired; a change to `Frame` is a
decision, not a fix.

## The seven frame steps

`docs/architecture.md` lists them. Step 5 is new and it is easy to miss:

    1. flush effects
    2. request layout
    3. layout
    4. prepaint          elements register hit regions here
    5. hit test          resolve the pointer against the regions just registered
    6. paint             IsHovered / IsActive / IsFocused answer here
    7. present

Step 5 is one line in GPUI (`window.rs:3157`) and it is what makes hover work without
elements keeping identity across frames. A region registered in prepaint is resolved
before paint runs, so an element asking "am I hovered" is asking about the regions
this frame registered. Get this wrong and hover silently never fires, which is how
`element` lost a day.

## Two frames, and two hit tests

A window holds **two** frames:

    rendered   what is on screen; hit regions, dispatch tree, focus
    next       what paint is filling in

Paint writes into `next`. At the end of a frame they swap and the new `next` clears.

One mutable frame is the obvious shape and it is wrong. Halfway through paint, a
single frame holds hit regions for the elements painted so far and none for the rest,
so a pointer event landing there hits nothing or hits the wrong thing. The bug is a
click that works except when it does not.

The two hit tests are separate and conflating them is the trap:

- **Step 5**, above, resolves the pointer against `next` so that paint can answer
  hover. It runs once per frame, inside the frame.
- **Event routing** resolves an arriving `platform.Event` against `rendered`, because
  that is what the user is looking at and clicking on. It runs whenever an event
  arrives, between frames.

Say in `doc.go` which structure holds which, because the names will be similar and
the next person will assume there is one.

## Decisions to record in doc.go

**What schedules a frame.** `app`'s effects flush and mark views dirty; `platform`
delivers paint and resize events. Two independent sources, one frame, and never zero.
Say what happens when a notification arrives mid-frame, and what stops a frame that
redraws only because the last frame redrew.

**Whether an unchanged frame presents.** Presenting with vsync blocks for up to a
frame interval, so doing it for an unchanged scene burns the main thread and the GPU.
Say what the idle cost of an open Facet window is, as a number.

**The order of resize.** `platform` reports logical pixels, `render` sizes its
swapchain in device pixels, and the scale factor can change in the same gesture when
a window crosses displays. Say the order, and what is on screen between the swapchain
resize and the frame that follows it.

**What a scale change invalidates.** Glyph masks are cached by `text` at a particular
scale; atlas tiles are allocated in device pixels. Both are dropped, and `window` is
what drops them.

## The wiring nobody else can do

    app.ForegroundExecutor  ->  platform.Dispatch    background work returns to the UI goroutine
    text coverage masks     ->  render.Upload        glyph rasters reach the GPU atlas
    platform.Event          ->  input.DispatchTree   raw events become actions
    app notification        ->  a dirty view         state changes become frames

The first is named in `platform`'s doc comment. The second is why neither `text` nor
`render` imports the other.

## The arena, later

`element` measured itself: 584 bytes per `Div`, one allocation, about 420 ns. A
thousand elements a frame is ~420 µs and ~640 KB of garbage, which is 38 MB/s at
60 fps. The wall-clock fits; the allocation rate does not, because it arrives as GC
pauses inside frames.

GPUI enters an `ElementArenaScope` in `draw` and resets it at frame end. That arena
belongs here. It is not first-milestone work, but shape the frame boundary so it can
be added without moving the reset afterwards. `NewDiv()` takes no arguments, so the
arena has to be reachable without one: `element`'s entry in `docs/packages.md`
records the expectation that `window` establishes an active per-frame scope on the UI
goroutine.

## Invariants

Implements `element.Frame`. The only package that sees both a `platform.Window` and a
`render.Renderer`.

The frame steps execute here, in order. No phase reaches backwards: prepaint may not
request layout, paint may not register hit regions, and the three `Is*` queries are
valid in paint only. Panic otherwise, as `element`'s own phases do.

Everything runs on the UI goroutine. `window` never touches `app` state from
elsewhere; that is what the foreground executor is for.

## Done when

    go build -o bin/ ./...
    go test ./...
    go test -tags facet_debug ./...
    go vet -unsafeptr=false ./...
    gofmt -l $(go list -f '{{.Dir}}' ./...)

The layering test passes.

A test drives a frame with a stub renderer and asserts the scene produced: the
primitives, in order, with their bounds. Not that drawing returned no error.

A test shows a hover style reaching the scene in the same frame, driven through the
real loop rather than by setting state directly.

A test shows input arriving between frames resolving against the frame on screen.

A test shows a notification during a flush producing one frame and not two.

An example puts a styled `div` on screen through the real loop, replacing the
hand-wired `examples/quad`.

Conventional commits, one file per commit, staged by path. A rename lands in one
commit whatever it touches.

## Habits from earlier rounds

Stay in your package. Both cross-package problems found from here last time were
real, and both were fixed properly at their source because they were raised rather
than patched.

When a struct of options exists, at least one test passes it empty. Two defects in
`platform` survived their suites because every test configured the field it was about
to check.

A test that needs to write unexported state to pass is telling you the design cannot
happen in production. `element` shipped a hover test that did exactly that, with a
comment admitting it, and the feature never fired.

Compiling is not evidence and not-erroring is not evidence. Three wrong D3D11 vtable
indices reached reviewed, fully green code. A frame loop that produces an empty scene
errors nowhere.
