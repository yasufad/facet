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
    element                                   geometry colour scene style layout text app
    window                                    all of the above
    ui                                        geometry colour style element app

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

Goroutine identity costs about six microseconds to read, so it is checked at update
boundaries and on exported methods, where a handful per frame is affordable. Context
accessors compare an update generation instead, which is a nanosecond and catches a
context used after its update ends. The case neither covers cheaply — a context used
from another goroutine while its update is still running — is caught by a full check
that a `facet_debug` build turns on at every accessor. Anything claiming a threading
guarantee has to say which of the three catches it.

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

The fixtures in `layout/testdata/flex/` come from Taffy at the commit pinned in
`upstream.pins`. Their README records what is excluded and why. A drift test fails
when that directory and the upstream checkout disagree on anything not on the
exclusion list, so bumping the pin surfaces new fixtures instead of ignoring them.

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

Style properties, the refinement model that layers them, and the fluent builder that
elements expose.

Invariants: no cascade and no stylesheet. Inheritance is explicit and confined to
text properties. Setting a property is a method on the element, resolved at build
time rather than looked up later.

## input

Keymaps, actions, the focus tree, and dispatch from a raw event to a handler.

Invariants: takes raw events from `platform` and resolves them against a context
stack. Knows about focus, not about geometry beyond hit-test results handed to it.

## element

The Element interface, the three-phase lifecycle, the element tree, and `div`.

Defines the `Frame` interface that elements use to request layout, register hit
regions and paint. `window` implements it. Elements never import `window` — that
would be a cycle, and the interface is what breaks it.

Invariants: elements are values built fresh each frame and discarded after paint.
Anything that must survive the frame belongs in an entity. Layout, prepaint and
paint run in that order and no phase may reach backwards.

## window

The frame loop. Owns a platform window, drives layout through paint, presents the
scene, and routes input into the dispatch tree.

Invariants: implements `element.Frame`. This is where the six frame steps in
`docs/architecture.md` are actually executed, and the only package that sees both a
`platform.Window` and a `render.Renderer`.

## ui

Buttons, labels, lists, text fields, scroll views.

Invariants: built entirely from the public API of `element` and `style`. If a widget
needs something those do not expose, the gap is in the framework and gets fixed
there. No widget registry — adding a widget adds a file and touches nothing else.

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
