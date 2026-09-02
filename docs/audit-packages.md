# Deep audit, layer by layer

`docs/audit.md` looked across the tree and found the defects that block the design.
This one goes down each package in dependency order, from the units up to the widgets.
Everything below was reproduced against the code as it stands; where a claim needed a
probe, the probe and its output are quoted.

One pattern runs through the whole stack and is worth stating before the detail.

**Facet is built wide and connected narrow.** Six scene primitives, six shaders, six
batch kinds, a pixel-readback assertion for each. Two of the six can be produced from
any API above `scene`. Sixty-odd style properties with a fluent builder over all of
them, and thirteen that nothing reads. A platform interface covering windows, menus,
dialogs, tray and notifications, with six methods returning "not implemented". Each
layer looks complete from inside itself. The vertical slice through them is one
primitive wide.

That is the shape of the remaining work, and it is not visible from any single package's
tests, because each package is testing the breadth it built.

---

## geometry

The best-reasoned package in the tree, and the claim it makes about itself holds.

`BoundsToDevicePixels` snaps both edges and derives the size from them, so rectangles
that touch in logical pixels touch in device pixels. I checked it rather than trusting
it, over 400 fractional origins across seven scale factors including 1.25, 1.5 and 1.75:

    adjacent-edge mismatches across 7 scale factors x 400 origins: 0

`PointToDevicePixels` really does round identically to the origin of the bounds
conversion, and `SizeToDevicePixels` says in its own doc comment that it does not, and
why. That is the standard the rest of the tree should be held to.

**`IsContainedWithin` is wrong.** It tests `outer.Contains(b.BottomRight())`, and
`Contains` is half-open, excluding the bottom and right edges. So:

    b.IsContainedWithin(b) = false

A rectangle is not contained within itself, and any rectangle flush with its container's
far edge reports as escaping it. The method has no callers anywhere in the tree, which is
why nothing has broken; it is exported API that is both dead and incorrect. Either fix
the comparison to be inclusive on the far corner or delete it.

`DevicePixels.ToBytes` converts through `uint32` without checking the sign, so a negative
extent produces about four billion bytes rather than an error. It is only ever called
with sizes, but nothing enforces that.

`Number` admits `~int` and `~int32`, so `Point.Div`, `Size.Centre` and the `/2` inside
`CentredAt` and `FromAnchorAndSize` truncate on integer units. That is the right
behaviour for device pixels and worth a sentence in the constraint's doc, since the same
call is exact on the float units.

---

## colour

Straight alpha stored, `Premultiply` as an explicit conversion, and a genuine source-over
`Blend` that accounts for both alphas. All correct, and the reasoning in
`docs/packages.md` matches the code.

**Every colour operation happens in gamma-encoded sRGB, and so does the GPU blend.** The
swapchain is created as `DXGI_FORMAT_B8G8R8A8_UNORM`, not the `_SRGB` variant, so the
output-merger blends encoded values directly. `Mix`, `Blend`, `Lighten` and `Darken` do
the same on the CPU. The polychrome atlas is `R8G8B8A8_UNORM`, so images are sampled and
blended encoded too.

This is what CSS does and what most toolkits do, so it is defensible. It is not what
produces good text. Greyscale coverage blended against a gamma-encoded backdrop makes
light text on a dark background look thin and dark text on a light background look heavy,
and the error is largest at the small sizes an editor uses. Zed-class text rendering is
mostly a story about getting this right.

What makes it a finding rather than a preference is that nobody decided it. There is no
sentence anywhere in `docs/` saying which space blending happens in. Pick one, write it
down, and if it stays sRGB, say that text AA is the thing that pays for it.

`Hsla.Mix` interpolates hue linearly rather than around the shorter arc, which the doc
comment states plainly. That is a defensible default and correctly flagged.

---

## layout

The strongest execution in the repository, and the model for how a port should be done.

2508 of 2656 upstream fixtures are carried, the 148 exclusions are enumerated by
category with a reason each, and `TestFlexFixtureDrift` compares the directory against
the pinned upstream checkout and fails on anything unaccounted for **in either
direction**. Bumping the pin surfaces new fixtures rather than silently ignoring them.
Aspect ratio, gap, baseline alignment and the intrinsic-size cache are all present.

The cache is a faithful port including Taffy's sign-bit key packing and its
second-chance replacement cursor. `window` allocates a fresh tree every frame and throws
it away, so it only ever serves the multiple passes within one solve. That is `window`'s
defect, recorded on its prompt, and it is worth repeating here that the thing being
discarded is good.

Nothing else to report. This package is done.

---

## app

Three findings beyond the ones already in `docs/audit.md`.

**A background task that calls `Update` after the app closes blocks for ever.** Verified:

    background updater BLOCKED for 2s after App.Close - Update enqueued work nothing drains

`AsyncApp.run` enqueues the closure onto the foreground queue and then blocks on a
result channel. The shutdown check that is supposed to return "application has been shut
down" lives *inside* that closure, so it can only run if something drains the queue,
and after `Close` nothing does. The error path the doc comment promises is unreachable by
construction. The check has to happen before the enqueue.

**`rc.shutdown` is read without its mutex.** `markShutdown` writes it under `r.mu` and
`refCounts.release` reads it under `r.mu`, but `async.go` reads it bare at lines 51 and
83, from background goroutines, on the one type documented as "the only context that may
cross an await point". It is a data race by inspection. It has not been caught because no
test exercises concurrent shutdown, and the deadlock above stops the reads happening at
all in the obvious test.

**`NewApp` binds the UI goroutine to whoever calls it, and nothing says it must be the
goroutine that later calls `platform.Run`.** Every `AsyncApp` operation ends up in
`app.update`, which calls `checkUI`. That only succeeds if the foreground queue is
drained on the goroutine that constructed the App. `examples/button` gets it right by
accident of ordering. Construct the App anywhere else (an `init`, a setup goroutine, a
test helper) and every background update panics from inside the platform dispatcher,
with a stack that names neither mistake. This is a cross-package invariant with no
statement and no check.

The rest is on `prompts/app.md`: the entity map hands out pointers to copies, the
goroutine check dominates the reactive core, the foreground executor's wake channel is
never closed.

---

## scene

The R-tree draw ordering and the six-way batch cursor are the best code here. The three
load-bearing properties are understood and written down, and the merge across per-type
slices by repeatedly taking the lowest `(order, kind)` pair is the non-obvious part and
is right.

**A zero-area clip does not clip.** `ContentMask` uses an empty rectangle to mean "no
clipping", which is a reasonable encoding until something pushes a genuinely empty clip.
Then the two meanings collide. Verified both ways:

    quads inserted under a zero-height clip: 1 (expected 0)
      its recorded mask: {Origin:{X:0 Y:0} Size:{Width:200 Height:0}}

    quads under (50x50 then 200x0): 1
      mask is the OUTER clip, not nothing: {Origin:{X:0 Y:0} Size:{Width:50 Height:50}}

Standalone, a collapsed container's children paint over the whole window. Nested, the
collapsed clip is discarded and the parent's is inherited, so children escape their own
container's bounds into their grandparent's. Both are reachable from ordinary layout: a
flex child that resolves to zero height, a panel mid-collapse, a scroll view before its
first solve.

The shaders agree with the CPU, since `if (mask.z > mask.x && mask.w > mask.y)` skips
clipping on a degenerate mask. So this is consistent, and consistently wrong. The fix is a
separate "no mask" state rather than overloading emptiness: a bool on the mask, or a
sentinel, or making the empty case clip everything and giving the unclipped case its own
representation.

Second, smaller: the CPU `Bounds.Contains` is half-open and the shader test is inclusive
on the maximum edges (`> mask.z` discards, so a fragment exactly at `mask.z` survives).
One pixel of disagreement between where a clip cuts and where the matching hit region
ends.

`boundsTree.clear` resets the node slice but each internal node's `children []int` is
freshly allocated on the next frame's inserts. Minor, on a per-frame path.

---

## text

Beyond what is on `prompts/text.md` (`ShapeLine` costing 37 µs on a cache hit, the
allocating cache key, and the two unbounded caches) there is a typographic defect.

**Half-leading is not implemented.** `ShapedLine` places the baseline at `ascent` from the
line's top, and `element.Text` paints glyphs at `bounds.Origin.Y + position.Y`. When the
line box is taller than `ascent + descent`, the whole difference lands below the text
rather than being split above and below it.

That is not an edge case, because `DefaultTextStyle` ships with `FontSize: 16` and
`LineHeight: 20`. Every text element in the framework's default configuration already has
leading it puts entirely on one side, and any UI that sets a line height of 1.4 or 1.5,
which is every editor, puts glyphs hard against the top of their line box.

The rule to implement is CSS's: `(lineHeight - (ascent + descent)) / 2` above the ascent.
It belongs in `element.Text` rather than here, since `text` reports the metrics correctly
and the line box is the element's.

Three subpixel buckets at thirds of a pixel is a reasonable choice and matches the atlas
key. Worth a note somewhere that it is a choice, since it caps horizontal positioning
precision at a third of a device pixel.

---

## platform

The window itself is thorough and the Win32 work is careful. Every method on the `Window`
interface is implemented. The handle map is keyed by `HWND` rather than stuffing a Go
pointer into `GWLP_USERDATA`, with a comment saying why, and the vendored `mainthread`
code kept its hidden-window dispatch and the issue link explaining the modal-loop message
swallowing. Those are scars that were read rather than cut.

**The shell around the window is mostly absent, and the README says "Windows".** Six
methods return errors and four are silent no-ops:

    NewSystemTray        not implemented
    ShowMessageDialog    not implemented
    ShowOpenDialog       not implemented
    ShowSaveDialog       not implemented
    SendNotification     not implemented
    SetApplicationMenu   TODO: per-window menus
    Hide / Show          TODO: enumerate windows
    SetIcon              TODO: load icon from bytes

No file open dialog means no "Open File…". That is not a nicety for an editor, it is the
first menu item. The status table reads as though the Windows backend is finished; what
is finished is a window, its input, the clipboard and the cursor.

**Nothing in the interface can minimise, maximise, restore or go fullscreen.** Not
missing from the backend. Missing from `platform.Window` itself. The only mention of a
window state anywhere is one `SIZE_MINIMIZED` test inside the wndproc. Any application
with `Decorated: false`, which is what a custom title bar requires and what Zed does,
must implement its own window controls, and there is nothing for them to call.

`Clipboard` is text only. Fine for an editor, worth knowing.

IME is on `prompts/platform.md`: the event type exists, nothing produces it.

---

## render

The verification discipline is right and should be copied into every backend that
follows: correctness is established by reading the swapchain back, not by checking for
errors, and that found three wrong vtable indices in code that was green and reviewed.
The blend state is correct premultiplied source-over (`ONE` / `INV_SRC_ALPHA`).

**Four of the six primitives have no producer.** Counting call sites in `element` and
`ui`:

    InsertQuad                1
    InsertMonochromeSprite    1
    InsertShadow              0
    InsertPath                0
    InsertUnderline           0
    InsertPolychromeSprite    0

Six shaders are compiled, embedded and asserted against real pixels. Four of them can
only be reached by a caller that does not exist. `architecture.md` says six primitives
are enough to draw a text editor, and that is true. Two of them are enough to draw a
button.

The rest is on `prompts/render.md`: a fresh instance slice per batch per frame, every
primitive written twice, `MAP_WRITE_DISCARD` once per batch instead of once per frame,
and an atlas whose shelf packer has no free list.

---

## style

The mask accounting is sound, and I checked it rather than assuming, because
`AGENTS.md` records a previous bug where a guard skipped everything above bit 63. The
low word uses bits 0 to 50 of 64 and the high word 64 to 79, with the `mask.lo != 0` and
`mask.hi != 0` guards correctly scoping each block. Four bits per compound property and
two per paired one, as documented. Zero allocations, measured.

`ToLayout` maps every layout-relevant property. Nothing is dropped on the way down.

**Thirteen properties have builder methods and no consumer.** The complete set of
resolved `Style` fields ever read above this package, across `element`, `window` and
`ui`, is: `Display`, `Overflow`, `MouseCursor`, `Background`, `BorderColour`,
`BorderWidths`, `BorderStyle`, `CornerRadii`, and eight text fields. Everything else is
set and forgotten:

    Visibility              Div.Visible(), Div.Invisible()
    Opacity                 Div.Opacity()
    BoxShadow               Div.BoxShadow()
    TextBackgroundColour    Text.TextBackgroundColour()
    Underline               Text.Underline()
    Strikethrough           Text.Strikethrough()
    WhiteSpace              set, unread
    TextOverflow            set, unread
    LineClamp               set, unread
    TextAlign               reaches layout, ignored in paint
    AllowConcurrentScroll   set, unread
    RestrictScrollToAxis    set, unread
    ScrollbarWidth          reaches layout, no scrollbar exists

These compile, read correctly at the call site, pass through the refinement mask
faithfully, and change nothing on screen. That is worse than a missing method, because a
missing method is a compile error and this is a silent one.

The fix is not to implement all thirteen at once. It is to decide, per property, whether
it is implemented or not yet offered, and to stop offering the ones that are not. A
builder method is a promise.

---

## input

Binding precedence, the four ordered rules, and `Explain` are the most thoughtful part of
this package, and the argument in `docs/packages.md` for why dispatch must be able to
explain itself is right.

The capture and bubble ordering is correct. I checked, because it is the classic place to
get a path backwards. `nodePath` builds target-to-root and then reverses, so the
capture loop ascending the slice really does go root to target.

`nodePath` allocates a `map[DispatchNodeID]bool` on every dispatch to guard against a
parent cycle. A cycle cannot occur: `PushNode` assigns `id = len(nodes)` and takes its
parent from the stack, so a parent index is always strictly less than its child's. The
map is a guard against an impossible state, paid for on every pointer move, alongside a
`path` slice that grows from nil each time. Both should be scratch buffers on the tree.

`DispatchText` delivers only to the focused node, with no capture or bubble phase, unlike
the other three dispatchers. A container cannot observe text input reaching a child. That
may be deliberate; it is not written down, and the asymmetry will surprise whoever writes
the second text widget.

The `platform` types in the handler signatures are already assigned on
`prompts/input.md` and are why the tree is red.

---

## element

Beyond the listener defect that `docs/audit.md` opens with and the width invalidation on
`prompts/element.md`:

`Div.Paint` reads `Display`, `Background`, `BorderColour`, `BorderWidths`, `BorderStyle`,
`CornerRadii` and `Overflow`. It does not read `Visibility`, `Opacity` or `BoxShadow`.
`Text.Paint` emits glyph sprites and nothing else: no text background quad, no underline,
no strikethrough, and no alignment.

`Opacity` is the interesting one of the three, because implementing it properly is not a
one-liner. CSS opacity is a group property: the subtree composites as a unit and then
fades, which needs an offscreen target or a layer. Multiplying each primitive's alpha
gives a different and wrong result wherever children overlap. Decide which one is being
offered before implementing it, and say so on the method.

`Div.Prepaint` returns early on `DisplayNone` without prepainting children, and
`RequestLayout` has already built their layout nodes by then. Taffy handles
`display: none` by not laying the subtree out, so the nodes are inert, but the element
tree work was still done.

---

## window

`staticView` and `fnView` both return an empty `app.Subscription` from `Observe`. So
`SetRoot` and `SetRootFn` attach nothing, and a window configured through either never
repaints in response to entity state, only in response to input, which sets `dirty`
directly on every event. It looks reactive because moving the mouse redraws everything.
Neither helper's doc comment mentions it.

Everything else is on `prompts/window.md`: no pointer capture, hit regions blind to the
clip stack, pointer-down on any non-focusable region blurring focus, four linear scans
where one would do, the layout tree discarded every frame, and the pump goroutine that
cannot be stopped.

---

## ui

Two widgets, and the defects above meet in both of them.

**`Button.Disabled(true)` is invisible.** It sets `Cursor(CursorNotAllowed)` and
`Opacity(0.5)`, and opacity is one of the thirteen unread properties. A disabled button
renders identically to an enabled one; the only signal is the cursor, and only while the
pointer is over it. The button's own tests do not catch it because they assert on
listeners and layout, not on emitted primitives.

That is the clearest illustration of why the unread-property finding matters. Nobody
wrote a bug. Someone reached for the property the framework offered, and the framework
was offering something it does not do.

`ScrollView` scrolls by putting a negative top inset on its content, so the entire
content is built, laid out and painted every frame and then clipped. Per the `scene`
finding above, it is not clipped at all if the viewport ever resolves to zero height.
It also writes entity state during paint, which works only because it does not notify.
Both are on `prompts/ui.md`.

---

## What this changes about the plan

The ordering in `docs/audit.md` still holds. The listener seam is still first and element
identity is still the decision the most depends on. Three things move.

**Settle the unread properties before more widgets are written.** Every
widget written against `Opacity` or `BoxShadow` today is written against a promise the
framework does not keep, and `Button.Disabled` already is. This is cheap to fix in the
direction of honesty, by removing the setters that have no consumer, and that is the
version to do first: it converts thirteen silent failures into thirteen compile errors
that say what is missing.

**The empty-clip encoding should be fixed alongside the clip-aware hit regions.** They
are the same subject and `window` is already opening that code. Fixing hit-region
clipping while leaving a collapsed container unclipped would produce a UI where the
pointer is correctly blocked from something still visibly painted over the window.

**Half-leading belongs with the text field, not after it.** A caret's height and position
come out of the same line-box arithmetic, so getting the leading wrong once means getting
the caret wrong twice.
