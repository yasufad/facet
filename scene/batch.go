package scene

import "iter"

// BatchKind identifies which per-type slice a Batch draws from.
type BatchKind uint8

const (
	BatchShadows BatchKind = iota
	BatchQuads
	BatchPaths
	BatchUnderlines
	BatchMonochromeSprites
	BatchPolychromeSprites
)

// Range is a half-open [Start, End) index range into one of the Scene's
// per-type slices.
type Range struct {
	Start int
	End   int
}

// Batch is a contiguous run of primitives of one kind, in draw order, that the
// renderer can draw with a single instanced call. For sprite batches, TextureID
// identifies the atlas texture all sprites in the batch share.
type Batch struct {
	Kind      BatchKind
	Range     Range
	TextureID AtlasTextureID // valid only for sprite batches
}

// batchKind orders kinds as a tiebreaker when two types share a draw order, so
// the batch order is deterministic. The order matches GPUI's PrimitiveKind
// discriminant order.
func (k BatchKind) order() int { return int(k) }

// Batches returns a pull iterator over the scene's primitives grouped into
// batches, in draw order. Consecutive primitives of the same kind — and, for
// sprites, the same texture — are merged into one batch. The iterator must be
// consumed after Finish has sorted the per-type slices.
func (s *Scene) Batches() iter.Seq[Batch] {
	return func(yield func(Batch) bool) {
		it := batchCursor{scene: s}
		for {
			b, ok := it.next()
			if !ok {
				return
			}
			if !yield(b) {
				return
			}
		}
	}
}

// batchCursor tracks the current position in each per-type slice and emits
// batches by repeatedly picking the type with the lowest draw order.
type batchCursor struct {
	scene      *Scene
	shadows    int
	quads      int
	paths      int
	underlines int
	mono       int
	poly       int
}

// peekOrder returns the draw order of the next unconsumed primitive of the
// given kind, or the maximum uint32 when that kind is exhausted. The kind's
// discriminant breaks ties so that two types sharing an order are drawn in a
// fixed order.
func (c *batchCursor) peekOrder(kind BatchKind) (DrawOrder, bool) {
	switch kind {
	case BatchShadows:
		if c.shadows < len(c.scene.shadows) {
			return c.scene.shadows[c.shadows].Order, true
		}
	case BatchQuads:
		if c.quads < len(c.scene.quads) {
			return c.scene.quads[c.quads].Order, true
		}
	case BatchPaths:
		if c.paths < len(c.scene.paths) {
			return c.scene.paths[c.paths].Order, true
		}
	case BatchUnderlines:
		if c.underlines < len(c.scene.underlines) {
			return c.scene.underlines[c.underlines].Order, true
		}
	case BatchMonochromeSprites:
		if c.mono < len(c.scene.monochromeSprites) {
			return c.scene.monochromeSprites[c.mono].Order, true
		}
	case BatchPolychromeSprites:
		if c.poly < len(c.scene.polychromeSprites) {
			return c.scene.polychromeSprites[c.poly].Order, true
		}
	}
	return 0, false
}

// next returns the next batch, or false if all primitives have been consumed.
func (c *batchCursor) next() (Batch, bool) {
	const maxOrder = ^DrawOrder(0)

	// Collect the (order, kind) of each non-exhausted type.
	var candidates [6]struct {
		order DrawOrder
		kind  BatchKind
	}
	n := 0
	for k := BatchShadows; k <= BatchPolychromeSprites; k++ {
		if order, ok := c.peekOrder(k); ok {
			candidates[n] = struct {
				order DrawOrder
				kind  BatchKind
			}{order, k}
			n++
		}
	}
	if n == 0 {
		return Batch{}, false
	}

	// Find the two lowest (order, kind) pairs. The second lowest is the upper
	// bound: the batch consumes primitives of the lowest kind while their
	// (order, kind) is below it.
	slice := candidates[:n]
	// Sort by (order, kind). n is at most 6, so a simple insertion sort is
	// fine and avoids allocating.
	for i := 1; i < n; i++ {
		for j := i; j > 0; j-- {
			a, b := slice[j-1], slice[j]
			if a.order > b.order || (a.order == b.order && a.kind.order() > b.kind.order()) {
				slice[j-1], slice[j] = slice[j], slice[j-1]
			} else {
				break
			}
		}
	}

	batchKind := slice[0].kind
	var upperBound struct {
		order DrawOrder
		kind  BatchKind
	}
	if n > 1 {
		upperBound.order = slice[1].order
		upperBound.kind = slice[1].kind
	} else {
		upperBound.order = maxOrder
		upperBound.kind = BatchKind(255)
	}

	return c.consume(batchKind, upperBound.order, upperBound.kind)
}

// below reports whether (order, kind) sorts before the upper bound.
func below(order DrawOrder, kind BatchKind, ubOrder DrawOrder, ubKind BatchKind) bool {
	if order != ubOrder {
		return order < ubOrder
	}
	return kind.order() < ubKind.order()
}

// consume emits one batch of the given kind, advancing the cursor past every
// consecutive primitive of that kind whose (order, kind) is below the upper
// bound. Sprite batches are additionally split by texture ID.
func (c *batchCursor) consume(kind BatchKind, ubOrder DrawOrder, ubKind BatchKind) (Batch, bool) {
	switch kind {
	case BatchShadows:
		start := c.shadows
		c.shadows++
		for c.shadows < len(c.scene.shadows) &&
			below(c.scene.shadows[c.shadows].Order, kind, ubOrder, ubKind) {
			c.shadows++
		}
		return Batch{Kind: kind, Range: Range{Start: start, End: c.shadows}}, true
	case BatchQuads:
		start := c.quads
		c.quads++
		for c.quads < len(c.scene.quads) &&
			below(c.scene.quads[c.quads].Order, kind, ubOrder, ubKind) {
			c.quads++
		}
		return Batch{Kind: kind, Range: Range{Start: start, End: c.quads}}, true
	case BatchPaths:
		start := c.paths
		c.paths++
		for c.paths < len(c.scene.paths) &&
			below(c.scene.paths[c.paths].Order, kind, ubOrder, ubKind) {
			c.paths++
		}
		return Batch{Kind: kind, Range: Range{Start: start, End: c.paths}}, true
	case BatchUnderlines:
		start := c.underlines
		c.underlines++
		for c.underlines < len(c.scene.underlines) &&
			below(c.scene.underlines[c.underlines].Order, kind, ubOrder, ubKind) {
			c.underlines++
		}
		return Batch{Kind: kind, Range: Range{Start: start, End: c.underlines}}, true
	case BatchMonochromeSprites:
		textureID := c.scene.monochromeSprites[c.mono].Tile.TextureID
		start := c.mono
		c.mono++
		for c.mono < len(c.scene.monochromeSprites) &&
			below(c.scene.monochromeSprites[c.mono].Order, kind, ubOrder, ubKind) &&
			c.scene.monochromeSprites[c.mono].Tile.TextureID == textureID {
			c.mono++
		}
		return Batch{Kind: kind, Range: Range{Start: start, End: c.mono}, TextureID: textureID}, true
	case BatchPolychromeSprites:
		textureID := c.scene.polychromeSprites[c.poly].Tile.TextureID
		start := c.poly
		c.poly++
		for c.poly < len(c.scene.polychromeSprites) &&
			below(c.scene.polychromeSprites[c.poly].Order, kind, ubOrder, ubKind) &&
			c.scene.polychromeSprites[c.poly].Tile.TextureID == textureID {
			c.poly++
		}
		return Batch{Kind: kind, Range: Range{Start: start, End: c.poly}, TextureID: textureID}, true
	}
	return Batch{}, false
}
