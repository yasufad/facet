# Packages

Each entry is a work assignment. It says what the package owns, what it is allowed
to import, and the invariants it must hold. An agent given one package should be
able to work from that entry plus the public API of its dependencies, without
reading the rest of the tree.

`docs/architecture.md` explains why the layers are shaped this way.

## The dependency rule

Imports run one way. A package may import anything listed under it, nothing above
it, and nothing that is not listed. A layering test enforces this; if it fails, the
design changed, not the test.

    geometry  colour                          nothing
    app                                       nothing
    layout                                    nothing
    scene                                     geometry colour
    platform                                  geometry colour third_party
    text                                      geometry colour
    render                                    geometry colour scene platform
    style                                     geometry colour layout text
    input                                     geometry platform
    element                                   geometry colour scene style layout text input app
    window                                    all of the above
    ui                                        geometry colour style element input app

Standard library imports are always allowed. Third-party imports are not, except
where an entry names one.

## geometry

Pixels, DevicePixels, ScaledPixels and Rems; Point, Size, Bounds, Edges and Corners
generic over them; Axis and Anchor for directions and reference points; and the
arithmetic over all of it.

Invariants: no dependencies at all, including on `colour`. Generic over a numeric
constraint so the same types serve logical pixels, device pixels and layout units.
Values, not pointers; everything here is copied freely.

Converting a `Bounds` to device pixels snaps both edges and derives the size from
them. Rectangles that touch in logical pixels touch in device pixels, for every
origin and scale factor. Rounding origin and size independently instead leaves
one-pixel gaps and overlaps along shared edges, which surface as hairlines and
double-blended alpha and get blamed on the renderer. A size converted on its own has
no edges to snap and is therefore approximate; convert the bounds when adjacency
matters.

## colour

Rgba and Hsla, conversion between them, blending, mixing and hex parsing.

Invariants: no dependencies, including on `geometry`. Note the spelling — it is
`colour` throughout, in the package name and in every identifier.

Components are float32 in the range 0 to 1, with straight alpha. The renderer wants
premultiplied components, so `Premultiply` is an explicit conversion rather than the
stored form; blending and interpolation stay accurate that way. `Blend` is a true
source-over composite accounting for both alphas, not the opaque-backdrop
simplification GPUI uses.

A named palette, gradients, colour spaces beyond sRGB and theming belong above this
layer.

## app

The entity map, `Entity[T]`, `Context[T]`, the effect queue, and the foreground and
background executors.

This is the reactive core and it knows nothing about drawing. Notification,
observation, subscription and the update cycle live here; so does `Task[T]`, because
the executor and the effect queue cannot be separated.

Invariants: the UI runs on one goroutine and a context never leaves it. Every
exported entry point that touches entity state, the effect queue or the subscriber
sets panics when called from another goroutine — an exported method is reachable
from anywhere, so none of them may assume a caller already inside an update. Effects
queue during an update and flush once at its end, so a burst of mutations produces
one frame. An `Entity[T]` is an identifier, never a pointer to the value.

Observers and subscribers fire in registration order. Go map iteration is random, so
this has to be arranged deliberately; without it, order-dependent bugs appear in one
run out of five and are close to impossible to reproduce. Hold them in order rather
than sorting when dispatching — dispatch is a per-frame path and must not allocate.

Goroutine identity costs about six microseconds to read, and every `UpdateEntity`
opens an update boundary, so checking it there cost most of a frame at a thousand
entities. It is now a `facet_debug`-only check. Release builds keep only the update
generation compare on `Context` accessors — a nanosecond, and it catches a context
stored and used after its update has ended.

A release build therefore does not catch a call from the wrong goroutine while an
update is still in flight. That gap is deliberate: the entity map and the reference
counts are unsynchronised by design, so cross-goroutine misuse is undefined behaviour
either way and the check only ever turned a race into a clean panic. Debug and race
builds keep it. Anything claiming a threading guarantee has to say which of the two
catches it, and in which build.

The map stores `*T`. `ReadEntity` returns the stored pointer rather than the address
of a copy, so a write through it is visible to every reader; `UpdateEntity` hands `f`
that same pointer. Leasing is unchanged and is what makes it safe — the value leaves
the map for the duration of an update, so a re-entrant update or a concurrent read
panics rather than racing.

Entity lifetime is reference counted by hand, and stays that way. Go 1.24's
`weak.Pointer[T]` with `runtime.AddCleanup` was spiked against this design and rejected
on evidence, so it does not need spiking again. A cleanup runs on its own goroutine at an
unspecified time, with no deadline and no guarantee of running before process exit, and
every exported entry point here asserts the UI goroutine — so `OnRelease` could not touch
state directly and would have to marshal through `AsyncApp`, which silently fails once
the foreground executor has stopped. An entity dying by collection after `App.Close`
would then never release at all. The lease does not survive either: it deletes the map
entry so a re-entrant access panics, and under a weak-keyed map that same deletion makes
a concurrent lookup by id report the entity dropped when it is merely being mutated.
Notification measured about twice as fast, which is real and does not pay for losing
deterministic release of GPU and platform handles. The decisive point is that `OnRelease`
would need an async-tolerant contract — a different contract, not a different storage
mechanism under the same one.

Every assertion on a value taken out of the map panics on failure, naming the entity
and the type. One of them returned silently instead, and when the map changed from `T`
to `*T` that assertion stopped matching: `OnRelease` simply never fired, on every
entity, with no diagnostic. A failed assertion here means the map holds something
other than what the entity's type says, which is unrecoverable — silence is not an
option any of them may take.

## scene

The renderer's input language: Quad, Shadow, MonochromeSprite, PolychromeSprite,
Path, Underline, and the Scene that collects them, orders them and groups them into
batches.

Draw order comes from an R-tree ported from GPUI: inserting bounds returns an order
one above anything they intersect. Three properties follow, and all three are load-
bearing. Overlapping primitives get strictly increasing orders, so occlusion is
correct. Primitives that share no screen space may reuse an order, which is what
makes batching possible at all. And batches must come out in draw order even where
types interleave — primitives live in per-type slices, so the obvious implementation
emits every quad then every sprite and quietly draws a glyph beneath the quad
painted over it.

Clips nest by intersection, and every primitive records the mask in force when it
was inserted.

Invariants: primitives carry no behaviour beyond construction. Adding a seventh
touches every renderer backend, so it is a decision to raise rather than a change to
make.

## layout

A port of Taffy's flexbox solver. Grid is out of scope.

The port carries its own types across with it — the style inputs, and Taffy's
`Size`, `Rect`, `Point` and `Line`. It imports nothing, not even `geometry`. Those
types are what the algorithm and its test suite are written against, and swapping
them for ours would make the port less faithful for no gain. `style` converts down
into this package and `element` converts results back out, so the vocabularies meet
at one boundary rather than throughout.

Invariants: Taffy's test suite is generated from browser behaviour and comes across
with the code. A behavioural difference from Taffy is a bug in the port, not a
local improvement. Every file carries its attribution header.

Every consumer of this package is outside it, so its tests are too: `layout_test`
builds a flex row, solves it and asserts the children's positions. It sees exactly
what a caller sees, which is the only way this package's real API gets exercised.
Internal tests hid a boundary that could not be driven at all — `ComputeLayout` named
an unexported type, and nine `Style` fields had unexported enums, so the flexbox
engine could not be told to do flexbox. `TestStyleAllNonDefaultFields` sets all
twenty-eight fields from outside; a field it cannot set is a dead field.

The fixtures in `layout/testdata/flex/` come from Taffy at the commit pinned in
`upstream.pins`. Their README records what is excluded and why. A drift test fails
when that directory and the upstream checkout disagree on anything not on the
exclusion list, so bumping the pin surfaces new fixtures instead of ignoring them.

`ComputeLeafLayout` and `OptF32` are exported deliberately so that higher layers
supplying custom leaf measurement functions (`MeasureFunction`) can perform standard
box-sizing, padding, border and min/max clamping arithmetic without duplicating the
solver's internal logic.

## platform

The operating system: windows, the native event loop, main-thread dispatch,
displays, cursors, clipboard, menus, tray, dialogs and notifications. Also raw
input, because with no webview in the picture nothing else is reading it.

    platform/win32     HWND, Win32 message loop     syscall
    platform/cocoa     NSWindow, layer-backed view  purego
    platform/gtk       GTK window                   purego

The interface lives in `platform`; each backend is a subpackage behind build tags,
following the layout Wails uses — `<feature>_{darwin,linux,windows}.go`.

Much of this is vendored from Wails rather than written. See `docs/sources.md` for
what comes across, and put it in `third_party/`.

Invariants: no cgo, here or anywhere. `CGO_ENABLED=0` builds on every target and
cross-compilation keeps working — see the requirements section of
`docs/architecture.md` for why that is a constraint rather than a preference. Win32
goes through `syscall` and `golang.org/x/sys/windows`; Cocoa and GTK go through
`github.com/ebitengine/purego`. If a platform turns out to be unreachable without
cgo, raise it rather than adding a build tag.

This is the only package permitted `unsafe` for platform calls, and only for memory
the operating system owns. A Go pointer never goes into OS storage — not into
`GWLP_USERDATA`, not into a callback's user data, nowhere the collector cannot see
it. The object is freed while the OS still holds its address, and the crash arrives
later under GC pressure looking like corruption. Keep a map keyed by the native
handle instead. Every `unsafe.Pointer` conversion carries a comment saying why it is
sound, because `vet`'s `unsafeptr` analyser is switched off — see the Commands
section of `AGENTS.md`.

It surfaces a window and an input stream, and never a rendering API.

Window geometry is in logical `Pixels`, not device pixels. A size in device pixels
means something different on each display, so a minimum size stops being a
constraint the moment the window is dragged to another monitor. `ScaleFactor()` is
there for the renderer, which does size its swapchain in device pixels.

Hiding a platform's units is the job; hiding what it measured is not. Wheel events
keep the distinction between an exact pixel delta from a trackpad and an inexact
line delta from a mouse notch, and carry the scroll phase, because neither can be
reconstructed once this layer has flattened them.

`WindowOptions.Size` is the client area, and a window reports back the size it was
asked for. On Windows that means deriving the native window style once and using
that same value both to adjust the frame and to create the window: the two drifted
apart while they were computed separately, and a request for 640×480 produced a
625×443 client area. Where a live value exists, read it rather than re-deriving —
resizing reads the window's current style rather than recomputing from options.

The cursor shape belongs to `Window`, the cursor's visibility to `Platform`. The split
is not stylistic: a pointer is over exactly one window, and Win32 answers the shape
per-window anyway, so `Platform.SetCursor` had no way to say which window it meant and
was silently wrong the moment a second one opened. Hiding the pointer really is
application-wide, so `SetCursorVisible` stayed. The rule the split follows: a method
belongs on `Platform` only when no window can be named for it.

IME composition is produced on Windows from `WM_IME_STARTCOMPOSITION`,
`WM_IME_COMPOSITION` and `WM_IME_ENDCOMPOSITION`, read through IMM32. `WM_CHAR` needs
no special handling — the IME generates it independently for the finalised text, so
dead keys on European layouts arrive through the ordinary text path and need no IME at
all. `IMECompositionEvent.Cursor` is a rune offset into `Text`, converted from the
UTF-16 code-unit offset IMM32 reports by counting units that are not low surrogates; a
character outside the basic plane is two units and one rune, and the conversion is
tested against one. IMM32 reports `0` rather than a sentinel for the cursor of an empty
composition, which is a legitimate answer rather than an error.

## render

The `Renderer` interface and the GPU side of the atlases, with a backend per
graphics API.

    render/d3d11       Windows
    render/metal       macOS
    render/vulkan      Linux

A backend takes a native handle from `platform`, owns the device and the swapchain,
and draws a `scene.Scene`.

Three packages touch the atlases and the split is deliberate. `text` rasterises a
glyph into a coverage mask and caches it by face, size and subpixel offset; it never
packs or uploads. `scene` carries only a reference — which texture, which tile, what
rectangle. `render` owns the GPU textures, allocates tiles within them, uploads
masks, and resolves the reference at draw time. Neither `text` nor `render` imports
the other; `window` sits above both and wires them together.

Shaders are compiled ahead of time and embedded with `go:embed`. HLSL, MSL and
SPIR-V bytecode is checked in; a user's build never runs `fxc`, `dxc`, `metal` or
`glslc`, because needing one would put a platform SDK back into the dependency list
through the side door. Compiling them is a job for our tooling and CI.

Correctness here is established by reading pixels back, not by checking for errors.
Almost everything in a graphics backend is an untyped number — a vtable slot, a
stride, a shader register, a format enum — and getting one wrong is a legal
operation that produces a black window. Three wrong vtable indices reached working,
reviewed, fully green code before a `facet_debug` readback path found them: the
swapchain was created by calling `IsCurrent`, and both draw calls were the indexed
variants drawing nothing with no index buffer bound.

So every primitive carries a pixel assertion, and each one fails if its own draw
path is disabled and no others. The readback copies the back buffer to a staging
texture and is compiled out of release builds.

One dynamic buffer serves every instance kind, written straight through a mapped
region rather than into an intermediate slice: `write` reserves a region and hands the
caller a slice over GPU memory to fill. It maps `DISCARD` on the first write of a
frame and `NO_OVERWRITE` at a rolling offset thereafter, so the driver renames the
buffer once per frame rather than once per batch.

Each reservation starts at a multiple of its own element size. Kinds share one buffer
and one byte offset, and D3D11 computes the address a draw reads from as
`StartInstanceLocation` times the stride bound for *that* draw — so a region left at
whatever odd offset a differently-sized batch ended on makes that multiplication land
outside the data just written. It is a legal read of the wrong bytes and reports no
error. `TestRenderMultiBatchSameKindInterleaved` pins it by chaining batches through
overlapping bounds so the scene cannot merge them, and reading back a pixel exclusive
to each; removing the rounding fails it on two pixels.

`ClearAtlas` invalidates every tile handed out for that kind. `scene.TileID` is
documented as the allocator's own handle and opaque to the Scene, so the backend packs
an 8-bit generation into its high byte and bumps it on clear; a stale tile panics under
`facet_debug` and is unchecked in release. Callers own dropping their references —
`window`'s glyph cache is the only one today, and it gets away with clearing in the same
handler, which is a property of that caller and not of the interface.

Invariants: `Renderer` is a layer boundary and changes by explicit decision.
Backends never see anything above `scene` — no elements, no styles, no entities. A
new backend is a new subpackage and nothing else.

Like `platform`, `render` backends are permitted `unsafe` for COM and graphics driver
interop, with the same condition: only for memory the OS or driver owns, never for Go
objects, and every conversion commented.

## text

Font loading and matching, script and bidi segmentation, shaping, line breaking,
and rasterising glyphs into the atlas.

Built on `github.com/go-text/typesetting` for font handling and shaping, and
`golang.org/x/image` for fixed-point types and outline rasterisation. This is the
package where third-party code is expected: text is the deepest problem in the
stack, and the established Go libraries for it are better than anything we would
write.

Invariants: shaped output is cached by run, not by string. What matters is the
boundary, not the dependency count — `text` exposes shaped lines, glyph runs and
coverage masks in our own types, so no layer above it knows what it is built on and
any of it can be replaced without reaching further up.

## style

Style properties, the refinement model that layers them, and the conversion into
`layout`'s vocabulary. The fluent builder is not here — `style` exposes mutators on
`*Refinement` and the chain hangs off the element, because `Refinement` is 504 bytes
and a value receiver copies it twice per call.

A refinement distinguishes unset from set-to-zero with a parallel 128-bit mask, one
bit per property and four for each compound property, so a hover refinement can set
one padding edge without disturbing the other three. Slice-valued properties are
immutable once set: setting replaces the slice, nothing appends to a stored one.

Invariants: no cascade and no stylesheet. Inheritance is explicit and confined to
text properties. Setting a property is a method on the element, resolved at build
time rather than looked up later. `Default()` is the only valid way to obtain a
`Style` — the zero value has opacity 0 and renders nothing, while `Refinement`'s zero
value means nothing set and is correct.

## input

Keymaps, actions, the focus tree, and dispatch from a raw event to a handler.

Invariants: takes raw events from `platform` and resolves them against a context
stack. Knows about focus, not about geometry beyond hit-test results handed to it.
Tab order belongs to `element`, which knows tree order; a focus hierarchy is not the
same thing, because a focus parent need not be the previous sibling in layout.

Binding precedence is four ordered rules and they are load-bearing: a deeper context
beats a shallower one, a later binding beats an earlier one at equal depth,
`NoAction` suppresses bindings at or below its precedence, and `Unbind` suppresses a
named action. An outer binding still fires from within an inner context; an unbound
chord neither matches nor pends; a chord that prefixes a longer one pends rather
than firing.

Dispatch has to be able to explain itself. `Explain` returns every candidate with
its depth, whether it matched, whether it won, and why — shadowed, context mismatch,
suppressed. A dispatch tree that cannot say why a keystroke did what it did is one
nobody can debug, and every framework that skipped this grew a worse version later.

`secondary` in a binding string is Cmd on macOS and Ctrl elsewhere, resolved by
build tag: `platform` reports which modifiers are held, never which system it is
running on.

`input` names the event vocabulary for everything above it: `KeyEvent`,
`PointerEvent`, `TextEvent` and `WheelEvent` are aliases for their `platform`
counterparts, declared here so that `element` and `ui` — both forbidden `platform`,
since it also hands out windows, the clipboard, menus and main-thread dispatch —
can write a handler without importing it. Aliases rather than distinct structs:
`platform`'s job is to hide a platform's units, not what it measured, and
`WheelEvent` deliberately keeps the distinction between a trackpad's exact pixel
delta and a mouse notch's inexact line delta. A re-declaration would risk losing
that distinction on some future edit; an alias cannot, because it is the same type.

The rule for what to alias is what a caller above has to *write*, not what this
package's own signatures happen to name. A handler signature naming `WheelEvent`
was not enough on its own — its body reads `event.Delta.Unit == platform.ScrollPixels`
to decide whether the delta is pixel-exact, so `ScrollUnit`, `ScrollPixels` and
`ScrollLines` are named here too. A constant cannot be aliased, so the two are
declarations, but because `ScrollUnit` is aliased, `input.ScrollPixels` and
`platform.ScrollPixels` are the same value of the same type and compare equal
across the boundary. `ScrollDelta` itself is not named: nothing above reaches it
except through a `WheelEvent`'s `Delta` field, so no caller has to write the type
name, and it gets added only once one does.

## element

The Element interface, the three-phase lifecycle, the element tree, and `div`.

Defines the `Frame` interface that elements use to request layout, register hit
regions and paint. `window` implements it. Elements never import `window` — that
would be a cycle, and the interface is what breaks it.

It owns the fluent style builder. `style` exposes mutators on `*Refinement` and the
chain — `NewDiv().Flex().Gap(4).Bg(c)` — hangs off the element, which is already
behind a pointer, so no style struct is copied per call.

Invariants: elements are values built fresh each frame and discarded after paint.
Anything that must survive the frame belongs in an entity. Layout, prepaint and
paint run strictly in that order and no phase may reach backwards. Out-of-order
calls panic.

The `Frame` contract: It is a layer boundary and changes by explicit decision.
`RequestMeasuredLayout` accepts a `MeasureFunc` allowing leaf nodes (such as `Text`)
to shape and measure content dynamically within flexbox constraints.
`RasteriseGlyph` resolves and uploads glyph masks to the GPU atlas texture during paint.
`PushDispatchNode(DispatchNode)` and `PopDispatchNode()` hand a node over whole,
so there is no implied "active node" to get wrong. The text style stack is carried
on `Frame` (`PushTextStyle`, `PopTextStyle`, `TextStyle`), pushed and popped around
children during layout and paint; container elements merge active pseudo-state
refinements (hover, active, focus) into the pushed style before children paint so
child `Text` elements immediately reflect parent state. `PushClip(Bounds)` and `PopClip()`
confine child primitives to container bounds during paint; layers are deliberately not
exposed yet. `IsHovered`, `IsActive`, `IsFocused`, `PushClip`, `PopClip` and `RasteriseGlyph`
are valid during paint only. `ShapeLine` is legal during layout solve and paint.

Overflow clipping: `Div` honours `style.Overflow`: when overflow is `Hidden`, `Clip`, or
`Scroll`, `Div` pushes its bounds onto `Frame`'s clip stack around painting its children
and pops it afterwards.

Tab order collection: Tab order requires tree order, collected in Prepaint as focusable
elements register dispatch nodes. Elements declare participation via `TrackFocus(id)`,
can opt out via `TabStop(false)`, or set an explicit ordering index via `TabIndex(n)`.
Window sorts active tab stops (positive indices first ascending, followed by index 0 in
tree order; negative indices excluded). Tab and Shift-Tab key events advance and reverse
focus along this sequence, wrapping at both ends and skipping unrendered elements.

`NodeID` alias: `element.NodeID` is an alias for `layout.NodeID` so widget authors
implementing `Element.RequestLayout` do not need to import `layout`.

Text rendering: `Text` is a single-line, left-aligned text element that measures
via `Frame.RequestMeasuredLayout`, inherits typographic styling from `f.TextStyle()`,
and emits `scene.MonochromeSprite` primitives for each glyph in paint with subpixel
offsets. The baseline sits half the difference above the ascent when the line box is
taller than the font's own metrics, as CSS does — caret height comes out of the same
arithmetic, so getting the leading wrong once gets the caret wrong twice.

Pseudo-state may change font metrics, not only colour. A container's `Hover` setting a
heavier weight is a supported case, so `Text` cannot shape once during layout and assume
it: `f.TextStyle()` in paint carries refinements merged in after prepaint, and hover is
resolved between prepaint and paint. `Text` therefore re-shapes when the resolved
`[]text.StyleRun` differs from the one it was shaped from. Comparing the runs rather than
a chosen list of style fields is what makes this stay correct — the runs *are*
`ShapeLine`'s input, so a new shaping input is covered the day it is added, and a colour
change never reaches them and never re-shapes. The alternative considered and rejected was
restricting pseudo-state to colour, which would have been a silent behavioural cut to
something `Div.Hover` already supports.

Available width is not part of that comparison. `ShapeLine` wraps nothing, so its output
is identical at every width, and the solver calls a leaf measure several times per solve
with different constraints.

Hit regions are per-frame: Monotonic, per-frame identifiers. A region registered
during prepaint is resolved by `window` at step 5 (the intra-frame hit test) and
queried in paint of the same frame. Elements keep no identity across frames and
there is no element-keyed map. The obvious implementation carries an ID forward,
and that cannot work when the element is rebuilt every frame. Visual paint styles
(backgrounds, borders, shadows, text colours) evaluate in paint with zero frame
lag; layout-altering hover styles lag by one frame when tracked in persistent
entity state, because `RequestLayout` precedes prepaint and hit testing.

`ClickEvent` is ours: A click is synthesised from down and up on the same target,
so `element` declares it in `geometry` units (`Position`, `LocalPosition`,
`MouseButton`, `Modifiers`) rather than naming a `platform` type. `element` never
imports `platform`.

Exported test double: `element/elementtest` provides an exported `Frame` double that
records layout requests, hit regions, dispatch nodes, and primitives, enabling widget
packages (such as `ui`) to test lifecycle and interaction without importing `platform`,
`scene`, `text`, or `render`.

What an element costs: 584 bytes per `Div` (embeds `style.Refinement`, 504 of them),
one allocation, about 420 ns baseline. Styling adds no allocations (~3% to 5% CPU
overhead). `NewDiv()` takes no arguments, which is what a future arena in `window`
has to work around: `window` establishes an active per-frame arena scope on the
single UI goroutine during frame draw.

It imports `input` because an element declares key contexts, focus handles and
action handlers, and that vocabulary lives there. Without it neither `element` nor
`ui` could express a click. `input` sits below both, so this runs downward like
everything else.

## window

The frame loop. Owns a platform window, drives layout through paint, presents the
scene, and routes input into the dispatch tree.

Implements `element.Frame` in full. This is where the seven frame steps are executed
in order:

1. `flush effects` — drains reactive notifications in `app`, runs observers, marks dirty views
2. `request layout` — walks root view, constructs `layout.TaffyTree` nodes
3. `layout` — solves flexbox layout, computes window-relative node bounds
4. `prepaint` — elements commit bounds, push dispatch nodes, register hit regions in `next`
5. `hit test` — resolves intra-frame pointer against `next.hitRegions` for zero-lag hover
6. `paint` — elements query `IsHovered`/`IsActive`/`IsFocused` and emit scene primitives
7. `present` — swaps `rendered` and `next`, resets `next` and layout tree, resizes swapchain if needed, submits scene to GPU

- Invariants and guarantees:
- Implements `element.Frame`. The only package that sees both a `platform.Window` and
  a `render.Renderer`.
- Two-frame isolation: `rendered` holds the on-screen scene, hit regions and dispatch
  tree for user event routing. `next` holds in-construction state. They swap on presentation.
- Two distinct hit tests: Step 5 intra-frame hit testing resolves the pointer against
  `next` before paint runs, so hover queries answer in the same frame without element
  persistent identity. Event routing resolves arriving `platform.Event` instances against
  `rendered`, because that is what the user sees and interacts with. Conflating the two
  breaks click handling or hover state.
- Phase ordering is enforced strictly:
  - `RequestLayout` and `RequestMeasuredLayout` are valid in layout only.
  - `PushDispatchNode`, `PopDispatchNode`, and `RegisterHitRegion` are valid in prepaint only.
  - `IsHovered`, `IsActive`, `IsFocused`, `PushClip`, `PopClip`, `RasteriseGlyph`, and `Insert*` primitive insertions are valid in paint only.
  - `ShapeLine` is valid in layout solve and paint.
  - `RequestFocus` is on `Frame` and is valid during event handlers, prepaint, and paint.
  - `phaseLayoutSolve` restriction: only `ShapeLine` is legal during layout solve. Calling any other `Frame` method (including `RequestLayout`, `RegisterHitRegion`, `Insert*`, `RasteriseGlyph`, or `RequestFocus`) from a measure callback panics immediately.
  - Under `facet_debug`, `window` asserts that the clip stack is empty at the end of paint and panics if an element pushed a clip without a matching pop.
- Measure seam: `RequestMeasuredLayout` registers a measurement callback against a Taffy leaf node, solved through `layout.ComputeLayoutWithMeasure` and `layout.ComputeLeafLayout`.
- Glyph rasterisation & texture atlas caching: `RasteriseGlyph` queries `text.Atlas` for coverage masks, uploads them to the GPU via `render.Renderer.Upload`, and caches the resulting `scene.AtlasTile`.
- Focus lifecycle, tab navigation, and unmount behaviour: Pointer down on a focusable hit region moves keyboard focus to that element's focus ID; clicking empty background blurs focus. Tab and Shift-Tab key events advance and reverse focus along the frame's tab order, wrapping at boundaries. Elements can request focus programmatically via `Frame.RequestFocus(id)`. When an element with active focus is not rendered in the subsequent frame (unmounted, conditionally removed, or scrolled out), focus drops to nothing (`Blur()`) at Step 7 presentation. Stale focus IDs never survive after leaving the tree.
- Pointer cursor resolution: Cursor shape is resolved in Step 5 (intra-frame hit test) from the hovered hit region's cursor property. The window calls `platform.Window.SetCursor` once per frame only when the cursor shape changes, avoiding redundant OS syscalls and mouse flicker during continuous motion within the same element.
- Frame scheduling: `frameScheduled` collapses an event or notification burst into
  exactly one frame turn. Idle cost is 0 GPU draw calls and 0 present calls per second;
  the event loop sleeps.
- Reactivity: `SetRootView` observes the root view's entity (`app.Observe`); entity
  notifications mark the window dirty and schedule a redraw automatically. Reactivity
  stops at the root view: secondary entities read during `Render` must be observed
  explicitly by the parent view.
- Re-rendering: the root view re-renders unconditionally on every frame; precise
  per-view dirty tracking and subtree caching are deferred to a future milestone.
- Resize order: `ResizeEvent` records logical dimensions; swapchain resize and atomic
  presentation happen during the frame loop without flashing blank or stretched buffers.
- Scale factor invalidation: `ScaleChangedEvent` clears the CPU glyph cache (`text.Atlas`),
  GPU glyph tile cache, and GPU texture atlas (`render.Renderer.ClearAtlas`), resizing the swapchain and relayouting.
- Threading: all frame loop operations run on the single UI goroutine. Background operations
  marshal back via `app.ForegroundExecutor` wired to `platform.Dispatch`. `ScheduleFrame`
  is thread-safe via an internal mutex so background tasks can request redraws safely.
- Verification by pixel readback: correctness across the integrated stack (`platform` ->
  `window` -> `element` -> `style` -> `layout` -> `scene` -> `render`) is established by
  reading swapchain backbuffer pixels back in `window_debug_test.go` under `facet_debug`,
  including both quad layouts and rasterised text glyphs.

## ui

Buttons, labels, lists, text fields, scroll views.

Invariants: built entirely from the public API of `element`, `style` and `input`. If
a widget needs something those do not expose, the gap is in the framework and gets
fixed there. No widget registry — adding a widget adds a file and touches nothing else.

## third_party

Vendored upstream source, one directory per project, keeping its original structure.
Every file carries an attribution header naming the upstream file and its licence,
and every project appears in `NOTICE`.

Invariants: changes here are ports and patches, not features. Record what was
changed and why, so the next update against a newer upstream is a merge rather than
an excavation. `third_party/README` is that record.

When vendored code fights you, work out what it knows before restructuring it. Its
awkward shapes are usually paid for: `mainthread_windows.go` posts to a hidden
window instead of the thread queue because a modal inner loop swallows thread-queued
messages, and it carries the issue link to prove it. Its same-thread requirement was
restructured away once, and the deadlock it prevented came straight back in another
form. Read the scars before cutting.

## internal

Repository-level test packages, belonging to no layer. They sit outside the
dependency table and may import anything; the layering test names them unconstrained
alongside `tools` and `examples`. Nothing here is part of the framework and nothing
here is imported by it.

`internal/layering` enforces the table above under three `GOOS` values, because a
build constraint hides a violation on the platform nobody happens to be compiling
for.

A test that spans the stack goes here too, and has nowhere else it could go. `window`
may not import `ui` and `ui` may not import `window`, so the path a user actually
writes — a widget, in a window, clicked — is reachable from no package's own tests.
That path is what `examples/button` demonstrates, and it had never once been executed
when `docs/audit.md` was written. `window`'s own tests reach far enough for the three
defects the audit names, which is why the assignment sits there; they do not reach the
widget library.

Keep these test-only. A package here that exports something for other packages to use
is the cross-package fixture `AGENTS.md` rules out, and it drifts from every package it
serves.

## Order of work

The dependency table above is also the order packages can be built in.

    depend on nothing        geometry  colour  app  layout
    then                     scene  text  platform
    then                     render  style  input
    last                     element  window  ui

A package whose dependencies are unwritten waits. Do not stub them — a placeholder
gets imported, drifts from the real API, and turns the merge into a rewrite.

`prompts/` holds the assignments currently in hand, and one is retired when its
package is done. It is not a list of what remains — a package with no prompt has
either been finished or not yet been assigned. `go list ./...` says which.
