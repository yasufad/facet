# Queued: the second round

Nothing in this file is assigned. It is the work `docs/audit-packages.md` found, ordered
by what blocks what, held here until the packages it touches are free.

`prompts/*.md` is what is in hand. This is what comes after, and it exists because the
gap between those two had nowhere to live: a finding with no assignment was either
written into a prompt someone was already working from, or lost. When a package reports
and its prompt retires, its section here becomes its next prompt.

Three decisions are taken below rather than left open, because each crosses a layer
boundary and none should be settled by whoever happens to pick the work up.

---

## Decided: the test for a widget in a window goes in internal/integration

`prompts/build.md` asks for one test that dispatches a click and asserts the state
changed, and puts it in `window`. That is right for what it describes: `window`'s fakes
reach the handler seam, the `ReadEntity` copy and the clip-blind hit testing, which are
three of the four blocking defects.

They do not reach a widget. `window` may not import `ui` and `ui` may not import
`window`, so nothing in the tree can test a `ui.Button` clicked in a real window — the
path `examples/button` demonstrates and the one that had never been run. `internal` is
already outside the layer stack, so `internal/integration` costs no change to the table
or the layering test; `docs/packages.md` records why it exists.

Unassigned deliberately: it cannot be written until `element` lands the listener seam and
`ui` migrates `Button` onto it. It goes to whoever has `ui` at that point, not to a
fourth party, because it is the same test they will already be writing one layer down.

Keep it to one test. The value is that the path executes at all.

## Decided: the content mask gets a real base rectangle

`ContentMask` currently encodes "no clipping" as an empty rectangle. A container that
resolves to zero height pushes a genuinely empty clip, the two meanings collide, and the
clip is ignored: standalone the children paint over the whole window, nested they inherit
the parent's mask and escape into their grandparent's bounds. `docs/audit-packages.md`
has both reproductions.

The obvious fix is a flag on `ContentMask` saying whether it masks. **Do not take it.**
Every shader infers the same thing from the same degenerate test:

    if (input.content_mask.z > input.content_mask.x && input.content_mask.w > input.content_mask.y)

so a flag means editing six `.hlsl` files and recompiling them, and `tools/compile_shaders`
is Windows-only and does not currently build at all. A correctness fix should not be
gated on a broken tool.

Instead, **give `Scene` its viewport and push it as the base mask**, so the clip stack is
never empty and every mask is a real rectangle. Empty then unambiguously means nothing is
visible, `place` already drops a primitive whose intersection with its mask is empty, and
`PushClip`'s "if either is empty return the other" special case goes away.

No shader changes. The existing guard stays as a safety net and stops being reachable,
because every mask that reaches the GPU is now non-degenerate.

Ordering: `scene` declares the viewport and `window` sets it, so `scene` lands first and
`window` follows. Two commits, each building.

While this is open, note the smaller mismatch it sits next to: `Bounds.Contains` is
half-open and the shader clip is inclusive on the maximum edges, so a clip cuts one pixel
further than the matching hit region ends. Fix it in whichever direction the readback
test can prove, and say which in the commit.

## Decided: colour keeps blending in sRGB, for now, and it gets written down

Everything blends in gamma-encoded sRGB on both sides. The swapchain is
`B8G8R8A8_UNORM` rather than the `_SRGB` variant, so the output-merger blends encoded
values, and `Mix`, `Blend`, `Lighten` and `Darken` do the same on the CPU.

Blending greyscale glyph coverage in linear space is the correct answer and the one that
makes small text look right. It is not this round, for three reasons that are about
sequencing rather than merit: it needs the shader toolchain, which does not build; it
needs every colour to reach the GPU linear, which is a `colour` and `render` change
together; and it shifts every tuned colour in `ui`, so it wants a UI worth re-tuning.

What happens now is that it stops being an accident. Record in `docs/architecture.md`
that blending is sRGB, that the choice is deliberate and provisional, and that text
anti-aliasing is what pays for it. An undocumented default and a documented one cost the
same to change and read very differently to whoever comes next.

Revisit it after `prompts/build.md` lands, alongside the atlas work.

---

## geometry

Free now. No dependencies, no package in flight, and small enough that it could ride
along with something else.

`IsContainedWithin` tests the bottom-right corner with the half-open `Contains`, so a
rectangle is not contained within itself and anything flush with its container's far edge
reports as escaping. It has no callers. Fix the far-corner comparison to be inclusive, or
delete the method; deleting is defensible and says more.

`DevicePixels.ToBytes` converts through `uint32` without checking the sign, so a negative
extent produces four billion bytes.

`Number` admits integer units, so `Point.Div`, `Size.Centre` and the halving inside
`CentredAt` and `FromAnchorAndSize` truncate. That is right for device pixels and belongs
in the constraint's doc comment, because the same call is exact on the float units.

## scene

Blocked on nothing but the decision above, which is taken. Can start as soon as someone
is free.

The base-mask change, then the half-open against inclusive edge mismatch.

Separately: `boundsTree.clear` resets the node slice but each internal node's children
slice is freshly allocated on the next frame's inserts, on a per-frame path.

## app

After `prompts/app.md` reports. Same files.

**`AsyncApp.Update` after `App.Close` blocks for ever.** `run` enqueues the closure and
then blocks on a result channel, and the shutdown check that returns "application has
been shut down" sits inside that closure, so it only runs if something drains a queue
nothing is draining any more. The error path the doc comment promises cannot be reached.
Check before the enqueue.

**`rc.shutdown` is read without its mutex** at `async.go:51` and `:83`, from background
goroutines, while `markShutdown` writes it under `r.mu`. A race by inspection, on the one
type documented as crossing await points. No test exercises concurrent shutdown, and the
deadlock above stops the reads happening at all in the obvious one, so write the test
that would have caught both.

**`NewApp` binds the UI goroutine to its caller and nothing says which caller that must
be.** Every `AsyncApp` operation reaches `app.update` and its `checkUI`, which only
succeeds if the foreground queue is drained on the constructing goroutine.
`examples/button` is correct by accident of ordering. Build the App anywhere else and
every background update panics from inside the platform dispatcher, naming neither
mistake. State the invariant on `NewApp`, and decide whether it can be checked rather
than only documented.

## input

After `prompts/input.md` reports.

`nodePath` allocates a `map[DispatchNodeID]bool` on every dispatch to guard against a
parent cycle that cannot occur, since `PushNode` assigns `id = len(nodes)` and takes its
parent from the stack, so a parent index is always strictly smaller. The map and the
growing path slice should both be scratch buffers on the tree. This is the pointer-move
path.

`DispatchText` delivers only to the focused node, with no capture or bubble phase, unlike
the other three dispatchers. Decide whether that is the contract and write it down, or
make it consistent. It is not a bug until someone writes a container that wants to see
text reaching a child, and then it is.

## style and element: the thirteen properties

After `prompts/element.md` reports. This is one landing across two packages and it needs
the decision below taken per property first, by me, not by whoever implements it.

Thirteen properties have builder methods and no consumer. They compile, they pass through
the refinement mask correctly, and they change nothing on screen:

    Visibility  Opacity  BoxShadow  TextBackgroundColour  Underline  Strikethrough
    WhiteSpace  TextOverflow  LineClamp  TextAlign  AllowConcurrentScroll
    RestrictScrollToAxis  ScrollbarWidth

`ui.Button.Disabled` is already broken by one of them: it communicates disabled state
through `Opacity(0.5)` and renders identically to an enabled button.

**The first move is deletion, not implementation.** Removing the setters that have no
consumer converts thirteen silent failures into thirteen compile errors that say what is
missing, and it is a fraction of the work. A builder method is a promise; the cheapest
way to stop breaking it is to stop making it.

Then implement, in this order, because they are not equally hard:

`Visibility`, `TextBackgroundColour`, `Underline` and `Strikethrough` are small and
local. The last three also give `scene.Underline` its first producer, which it has never
had.

`BoxShadow` gives `scene.Shadow` its first producer. The primitive, the shader and its
readback assertion all exist and nothing has ever emitted one.

`Opacity` is not small and should not be treated as though it is. CSS opacity is a group
property: the subtree composites as a unit and then fades. Multiplying each primitive's
alpha gives a visibly different and wrong result wherever children overlap, so a correct
implementation needs a layer or an offscreen target, which is the same machinery popups
need. Decide which semantic is being offered before writing it, and say so on the method.

`WhiteSpace`, `TextOverflow`, `LineClamp` and `TextAlign` all belong to multi-line text
and should wait for it rather than being faked on a single line.

`ScrollbarWidth` reaches layout already and has no scrollbar to size. It waits for one.

## element: half-leading

After `prompts/element.md`, and before the text field rather than after it.

The baseline goes at `bounds.Origin.Y + ascent`, so when the line box is taller than
ascent plus descent the whole difference lands below the glyphs. `DefaultTextStyle` ships
`FontSize: 16` with `LineHeight: 20`, so the default configuration already has leading it
puts on one side, and an editor's line height of 1.5 pins glyphs to the top of the box.

The rule is CSS's: half the difference above the ascent. `text` reports the metrics
correctly; the line box belongs to the element.

Do it with the text field, not after. Caret height and position come out of the same
arithmetic, so getting the leading wrong once means getting the caret wrong twice.

## window

After `prompts/window.md` reports.

`staticView` and `fnView` both return an empty `app.Subscription` from `Observe`, so
`SetRoot` and `SetRootFn` attach nothing and a window configured through either never
repaints in response to entity state. It looks reactive because every input event sets
`dirty` directly. Neither doc comment mentions it. Either wire them or say plainly that
they are for static content.

## platform

After `prompts/platform.md` reports, and larger than it.

**Nothing in the interface can minimise, maximise, restore or go fullscreen.** Not
missing from the Windows backend, missing from `platform.Window`. Any application with
`Decorated: false`, which is what a custom title bar requires, must draw its own window
controls and has nothing to call. This is an interface change and therefore a decision;
the shape to aim at is a window state that can be read and set, not four booleans.

**The shell is six stubs and four no-ops.** Tray, message dialog, open dialog, save
dialog and notifications return "not implemented"; application menu, hide, show and icon
are silent TODOs. The status table says "Windows", which reads as finished.

Take the file dialogs first. No open dialog means no "Open File…", which is the first
menu item in the application this framework exists to make possible. `ShowOpenDialog`
already carries the threading note about not blocking the platform thread, so the
contract is written; only the implementation is missing.

Tray and notifications can wait. Menus wait for the window-state work, since a custom
title bar changes what a menu bar even is.

## render: the atlas free list, with a number behind it

`prompts/render.md` is retired; this is the one item it could not close, because the fix
is not `render`'s alone. `render` cannot know when a tile stops being referenced — the
reference lives in `window.glyphTileCache` — so eviction needs a way for the holder to say
a tile is done, which is a `Renderer` interface change, and the policy belongs with
whoever owns that cache.

The measurement asked for came back: over 20,000 glyph-sized tiles, **2 pages at ~69%
average occupancy, 58% on the final page.**

That answers the question the prompt posed and the answer is the awkward one. A packer
that filled its pages would make this purely an eviction problem. At 58% it is two
problems: tiles are never freed, *and* the shelf packer leaves two fifths of a page on the
floor before opening another. Sizing an eviction policy against pages that are half empty
would tune the wrong number.

So take them together, after `window` owns its caching policy: a free list keyed by the
same tile handle, a way for the holder to release, and shelf packing measured again once
tiles can be reclaimed — occupancy under reuse is a different number from occupancy under
append-only growth, and it is the one the policy should be sized against.

The generation tag landed this round and is what makes any of this detectable: a stale
tile panics under `facet_debug` rather than sampling whatever overwrote it.

## render: the rest

After `scene`'s base-mask change lands.

Four of six primitives have no producer above `scene`. That is `element`'s to fix, not
yours, and it is above. What is yours is that the readback assertions for shadow, path
and underline are currently the only exercise those paths get, so they are the only thing
standing between an unused shader and a broken one. Keep them running.

The atlas free list waits on the caching policy decision, which waits on `window` owning
its glyph cache. Unchanged from your prompt.

## ui

After `element` lands both the property decision and the listener seam.

`Button.Disabled` needs a real visual affordance rather than an opacity that does
nothing. Whether that is a colour or a real opacity depends on the decision above.

---

## What this round is really about

Round one is defects that stop the framework working. This one is mostly a different
thing: promises the API makes and does not keep. Thirteen style setters, four primitives
with no producer, six platform methods that return an error, a status table that says
Windows.

None of them is a bug in the sense that something behaves incorrectly at runtime. All of
them are worse than a bug for anyone building on top, because a missing feature is a
compile error and a hollow one is an afternoon.
