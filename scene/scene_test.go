package scene

import (
	"slices"
	"testing"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
)

func quadAt(x, y, w, h float32) Quad {
	return Quad{Bounds: spBounds(x, y, w, h)}
}

func shadowAt(x, y, w, h float32) Shadow {
	return Shadow{Bounds: spBounds(x, y, w, h)}
}

func underlineAt(x, y, w, h float32) Underline {
	return Underline{Bounds: spBounds(x, y, w, h)}
}

func monoSpriteAt(x, y, w, h float32, tileID TileID) MonochromeSprite {
	return MonochromeSprite{
		Bounds: spBounds(x, y, w, h),
		Tile:   AtlasTile{TileID: tileID},
	}
}

func polySpriteAt(x, y, w, h float32, texIdx uint32, tileID TileID) PolychromeSprite {
	return PolychromeSprite{
		Bounds: spBounds(x, y, w, h),
		Tile: AtlasTile{
			TextureID: AtlasTextureID{Index: texIdx},
			TileID:    tileID,
		},
	}
}

// TestSpatialOrdering verifies that overlapping primitives receive strictly
// increasing draw orders, while non-overlapping primitives may reuse an order.
func TestSpatialOrdering(t *testing.T) {
	s := New()
	defer s.Clear()

	// Two overlapping quads: the second must get a higher order.
	s.InsertQuad(quadAt(0, 0, 10, 10))
	s.InsertQuad(quadAt(5, 5, 10, 10))
	if got := s.quads[0].Order; got != 1 {
		t.Fatalf("first quad order: got %d, want 1", got)
	}
	if got := s.quads[1].Order; got != 2 {
		t.Fatalf("second quad order: got %d, want 2", got)
	}

	// A non-overlapping quad reuses order 1.
	s.InsertQuad(quadAt(100, 100, 10, 10))
	if got := s.quads[2].Order; got != 1 {
		t.Fatalf("non-overlapping quad order: got %d, want 1", got)
	}
}

// TestStackingContext verifies that primitives inside a layer share the
// layer's order, and that a second layer overlapping the first gets a higher
// order.
func TestStackingContext(t *testing.T) {
	s := New()
	defer s.Clear()

	// Layer 1 at (0,0,50,50).
	s.PushLayer(spBounds(0, 0, 50, 50))
	s.InsertQuad(quadAt(0, 0, 10, 10))
	s.InsertQuad(quadAt(10, 10, 10, 10))
	s.InsertShadow(shadowAt(5, 5, 20, 20))
	s.PopLayer()

	// All primitives inside the layer share its order.
	layerOrder := s.quads[0].Order
	if layerOrder != 1 {
		t.Fatalf("layer order: got %d, want 1", layerOrder)
	}
	if s.quads[1].Order != layerOrder {
		t.Fatalf("second quad in layer: got order %d, want %d", s.quads[1].Order, layerOrder)
	}
	if s.shadows[0].Order != layerOrder {
		t.Fatalf("shadow in layer: got order %d, want %d", s.shadows[0].Order, layerOrder)
	}

	// A second layer overlapping the first gets a higher order.
	s.PushLayer(spBounds(0, 0, 50, 50))
	s.InsertQuad(quadAt(0, 0, 10, 10))
	s.PopLayer()
	if got := s.quads[2].Order; got != 2 {
		t.Fatalf("overlapping layer order: got %d, want 2", got)
	}
}

// TestNestedStackingContexts verifies that layers nest: an inner layer inside
// an outer layer gets the outer layer's order plus the spatial contribution.
func TestNestedStackingContexts(t *testing.T) {
	s := New()
	defer s.Clear()

	// Outer layer.
	s.PushLayer(spBounds(0, 0, 100, 100))
	s.InsertQuad(quadAt(0, 0, 10, 10)) // order = outer layer order
	outerOrder := s.quads[0].Order

	// Inner layer overlapping the quad inside the outer layer.
	s.PushLayer(spBounds(0, 0, 50, 50))
	s.InsertQuad(quadAt(0, 0, 10, 10))
	s.PopLayer()

	s.PopLayer()

	innerOrder := s.quads[1].Order
	if innerOrder <= outerOrder {
		t.Fatalf("inner layer order %d should exceed outer order %d",
			innerOrder, outerOrder)
	}
}

// TestStableSortWithinLayer verifies that primitives sharing a draw order
// (because they are in the same layer) preserve their insertion order after
// Finish.
func TestStableSortWithinLayer(t *testing.T) {
	s := New()
	defer s.Clear()

	s.PushLayer(spBounds(0, 0, 100, 100))
	// Insert quads with distinguishable backgrounds in insertion order.
	for i := range 5 {
		q := quadAt(float32(i*10), 0, 10, 10)
		q.Background = colour.Rgba{R: float32(i) / 10}
		s.InsertQuad(q)
	}
	s.PopLayer()

	// All share the layer's order.
	for i, q := range s.quads {
		if q.Order != s.quads[0].Order {
			t.Fatalf("quad %d order %d differs from first %d", i, q.Order, s.quads[0].Order)
		}
	}

	s.Finish()

	// Insertion order preserved after stable sort.
	for i, q := range s.quads {
		want := float32(i) / 10
		if q.Background.R != want {
			t.Fatalf("quad %d after sort: R %v, want %v (insertion order not preserved)",
				i, q.Background.R, want)
		}
	}
}

// TestClipStack verifies that the clip stack intersects and that primitives
// outside the clip are skipped.
func TestClipStack(t *testing.T) {
	s := New()
	defer s.Clear()

	// Push a clip covering (10,10,50,50).
	s.PushClip(ContentMask[geometry.ScaledPixels]{Bounds: spBounds(10, 10, 50, 50)})

	// A quad inside the clip is kept.
	s.InsertQuad(quadAt(20, 20, 10, 10))
	if len(s.quads) != 1 {
		t.Fatalf("quad inside clip: got %d quads, want 1", len(s.quads))
	}
	if got := s.quads[0].ContentMask.Bounds; !boundsEq(got, spBounds(10, 10, 50, 50)) {
		t.Fatalf("quad content mask: got %v, want clip bounds", got)
	}

	// A quad outside the clip is skipped.
	s.InsertQuad(quadAt(100, 100, 10, 10))
	if len(s.quads) != 1 {
		t.Fatalf("quad outside clip: got %d quads, want 1 (skipped)", len(s.quads))
	}

	// A quad partially inside the clip is kept, with clipped bounds recorded
	// in the tree (its order reflects the clipped bounds).
	s.InsertQuad(quadAt(40, 40, 30, 30)) // intersects clip at (40,40,20,20)
	if len(s.quads) != 2 {
		t.Fatalf("partially clipped quad: got %d quads, want 2", len(s.quads))
	}

	s.PopClip()

	// After popping, a quad anywhere is kept.
	s.InsertQuad(quadAt(200, 200, 10, 10))
	if len(s.quads) != 3 {
		t.Fatalf("after pop clip: got %d quads, want 3", len(s.quads))
	}
}

// TestNestedClipIntersection verifies that nested clips intersect.
func TestNestedClipIntersection(t *testing.T) {
	s := New()
	defer s.Clear()

	s.PushClip(ContentMask[geometry.ScaledPixels]{Bounds: spBounds(0, 0, 100, 100)})
	s.PushClip(ContentMask[geometry.ScaledPixels]{Bounds: spBounds(50, 50, 100, 100)})

	// Effective clip is (50,50,50,50).
	s.InsertQuad(quadAt(20, 20, 10, 10)) // outside effective clip → skipped
	if len(s.quads) != 0 {
		t.Fatalf("quad outside nested clip: got %d, want 0", len(s.quads))
	}

	s.InsertQuad(quadAt(60, 60, 10, 10)) // inside effective clip → kept
	if len(s.quads) != 1 {
		t.Fatalf("quad inside nested clip: got %d, want 1", len(s.quads))
	}

	s.PopClip()
	s.PopClip()
}

// TestBatchOrder verifies that the batch iterator yields batches in ascending
// draw order.
func TestBatchOrder(t *testing.T) {
	s := New()
	defer s.Clear()

	// Insert a shadow and a quad that overlap, so the quad gets a higher order.
	s.InsertShadow(shadowAt(0, 0, 10, 10)) // order 1
	s.InsertQuad(quadAt(5, 5, 10, 10))     // order 2
	s.Finish()

	batches := collectBatches(s)
	if len(batches) != 2 {
		t.Fatalf("got %d batches, want 2", len(batches))
	}
	if batches[0].Kind != BatchShadows {
		t.Fatalf("first batch kind: got %d, want BatchShadows", batches[0].Kind)
	}
	if batches[1].Kind != BatchQuads {
		t.Fatalf("second batch kind: got %d, want BatchQuads", batches[1].Kind)
	}
}

// TestBatchKindTiebreak verifies that when two primitive types share a draw
// order, the kind tiebreaker gives a deterministic order (shadows before
// quads).
func TestBatchKindTiebreak(t *testing.T) {
	s := New()
	defer s.Clear()

	// Insert a shadow and a quad at the same location, both outside any layer.
	// They overlap, so the quad gets order 2. Use a layer to force same order.
	s.PushLayer(spBounds(0, 0, 100, 100))
	s.InsertShadow(shadowAt(0, 0, 10, 10)) // same layer order
	s.InsertQuad(quadAt(0, 0, 10, 10))     // same layer order
	s.PopLayer()
	s.Finish()

	batches := collectBatches(s)
	if len(batches) != 2 {
		t.Fatalf("got %d batches, want 2", len(batches))
	}
	// Shadow kind (0) < Quad kind (1), so shadow batch comes first.
	if batches[0].Kind != BatchShadows {
		t.Fatalf("first batch: got kind %d, want BatchShadows", batches[0].Kind)
	}
	if batches[1].Kind != BatchQuads {
		t.Fatalf("second batch: got kind %d, want BatchQuads", batches[1].Kind)
	}
}

// TestBatchMerging verifies that consecutive primitives of the same kind are
// merged into one batch.
func TestBatchMerging(t *testing.T) {
	s := New()
	defer s.Clear()

	// Three non-overlapping quads, all order 1.
	s.InsertQuad(quadAt(0, 0, 10, 10))
	s.InsertQuad(quadAt(100, 0, 10, 10))
	s.InsertQuad(quadAt(200, 0, 10, 10))
	s.Finish()

	batches := collectBatches(s)
	if len(batches) != 1 {
		t.Fatalf("got %d batches, want 1 (all quads merged)", len(batches))
	}
	if batches[0].Kind != BatchQuads {
		t.Fatalf("batch kind: got %d, want BatchQuads", batches[0].Kind)
	}
	if batches[0].Range.Start != 0 || batches[0].Range.End != 3 {
		t.Fatalf("batch range: got [%d,%d), want [0,3)", batches[0].Range.Start, batches[0].Range.End)
	}
}

// TestBatchSplitByOrder verifies that a batch of one kind is split when another
// kind with a lower order appears between its primitives in draw order.
func TestBatchSplitByOrder(t *testing.T) {
	s := New()
	defer s.Clear()

	// Quad at order 1, shadow overlapping it at order 2, quad overlapping
	// shadow at order 3. The quads cannot merge because the shadow is drawn
	// between them.
	s.InsertQuad(quadAt(0, 0, 10, 10))     // order 1
	s.InsertShadow(shadowAt(5, 5, 10, 10)) // order 2
	s.InsertQuad(quadAt(10, 10, 10, 10))   // order 3
	s.Finish()

	batches := collectBatches(s)
	if len(batches) != 3 {
		t.Fatalf("got %d batches, want 3 (split by shadow)", len(batches))
	}
	if batches[0].Kind != BatchQuads || batches[0].Range.End != 1 {
		t.Fatalf("batch 0: got kind %d range [%d,%d), want BatchQuads [0,1)",
			batches[0].Kind, batches[0].Range.Start, batches[0].Range.End)
	}
	if batches[1].Kind != BatchShadows {
		t.Fatalf("batch 1: got kind %d, want BatchShadows", batches[1].Kind)
	}
	if batches[2].Kind != BatchQuads || batches[2].Range.Start != 1 {
		t.Fatalf("batch 2: got kind %d range [%d,%d), want BatchQuads [1,2)",
			batches[2].Kind, batches[2].Range.Start, batches[2].Range.End)
	}
}

// TestPolychromeSpriteBatchSplitByTexture verifies that polychrome sprite
// batches are split by texture ID.
func TestPolychromeSpriteBatchSplitByTexture(t *testing.T) {
	s := New()
	defer s.Clear()

	// Three non-overlapping sprites on two textures, all order 1.
	s.InsertPolychromeSprite(polySpriteAt(0, 0, 10, 10, 0, 1))   // texture 0
	s.InsertPolychromeSprite(polySpriteAt(100, 0, 10, 10, 0, 2)) // texture 0
	s.InsertPolychromeSprite(polySpriteAt(200, 0, 10, 10, 1, 3)) // texture 1
	s.Finish()

	batches := collectBatches(s)
	if len(batches) != 2 {
		t.Fatalf("got %d batches, want 2 (split by texture)", len(batches))
	}
	if batches[0].TextureID.Index != 0 || batches[0].Range.End != 2 {
		t.Fatalf("batch 0: texture %d range [%d,%d), want texture 0 [0,2)",
			batches[0].TextureID.Index, batches[0].Range.Start, batches[0].Range.End)
	}
	if batches[1].TextureID.Index != 1 || batches[1].Range.Start != 2 {
		t.Fatalf("batch 1: texture %d range [%d,%d), want texture 1 [2,3)",
			batches[1].TextureID.Index, batches[1].Range.Start, batches[1].Range.End)
	}
}

// TestMonochromeSpriteSortByTileID verifies that monochrome sprites are sorted
// by (order, tile_id) within the per-type slice.
func TestMonochromeSpriteSortByTileID(t *testing.T) {
	s := New()
	defer s.Clear()

	// Insert sprites with tile IDs out of order, all non-overlapping (order 1).
	s.InsertMonochromeSprite(monoSpriteAt(0, 0, 10, 10, 3))
	s.InsertMonochromeSprite(monoSpriteAt(100, 0, 10, 10, 1))
	s.InsertMonochromeSprite(monoSpriteAt(200, 0, 10, 10, 2))
	s.Finish()

	got := make([]TileID, len(s.monochromeSprites))
	for i, sp := range s.monochromeSprites {
		got[i] = sp.Tile.TileID
	}
	want := []TileID{1, 2, 3}
	if !slices.Equal(got, want) {
		t.Fatalf("tile IDs after sort: got %v, want %v", got, want)
	}
}

// TestFullyClippedSkipped verifies that a primitive fully outside the clip
// stack is never inserted.
func TestFullyClippedSkipped(t *testing.T) {
	s := New()
	defer s.Clear()

	s.PushClip(ContentMask[geometry.ScaledPixels]{Bounds: spBounds(0, 0, 10, 10)})
	s.InsertQuad(quadAt(100, 100, 10, 10))
	s.PopClip()

	if s.Len() != 0 {
		t.Fatalf("fully clipped quad: got len %d, want 0", s.Len())
	}
}

// TestPathInsertion verifies that a path is inserted with an ID and order.
func TestPathInsertion(t *testing.T) {
	s := New()
	defer s.Clear()

	p := NewPath(geometry.Point[geometry.Pixels]{X: 0, Y: 0})
	p.LineTo(geometry.Point[geometry.Pixels]{X: 10, Y: 0})
	p.LineTo(geometry.Point[geometry.Pixels]{X: 10, Y: 10})
	p.Colour = colour.Rgba{R: 1, A: 1}

	s.InsertPath(ScalePath(p, 1.0))

	if len(s.paths) != 1 {
		t.Fatalf("got %d paths, want 1", len(s.paths))
	}
	if s.paths[0].ID != 0 {
		t.Fatalf("path ID: got %d, want 0", s.paths[0].ID)
	}
	if s.paths[0].Order != 1 {
		t.Fatalf("path order: got %d, want 1", s.paths[0].Order)
	}
	if len(s.paths[0].Vertices) != 3 {
		t.Fatalf("path vertices: got %d, want 3 (one triangle)", len(s.paths[0].Vertices))
	}
}

// TestClear verifies that Clear empties the scene but keeps it usable.
func TestClear(t *testing.T) {
	s := New()
	s.InsertQuad(quadAt(0, 0, 10, 10))
	s.PushLayer(spBounds(0, 0, 50, 50))
	s.PushClip(ContentMask[geometry.ScaledPixels]{Bounds: spBounds(0, 0, 50, 50)})
	s.Clear()

	if !s.IsEmpty() {
		t.Fatalf("after Clear: not empty")
	}
	if len(s.layers) != 0 || len(s.clips) != 0 {
		t.Fatalf("after Clear: layers %d clips %d, want 0 0", len(s.layers), len(s.clips))
	}

	// Reusable after Clear.
	s.InsertQuad(quadAt(0, 0, 10, 10))
	if s.Len() != 1 {
		t.Fatalf("after Clear and re-insert: len %d, want 1", s.Len())
	}
}

// --- helpers ---

func collectBatches(s *Scene) []Batch {
	var batches []Batch
	for b := range s.Batches() {
		batches = append(batches, b)
	}
	return batches
}

func boundsEq(a, b geometry.Bounds[geometry.ScaledPixels]) bool {
	return a.Origin == b.Origin && a.Size == b.Size
}
