package app

import "sort"

// subscriberSet is a multi-map from a key to a set of subscribers, where each
// subscriber may be inert (not yet activated), active, or dropped.
//
// It mirrors GPUI's SubscriberSet. Subscriptions are inserted inert and
// activated later, so registering an observer during an update does not fire
// it for the very notification that is being processed: activation is deferred
// to the end of the flush. A subscriber may drop itself (or be dropped by an
// earlier subscriber) during a retain pass; the set copes by re-merging any
// subscribers added while the pass was running.
type subscriberSet[K comparable, C any] struct {
	byKey map[K]map[uint64]*subscriber[C]
	next  uint64
}

type subscriber[C any] struct {
	active  bool
	dropped bool
	cb      C
}

func newSubscriberSet[K comparable, C any]() *subscriberSet[K, C] {
	return &subscriberSet[K, C]{byKey: make(map[K]map[uint64]*subscriber[C])}
}

// insert adds an inert subscriber for key and returns a Subscription that
// cancels it on drop, plus an activate function to call once the subscriber
// should start firing. The activate function is deferred by the caller (the
// App queues it as a Defer effect) so that observers registered during an
// update do not fire mid-flush.
func (s *subscriberSet[K, C]) insert(key K, cb C) (Subscription, func()) {
	s.next++
	id := s.next
	sub := &subscriber[C]{cb: cb}
	if s.byKey[key] == nil {
		s.byKey[key] = make(map[uint64]*subscriber[C])
	}
	s.byKey[key][id] = sub

	activate := func() { sub.active = true }
	unsubscribe := func() {
		sub.dropped = true
		if subs, ok := s.byKey[key]; ok {
			delete(subs, id)
			if len(subs) == 0 {
				delete(s.byKey, key)
			}
		}
	}
	return Subscription{state: &subscriptionState{unsubscribe: unsubscribe}}, activate
}

// remove returns the callbacks of all active subscribers for key and clears
// them, in registration order. Used when an entity is dropped: its observers
// and subscribers go with it.
func (s *subscriberSet[K, C]) remove(key K) []C {
	subs := s.byKey[key]
	delete(s.byKey, key)
	ids := make([]uint64, 0, len(subs))
	for id := range subs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	var out []C
	for _, id := range ids {
		sub := subs[id]
		if sub.active && !sub.dropped {
			out = append(out, sub.cb)
		}
	}
	return out
}

// retain calls f for each active, non-dropped subscriber for key. If f returns
// false the subscriber is removed. Subscribers dropped during the pass (by f
// itself or by an earlier subscriber) are removed. New subscribers inserted
// during the pass are preserved: they are merged back afterwards, so an
// observer that registers another observer mid-flush does not lose it.
func (s *subscriberSet[K, C]) retain(key K, f func(*C) bool) {
	subs := s.byKey[key]
	if subs == nil {
		return
	}
	// Take the bucket out so inserts during the pass land in a fresh one and
	// are merged back at the end.
	delete(s.byKey, key)

	// Dispatch in registration order (by monotonic id), not map iteration
	// order. Map iteration is nondeterministic in Go; pinning to id makes
	// observer dispatch reproducible across runs.
	ids := make([]uint64, 0, len(subs))
	for id := range subs {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	for _, id := range ids {
		sub := subs[id]
		if sub == nil {
			continue
		}
		if !sub.active {
			continue
		}
		if sub.dropped {
			delete(subs, id)
			continue
		}
		keep := f(&sub.cb) && !sub.dropped
		if !keep {
			delete(subs, id)
		}
	}
	// Merge any subscribers inserted during the pass into the bucket.
	if fresh, ok := s.byKey[key]; ok {
		for id, sub := range fresh {
			subs[id] = sub
		}
		delete(s.byKey, key)
	}
	if len(subs) > 0 {
		s.byKey[key] = subs
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
