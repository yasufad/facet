package geometry

import "testing"

// TestBoundsToDevicePixelsAdjacentRectanglesShareEdges walks a chain of
// rectangles that tile a line in logical pixels and asserts that each one's
// far edge equals the next one's near edge in device pixels. This is the
// property independent rounding breaks: a neighbour's near edge is round(o*f)
// and this rectangle's far edge is round((o+s)*f), and when the neighbour's
// origin equals this rectangle's far edge the two expressions are identical.
//
// Origins are non-zero and fractional. A probe whose first rectangle starts
// at 0 cannot tell edge-snapping from independent rounding, because round(0)
// is 0 either way; the 0.4/9.2 at 1.5 case is included because it is a known
// failing case for independent rounding.
func TestBoundsToDevicePixelsAdjacentRectanglesShareEdges(t *testing.T) {
	cases := []struct {
		name   string
		x0, w  float32
		factor float32
		n      int
	}{
		{"0.4_9.2_at_1.5", 0.4, 9.2, 1.5, 8},
		{"1.3_7.7_at_1.25", 1.3, 7.7, 1.25, 8},
		{"0.4_9.2_at_2.5", 0.4, 9.2, 2.5, 8},
		{"3.1_11.9_at_1.75", 3.1, 11.9, 1.75, 8},
		{"0.6_5.3_at_1.333", 0.6, 5.3, 1.333, 8},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Horizontal chain: rectangles tile along x, sharing vertical extent.
			hPrev := BoundsToDevicePixels(
				NewBounds(NewPoint(Pixels(c.x0), Pixels(2.0)), NewSize(Pixels(c.w), Pixels(5.0))),
				c.factor,
			)
			for i := 1; i < c.n; i++ {
				x := c.x0 + float32(i)*c.w
				hCur := BoundsToDevicePixels(
					NewBounds(NewPoint(Pixels(x), Pixels(2.0)), NewSize(Pixels(c.w), Pixels(5.0))),
					c.factor,
				)
				if hCur.Left() != hPrev.Right() {
					t.Fatalf("x chain i=%d: prev right %d != cur left %d (origin %v size %v factor %v)",
						i, hPrev.Right(), hCur.Left(), x, c.w, c.factor)
				}
				hPrev = hCur
			}

			// Vertical chain: rectangles tile along y, sharing horizontal extent.
			vPrev := BoundsToDevicePixels(
				NewBounds(NewPoint(Pixels(2.0), Pixels(c.x0)), NewSize(Pixels(5.0), Pixels(c.w))),
				c.factor,
			)
			for i := 1; i < c.n; i++ {
				y := c.x0 + float32(i)*c.w
				vCur := BoundsToDevicePixels(
					NewBounds(NewPoint(Pixels(2.0), Pixels(y)), NewSize(Pixels(5.0), Pixels(c.w))),
					c.factor,
				)
				if vCur.Top() != vPrev.Bottom() {
					t.Fatalf("y chain i=%d: prev bottom %d != cur top %d (origin %v size %v factor %v)",
						i, vPrev.Bottom(), vCur.Top(), y, c.w, c.factor)
				}
				vPrev = vCur
			}
		})
	}
}

// TestDeviceBoundsToPixelsRoundTripPreservesAdjacency converts a chain of
// touching logical rectangles to device pixels and back, and asserts the
// rectangles still touch in logical space. Deriving the size from the edges
// on the way back keeps a rectangle's far edge equal to its neighbour's near
// edge; converting the size independently can leave a one-ULP gap that rounds
// to a device pixel on the next forward pass.
func TestDeviceBoundsToPixelsRoundTripPreservesAdjacency(t *testing.T) {
	const (
		x0, w  = 0.4, 9.2
		factor = 1.5
		n      = 8
	)
	prev := DeviceBoundsToPixels(
		BoundsToDevicePixels(
			NewBounds(NewPoint(Pixels(x0), Pixels(1.0)), NewSize(Pixels(w), Pixels(4.0))),
			factor,
		),
		factor,
	)
	for i := 1; i < n; i++ {
		x := x0 + float32(i)*w
		cur := DeviceBoundsToPixels(
			BoundsToDevicePixels(
				NewBounds(NewPoint(Pixels(x), Pixels(1.0)), NewSize(Pixels(w), Pixels(4.0))),
				factor,
			),
			factor,
		)
		if cur.Left() != prev.Right() {
			t.Fatalf("round trip i=%d: prev right %v != cur left %v", i, prev.Right(), cur.Left())
		}
		prev = cur
	}
}
