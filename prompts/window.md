# Assignment: window

Build `window`: the frame loop. It owns a platform window, drives layout through
paint, presents the scene, and routes input into the dispatch tree.

This is the package where everything below finally runs. `platform` produces events,
`app` produces notifications, `layout` solves, `text` shapes, `scene` collects,
`render` draws — and none of them know about each other. `window` is the only place
that sees all of it, so every seam in `docs/architecture.md` is closed here or not at
all.

## Response to the plan — read this before anything else

**You are blocked, and waiting is the work.** Layer 2 of the plan creates
`element/doc.go`, `element/frame.go`, `element/element.go` and `element/div.go`.
`element` is assigned to another agent and that agent is writing those files now. Two
agents on the same four files collide, and the plan's `Element` — three methods, no
per-phase state mechanism, no identity — quietly settles the two decisions
`prompts/element.md` exists to have decided properly.

Drop Layer 2 entirely. `Frame` arrives from `element`; you implement it. Nothing in
`window/` can be written until it lands. The section below on `Frame` says what
`window` is able to supply, and that is your input to that agent's design, not a
licence to write it yourself.

**Two things you found are real, and both belong to other packages.**

`layout.ComputeLayout` genuinely cannot be called from outside `layout` —
`availableSpace` is unexported and `definiteAvail`, `minContent` and `maxContent`
are all lower-case, so no caller can name the type or obtain a value. Both
`ComputeLayout` and `ComputeLayoutWithMeasure` are dead API to every package above.
That is a good find in a package marked finished. It is `layout`'s to fix, under its
own prompt. Do not export it from here.

The `Frame` you proposed takes `input.DispatchNodeID` and returns
`*input.DispatchTree`, which `element` was not permitted to import. Chasing that
through, neither `element` nor `ui` could name an action, a key context or a focus
handle, so no widget could declare a click. The table was wrong, not your instinct:
`element` and `ui` now import `input`, as of `8b223e1`. Report a layering conflict
next time rather than routing around it — that test failing is the design asking to
be decided.

**Decisions 1, 2, 3, 5 and 6 are accepted.** The two-frame model is understood
correctly, including that pointer and key events resolve against `rendered`. The idle
answer — zero draws, zero presents, event loop asleep — is right and stated as a
number, which is what makes it checkable. Decision 3 puts surviving state in entities
with no unevicted map, and matches what `element`'s agent is being asked for
independently.

**Decision 4 has a gap.** The swapchain resizes on the event and relayout waits for
the next frame. Say what is on screen in between — a stretched present, a stale one,
or nothing.

**Two corrections to the plan's `Frame`.** `TextSystem() *text.System` and
`TextAtlas() *text.Atlas` hand out mutable internals wholesale; the need is to shape
text at the window's scale factor, so expose that. And a scene assertion is the
verification here — running `examples/quad` and looking at it is not evidence, for
the reason `docs/packages.md` gives about the render backend.

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

Write the answers in `doc.go`. Five of the six are settled above; what remains is
decision 4's gap, and everything here stays as the record of what had to be answered.

**What schedules a frame.** `app`'s effects flush and mark views dirty; `platform`
delivers a paint or resize event. Those are two independent sources and they have to
produce one frame, not two, and not zero.

**Whether a frame that changed nothing presents.** Presenting with vsync blocks for
up to a frame interval, so doing it for an unchanged scene burns the main thread and
the GPU for nothing.

**Where per-element state that survives the frame lives.** Entities, with no
element-keyed map — agreed, and `element` is answering the same question.

**The order of resize.** `platform` reports logical pixels; `render` sizes its
swapchain in device pixels; the scale factor can change in the same gesture when a
window is dragged between displays. Say the order, and what is on screen between the
swapchain resize and the frame that follows it.

**What a scale-factor change invalidates.** Glyph masks are rasterised at a
particular scale and cached by `text`; atlas tiles are allocated in device pixels.
Both are dropped, and `window` is what drops them.

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
any phase, and the phase ordering invariant is enforced here or not at all. Hand out
capabilities, not internals: shape a line, do not return the text system.

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

Stay in your package. Where you need something another package does not expose, say
so and stop — that is a design decision, and both of the ones you found this round
were real and were fixed properly at the source.

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
