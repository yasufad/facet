package text

import "testing"

// TestByteLRUEvictsOldest fills a cache past its byte ceiling and checks that
// the least-recently-used entry is gone while a recently-touched one
// survives. Getting "a" before inserting "c" makes "b" the least-recently-used
// entry, not "a" — a cache that evicts by insertion order (FIFO) rather than
// use order would evict "a" instead and this test would still pass a naive
// "something got evicted" check, so it asserts on which key survives.
func TestByteLRUEvictsOldest(t *testing.T) {
	l := newByteLRU[string, int](2, func(int) int64 { return 1 })
	l.Put("a", 1)
	l.Put("b", 2)
	if _, ok := l.Get("a"); !ok {
		t.Fatal("a should still be present before eviction")
	}
	// a is now most-recently-used; b is least-recently-used.
	l.Put("c", 3)

	if _, ok := l.Get("b"); ok {
		t.Fatal("b should have been evicted as the least-recently-used entry")
	}
	if _, ok := l.Get("a"); !ok {
		t.Fatal("a should have survived eviction, it was touched most recently")
	}
	if _, ok := l.Get("c"); !ok {
		t.Fatal("c should be present, it was just inserted")
	}
	if got := l.Len(); got != 2 {
		t.Fatalf("cache should hold 2 entries at the ceiling, got %d", got)
	}
}

// TestByteLRURespectsSize checks eviction is driven by total byte weight, not
// entry count: a single oversized value should evict as many prior entries as
// needed to fit.
func TestByteLRURespectsSize(t *testing.T) {
	l := newByteLRU[string, int](10, func(v int) int64 { return int64(v) })
	l.Put("small1", 3)
	l.Put("small2", 3)
	l.Put("big", 8)

	if _, ok := l.Get("big"); !ok {
		t.Fatal("big should be present")
	}
	if l.Len() != 1 {
		t.Fatalf("big alone should have evicted both small entries, got %d entries", l.Len())
	}
}

// TestByteLRUSetMaxBytesEvicts checks that lowering the ceiling evicts
// immediately rather than waiting for the next Put.
func TestByteLRUSetMaxBytesEvicts(t *testing.T) {
	l := newByteLRU[string, int](10, func(int) int64 { return 5 })
	l.Put("a", 1)
	l.Put("b", 2)
	if l.Len() != 2 {
		t.Fatalf("expected 2 entries before shrinking, got %d", l.Len())
	}
	l.SetMaxBytes(5)
	if l.Len() != 1 {
		t.Fatalf("expected 1 entry after shrinking ceiling to fit one, got %d", l.Len())
	}
	if _, ok := l.Get("b"); !ok {
		t.Fatal("b was the most-recently-used entry and should have survived")
	}
}
