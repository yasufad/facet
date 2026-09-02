package app

import (
	"fmt"
	"reflect"
)

// Context is the context for working on one entity. It dereferences to the
// App (every App method is reachable through it) and adds the operations
// scoped to the entity: Notify, Emit, Observe, Subscribe and OnRelease.
//
// A Context only exists for the duration of the update or constructor that
// produced it, and only on the UI goroutine. To react to changes from outside
// an update — or to reach entity state from a background task — use the
// AsyncApp returned by App.Async or the Spawn function.
//
// The typed operations that introduce a second type parameter (Observe,
// Subscribe, Emit, Spawn) are top-level functions rather than methods, because
// Go does not permit methods to declare type parameters of their own. They
// take the Context as their first argument.
type Context[T any] struct {
	app        *App
	self       WeakEntity[T]
	generation int64
}

// App returns the underlying App. Every App method is reachable through a
// Context.
func (c *Context[T]) App() *App { return c.app }

// Entity returns a strong handle to the entity this context is for. The
// context holds only a weak handle, so this upgrades; it panics if the entity
// has been dropped, which can only happen through a programming error during
// construction or release.
func (c *Context[T]) Entity() Entity[T] {
	e, ok := c.self.Upgrade()
	if !ok {
		panic("app: context used after its entity was dropped")
	}
	// Upgrade added a strong reference; the caller now owns it.
	return e
}

// WeakEntity returns a weak handle to the entity this context is for.
func (c *Context[T]) WeakEntity() WeakEntity[T] { return c.self }

// EntityID returns the id of the entity this context is for.
func (c *Context[T]) EntityID() entityID { return c.self.id }

// Notify marks this entity dirty and schedules its observers.
func (c *Context[T]) Notify() {
	c.checkGeneration()
	c.app.notify(c.self.id)
}

// OnRelease registers a callback to run when this entity is dropped. It is the
// place to Release handles the entity owns and to Close subscriptions it
// holds. The callback receives a pointer to the value and the App.
func (c *Context[T]) OnRelease(onRelease func(v *T, app *App)) Subscription {
	c.checkGeneration()
	id := c.self.id
	return c.app.onRelease(AnyEntity{id: id}, func(value any, app *App) {
		t, ok := value.(*T)
		if !ok {
			// The entity map holds something other than what this entity's
			// type says it holds: a programmer error with no recovery, the
			// same case ReadEntity and UpdateEntity panic on.
			panic(fmt.Sprintf("app: entity %d is not of type %T", id, *new(T)))
		}
		onRelease(t, app)
	})
}

// Defer schedules f to run at the end of the current flush, with a Context for
// this entity. Use it to run after entities currently on the stack have been
// returned to the map.
func (c *Context[T]) Defer(f func(v *T, cx *Context[T])) {
	c.checkGeneration()
	self := c.self
	c.app.deferFn(func(app *App) {
		s, ok := self.Upgrade()
		if !ok {
			return
		}
		UpdateEntity(app, s, func(v *T, cx *Context[T]) {
			f(v, cx)
		})
		s.Release()
	})
}

// Emit emits a typed event from the entity cx is for, delivered to its
// subscribers during the flush.
//
// It is a top-level function because Go does not permit methods to declare
// type parameters.
func Emit[T any, Evt any](cx *Context[T], event Evt) {
	cx.checkGeneration()
	cx.app.emit(cx.self.id, &event, reflect.TypeOf((*Evt)(nil)))
}

// Observe registers a callback to run when entity notifies. The callback
// receives the entity cx is for (upgraded through the weak handle cx holds, so
// observing does not keep the observer alive), a handle to the entity that
// notified, and a Context for the observer.
//
// The registration is activated at the end of the current flush, so an
// observer registered during a notification does not fire for that very
// notification. When the observer entity is dropped, its weak handle no longer
// upgrades and the registration removes itself.
//
// It is a top-level function because Go does not permit methods to declare
// type parameters.
func Observe[T any, W any](cx *Context[T], entity Entity[W], onNotify func(v *T, e Entity[W], cx *Context[T])) Subscription {
	cx.checkGeneration()
	observer := cx.self
	observedWeak := entity.Downgrade()
	observedID := entity.id
	return cx.app.observe(AnyEntity{id: observedID}, func(app *App) bool {
		obs, ok := observedWeak.Upgrade()
		if !ok {
			return false
		}
		self, ok := observer.Upgrade()
		if !ok {
			obs.Release()
			return false
		}
		UpdateEntity(app, self, func(v *T, icx *Context[T]) {
			onNotify(v, obs, icx)
		})
		// The handles passed to the callback were borrows: release the
		// upgrades that produced them. Callers that need to keep a handle
		// must Clone it inside the callback.
		self.Release()
		obs.Release()
		return true
	})
}

// Subscribe registers a callback for typed events emitted by entity. The
// callback receives the entity cx is for, a handle to the emitter, the event,
// and a Context for the subscriber. As with Observe, the registration is
// activated at the end of the flush and removes itself when the subscriber is
// dropped.
//
// It is a top-level function because Go does not permit methods to declare
// type parameters.
func Subscribe[T any, E any, Evt any](cx *Context[T], entity Entity[E], onEvent func(v *T, e Entity[E], evt *Evt, cx *Context[T])) Subscription {
	cx.checkGeneration()
	observer := cx.self
	emitterWeak := entity.Downgrade()
	emitterID := entity.id
	return cx.app.subscribe(AnyEntity{id: emitterID}, reflect.TypeOf((*Evt)(nil)), func(app *App, event any) bool {
		em, ok := emitterWeak.Upgrade()
		if !ok {
			return false
		}
		self, ok := observer.Upgrade()
		if !ok {
			em.Release()
			return false
		}
		evt := event.(*Evt)
		UpdateEntity(app, self, func(v *T, icx *Context[T]) {
			onEvent(v, em, evt, icx)
		})
		self.Release()
		em.Release()
		return true
	})
}

// Spawn runs f on the foreground executor with an AsyncApp and a weak handle
// to the entity cx is for. The task runs on the UI goroutine; blocking work
// should use the background executor inside f instead. The returned Task must
// be Awaited (off the UI goroutine) or Detached.
//
// It is a top-level function because Go does not permit methods to declare
// type parameters.
func Spawn[T any, R any](cx *Context[T], f func(self WeakEntity[T], async *AsyncApp) R) Task[R] {
	self := cx.self
	async := cx.app.Async()
	return fgSpawn(cx.app.fg, func() R {
		return f(self, async)
	})
}
