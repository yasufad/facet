package scene

import (
	"math/rand"
	"testing"

	"github.com/yasufad/facet/geometry"
)

func spBounds(x, y, w, h float32) geometry.Bounds[geometry.ScaledPixels] {
	return geometry.Bounds[geometry.ScaledPixels]{
		Origin: geometry.Point[geometry.ScaledPixels]{
			X: geometry.ScaledPixels(x),
			Y: geometry.ScaledPixels(y),
		},
		Size: geometry.Size[geometry.ScaledPixels]{
			Width:  geometry.ScaledPixels(w),
			Height: geometry.ScaledPixels(h),
		},
	}
}

// TestBoundsTreeInsert ports GPUI's test_insert: overlapping bounds receive
// increasing orders, non-overlapping bounds may reuse an order.
func TestBoundsTreeInsert(t *testing.T) {
	var tree boundsTree

	b1 := spBounds(0, 0, 10, 10)
	b2 := spBounds(5, 5, 10, 10)
	b3 := spBounds(10, 10, 10, 10)

	if order := tree.Insert(b1); order != 1 {
		t.Fatalf("b1: got order %d, want 1", order)
	}
	if order := tree.Insert(b2); order != 2 {
		t.Fatalf("b2: got order %d, want 2", order)
	}
	if order := tree.Insert(b3); order != 3 {
		t.Fatalf("b3: got order %d, want 3", order)
	}

	// Non-overlapping bounds reuse order 1.
	b4 := spBounds(20, 20, 10, 10)
	b5 := spBounds(40, 40, 10, 10)
	b6 := spBounds(25, 25, 10, 10)

	if order := tree.Insert(b4); order != 1 {
		t.Fatalf("b4: got order %d, want 1", order)
	}
	if order := tree.Insert(b5); order != 1 {
		t.Fatalf("b5: got order %d, want 1", order)
	}
	// b6 overlaps b4, so it gets order 2.
	if order := tree.Insert(b6); order != 2 {
		t.Fatalf("b6: got order %d, want 2", order)
	}
}

// TestBoundsTreeRandomIterations ports GPUI's test_random_iterations: for a
// thousand seeded random sequences, the tree's assigned order must equal one
// plus the maximum order of all previously inserted bounds that intersect the
// new ones.
func TestBoundsTreeRandomIterations(t *testing.T) {
	const maxBounds = 100
	for seed := int64(1); seed <= 1000; seed++ {
		var tree boundsTree
		r := rand.New(rand.NewSource(seed))

		type entry struct {
			bounds geometry.Bounds[geometry.ScaledPixels]
			order  DrawOrder
		}
		var entries []entry

		numBounds := r.Intn(maxBounds) + 1
		for i := 0; i < numBounds; i++ {
			minX := float32(r.Float64()*200 - 100)
			minY := float32(r.Float64()*200 - 100)
			w := float32(r.Float64() * 50)
			h := float32(r.Float64() * 50)
			bounds := spBounds(minX, minY, w, h)

			var expected DrawOrder
			for _, e := range entries {
				if e.bounds.Intersects(bounds) && e.order > expected {
					expected = e.order
				}
			}
			expected++

			actual := tree.Insert(bounds)
			if actual != expected {
				t.Fatalf("seed %d, insert %d: got order %d, want %d",
					seed, i, actual, expected)
			}
			entries = append(entries, entry{bounds, actual})
		}
	}
}
