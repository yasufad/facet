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
    platform                                  geometry colour
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

Pixels, DevicePixels, Rems, Point, Size, Bounds, Edges, Corners, Axis, and the
arithmetic over them.

Invariants: no dependencies at all, including on `colour`. Generic over a numeric
constraint so the same types serve logical pixels, device pixels and layout units.
Values, not pointers; everything here is copied freely.

## colour

Rgba and Hsla, conversion between them, and blending.

Invariants: no dependencies, including on `geometry`. Note the spelling — it is
`colour` throughout, in the package name and in every identifier.

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

## scene

The renderer's input language: Quad, Shadow, MonochromeSprite, PolychromeSprite,
Path, Underline, and the Scene that holds them in draw order.

Invariants: plain data, no behaviour beyond construction and ordering. Adding a
primitive is a decision that touches every renderer backend, so it is not a local
change.

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

## platform

The operating system: windows, the native event loop, main-thread dispatch,
displays, cursors, clipboard, menus, tray, dialogs and notifications. Also raw
input, because with no webview in the picture nothing else is reading it.

    platform/win32     HWND, Win32 message loop
    platform/cocoa     NSWindow, layer-backed view
    platform/gtk       GTK window, cgo bridge

The interface lives in `platform`; each backend is a subpackage behind build tags,
following the layout Wails uses — `<feature>_{darwin,linux,windows}.go` with the
cgo bridge isolated in its own file.

Much of this is vendored from Wails rather than written. See `docs/sources.md` for
what comes across, and put it in `third_party/`.

Invariants: this is the only package permitted to use cgo or `unsafe` for platform
calls. It surfaces a window and an input stream, and never a rendering API.

## render

The `Renderer` interface and the glyph and image atlases, with a backend per
graphics API.

    render/d3d11       Windows
    render/metal       macOS
    render/vulkan      Linux

A backend takes a native handle from `platform`, owns a swapchain, and draws a
`scene.Scene`.

Invariants: `Renderer` is a layer boundary and changes by explicit decision.
Backends never see anything above `scene` — no elements, no styles, no entities. A
new backend is a new subpackage and nothing else.

## text

Font loading and matching, script and bidi segmentation, shaping, line breaking,
and rasterising glyphs into the atlas.

Wraps `github.com/go-text/typesetting`, which is the one third-party dependency
here. Nothing above `text` knows that package exists.

Invariants: the shaped output is cached by run, not by string. Rasterisation is
still an open choice — see `docs/architecture.md`.

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
an excavation.

## Where to start

Four packages depend on nothing and start immediately:

    geometry   colour   app   layout

`geometry` and `colour` are both small, and until they land nothing that speaks in
pixels or colours can compile. They are the critical path, not the warm-up.

Once they are in, a second wave opens:

    scene   text   platform

`app` is large enough to run across both waves. `render` waits on `platform` and
`scene`; `style`, `input`, `element`, `window` and `ui` wait on the second wave.

A package whose dependencies are unwritten waits. Do not stub them — a placeholder
gets imported, drifts from the real API, and turns the merge into a rewrite.
