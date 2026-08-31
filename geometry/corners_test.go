package geometry

import "testing"

func TestCornersCornerCentresAverage(t *testing.T) {
	c := NewCorners(10, 20, 40, 30) // topLeft, topRight, bottomRight, bottomLeft

	cases := []struct {
		anchor Anchor
		want   int
	}{
		{TopLeft, 10},
		{TopRight, 20},
		{BottomLeft, 30},
		{BottomRight, 40},
		{TopCentre, 15},    // (topLeft + topRight) / 2
		{BottomCentre, 35}, // (bottomLeft + bottomRight) / 2
		{LeftCentre, 20},   // (topLeft + bottomLeft) / 2
		{RightCentre, 30},  // (topRight + bottomRight) / 2
	}
	for _, tc := range cases {
		if got := c.Corner(tc.anchor); got != tc.want {
			t.Fatalf("Corner(%v) = %d, want %d", tc.anchor, got, tc.want)
		}
	}
}

func TestSymmetricCorners(t *testing.T) {
	got := SymmetricCorners(5, 9)
	want := Corners[int]{TopLeft: 5, TopRight: 5, BottomRight: 9, BottomLeft: 9}
	if got != want {
		t.Fatalf("SymmetricCorners = %v, want %v", got, want)
	}
}

func TestCornersMax(t *testing.T) {
	c := NewCorners(3, 9, 2, 5)
	if got := c.Max(); got != 9 {
		t.Fatalf("Max = %d, want 9", got)
	}
}
