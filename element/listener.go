package element

import (
	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/input"
)

// Listener adapts a handler that needs its view's state into the plain
// callback shape the element tree stores and window dispatches.
//
// Render runs inside app.UpdateEntity, so a handler written the ordinary Go
// way — capturing the *T and *app.Context[T] that Render was given — captures
// a pointer to a leased copy and a context whose generation expires the
// moment Render returns. The closure is invoked later, from
// window.DispatchEvent, outside any update, so a captured Context panics on
// use and a captured pointer's writes land on a value the entity map has
// already superseded.
//
// Listener closes over a weak handle to the view's entity instead of the
// entity itself. At dispatch time it upgrades the handle and re-enters
// app.UpdateEntity, so f runs against live state and a live Context. If the
// view has been dropped by the time the event fires, the handle no longer
// upgrades and the event is reported unhandled rather than panicking.
//
// This is a mitigation at the one place Render's update boundary bites
// callbacks, not a general escape from it — see docs/audit.md.
func Listener[T, E any](cx *app.Context[T], f func(v *T, e E, cx *app.Context[T]) bool) func(E) bool {
	weak := cx.WeakEntity()
	a := cx.App()
	return func(e E) bool {
		entity, ok := weak.Upgrade()
		if !ok {
			return false
		}
		defer entity.Release()
		handled := false
		app.UpdateEntity(a, entity, func(v *T, icx *app.Context[T]) {
			handled = f(v, e, icx)
		})
		return handled
	}
}

// PhasedListener is Listener for the handlers that carry an
// input.DispatchPhase alongside their event, such as key, pointer and wheel
// listeners.
func PhasedListener[T, E any](cx *app.Context[T], f func(v *T, e E, phase input.DispatchPhase, cx *app.Context[T]) bool) func(E, input.DispatchPhase) bool {
	weak := cx.WeakEntity()
	a := cx.App()
	return func(e E, phase input.DispatchPhase) bool {
		entity, ok := weak.Upgrade()
		if !ok {
			return false
		}
		defer entity.Release()
		handled := false
		app.UpdateEntity(a, entity, func(v *T, icx *app.Context[T]) {
			handled = f(v, e, phase, icx)
		})
		return handled
	}
}
