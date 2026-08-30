package geometry

import "testing"

// boundsFromInts builds a Bounds[int] from raw coordinates for test brevity.
func boundsFromInts(x, y, w, h int) Bounds[int] {
	return NewBounds(NewPoint(x, y), NewSize(w, h))
}

func TestIntersectOverlap(t *testing.T) {
	a := boundsFromInts(0, 0, 10, 10)
	b := boundsFromInts(5, 5, 10, 10)

	got := a.Intersect(b)
	want := boundsFromInts(5, 5, 5, 5)
	if got != want {
		t.Fatalf("Intersect = %v, want %v", got, want)
	}
}

func TestIntersectNoOverlapIsEmpty(t *testing.T) {
	a := boundsFromInts(0, 0, 10, 10)
	b := boundsFromInts(20, 20, 10, 10)

	got := a.Intersect(b)
	if !got.IsEmpty() {
		t.Fatalf("Intersect of disjoint bounds = %v, want empty", got)
	}
}

// Edge-touching bounds share a boundary but no area, so they neither intersect
// nor produce a non-empty intersection. This is the half-open convention.
func TestIntersectEdgeTouchingIsEmpty(t *testing.T) {
	a := boundsFromInts(0, 0, 10, 10)
	b := boundsFromInts(10, 0, 10, 10)

	if a.Intersects(b) {
		t.Fatalf("edge-touching bounds should not Intersect")
	}
	got := a.Intersect(b)
	if !got.IsEmpty() {
		t.Fatalf("Intersect of edge-touching bounds = %v, want empty", got)
	}
}

func TestUnion(t *testing.T) {
	a := boundsFromInts(0, 0, 10, 10)
	b := boundsFromInts(5, 5, 15, 15)

	got := a.Union(b)
	want := boundsFromInts(0, 0, 20, 20)
	if got != want {
		t.Fatalf("Union = %v, want %v", got, want)
	}
}

func TestContainsHalfOpen(t *testing.T) {
	b := boundsFromInts(0, 0, 10, 10)

	cases := []struct {
		p    Point[int]
		want bool
	}{
		{NewPoint(0, 0), true},   // top-left included
		{NewPoint(9, 9), true},   // interior
		{NewPoint(10, 5), false}, // right edge excluded
		{NewPoint(5, 10), false}, // bottom edge excluded
		{NewPoint(-1, 5), false}, // left of origin
		{NewPoint(5, -1), false}, // above origin
	}
	for _, c := range cases {
		if got := b.Contains(c.p); got != c.want {
			t.Fatalf("Contains(%v) = %v, want %v", c.p, got, c.want)
		}
	}
}

func TestDilateAndInset(t *testing.T) {
	b := boundsFromInts(10, 10, 10, 10)

	if got := b.Dilate(2); got != boundsFromInts(8, 8, 14, 14) {
		t.Fatalf("Dilate = %v", got)
	}
	// Inset is Dilate negated: origin moves in, size shrinks by twice amount.
	if got := b.Inset(2); got != boundsFromInts(12, 12, 6, 6) {
		t.Fatalf("Inset = %v", got)
	}
}

func TestFromCorners(t *testing.T) {
	got := FromCorners(NewPoint(3, 4), NewPoint(13, 19))
	want := boundsFromInts(3, 4, 10, 15)
	if got != want {
		t.Fatalf("FromCorners = %v, want %v", got, want)
	}
}

func TestCenter(t *testing.T) {
	b := boundsFromInts(0, 0, 10, 20)
	if got := b.Center(); got != NewPoint(5, 10) {
		t.Fatalf("Center = %v, want (5,10)", got)
	}
}

func TestCenteredAt(t *testing.T) {
	got := CenteredAt(NewPoint(10, 10), NewSize(4, 6))
	want := boundsFromInts(8, 7, 4, 6)
	if got != want {
		t.Fatalf("CenteredAt = %v, want %v", got, want)
	}
}
