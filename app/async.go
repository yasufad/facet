package app

import (
	"context"
	"fmt"
	"reflect"
)

// AsyncApp is the only context that may cross an await point. It holds the App
// and the executors, and marshals every entity operation back onto the UI
// goroutine through the foreground executor. Calls are fallible: if the App
// has been shut down, they return an error rather than touching freed state.
//
// AsyncApp does not itself check the UI goroutine, because it is meant to be
// used from background tasks. Its methods either dispatch onto the UI
// goroutine (Update, Observe, Subscribe) or are pure executor access
// (Foreground, Background). The typed operations that introduce a type
// parameter (Read, UpdateEntity, UpdateWeak, ReadEntity, Spawn,
// BackgroundSpawn) are top-level functions, because Go does not permit methods
// to declare type parameters.
type AsyncApp struct {
	app *App
}

// Foreground returns the foreground executor.
func (a *AsyncApp) Foreground() *ForegroundExecutor { return a.app.fg }

// Background returns the background executor.
func (a *AsyncApp) Background() *BackgroundExecutor { return a.app.bg }

// run runs f against the App. If the caller is already on the UI goroutine —
// for example a foreground continuation dispatched by Then — f runs inline.
// Otherwise f is marshalled onto the UI goroutine and run blocks until it has
// completed. This is the seam that keeps the single-goroutine invariant:
// background tasks touch entity state only through closures that run on the UI
// goroutine, while foreground continuations do not re-enter the dispatcher.
func (a *AsyncApp) run(f func() error) error {
	if goroutineID() == a.app.uiGoroutine {
		return f()
	}
	res := make(chan error, 1)
	a.app.fg.enqueue(func() { res <- f() })
	return <-res
}

// Update runs f against the App on the UI goroutine, inside an update
// boundary, and returns once it has completed. Effects raised inside f flush
// before Update returns. It is the way a background task touches state.
func (a *AsyncApp) Update(f func(app *App)) error {
	return a.run(func() error {
		if a.app.rc.shutdown {
			return fmt.Errorf("app: application has been shut down")
		}
		a.app.update(f)
		return nil
	})
}

// Observe registers a notify callback on the UI goroutine. The subscription is
// activated at the end of the next flush.
func (a *AsyncApp) Observe(entity AnyEntity, onNotify func(app *App) bool) (Subscription, error) {
	var sub Subscription
	err := a.Update(func(app *App) {
		sub = app.Observe(entity, onNotify)
	})
	return sub, err
}

// Subscribe registers a typed-event callback on the UI goroutine.
func (a *AsyncApp) Subscribe(entity AnyEntity, eventType reflect.Type, onEvent func(app *App, event any) bool) (Subscription, error) {
	var sub Subscription
	err := a.Update(func(app *App) {
		sub = app.Subscribe(entity, eventType, onEvent)
	})
	return sub, err
}

// AsyncRead runs f against the App on the UI goroutine and returns its result.
// Reads do not open an update boundary; use Update for mutations.
func AsyncRead[R any](a *AsyncApp, f func(app *App) R) (R, error) {
	var out R
	err := a.run(func() error {
		if a.app.rc.shutdown {
			return fmt.Errorf("app: application has been shut down")
		}
		a.app.checkUI()
		out = f(a.app)
		return nil
	})
	return out, err
}

// AsyncUpdateEntity runs f against the entity on the UI goroutine, inside an
// update boundary. It is the fallible counterpart of Entity.Update, for use
// from background tasks.
func AsyncUpdateEntity[T any](a *AsyncApp, handle Entity[T], f func(v *T, cx *Context[T])) error {
	return a.Update(func(app *App) {
		UpdateEntity(app, handle, f)
	})
}

// AsyncUpdateWeak runs f against the entity on the UI goroutine if it is still
// alive. It upgrades a weak handle on the UI goroutine and reports false if the
// entity has been dropped.
func AsyncUpdateWeak[T any](a *AsyncApp, weak WeakEntity[T], f func(v *T, cx *Context[T])) (bool, error) {
	var ok bool
	err := a.Update(func(app *App) {
		handle, up := weak.Upgrade()
		if !up {
			return
		}
		UpdateEntity(app, handle, func(v *T, cx *Context[T]) {
			f(v, cx)
		})
		handle.Release()
		ok = true
	})
	return ok, err
}

// AsyncReadEntity returns a pointer to the entity's value, read on the UI
// goroutine. The pointer is only valid until the next update; do not retain
// it across an await.
func AsyncReadEntity[T any](a *AsyncApp, handle Entity[T]) (*T, error) {
	return AsyncRead(a, func(app *App) *T {
		return ReadEntity(app, handle)
	})
}

// AsyncSpawn runs f on the foreground executor with an AsyncApp. The task runs
// on the UI goroutine; blocking work should use the background executor inside
// f.
func AsyncSpawn[R any](a *AsyncApp, f func(async *AsyncApp) R) Task[R] {
	async := &AsyncApp{app: a.app}
	return fgSpawn(a.app.fg, func() R {
		return f(async)
	})
}

// AsyncBackgroundSpawn runs f on the background executor with an AsyncApp and a
// context that is cancelled when the returned Task is cancelled. f reaches
// entity state through async, which marshals onto the UI goroutine.
func AsyncBackgroundSpawn[R any](a *AsyncApp, f func(ctx context.Context, async *AsyncApp) R) Task[R] {
	async := &AsyncApp{app: a.app}
	return BackgroundSpawn(a.app.bg, func(ctx context.Context) R {
		return f(ctx, async)
	})
}
