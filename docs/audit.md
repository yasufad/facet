# Audit

Written after reading every package, running the suite, and probing the claims the
documentation makes about itself. It records what works, what does not, and the order
the remaining decisions have to be taken in.

The short version: the layer design is sound and several packages are genuinely good,
but the reactive model does not survive contact with an event handler. A view cannot
mutate its own state from a click. That is not a missing feature — it is the central
idiom of the design, and it panics. Everything downstream of it is blocked on the same
decision.

## Verification

Every claim below was reproduced. Commands, probes and measurements are named where they
matter, so the next person can check them rather than trust them.

Measurements ran on the machine that produced them; the absolute numbers will move on
other hardware. The ratios are what the argument rests on.

## The tree is red

`go build ./...` and `go test ./...` both fail on a Linux or macOS checkout:

    FAIL  internal/layering            ui imports platform, which its entry does not permit
    FAIL  examples/button              undefined: platform.New
    FAIL  examples/quad                undefined: platform.New
    FAIL  tools/compile_shaders        undefined: syscall.NewLazyDLL

The layering failure is known and assigned in `prompts/input.md`. The other three are
not. `examples/` and `tools/compile_shaders` carry Windows-only code with no build
constraint, so the two commands `AGENTS.md` names as the build and test commands do not
pass anywhere except Windows. Neither does `go vet`.

There is no `.github/`. No continuous integration runs anything, on any platform. That
is the root cause of everything in the next section: a broken example can sit in the
tree indefinitely because nothing ever builds it, let alone runs it.

## The reactive model does not work

`examples/button` is the flagship example. It cannot ever have been run.

```go
func (c *CounterView) Render(cx *app.Context[CounterView]) element.Element {
    return element.NewDiv().Children(
        ui.NewButton("Click Me").
            OnClick(func(event element.ClickEvent) bool {
                c.count++
                cx.Notify()
                return true
            }),
    )
}
```

`Render` runs inside `UpdateEntity`. `c` is a pointer to the leased copy on the stack;
`cx` is a context whose generation expires when that update returns. The closure is
stored in the dispatch tree and invoked later, from `DispatchEvent`, outside any update.
Reproduced with a test that builds a view, captures the handler the way `window` does,
and calls it:

    PANIC when the click handler ran: app: context used after its update has ended
    stored count after the click handler ran: 0 (expected 1)

Both halves fail. `cx.Notify()` trips the generation check, and the increment lands on a
copy that `endLease` has already superseded. There is no way to write a correct handler
with the signature the framework offers, because `func(ClickEvent) bool` gives the
callback no route back into the entity. Capturing is the only option, and capturing is
wrong.

GPUI does not have this problem because the framework re-enters the entity update when
the event fires, so the handler is handed live state and a live context rather than
captured ones. Facet copied the element lifecycle and the entity map but not the part
that joins them.

Confirmed at the source. `Context::listener` in `crates/gpui/src/app/context.rs` is the
whole mechanism, and it is an adapter rather than a change to any handler signature:

```rust
pub fn listener<E: ?Sized>(
    &self,
    f: impl Fn(&mut T, &E, &mut Window, &mut Context<T>) + 'static,
) -> impl Fn(&E, &mut Window, &mut App) + 'static {
    let view = self.entity().downgrade();
    move |e: &E, window: &mut Window, cx: &mut App| {
        view.update(cx, |view, cx| f(view, e, window, cx)).ok();
    }
}
```

The handle is downgraded at render time, upgraded and updated at dispatch time, and a
dropped view is a silent no-op rather than an error. `Window::listener_for` is the same
body for callers who hold the entity rather than a context. Dispatch itself runs at the
app level — `dispatch_event(&mut self, event, cx: &mut App)` — outside any update, which
is what makes the re-entry legal.

One thing does not carry over, and it is in Facet's favour. GPUI threads `&mut App`
through every callback because the borrow checker gives the closure no other way to reach
it. Go has no such constraint: the adapter captures `cx.App()` and `cx.WeakEntity()` at
registration, so nothing needs adding to the handler signature. Probed directly — the
adapter compiles against today's `app`, the increment persists across two dispatches, and
`cx.Notify()` from inside it reaches an observer. That makes step 1 an addition to
`element` rather than a change to it.

`ui.Button` has the same signature, so the only interactive widget in the library is
unusable from a view. Its test passes because it closes over a test-local `clicked`
variable and never goes near an entity. `AGENTS.md` warns about exactly this pattern —
"interaction needs a test that sets one thing and checks another" — and the warning was
not applied to the case it was written for.

## Reads silently discard writes

`ReadEntity` returns `*T`, which every Go programmer reads as a mutable view of the
stored value. It is a pointer to a copy:

```go
v := app.entities.read(handle.id)   // any
t, ok := v.(T)                      // copies out of the interface box
return &t                           // address of the local
```

Probed directly: write 42 through the returned pointer, read again, get 1. No panic, no
warning. The doc comment says nothing about it.

This is a worse trap than the handler problem, because it fails quietly. Either return
`T` by value and let the compiler stop the write, or store `*T` in the map so the
pointer aliases the real thing. The second is better for other reasons; see below.

## Ownership is manual in a garbage-collected language

`Entity[T]` is reference counted by hand. `Clone` increments, `Release` decrements, and
every long-lived holder is expected to pair them. Go has no destructor, so there is no
equivalent of the Rust `Drop` that makes this work upstream, and both failure modes are
bad: a missed `Release` leaks the entity and every observer registered against it, with
no diagnostic, and a double `Release` panics at a point unrelated to the mistake.

`Context.Observe` already shows how much bookkeeping this costs. Every notification
upgrades two weak handles, calls the callback, then releases both, with a comment
explaining that the handles passed to the callback were borrows. That is four
refcount operations per observer per notification, and one wrong branch leaks silently.

This is worth reopening rather than accepting. Go 1.24 added `weak.Pointer[T]` and
`runtime.AddCleanup`, which between them give exactly the semantic the entity map wants:
the map holds the value, handles are ordinary reachable references, and an entity is
dropped when nothing can reach it. The alternative is to keep counting but make the count
unnecessary in the common case, by giving entities an explicit owner and an explicit
`Drop`. Either beats asking every caller to balance a counter.

The decision does not block anything today, because the tree is small enough that leaks
do not show. It gets much harder to change once `ui` has thirty widgets in it.

## Hit testing ignores clipping

`PushClip` pushes a mask onto the scene. Hit regions are a flat slice with no mask at
all, scanned in reverse insertion order:

```go
func hitTest(regions []hitRegion, pt geometry.Point[geometry.Pixels]) (...) {
    for i := len(regions) - 1; i >= 0; i-- {
        if regions[i].bounds.Contains(pt) { return ... }
    }
}
```

A button scrolled out of a `ScrollView` is invisible and still clickable. So is anything
clipped by an overflow-hidden container. The fix is small — carry the clip rectangle in
force onto the region at registration and test against the intersection — but it has to
happen before any list, panel or scrolling editor is trusted.

## Pointer capture does not exist

Already found by `ui` and recorded in its prompt, and it deserves to be here too because
it is the widest single gap in the input layer. Press inside an element, drag outside it,
and the move events go to whatever is now under the pointer. Text selection, sliders,
resize handles, scrollbar thumbs, drag and drop and window-edge resizing all need the
window to route to the pressed element until release. `prompts/ui.md` records that GPUI
holds a captured hitbox on the window for exactly this.

## IME is declared and not wired

`platform.IMECompositionEvent` exists, with a careful doc comment about `WM_IME_*` on
Windows and `NSTextInputClient` on macOS. Nothing produces it: `platform/input_windows.go`
never handles a `WM_IME_` message, and `window.DispatchEvent` has no case for it. A type
with no producer and no consumer is a plan written in Go rather than in prose, and it
reads as done to anyone scanning the package.

Any editor that expects to be used outside Latin scripts needs this. So does dead-key
input for European keyboards.

## What the frame actually costs

The frame loop rebuilds everything, every time, for any reason. A mouse move sets
`w.dirty` and schedules a full turn: re-render the root view, rebuild the element tree,
allocate a fresh Taffy tree, solve flexbox, prepaint, hit test, paint, present.

    w.rendered, w.next = w.next, w.rendered
    w.next.clear()
    w.layoutTree = layout.NewTaffyTree()

That last line throws away the layout tree at the end of every frame. `layout` carries a
faithful port of Taffy's per-node cache, with the sign-bit key packing and the
second-chance replacement cursor intact. It is discarded before it can help across
frames, so it only ever serves the multiple passes within one solve. A hover colour
change re-solves the whole flexbox tree from nothing.

Measured costs, from the benchmarks already in the tree:

| What | Cost | Per frame at scale |
| --- | --- | --- |
| `UpdateEntity` + `Notify`, 1000 entities | 4.87 ms, 3745 allocs | 29% of a 16.6 ms budget |
| `checkUI` (goroutine identity) | 3.16 µs | two per entity update |
| Element tree build | ~1 KB, 1.5 allocs per element | ~950 KB at 1000 elements |
| `Text` measure, width varies | 37.5 µs, 32 KB, 92 allocs | per text element |
| `Text` measure, fresh each frame | 18.6 µs, 17 KB, 72 allocs | per text element |

The first row is the one to look at. `app`'s doc explains that the 6 µs goroutine check
is affordable because it runs "at update boundaries" and "a frame opens a few update
boundaries". That is not what happens. Every `UpdateEntity` is an update boundary, every
`Notify` is an exported method, and every observer firing opens another `UpdateEntity`
of its own. The check is the dominant cost of the reactive core, and the sentence
claiming otherwise is prose that was true of the design and never true of the code.

`checkGeneration` already demonstrates the right shape: an integer compare in release,
the full check under `facet_debug`. `checkUI` should follow it. The generation counter
plus `-race` covers what matters, and release builds should pay nothing.

The text numbers are the other wall. Flexbox calls a leaf measure function several times
with different available widths; `element.Text` caches on `lastAvailWidth`, so every
distinct width re-shapes. Fifty visible text elements at 18.6 µs is a millisecond before
anything is drawn. A code editor has several hundred.

## Allocation on every hot path

One allocation per `UpdateEntity`, measured. The entity map stores `any`, so `endLease`
re-boxes the value on every update. Storing `*T` removes it and removes the copy in both
directions.

The renderer allocates a fresh instance slice per batch per frame:

```go
data := make([]quadInstance, len(quads))
```

then copies it into the mapped GPU buffer, so every primitive is written twice. Each
batch maps with `D3D11_MAP_WRITE_DISCARD` from offset zero, which forces the driver to
rename the buffer once per batch rather than once per frame. The standard shape is a
persistent scratch buffer written straight into a mapped region, `DISCARD` on the first
map of a frame and `NO_OVERWRITE` with a rolling offset thereafter. At five thousand
glyphs the current path is roughly half a megabyte of garbage per frame from the
renderer alone.

`input.DispatchTree.nodePath` allocates a `map[DispatchNodeID]bool` per dispatch, on the
pointer and key paths. `text`'s shape cache key builds a string on every lookup, hit or
miss, through `string(in.language)` and `featuresKey`. Pointer resolution scans the hit
region slice linearly and then scans it two or three more times to fetch the cursor, the
focus id and the bounds of the region it just found.

None of these individually matters at ten elements. All of them together set the ceiling
on how large a UI Facet can carry, and that ceiling is the thing the project is
ultimately being judged on.

## Caches that only grow

Four unbounded caches, none with eviction:

The glyph tile cache in `window`, keyed by face, glyph, size bits and subpixel offset.
The CPU coverage-mask cache in `text.Atlas`. The shaped-run cache in `text`, which will
accumulate an entry per distinct word per face per size for the life of the process. And
the GPU atlas in `render/d3d11`, whose shelf packer has no free list at all: `allocate`
walks a cursor forward and `clear` resets everything. When a page fills, another page is
created and the old one is never reclaimed.

For a demo this is invisible. For an editor left open across a day, with theme changes,
font size changes and several languages, it is a slow leak by construction. An atlas that
never frees a tile is only correct for a process that never changes what it draws.

## Smaller defects, all reproducible

Each of these is narrow enough to fix in one landing, and each is real.

`window.New` starts a goroutine that ranges over `app.Foreground().Pending()` and never
stops it. `ForegroundExecutor.stop` closes `done`, not `wake`, so the range never ends;
the goroutine holds the platform, the app and the window alive for the process lifetime,
and `Window.Close` does not touch it. `ForegroundExecutor.stop` also lacks the
`sync.Once` its background counterpart has, so calling `App.Close` twice closes a closed
channel and panics.

Pointer-down on any hit region without a focus id calls `focusTree.Blur()`. Clicking a
button therefore blurs the text field next to it, because a button registers a hit region
and does not track focus. The rule wanted is that clicking a focusable element moves
focus and clicking anything else leaves focus alone unless it explicitly asks to take it.

`Window.Draw` returns early when the root view renders nil, after setting `w.phase =
phaseLayout` and before restoring it; the reset on that path sets `phaseNone` but the
scene, the tab order and the clip depth are left as they were. `Draw` also has no
re-entrancy guard, and `RequestFocus` is legal during paint and calls `ScheduleFrame`.

`ui.ScrollView.Paint` calls `state.Update` during the paint phase to record its viewport
and content heights. It works because it does not notify, and it is one line away from an
infinite frame loop if anyone adds one. Writing entity state from paint should either be
a supported operation with a stated rule or a panic, not an accident that holds.

The subscriber slice compaction in `app` nils the tail on every `retain` pass, which is
correct, but `effectQueue.pop` advances with `q.items = q.items[1:]` and so hands the
backing array back to the allocator after each full drain. A head index would keep it.

`Div` resolves its full style three times per frame — once in `RequestLayout`, once in
`Prepaint`, once in `Paint` — building a 488-byte `Style` from a 504-byte `Refinement`
each time. Resolving once in `RequestLayout` and carrying the result on the element costs
nothing, since the element is already behind a pointer.

## What is missing to build something like Zed

Set aside the defects. These are things nothing in the tree attempts yet, ordered by how
much they block.

There is no virtualisation. `ui.ScrollView` scrolls by putting a negative top inset on a
relatively positioned content div, which means the entire content is built, laid out and
painted every frame and then clipped. A hundred thousand lines is a hundred thousand
elements. GPUI's `uniform_list` and `list` exist precisely for this, backed by
`sum_tree`, and the seam they need is an element that learns its viewport before it
builds its children. Facet's `Element` interface gives `Prepaint` the bounds, but
`RequestLayout` has already built the whole subtree by then. Closing that gap is a
change to the element contract, not a widget.

There is no way to draw above your siblings. `scene` has `PushLayer` and `PopLayer` and
they work; `element.Frame` does not expose them, and the package doc says layers are
deliberately not exposed yet. Without them there are no menus, no context menus, no
tooltips, no autocomplete popups, no dialogs, no drag ghosts, and no combo boxes. That is
most of a real application's chrome.

There is no image. `scene.PolychromeSprite` exists, the D3D11 pipeline draws it, and no
element anywhere constructs one. No decoding, no asset path, no `Img`.

There is no animation and no timer. No frame ticker, no easing, no transition, no way to
schedule work at a wall-clock deadline on the UI goroutine. A blinking caret is out of
reach, and so is any hover transition, spinner or scroll inertia.

Text stops at one line with one style run. No wrapping, no selection geometry, no bidi at
the element level, no mixed styling within a paragraph. `text` handles bidi and script
segmentation already, so this is element work, and it is the work an editor is made of.

There is no accessibility of any kind. No UI Automation on Windows, no `NSAccessibility`,
no AT-SPI. Nothing in the tree so much as names a role. Retrofitting this later means
touching every widget, because it needs a tree that mirrors the element tree and survives
the frame — which is the same identity problem virtualisation has.

Multiple windows are half-there. `platform` keeps a map of windows keyed by `HWND`, but
`SetCursor` is a method on `Platform` rather than on `Window`, and `window.Window` owns a
single root view, a single focus tree and a single renderer. Two windows would fight over
the cursor.

macOS and Linux do not exist. This is known and recorded.

## What is right

Saying this plainly matters, because the list above is long and the work underneath it is
not bad work.

`scene` is the strongest package in the tree. The R-tree draw-order assignment is ported
from GPUI with its three load-bearing properties understood and written down: strictly
increasing orders for overlap, shared orders for disjoint bounds so batching is possible,
and batches emitted in draw order even where types interleave. The batch cursor merges
across six per-type slices by repeatedly picking the lowest `(order, kind)` pair, which
is the non-obvious part and is correct.

`layout` is a real port rather than a reimplementation, including the browser-derived
fixtures, and it has a drift test that fails when the checked-in fixtures and the pinned
upstream disagree. That is the difference between a port and a rewrite that resembles
one.

The Windows backend keeps its window map keyed by `HWND` rather than stuffing a Go
pointer into `GWLP_USERDATA`, with a comment saying why. The vendored `mainthread` code kept its
hidden-window dispatch and the issue link explaining the modal-loop message swallowing.
Both are scars that were read rather than cut.

`style` earns its design: mask-based refinement with four bits per compound property,
measured at zero allocations, and unstyled and fully styled element construction within
2% of each other.

`internal/layering` checks imports under three `GOOS` values, which is the only way to
catch a violation hidden behind a build tag. It is currently failing, which is the test
doing its job.

The D3D11 backend is verified by reading the swapchain back rather than by checking for
errors, and that found three wrong vtable indices in code that was green and reviewed.
That practice should be copied into every backend that follows.

And the documentation is better than the code. `architecture.md`, `packages.md` and
`sources.md` are unusually clear about what was decided and why. The gap this audit found
is not that people wrote things down carelessly; it is that some of what is written down
was true of the plan and was never checked against the result.

## The path

Ordered by dependency. Each of these is an architectural decision first and a diff
second, which is the right way round given where the effort actually goes.

### 1. Give handlers a way back into the entity

Nothing else in the reactive layer can be judged until this is settled, and every widget
written before it will be rewritten after it.

A listener registered by a view closes over a `WeakEntity[T]`, and the framework
re-enters `UpdateEntity` when the event fires, handing the callback `(v *T, cx
*Context[T])`. In Go that is a helper in `element`:

```go
func Listener[T, E any](cx *Context[T], f func(v *T, e E, cx *Context[T]) bool) func(E) bool
```

which upgrades the weak handle at dispatch time, opens the update, calls `f`, and drops
the handle. Handlers registered outside a view keep the plain signature.

This is a change to `element`'s public API and to how `ui` writes every widget. Decide
it, write it into `docs/packages.md`, then change `element`, `ui` and both examples in
one landing.

While the decision is open, fix `examples/button` to whatever the current API can express
correctly, or delete it. A broken example is worse than none.

### 2. Store pointers in the entity map

`map[entityID]any` holding `*T` rather than `T`. One box at construction and none after.
`lease` takes the pointer out of the map, which keeps the re-entrancy guard exactly as it
is, and `endLease` puts the same pointer back rather than writing a value through. The
guard survives; only the copying goes. `ReadEntity` then aliases the stored value
honestly, and the copy trap disappears with it.

Then move `checkUI` behind `facet_debug`, as `checkGeneration` already is, and re-run
`BenchmarkFrameSimulatesUpdateNotify`. Expect most of the 4.87 ms to go.

### 3. Make the frame incremental

Three tiers of invalidation, because they cost different amounts and the current code
pays the highest one for all three:

Paint only, for hover, active, focus and pure colour changes. Keep the element tree and
the solved layout; re-run paint. This is the mouse-move path and it is currently the
most expensive thing a user can do by accident.

Layout, for content and size changes. Keep the element tree structure and the Taffy
nodes; re-solve. Requires that the layout tree survive the frame, which requires stable
node identity across frames.

Structure, for a view that actually re-rendered. What happens today, unconditionally.

Stop allocating `layout.NewTaffyTree()` per frame. Replace `nodeBounds` and
`measureCallbacks` with slices indexed by node id rather than maps; the ids are dense
integers already.

### 4. Decide element identity

More depends on this than on anything else in the list, and it should be taken
deliberately rather than arrived at.

Facet's current answer is that elements have no identity: hover works because hit
regions are resolved intra-frame, and the reasoning is written down and is good. But
virtualised lists need to know which item an element was, animations need to know what a
value was interpolating from, layout caching needs to know which node this is the same as,
and accessibility needs a tree that outlives the frame. All four are the same requirement.

GPUI's answer, read rather than assumed: `GlobalElementId` is an `Arc<[ElementId]>`, a
path from the root rather than a single id, and the frame holds `element_states:
HashMap<(GlobalElementId, TypeId), ElementStateBox>`. `Window::with_element_state` looks
the entry up in the frame being built, falls back to the rendered frame, and records the
key in `accessed_element_states`; anything not accessed during a frame is dropped when
the two frames swap. State therefore lives exactly as long as an element keeps being
built at the same path, and eviction needs no policy.

Two details matter for what follows. State is reachable in prepaint as well as paint
(`debug_assert_paint_or_prepaint`), which is what lets a list learn its viewport and then
decide what to build. And the carry-forward is between the same two frames Facet already
keeps, so the mechanism lands on `Frame` rather than changing the `Element` interface.

Take that shape or a better one, but take it before the list element rather than during
it.

### 5. Expose layers, and add deferred painting

`Frame.PushLayer` and `Frame.PopLayer`, plus a way for an element to defer a paint
closure to run after its siblings. That combination is what makes a popup anchored to a
button that draws above the panel it lives in. It is a change to the `Frame` contract, so
it is a decision, not an implementation.

### 6. Pointer capture on the window

Capture on press, route pointer moves and the release to the captured region regardless of
where the pointer is, clear on release. Small, and it unblocks selection, sliders, resize
handles, scrollbar thumbs and drag and drop at once.

Carry the clip rectangle onto hit regions in the same landing, since both are the same
kind of fix to the same slice.

### 7. The list element

With identity and layers in place, build the virtualised list: measure one item, learn
the viewport in prepaint, build only what is visible plus an overscan margin. This is the
element that decides whether an editor is possible.

### 8. Bound the caches

Reference-counted atlas tiles with a free list, an LRU on the shape cache and the glyph
cache, and a size ceiling on each expressed in bytes rather than entries. Measure the
steady-state footprint of a long-running window and publish the number.

### 9. Arena-allocate elements

Already scoped in `architecture.md`. Take it after the frame is incremental, not before,
because incremental invalidation changes how many elements are built per frame and
therefore what the arena is sized for.

### 10. macOS before Linux

Not for reach — for design pressure. Metal's pipeline state and command encoder model
differ enough from D3D11's immediate context that `render.Renderer` may need to change,
and it is far cheaper to find that out with one backend written than three. Cocoa also
forces the main-thread and run-loop questions that `platform` has so far only had to
answer in the Win32 shape. If purego cannot carry `objc_msgSend` with struct returns,
that is a decision to record, and the sooner it is recorded the less is built on top of
the assumption.

Linux third, with OpenGL rather than Vulkan as the first backend, for the reason
`architecture.md` already gives.

## Process, which is the actual bottleneck

The instruction that code is not the bottleneck is right, and the evidence for it is in
this audit: every package passes its own tests, and the application does not run. The
process produces locally correct, globally broken software, and it does so reliably.

Three changes fix that.

Add continuous integration. Build and test on Windows, Linux and macOS, run `go vet`,
run `gofmt -l`, and run the `facet_debug` suite. The layering test has been failing on
`main` and the only reason anyone knows is that a prompt says so.

Make an integration test that drives a real frame the way a user does. Not a package
test — a test that builds a view with state, dispatches a synthetic click through
`window.DispatchEvent`, and asserts the state changed and the next frame reflects it. That
one test would have caught the handler defect, the `ReadEntity` copy, and the clip-blind
hit testing. Three of the four blocking defects in this document are integration
failures between packages that individually work.

Fix the examples so they build everywhere and run them in CI under a headless or
software-rasterised path where possible. An example that cannot be built by the people
maintaining it is documentation of an API nobody has used.

The per-package assignment model is working. The seams between packages are where
everything has gone wrong, and the seams are exactly what no single agent's assignment
covers.
