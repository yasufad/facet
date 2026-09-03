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

## input

After `prompts/input.md` reports.

`nodePath` allocates a `map[DispatchNodeID]bool` on every dispatch to guard against a
parent cycle that cannot occur, since `PushNode` assigns `id = len(nodes)` and takes its
parent from the stack, so a parent index is always strictly smaller. The map and the
growing path slice should both be scratch buffers on the tree. This is the pointer-move
path.


## text and element: stop handing out mutable glyphs

`text`'s line cache clones every `[]ShapedLine` it returns, on both the hit and the miss
path, because `ShapedRun.Glyphs` is an exported mutable slice and a caller who touches one
glyph would otherwise corrupt the entry every later caller reads. That was a real bug — I
reproduced it through the exported API alone, and it passed at the commit before the cache
landed.

The clone is the whole remaining cost of a cache hit: 2,258 ns and 1,552 B, three
allocations, all of it copying glyphs nobody intends to write to. Optimising around it
inside `text` is not possible, because the hazard is in the type rather than the cache.

The fix is to stop offering the mutation: unexport `Glyphs` behind an accessor that
returns a read-only view, so a cached line can be handed out directly. That crosses into
`element`, which walks glyphs every frame to emit sprites and is the only real consumer,
so it is one landing across two packages and a `text` API change.

Not now, and the reason is sequencing rather than doubt: `ui` is building a text field
against `element.TextLayout` right now, and `element` is the package that would have to
absorb the accessor change. Take it once the text field has landed and we know what a
caret and a selection actually need from a shaped line — changing the glyph API twice
would be worse than paying the clone for another round.

`BenchmarkShapeLineLineCacheHit` is what proves it when it happens.



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
