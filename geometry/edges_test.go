package geometry

import "testing"

func TestExtend(t *testing.T) {
	b := boundsFromInts(0, 0, 10, 10)
	edges := NewEdges(1, 2, 3, 4) // top, right, bottom, left

	got := b.Extend(edges)
	// Origin moves out by (left, top); size grows by (left+right, top+bottom).
	want := boundsFromInts(-4, -1, 16, 14)
	if got != want {
		t.Fatalf("Extend = %v, want %v", got, want)
	}
}

func TestSpaceWithin(t *testing.T) {
	outer := boundsFromInts(0, 0, 10, 10)
	inner := boundsFromInts(2, 2, 6, 6)

	got := inner.SpaceWithin(outer)
	want := Edges[int]{Top: 2, Right: 2, Bottom: 2, Left: 2}
	if got != want {
		t.Fatalf("SpaceWithin = %v, want %v", got, want)
	}
}

func TestSymmetricEdges(t *testing.T) {
	got := SymmetricEdges(3, 5)
	want := Edges[int]{Top: 3, Right: 5, Bottom: 3, Left: 5}
	if got != want {
		t.Fatalf("SymmetricEdges = %v, want %v", got, want)
	}
}

func TestAllEdges(t *testing.T) {
	got := AllEdges(7)
	want := Edges[int]{Top: 7, Right: 7, Bottom: 7, Left: 7}
	if got != want {
		t.Fatalf("AllEdges = %v, want %v", got, want)
	}
}

func TestEdgesMax(t *testing.T) {
	e := NewEdges(3, 9, 2, 5)
	if got := e.Max(); got != 9 {
		t.Fatalf("Max = %d, want 9", got)
	}
}
