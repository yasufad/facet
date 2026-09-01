# render: every batch allocates, and every batch renames the buffer

Nothing above you changes and nothing below you moves, so this runs in parallel with the
rest. All of it is inside `render/d3d11` and none of it touches the `Renderer` interface.

`docs/audit.md` has the measurements. The work is below.

## 1. Stop allocating per batch

Every draw path starts the same way:

```go
data := make([]quadInstance, len(quads))
```

then fills it, then copies it into the mapped GPU buffer. Two writes per primitive and a
fresh slice per batch per frame. A text-heavy frame with five thousand glyphs is roughly
half a megabyte of garbage from the renderer alone, arriving as pauses inside frames
rather than as a slower average.

Keep a scratch buffer per instance kind on the renderer, sized once and reused, or write
the instances straight into the mapped region and skip the intermediate entirely. The
second is better and is the shape the third item wants anyway.

## 2. One discard per frame, not one per batch

`dynamicBuffer.write` maps with `D3D11_MAP_WRITE_DISCARD` from offset zero on every call,
so the driver renames the buffer once per batch. That is the pattern for a buffer written
once a frame, not for one written six or sixty times.

The standard shape: `DISCARD` on the first map of a frame, then `NO_OVERWRITE` with a
rolling offset for every batch after it, resetting the offset when the frame ends or the
buffer wraps. Each batch's draw call then takes a start-instance offset rather than
assuming zero.

**This is the change most likely to draw the wrong thing while erroring nowhere.** An
offset that is right for the first batch and wrong for the second draws the previous
batch's instances at the current batch's shader, which is a legal operation producing
plausible pixels. This package has already had three wrong vtable indices reach green,
reviewed code, and the readback path is what found them.

So: extend the `facet_debug` readback assertions to cover a frame with at least two
batches of the same kind and one of a different kind interleaved between them, and check
pixels from both. A test that draws one batch cannot fail on this bug.

## 3. ClearAtlas invalidates every outstanding tile, silently

`atlasManager.clear` resets each page's packer cursor and keeps the textures. The next
upload lands at the same coordinates as a tile someone is still holding, and that holder
now samples whatever overwrote it.

`window` gets away with it today because `ScaleChangedEvent` clears its glyph tile cache
in the same handler. That is a coincidence of one caller, not a property of the
interface, and `ClearAtlas` is exported.

Say it in the doc comment: every `scene.AtlasTile` handed out before a `ClearAtlas` of
that kind is invalid afterwards, and the caller owns dropping its references. Then make
it detectable rather than silent — a generation counter on the atlas, stamped into the
tile, checked on use under `facet_debug`. A stale tile should panic where it is used, not
render the wrong glyph.

## 4. Pages only ever grow

`shelfPacker.allocate` walks a cursor forward and has no free list. When a page fills,
`upload` creates another and the old one is never reclaimed. A window left open across a
day, with theme changes, font size changes and several scripts, accumulates 4 MB mono
pages and 16 MB polychrome ones until the process ends.

Not this round, and the reason is that the fix is not yours alone. `render` cannot know
when a tile stops being referenced, because the reference lives in `window.glyphTileCache`.
Evicting needs a way for the holder to say a tile is done, which is a `Renderer` interface
change, and the eviction policy belongs with whoever owns the cache. I will decide that
alongside the caching work in `window`.

What you can do now is measure it so the decision has a number: report how many pages a
realistic session allocates, and how full the last one is when a new one is created. A
shelf packer that starts a new page at 60% occupancy is a different problem from one that
fills them.

## Done when

    go test ./render/...
    GOOS=windows go build ./...

pass, and on a Windows machine:

    go test -tags facet_debug ./render/d3d11

passes with the multi-batch readback assertions, each of which fails if its own draw path
is disabled and no others. That property is what makes these tests worth having and it is
easy to lose when the buffer becomes shared.

Report the before and after allocation counts per frame, and the page occupancy numbers
from item 4.
