package text

// byteLRU is a least-recently-used cache bounded by total byte size rather
// than entry count. Entries in this package's caches vary by orders of
// magnitude — a six-glyph word and a 96px heading mask are not the same
// weight — so a count-based ceiling would either starve small entries or let
// large ones blow the budget. sizeOf reports the weight of a value; the cache
// evicts the least-recently-used entries until the total is at or under
// maxBytes.
//
// Not safe for concurrent use, matching every other type in this package.
type byteLRU[K comparable, V any] struct {
	entries    map[K]*lruEntry[K, V]
	head, tail *lruEntry[K, V] // sentinels; head.prev is most-recently-used, tail.next is least
	size       int64
	maxBytes   int64
	sizeOf     func(V) int64
}

type lruEntry[K comparable, V any] struct {
	key        K
	value      V
	size       int64
	next, prev *lruEntry[K, V]
}

// newByteLRU returns an empty cache with the given byte ceiling. sizeOf must
// return a value's weight in bytes; it is called once per Put.
func newByteLRU[K comparable, V any](maxBytes int64, sizeOf func(V) int64) *byteLRU[K, V] {
	l := &byteLRU[K, V]{
		entries:  make(map[K]*lruEntry[K, V]),
		maxBytes: maxBytes,
		sizeOf:   sizeOf,
	}
	l.head = &lruEntry[K, V]{}
	l.tail = &lruEntry[K, V]{}
	l.head.prev = l.tail
	l.tail.next = l.head
	return l
}

// Get returns the value for k, if present, and marks it most recently used.
func (l *byteLRU[K, V]) Get(k K) (V, bool) {
	if e, ok := l.entries[k]; ok {
		l.unlink(e)
		l.pushFront(e)
		return e.value, true
	}
	var zero V
	return zero, false
}

// Put inserts or replaces the value for k, marks it most recently used, and
// evicts the least-recently-used entries until the cache is at or under its
// byte ceiling.
func (l *byteLRU[K, V]) Put(k K, v V) {
	size := l.sizeOf(v)
	if e, ok := l.entries[k]; ok {
		l.size += size - e.size
		e.value = v
		e.size = size
		l.unlink(e)
		l.pushFront(e)
	} else {
		e := &lruEntry[K, V]{key: k, value: v, size: size}
		l.entries[k] = e
		l.pushFront(e)
		l.size += size
	}
	l.evict()
}

// SetMaxBytes changes the byte ceiling, evicting immediately if the cache is
// now over it. Zero means the cache holds nothing, not that it is unbounded:
// evict runs while size > maxBytes, and every non-empty entry has size > 0.
func (l *byteLRU[K, V]) SetMaxBytes(maxBytes int64) {
	l.maxBytes = maxBytes
	l.evict()
}

// Len returns the number of entries held.
func (l *byteLRU[K, V]) Len() int { return len(l.entries) }

// Clear empties the cache.
func (l *byteLRU[K, V]) Clear() {
	l.entries = make(map[K]*lruEntry[K, V])
	l.head.prev = l.tail
	l.tail.next = l.head
	l.size = 0
}

func (l *byteLRU[K, V]) evict() {
	for l.size > l.maxBytes && l.tail.next != l.head {
		oldest := l.tail.next
		l.unlink(oldest)
		delete(l.entries, oldest.key)
		l.size -= oldest.size
	}
}

func (l *byteLRU[K, V]) unlink(e *lruEntry[K, V]) {
	e.next.prev = e.prev
	e.prev.next = e.next
}

// pushFront inserts e as the most-recently-used entry.
func (l *byteLRU[K, V]) pushFront(e *lruEntry[K, V]) {
	e.next = l.head
	e.prev = l.head.prev
	e.prev.next = e
	e.next.prev = e
}
