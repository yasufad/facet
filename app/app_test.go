package app

import (
	"reflect"
	"testing"
)

// observerCount is a tiny entity that counts how many times its observer fires.
type observerCount struct {
	hits int
}

// pingEvent is a typed event used by the subscribe tests.
type pingEvent struct {
	n int
}

func TestNotifyDeduplicatesPerFlush(t *testing.T) {
	// A hundred Notify calls during one update must collapse to a single
	// observer run.
	app := NewApp()
	defer app.Close()

	observed := newCounter(t, app, 0)
	observer := New(app, func(cx *Context[observerCount]) observerCount {
		Observe(cx, observed, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			v.hits++
		})
		return observerCount{}
	})

	for i := 0; i < 100; i++ {
		observed.Update(app, func(v *counter, cx *Context[counter]) {
			v.count++
			cx.Notify()
		})
	}
	// Each Update is its own flush, so 100 updates → 100 hits. The
	// deduplication is per-flush, so check that within a single update.
	observer.Update(app, func(v *observerCount, cx *Context[observerCount]) {
		v.hits = 0
	})

	observed.Update(app, func(v *counter, cx *Context[counter]) {
		v.count += 100
		for i := 0; i < 100; i++ {
			cx.Notify()
		}
	})
	if got := observer.Read(app).hits; got != 1 {
		t.Fatalf("got %d hits, want 1 (notify is deduplicated per flush)", got)
	}

	observer.Release()
	observed.Release()
}

func TestObserverFiresOnNotify(t *testing.T) {
	app := NewApp()
	defer app.Close()

	observed := newCounter(t, app, 0)
	observer := New(app, func(cx *Context[observerCount]) observerCount {
		Observe(cx, observed, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			v.hits++
		})
		return observerCount{}
	})

	observed.Update(app, func(v *counter, cx *Context[counter]) {
		cx.Notify()
	})
	if got := observer.Read(app).hits; got != 1 {
		t.Fatalf("got %d hits, want 1", got)
	}

	observer.Release()
	observed.Release()
}

func TestObserverStopsWhenObserverDropped(t *testing.T) {
	// When the observer entity is dropped, its weak handle no longer upgrades
	// and the registration removes itself on the next notify.
	app := NewApp()
	defer app.Close()

	observed := newCounter(t, app, 0)
	hits := 0

	observer := New(app, func(cx *Context[observerCount]) observerCount {
		Observe(cx, observed, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			hits++
		})
		return observerCount{}
	})

	observed.Update(app, func(v *counter, cx *Context[counter]) { cx.Notify() })
	if hits != 1 {
		t.Fatalf("got %d hits, want 1 before drop", hits)
	}

	observer.Release()
	// Notifying after the observer is dropped must not fire the callback. The
	// registration removes itself when the weak handle fails to upgrade.
	observed.Update(app, func(v *counter, cx *Context[counter]) { cx.Notify() })
	if hits != 1 {
		t.Fatalf("got %d hits, want 1 after observer dropped", hits)
	}

	observed.Release()
}

func TestObserverStopsWhenObservedDropped(t *testing.T) {
	// When the observed entity is dropped, its observer set is cleared, so the
	// observer's registration is gone and no dangling reference remains.
	app := NewApp()
	defer app.Close()

	observed := newCounter(t, app, 0)
	hits := 0

	observer := New(app, func(cx *Context[observerCount]) observerCount {
		Observe(cx, observed, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			hits++
		})
		return observerCount{}
	})
	defer observer.Release()

	observed.Update(app, func(v *counter, cx *Context[counter]) { cx.Notify() })
	if hits != 1 {
		t.Fatalf("got %d hits, want 1 before drop", hits)
	}

	observed.Release()
	// The observer set for the dropped entity is gone; the observer's local
	// hit counter is unaffected (still 1, not incremented again).
	if hits != 1 {
		t.Fatalf("observer fired after observed was dropped: got %d hits, want 1", hits)
	}
}

func TestSubscribeReceivesTypedEvents(t *testing.T) {
	app := NewApp()
	defer app.Close()

	emitter := newCounter(t, app, 0)
	received := []pingEvent{}

	subscriber := New(app, func(cx *Context[observerCount]) observerCount {
		Subscribe(cx, emitter, func(v *observerCount, e Entity[counter], evt *pingEvent, cx *Context[observerCount]) {
			received = append(received, *evt)
		})
		return observerCount{}
	})
	defer subscriber.Release()

	emitter.Update(app, func(v *counter, cx *Context[counter]) {
		Emit(cx, pingEvent{n: 1})
		Emit(cx, pingEvent{n: 2})
	})
	if !reflect.DeepEqual(received, []pingEvent{{1}, {2}}) {
		t.Fatalf("got %v, want [{1} {2}]", received)
	}
}

func TestSubscriberStopsWhenSubscriberDropped(t *testing.T) {
	app := NewApp()
	defer app.Close()

	emitter := newCounter(t, app, 0)
	received := 0

	subscriber := New(app, func(cx *Context[observerCount]) observerCount {
		Subscribe(cx, emitter, func(v *observerCount, e Entity[counter], evt *pingEvent, cx *Context[observerCount]) {
			received++
		})
		return observerCount{}
	})

	emitter.Update(app, func(v *counter, cx *Context[counter]) { Emit(cx, pingEvent{}) })
	if received != 1 {
		t.Fatalf("got %d, want 1 before drop", received)
	}

	subscriber.Release()
	emitter.Update(app, func(v *counter, cx *Context[counter]) { Emit(cx, pingEvent{}) })
	if received != 1 {
		t.Fatalf("got %d, want 1 after subscriber dropped", received)
	}
	emitter.Release()
}

func TestSubscriptionCloseCancels(t *testing.T) {
	app := NewApp()
	defer app.Close()

	observed := newCounter(t, app, 0)
	hits := 0

	var sub Subscription
	observer := New(app, func(cx *Context[observerCount]) observerCount {
		sub = Observe(cx, observed, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			hits++
		})
		return observerCount{}
	})
	defer observer.Release()

	sub.Close()
	observed.Update(app, func(v *counter, cx *Context[counter]) { cx.Notify() })
	if hits != 0 {
		t.Fatalf("got %d hits, want 0 after subscription closed", hits)
	}
	observed.Release()
}

func TestNotificationDuringFlushChains(t *testing.T) {
	// An observer that notifies another entity must cause a follow-on notify
	// effect processed in the same flush: effects cause effects, and the flush
	// only ends when the queue is empty.
	app := NewApp()
	defer app.Close()

	a := newCounter(t, app, 0)
	b := newCounter(t, app, 0)

	// When A notifies, B's observer bumps B and notifies B. A second observer
	// on B records that B's notify was reached in the same flush.
	bReached := 0
	host := New(app, func(cx *Context[observerCount]) observerCount {
		Observe(cx, a, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			b.Update(app, func(v *counter, cx *Context[counter]) {
				v.count++
				cx.Notify()
			})
		})
		Observe(cx, b, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			bReached++
		})
		return observerCount{}
	})
	defer host.Release()

	a.Update(app, func(v *counter, cx *Context[counter]) {
		cx.Notify()
	})
	if got := b.Read(app).count; got != 1 {
		t.Fatalf("B count got %d, want 1", got)
	}
	if bReached != 1 {
		t.Fatalf("B's notify was not reached in the same flush: got %d, want 1", bReached)
	}
	a.Release()
	b.Release()
}

func TestEffectOrderingFIFO(t *testing.T) {
	// Effects are processed in FIFO order. Notify A then emit on A in one
	// update: the observer fires before the subscriber, because notify was
	// queued first.
	app := NewApp()
	defer app.Close()

	emitter := newCounter(t, app, 0)
	order := []string{}

	host := New(app, func(cx *Context[observerCount]) observerCount {
		Observe(cx, emitter, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			order = append(order, "observe")
		})
		Subscribe(cx, emitter, func(v *observerCount, e Entity[counter], evt *pingEvent, cx *Context[observerCount]) {
			order = append(order, "subscribe")
		})
		return observerCount{}
	})
	defer host.Release()

	emitter.Update(app, func(v *counter, cx *Context[counter]) {
		cx.Notify()           // queued first
		Emit(cx, pingEvent{}) // queued second
	})
	if !reflect.DeepEqual(order, []string{"observe", "subscribe"}) {
		t.Fatalf("got %v, want [observe subscribe]", order)
	}
	emitter.Release()
}

func TestObserveRegisteredDuringNotifyDoesNotFireImmediately(t *testing.T) {
	// An observer registered during a flush is activated at the end of the
	// flush, so it does not fire for the notification in flight when it was
	// created.
	app := NewApp()
	defer app.Close()

	observed := newCounter(t, app, 0)
	laterHits := 0

	host := New(app, func(cx *Context[observerCount]) observerCount {
		Observe(cx, observed, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			// Register another observer mid-flush. It must not fire for this
			// very notification.
			Observe(cx, observed, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
				laterHits++
			})
		})
		return observerCount{}
	})
	defer host.Release()

	observed.Update(app, func(v *counter, cx *Context[counter]) { cx.Notify() })
	if laterHits != 0 {
		t.Fatalf("observer registered mid-flush fired immediately: got %d, want 0", laterHits)
	}
	// A subsequent notify should fire it.
	observed.Update(app, func(v *counter, cx *Context[counter]) { cx.Notify() })
	if laterHits != 1 {
		t.Fatalf("observer registered mid-flush did not fire on next notify: got %d, want 1", laterHits)
	}
	observed.Release()
}

func TestObserverDispatchOrderIsRegistrationOrder(t *testing.T) {
	// Multiple observers of the same entity must fire in registration order,
	// not map iteration order. This is the kind of regression that reappears
	// quietly, so the test pins the order explicitly.
	app := NewApp()
	defer app.Close()

	observed := newCounter(t, app, 0)
	defer observed.Release()

	var order []string
	host := New(app, func(cx *Context[observerCount]) observerCount {
		Observe(cx, observed, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			order = append(order, "a")
		})
		Observe(cx, observed, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			order = append(order, "b")
		})
		Observe(cx, observed, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			order = append(order, "c")
		})
		Observe(cx, observed, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			order = append(order, "d")
		})
		Observe(cx, observed, func(v *observerCount, e Entity[counter], cx *Context[observerCount]) {
			order = append(order, "e")
		})
		return observerCount{}
	})
	defer host.Release()

	// Run several notifies; the order must be identical every time.
	for run := 0; run < 5; run++ {
		order = nil
		observed.Update(app, func(v *counter, cx *Context[counter]) { cx.Notify() })
		want := []string{"a", "b", "c", "d", "e"}
		if !reflect.DeepEqual(order, want) {
			t.Fatalf("run %d: got %v, want %v", run, order, want)
		}
	}
}
