package app

import (
	"fmt"
	"reflect"
)

// observerHandler is a notify callback. It returns false to remove itself (for
// example when the observer it forwards to has been dropped and its weak
// handle no longer upgrades).
type observerHandler func(*App) bool

// subscriberEntry is a typed event subscription: the event type it matches and
// the handler that receives the event (a *Evt).
type subscriberEntry struct {
	eventType reflect.Type
	handler   func(*App, any) bool
}

// releaseHandler runs against an entity's value when the entity is dropped.
type releaseHandler func(value any, app *App)

// App is the global context: it owns the entity map, the effect queue and the
// executors, and mediates every read and write of entity state.
//
// App is not safe to use from goroutines other than the one that created it.
// Background work reaches entity state through AsyncApp, which marshals onto
// the UI goroutine.
type App struct {
	rc       *refCounts
	entities *entityMap

	effects              effectQueue
	pendingNotifications map[entityID]struct{}

	observers   *subscriberSet[entityID, observerHandler]
	subscribers *subscriberSet[entityID, subscriberEntry]
	releases    *subscriberSet[entityID, releaseHandler]

	flushingEffects bool
	pendingUpdates  int

	// generation is incremented at the start and end of each outermost
	// update. A Context records the generation at creation and checks it at
	// every accessor: if the generation has moved on, the context has
	// escaped its update and must not be used. The check is an integer
	// compare (~1ns), so it is cheap enough to run at every Context accessor
	// on the per-frame path. The full goroutine check (checkUI, ~6µs) runs
	// at update boundaries and on exported App methods, where a few
	// microseconds a few times per frame is affordable.
	generation int64

	fg *ForegroundExecutor
	bg *BackgroundExecutor

	uiGoroutine int64
}

// NewApp constructs an App bound to the calling goroutine. The calling
// goroutine becomes the UI goroutine: every context method invoked from
// another goroutine panics. The foreground and background executors are
// created here so that App is self-contained and testable before platform is
// wired in.
func NewApp() *App {
	app := &App{
		rc:                   newRefCounts(),
		entities:             newEntityMap(),
		pendingNotifications: make(map[entityID]struct{}),
		observers:            newSubscriberSet[entityID, observerHandler](),
		subscribers:          newSubscriberSet[entityID, subscriberEntry](),
		releases:             newSubscriberSet[entityID, releaseHandler](),
		uiGoroutine:          goroutineID(),
	}
	app.fg = newForegroundExecutor()
	app.bg = newBackgroundExecutor()
	return app
}

// Foreground returns the executor that runs tasks on the UI goroutine.
func (app *App) Foreground() *ForegroundExecutor { return app.fg }

// Background returns the executor that runs tasks on a background goroutine
// pool.
func (app *App) Background() *BackgroundExecutor { return app.bg }

// Async returns an AsyncApp that can be held across await points. Every entity
// operation on it is marshalled back onto the UI goroutine and is fallible if
// the App has been shut down.
func (app *App) Async() *AsyncApp {
	return &AsyncApp{app: app}
}

// Close shuts the App down: the executors stop and further releases do not
// queue drops. The App and its entity map become unusable afterwards.
func (app *App) Close() {
	app.checkUI()
	app.fg.stop()
	app.bg.stop()
	app.rc.markShutdown()
}

// checkUI panics if the calling goroutine is not the UI goroutine. It is
// called from every exported entry point that touches entity state, the
// effect queue or the subscriber sets. The check costs about 6µs
// (runtime.Stack to read the goroutine header); see goroutine.go for the
// measurement and the plan for reducing the per-frame cost.
func (app *App) checkUI() {
	if goroutineID() != app.uiGoroutine {
		panic("app: context used from a goroutine other than the UI goroutine")
	}
}

// update runs f against the App inside an update boundary: effects raised
// inside f (and any they cause) flush once f returns. Updates nest; only the
// outermost flushes, so a burst of mutations produces one frame.
//
// The generation is incremented at the start of the outermost update and at
// the end of the outermost update. Contexts created during the update record
// the start generation; after the update ends, the generation has moved on
// and any escaped context fails the generation check.
func (app *App) update(f func(*App)) {
	app.checkUI()
	if app.pendingUpdates == 0 {
		app.generation++
	}
	app.pendingUpdates++
	f(app)
	app.finishUpdate()
}

func (app *App) finishUpdate() {
	if !app.flushingEffects && app.pendingUpdates == 1 {
		app.flushingEffects = true
		app.flushEffects()
		app.flushingEffects = false
	}
	app.pendingUpdates--
	if app.pendingUpdates == 0 {
		app.generation++
	}
}

// Flush reaps dropped entities and processes any queued effects without opening
// an update boundary. It is for tests and for the platform loop between frames:
// ordinary code drops handles inside an update, where the flush at the end of
// the update reaps them. A handle released at the top level is not reaped
// until the next Flush (or update).
func (app *App) Flush() {
	app.checkUI()
	if app.flushingEffects {
		return
	}
	app.flushingEffects = true
	app.flushEffects()
	app.flushingEffects = false
}

// flushEffects drains the effect queue. Effects may raise further effects,
// which are appended and processed in turn; the flush only ends when the queue
// is empty. Dropped entities are reaped at the top of each pass so that
// release observers see a consistent map and observers of a dropped entity do
// not fire.
func (app *App) flushEffects() {
	for {
		app.releaseDropped()
		eff, ok := app.effects.pop()
		if !ok {
			break
		}
		eff.apply(app)
	}
}

// releaseDropped reaps entities whose strong count reached zero since the last
// pass. For each, the release observers run against the value (so an entity
// can release the handles it owns), then the value is removed from the map and
// the entity's own observers and subscribers are cleared.
func (app *App) releaseDropped() {
	for {
		dropped := app.entities.takeDropped(app.rc)
		if len(dropped) == 0 {
			return
		}
		for _, id := range dropped {
			// Run release observers before removing the value, so they can
			// read the entity one last time and release the handles it owns.
			for _, rh := range app.releases.remove(id) {
				value := app.entities.values[id]
				rh(value, app)
			}
			app.entities.remove(id)
			app.observers.remove(id)
			app.subscribers.remove(id)
			// releases were already removed above when collecting handlers.
			delete(app.rc.counts, id)
		}
	}
}

// pushEffect queues an effect, deduplicating Notify effects per entity per
// flush so a burst of Notify calls collapses to a single observer run.
func (app *App) pushEffect(e effect) {
	if ne, ok := e.(notifyEffect); ok {
		if _, pending := app.pendingNotifications[ne.emitter]; pending {
			return
		}
		app.pendingNotifications[ne.emitter] = struct{}{}
	}
	app.effects.push(e)
}

// New builds an entity owned by the App. The constructor runs with a Context
// for the new entity, so it can observe or subscribe to others as it is
// created; it returns the value, which the App then stores. The returned
// handle is owning (count one).
func New[T any](app *App, build func(cx *Context[T]) T) Entity[T] {
	var handle Entity[T]
	app.update(func(cx *App) {
		id := cx.entities.reserveID(cx.rc)
		handle = Entity[T]{id: id, rc: cx.rc}
		ctx := &Context[T]{app: cx, self: handle.Downgrade(), generation: cx.generation}
		value := build(ctx)
		cx.entities.insert(id, value)
	})
	return handle
}

// ReadEntity returns a pointer to the value for handle. It panics if the entity
// is being updated (a re-entrant update) or has been dropped.
//
// It is a top-level function because Go does not permit methods to declare
// type parameters; Entity.Read wraps it.
func ReadEntity[T any](app *App, handle Entity[T]) *T {
	app.checkUI()
	v := app.entities.read(handle.id)
	t, ok := v.(T)
	if !ok {
		panic(fmt.Sprintf("app: entity %d is not of type %T", handle.id, *new(T)))
	}
	return &t
}

// UpdateEntity leases the value out, runs f against it with a Context for the
// entity, and restores it. Effects raised inside f flush once f returns. A
// re-entrant update of the same entity panics, because the value is on lease.
//
// It is a top-level function because Go does not permit methods to declare
// type parameters; Entity.Update wraps it.
func UpdateEntity[T any](app *App, handle Entity[T], f func(v *T, cx *Context[T])) {
	app.update(func(cx *App) {
		v := cx.entities.lease(handle.id)
		t, ok := v.(T)
		if !ok {
			// Restore before panicking so the map is not left with a hole.
			cx.entities.endLease(handle.id, v)
			panic(fmt.Sprintf("app: entity %d is not of type %T", handle.id, *new(T)))
		}
		ctx := &Context[T]{app: cx, self: handle.Downgrade(), generation: cx.generation}
		f(&t, ctx)
		cx.entities.endLease(handle.id, t)
	})
}

// notify is the unexported inner of Notify. It does not checkUI; callers
// must have verified the goroutine (App.Notify) or the generation (Context).
func (app *App) notify(id entityID) {
	app.pushEffect(notifyEffect{emitter: id})
}

// Notify marks the entity dirty and schedules its observers. Deduplicated per
// entity per flush: a hundred Notify calls during one update collapse to a
// single observer run.
func (app *App) Notify(id entityID) {
	app.checkUI()
	app.notify(id)
}

// observe is the unexported inner of Observe.
func (app *App) observe(entity anyEntity, onNotify func(cx *App) bool) Subscription {
	sub, activate := app.observers.insert(entity.id, observerHandler(onNotify))
	app.deferFn(func(*App) { activate() })
	return sub
}

// Observe registers a callback to run when the entity notify is called on
// entity. The callback returns false to remove itself. The registration is
// activated at the end of the current flush, so an observer registered during a
// notification does not fire for that very notification.
//
// This is the low-level, App-scoped form. Context.Observe wraps it to forward
// to a specific observer entity through a weak handle, so the observer is not
// kept alive by what it watches.
func (app *App) Observe(entity anyEntity, onNotify func(cx *App) bool) Subscription {
	app.checkUI()
	return app.observe(entity, onNotify)
}

// subscribe is the unexported inner of Subscribe.
func (app *App) subscribe(entity anyEntity, eventType reflect.Type, onEvent func(cx *App, event any) bool) Subscription {
	sub, activate := app.subscribers.insert(entity.id, subscriberEntry{eventType: eventType, handler: onEvent})
	app.deferFn(func(*App) { activate() })
	return sub
}

// Subscribe registers a callback for typed events emitted by entity. The
// registration is activated at the end of the current flush.
func (app *App) Subscribe(entity anyEntity, eventType reflect.Type, onEvent func(cx *App, event any) bool) Subscription {
	app.checkUI()
	return app.subscribe(entity, eventType, onEvent)
}

// emit is the unexported inner of Emit.
func (app *App) emit(id entityID, event any, eventType reflect.Type) {
	app.pushEffect(emitEffect{emitter: id, eventType: eventType, event: event})
}

// Emit queues delivery of event to the entity's subscribers of its type. The
// event is delivered during the flush, after the update returns.
func (app *App) Emit(id entityID, event any, eventType reflect.Type) {
	app.checkUI()
	app.emit(id, event, eventType)
}

// deferFn is the unexported inner of Defer.
func (app *App) deferFn(f func(*App)) {
	app.pushEffect(deferEffect{callback: f})
}

// Defer schedules f to run at the end of the current flush. It is how the
// framework activates subscriptions registered during an update, and it is
// available to user code that needs to run after entities on the stack have
// been returned to the map.
func (app *App) Defer(f func(*App)) {
	app.checkUI()
	app.deferFn(f)
}

// onRelease is the unexported inner of OnRelease.
func (app *App) onRelease(entity anyEntity, onRelease func(value any, app *App)) Subscription {
	sub, activate := app.releases.insert(entity.id, releaseHandler(onRelease))
	app.deferFn(func(*App) { activate() })
	return sub
}

// OnRelease registers a callback to run when the entity is dropped. The
// callback receives the entity's value and the App, and is the place to
// Release handles the entity owns and to Close subscriptions it holds. The
// registration is activated at the end of the current flush.
func (app *App) OnRelease(entity anyEntity, onRelease func(value any, app *App)) Subscription {
	app.checkUI()
	return app.onRelease(entity, onRelease)
}

// anyEntity is the type-erased view of a handle that the App needs to register
// observers, subscribers and release callbacks: just an id into the map. It is
// constructed from a typed Entity through the entityAny helper.
type anyEntity struct {
	id entityID
}

func entityAny[T any](e Entity[T]) anyEntity { return anyEntity{id: e.id} }
