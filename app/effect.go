package app

import "reflect"

// effect is a side effect queued during an update and applied during the
// flush at its end. Effects can themselves raise effects, which are appended
// to the queue and processed in order; the flush only ends when the queue is
// empty.
type effect interface {
	apply(*App)
}

// notifyEffect schedules a run of an entity's observers. Deduplicated per
// entity per flush through App.pendingNotifications, so a burst of Notify
// calls collapses to a single observer run.
type notifyEffect struct {
	emitter entityID
}

func (e notifyEffect) apply(app *App) {
	delete(app.pendingNotifications, e.emitter)
	app.observers.retain(e.emitter, func(cb *observerHandler) bool {
		return (*cb)(app)
	})
}

// emitEffect delivers a typed event to an entity's subscribers of that event
// type.
type emitEffect struct {
	emitter   entityID
	eventType reflect.Type
	event     any // a *Evt, so the subscriber receives a pointer
}

func (e emitEffect) apply(app *App) {
	app.subscribers.retain(e.emitter, func(sub *subscriberEntry) bool {
		if sub.eventType != e.eventType {
			return true
		}
		return sub.handler(app, e.event)
	})
}

// deferEffect runs a closure at the end of the flush. The framework uses it to
// activate subscriptions registered during an update, so they do not fire for
// the notification in flight when they were created.
type deferEffect struct {
	callback func(*App)
}

func (e deferEffect) apply(app *App) {
	e.callback(app)
}

// effectQueue is a FIFO of effects. New effects raised while flushing are
// pushed to the back; the flush drains from the front.
type effectQueue struct {
	items []effect
}

func (q *effectQueue) push(e effect) { q.items = append(q.items, e) }
func (q *effectQueue) pop() (effect, bool) {
	if len(q.items) == 0 {
		return nil, false
	}
	e := q.items[0]
	q.items[0] = nil
	q.items = q.items[1:]
	return e, true
}
func (q *effectQueue) len() int { return len(q.items) }
