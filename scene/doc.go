// Package scene is the renderer's input language. During the paint phase,
// elements emit primitives into a Scene; during present, the renderer consumes
// that Scene and draws it onto a GPU surface.
//
// # Primitives
//
// Six primitives, and no more:
//
//   - Quad             a filled, optionally rounded rectangle with a per-edge border
//   - Shadow           a blurred rounded rectangle, with offset and spread baked into its bounds
//   - MonochromeSprite a glyph: an atlas tile tinted with a colour
//   - PolychromeSprite an image or emoji: an atlas tile drawn at full colour
//   - Path             a filled bezier, for what the others cannot express
//   - Underline        a straight or wavy line beneath text
//
// Adding a seventh primitive touches every renderer backend, so it is a
// decision to raise rather than a change to make. If something cannot be drawn
// with the six, say so.
//
// # Draw order
//
// Primitives arrive in tree order but must be drawn in layer order. Each
// primitive carries a DrawOrder: lower values are drawn first, higher values on
// top. The Scene assigns orders from a spatial tree so that overlapping
// primitives receive strictly increasing orders, while primitives that share no
// screen space may reuse an order and be batched together.
//
// A caller opens a stacking context with PushLayer and closes it with PopLayer.
// Every primitive inserted between the two inherits the layer's order, so a
// whole subtree draws above anything beneath the layer's bounds. Primitives
// inserted outside any layer are ordered by their own bounds through the
// spatial tree.
//
// Finish sorts each per-type slice by draw order, stably, so primitives within
// the same layer — which share an order — keep their insertion order. The
// renderer then reads the sorted slices through Batches, which groups
// consecutive primitives of one kind (and, for sprites, one texture) into a
// single instanced draw call.
//
// # Clipping
//
// The Scene maintains a clip stack. PushClip intersects a content mask with the
// one already on top; PopClip removes it. Every inserted primitive records the
// stack's current mask in its ContentMask field. A primitive whose bounds are
// fully clipped away is skipped and never reaches a per-type slice.
//
// A ContentMask with empty bounds means "no clipping". The Scene never inserts
// a primitive carrying a mask that clips it to nothing — the intersection is
// empty and the primitive is dropped first.
//
// # Colour
//
// Primitives carry straight (non-premultiplied) alpha, as stored by the colour
// package. The renderer premultiplies when uploading instance data; the Scene
// never does.
//
// # Atlas tiles
//
// Sprites reference atlas tiles by identifier and rectangle. The Scene does not
// own an atlas, allocate from one, or know how glyphs get into it — that is the
// job of the text and render packages. It only records the reference so the
// renderer can look the tile up.
//
// # Invariants
//
//   - Plain data. No behaviour beyond construction, ordering and batching. A
//     primitive does not know what produced it.
//   - Imports only geometry and colour, enforced by the layering test.
//   - All geometric fields are in ScaledPixels, the intermediate unit between
//     layout (Pixels) and the renderer (DevicePixels). The caller converts
//     before inserting.
//   - Finish must be called after all insertions and before Batches.
package scene
