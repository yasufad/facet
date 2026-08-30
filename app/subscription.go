package app

// subscriberSet is a multi-map from a key to a set of subscribers, where each
// subscriber may be inert (not yet activated), active, or dropped.
//
// It mirrors GPUI's SubscriberSet. Subscriptions are inserted inert and
// activated later, so registering an observer during an update does not fire
// it for the very notification that is being processed: activation is deferred
// to the end of the flush. A subscriber may drop itself (or be dropped by an
// earlier subscriber) during a retain pass; the set copes by re-merging any
// subscribers added while the pass was running.
//
// Subscribers are held in a slice per key, in registration order. Dispatch
// walks the slice directly — no sort, no map iteration, no allocation. GPUI
// gets the same property from BTreeMap; a slice gets it without the tree.
type subscriberSet[K comparable, C any] struct {
	byKey map[K]*subscriberBucket[C]
	next  uint64
}

// subscriberBucket holds the subscribers for one key in registration order.
type subscriberBucket[C any] struct {
	subs []*subscriber[C]
}

type subscriber[C any] struct {
	id      uint64
	active  bool
	dropped bool
	cb      C
}

func newSubscriberSet[K comparable, C any]() *subscriberSet[K, C] {
	return &subscriberSet[K, C]{byKey: make(map[K]*subscriberBucket[C])}
}

// insert adds an inert subscriber for key and returns a Subscription that
// cancels it on drop, plus an activate function to call once the subscriber
// should start firing. The activate function is deferred by the caller (the
// App queues it as a Defer effect) so that observers registered during an
// update do not fire mid-flush.
func (s *subscriberSet[K, C]) insert(key K, cb C) (Subscription, func()) {
	s.next++
	id := s.next
	sub := &subscriber[C]{id: id, cb: cb}
	bucket := s.byKey[key]
	if bucket == nil {
		bucket = &subscriberBucket[C]{}
		s.byKey[key] = bucket
	}
	bucket.subs = append(bucket.subs, sub)

	activate := func() { sub.active = true }
	unsubscribe := func() {
		sub.dropped = true
		// Do not remove from the slice here; retain and remove compact
		// dropped entries when they next walk the slice. This keeps
		// unsubscribe O(1) and avoids shifting slice elements on the
		// uncommon path.
	}
	return Subscription{state: &subscriptionState{unsubscribe: unsubscribe}}, activate
}

// remove returns the callbacks of all active subscribers for key and clears
// them, in registration order. Used when an entity is dropped: its observers
// and subscribers go with it.
func (s *subscriberSet[K, C]) remove(key K) []C {
	bucket := s.byKey[key]
	delete(s.byKey, key)
	if bucket == nil {
		return nil
	}
	var out []C
	for _, sub := range bucket.subs {
		if sub.active && !sub.dropped {
			out = append(out, sub.cb)
		}
	}
	return out
}

// retain calls f for each active, non-dropped subscriber for key, in
// registration order. If f returns false the subscriber is removed. Subscribers
// dropped during the pass (by f itself or by an earlier subscriber) are
// removed. New subscribers inserted during the pass are preserved: they land in
// a fresh bucket and are merged back afterwards, so an observer that registers
// another observer mid-flush does not lose it.
//
// Dispatch walks the slice directly and filters in place. It allocates
// nothing when no subscribers are added or removed during the pass.
func (s *subscriberSet[K, C]) retain(key K, f func(*C) bool) {
	bucket := s.byKey[key]
	if bucket == nil {
		return
	}
	// Take the bucket out so inserts during the pass land in a fresh one
	// and are merged back at the end.
	delete(s.byKey, key)

	// Walk in registration order. Filter in place: compact surviving
	// subscribers to the front of the slice, then truncate.
	write := 0
	for _, sub := range bucket.subs {
		if !sub.active || sub.dropped {
			continue
		}
		keep := f(&sub.cb) && !sub.dropped
		if keep {
			bucket.subs[write] = sub
			write++
		}
	}
	// Nil out the tail so dropped subscribers are not pinned by the
	// underlying array.
	for i := write; i < len(bucket.subs); i++ {
		bucket.subs[i] = nil
	}
	bucket.subs = bucket.subs[:write]

	// Merge any subscribers inserted during the pass.
	if fresh, ok := s.byKey[key]; ok {
		bucket.subs = append(bucket.subs, fresh.subs...)
		delete(s.byKey, key)
	}

	if len(bucket.subs) > 0 {
		s.byKey[key] = bucket
	}
}

// Subscription is a handle to an observer or subscriber registration. Dropping
// it (calling Close) cancels the registration. Detach leaves the registration
// running until the entity it watches is dropped.
//
// Go has no RAII, so cancellation is explicit. The framework closes
// subscriptions tied to an entity when that entity is dropped; code that holds
// a Subscription for early cancellation should Close it when done.
//
// The state lives behind a pointer so that Close and Detach have value
// receivers and can be called inline on a returned value:
// app.Subscribe(...).Close(). A pointer receiver would not compile because the
// return value is not addressable.
type Subscription struct {
	state *subscriptionState
}

type subscriptionState struct {
	unsubscribe func()
	closed      bool
}

// Close cancels the subscription. It is safe to call multiple times and on a
// zero-value Subscription.
func (s Subscription) Close() {
	if s.state == nil || s.state.closed {
		return
	}
	s.state.closed = true
	s.state.unsubscribe()
}

// Detach leaves the subscription running independently of this handle. The
// callback continues to fire until the entity it watches is dropped.
func (s Subscription) Detach() {
	if s.state == nil {
		return
	}
	s.state.closed = true
	// Drop the unsubscribe closure without invoking it.
	s.state.unsubscribe = nil
}
