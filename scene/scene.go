package scene

import (
	"slices"

	"github.com/yasufad/facet/geometry"
)

// Scene is the renderer's entire input: a flat, draw-ordered collection of
// primitives. Elements paint into it during the paint phase; the renderer
// consumes it during present.
//
// A Scene is built once per frame and discarded. Reuse Clear between frames
// rather than allocating a new Scene, so the per-type slices keep their
// capacity.
type Scene struct {
	quads             []Quad
	shadows           []Shadow
	paths             []Path[geometry.ScaledPixels]
	underlines        []Underline
	monochromeSprites []MonochromeSprite
	polychromeSprites []PolychromeSprite

	tree   boundsTree
	layers []DrawOrder
	clips  []ContentMask[geometry.ScaledPixels]
}

// New returns an empty Scene ready to paint into.
func New() *Scene {
	return &Scene{}
}

// Clear removes every primitive, layer and clip, keeping the slice capacity for
// the next frame.
func (s *Scene) Clear() {
	s.quads = s.quads[:0]
	s.shadows = s.shadows[:0]
	s.paths = s.paths[:0]
	s.underlines = s.underlines[:0]
	s.monochromeSprites = s.monochromeSprites[:0]
	s.polychromeSprites = s.polychromeSprites[:0]
	s.tree.clear()
	s.layers = s.layers[:0]
	s.clips = s.clips[:0]
}

// Len returns the total number of primitives in the scene.
func (s *Scene) Len() int {
	return len(s.quads) + len(s.shadows) + len(s.paths) +
		len(s.underlines) + len(s.monochromeSprites) + len(s.polychromeSprites)
}

// IsEmpty reports whether the scene contains no primitives.
func (s *Scene) IsEmpty() bool { return s.Len() == 0 }

// PushLayer opens a stacking context. The bounds are inserted into the spatial
// tree and the assigned draw order is pushed onto the layer stack; every
// primitive inserted until the matching PopLayer inherits that order and is
// drawn above anything beneath the layer's bounds.
func (s *Scene) PushLayer(bounds geometry.Bounds[geometry.ScaledPixels]) {
	order := s.tree.Insert(bounds)
	s.layers = append(s.layers, order)
}

// PopLayer closes the most recently opened stacking context.
func (s *Scene) PopLayer() {
	if len(s.layers) > 0 {
		s.layers = s.layers[:len(s.layers)-1]
	}
}

// PushClip pushes a content mask onto the clip stack, intersected with the mask
// already on top. Every primitive inserted until the matching PopClip is
// clipped to the intersection.
func (s *Scene) PushClip(mask ContentMask[geometry.ScaledPixels]) {
	if len(s.clips) > 0 {
		mask = mask.Intersect(s.clips[len(s.clips)-1])
	}
	s.clips = append(s.clips, mask)
}

// PopClip removes the most recently pushed content mask.
func (s *Scene) PopClip() {
	if len(s.clips) > 0 {
		s.clips = s.clips[:len(s.clips)-1]
	}
}

// currentClip returns the top of the clip stack, or a zero mask (no clipping)
// if the stack is empty.
func (s *Scene) currentClip() ContentMask[geometry.ScaledPixels] {
	if len(s.clips) == 0 {
		return ContentMask[geometry.ScaledPixels]{}
	}
	return s.clips[len(s.clips)-1]
}

// place computes the draw order and resolved content mask for a primitive with
// the given bounds. It intersects the bounds with the current clip, skips the
// primitive if nothing is visible, and assigns an order from the layer stack or
// the spatial tree. The bool is false when the primitive is fully clipped.
func (s *Scene) place(bounds geometry.Bounds[geometry.ScaledPixels]) (DrawOrder, ContentMask[geometry.ScaledPixels], bool) {
	mask := s.currentClip()
	if !mask.Bounds.IsEmpty() {
		bounds = bounds.Intersect(mask.Bounds)
	}
	if bounds.IsEmpty() {
		return 0, mask, false
	}
	var order DrawOrder
	if len(s.layers) > 0 {
		order = s.layers[len(s.layers)-1]
	} else {
		order = s.tree.Insert(bounds)
	}
	return order, mask, true
}

// InsertQuad adds a quad to the scene, clipped and ordered by the current
// clip and layer state. Quads with no visible area are skipped.
func (s *Scene) InsertQuad(q Quad) {
	order, mask, ok := s.place(q.Bounds)
	if !ok {
		return
	}
	q.Order = order
	q.ContentMask = mask
	s.quads = append(s.quads, q)
}

// InsertShadow adds a shadow to the scene.
func (s *Scene) InsertShadow(sh Shadow) {
	order, mask, ok := s.place(sh.Bounds)
	if !ok {
		return
	}
	sh.Order = order
	sh.ContentMask = mask
	s.shadows = append(s.shadows, sh)
}

// InsertPath adds a path to the scene. The path is assigned an ID from the
// current length of the paths slice before appending.
func (s *Scene) InsertPath(p Path[geometry.ScaledPixels]) {
	order, mask, ok := s.place(p.Bounds)
	if !ok {
		return
	}
	p.Order = order
	p.ContentMask = mask
	p.ID = PathID(len(s.paths))
	s.paths = append(s.paths, p)
}

// InsertUnderline adds an underline to the scene.
func (s *Scene) InsertUnderline(u Underline) {
	order, mask, ok := s.place(u.Bounds)
	if !ok {
		return
	}
	u.Order = order
	u.ContentMask = mask
	s.underlines = append(s.underlines, u)
}

// InsertMonochromeSprite adds a monochrome sprite (a glyph) to the scene.
func (s *Scene) InsertMonochromeSprite(sp MonochromeSprite) {
	order, mask, ok := s.place(sp.Bounds)
	if !ok {
		return
	}
	sp.Order = order
	sp.ContentMask = mask
	s.monochromeSprites = append(s.monochromeSprites, sp)
}

// InsertPolychromeSprite adds a polychrome sprite (an image or emoji) to the
// scene.
func (s *Scene) InsertPolychromeSprite(sp PolychromeSprite) {
	order, mask, ok := s.place(sp.Bounds)
	if !ok {
		return
	}
	sp.Order = order
	sp.ContentMask = mask
	s.polychromeSprites = append(s.polychromeSprites, sp)
}

// Finish sorts each per-type slice by draw order so the renderer can batch
// primitives into instanced calls. The sort is stable, so primitives within
// the same layer — which share an order — keep their insertion order. Sprites
// are secondarily sorted by tile ID for atlas locality.
//
// Finish must be called after all primitives are inserted and before Batches.
func (s *Scene) Finish() {
	slices.SortStableFunc(s.shadows, func(a, b Shadow) int { return cmpOrder(a.Order, b.Order) })
	slices.SortStableFunc(s.quads, func(a, b Quad) int { return cmpOrder(a.Order, b.Order) })
	slices.SortStableFunc(s.paths, func(a, b Path[geometry.ScaledPixels]) int {
		return cmpOrder(a.Order, b.Order)
	})
	slices.SortStableFunc(s.underlines, func(a, b Underline) int { return cmpOrder(a.Order, b.Order) })
	slices.SortStableFunc(s.monochromeSprites, func(a, b MonochromeSprite) int {
		if c := cmpOrder(a.Order, b.Order); c != 0 {
			return c
		}
		return cmpUint32(uint32(a.Tile.TileID), uint32(b.Tile.TileID))
	})
	slices.SortStableFunc(s.polychromeSprites, func(a, b PolychromeSprite) int {
		if c := cmpOrder(a.Order, b.Order); c != 0 {
			return c
		}
		return cmpUint32(uint32(a.Tile.TileID), uint32(b.Tile.TileID))
	})
}

func cmpOrder(a, b DrawOrder) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

func cmpUint32(a, b uint32) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// Quads returns the sorted quad slice, for direct access by the renderer.
func (s *Scene) Quads() []Quad { return s.quads }

// Shadows returns the sorted shadow slice.
func (s *Scene) Shadows() []Shadow { return s.shadows }

// Paths returns the sorted path slice.
func (s *Scene) Paths() []Path[geometry.ScaledPixels] { return s.paths }

// Underlines returns the sorted underline slice.
func (s *Scene) Underlines() []Underline { return s.underlines }

// MonochromeSprites returns the sorted monochrome sprite slice.
func (s *Scene) MonochromeSprites() []MonochromeSprite { return s.monochromeSprites }

// PolychromeSprites returns the sorted polychrome sprite slice.
func (s *Scene) PolychromeSprites() []PolychromeSprite { return s.polychromeSprites }
