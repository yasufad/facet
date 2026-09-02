package element

import (
	"testing"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/input"
)

type listenerCounter struct {
	count int
}

// TestListenerReachesEntityState reproduces the shape of examples/button: a
// handler registered during Render, stored the way window stores it, and
// invoked later from outside any update. Before Listener existed this
// panicked on Notify and silently discarded the write; see docs/audit.md.
func TestListenerReachesEntityState(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	entity := app.New(a, func(cx *app.Context[listenerCounter]) listenerCounter {
		return listenerCounter{}
	})
	defer entity.Release()

	var handler func(ClickEvent) bool
	entity.Update(a, func(v *listenerCounter, cx *app.Context[listenerCounter]) {
		handler = Listener(cx, func(v *listenerCounter, e ClickEvent, cx *app.Context[listenerCounter]) bool {
			v.count++
			cx.Notify()
			return true
		})
	})

	if !handler(ClickEvent{}) {
		t.Fatalf("expected listener to report the event handled")
	}
	if !handler(ClickEvent{}) {
		t.Fatalf("expected listener to report the event handled on a second dispatch")
	}

	if got := entity.Read(a).count; got != 2 {
		t.Fatalf("count = %d, want 2: listener's write did not reach entity state", got)
	}
}

// TestListenerOnDroppedEntityIsUnhandled confirms a listener whose view has
// been released does not panic; it reports the event unhandled, matching
// GPUI's Context::listener.
func TestListenerOnDroppedEntityIsUnhandled(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	entity := app.New(a, func(cx *app.Context[listenerCounter]) listenerCounter {
		return listenerCounter{}
	})

	var handler func(ClickEvent) bool
	entity.Update(a, func(v *listenerCounter, cx *app.Context[listenerCounter]) {
		handler = Listener(cx, func(v *listenerCounter, e ClickEvent, cx *app.Context[listenerCounter]) bool {
			v.count++
			return true
		})
	})

	entity.Release()

	if handler(ClickEvent{}) {
		t.Fatalf("expected listener on a dropped entity to report unhandled")
	}
}

// TestPhasedListenerReachesEntityState mirrors TestListenerReachesEntityState
// for the key/pointer/wheel handler shape, and checks the adapter's return
// type is assignable to input.KeyEventHandler with no conversion, since that
// is what lets it sit in Div's listener slices unchanged.
func TestPhasedListenerReachesEntityState(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	entity := app.New(a, func(cx *app.Context[listenerCounter]) listenerCounter {
		return listenerCounter{}
	})
	defer entity.Release()

	var handler input.KeyEventHandler
	entity.Update(a, func(v *listenerCounter, cx *app.Context[listenerCounter]) {
		handler = PhasedListener(cx, func(v *listenerCounter, e input.KeyEvent, phase input.DispatchPhase, cx *app.Context[listenerCounter]) bool {
			if phase != input.Bubble {
				return false
			}
			v.count++
			cx.Notify()
			return true
		})
	})

	if handler(input.KeyEvent{}, input.Capture) {
		t.Fatalf("expected the capture-phase dispatch to report unhandled")
	}
	if !handler(input.KeyEvent{}, input.Bubble) {
		t.Fatalf("expected the bubble-phase dispatch to report handled")
	}

	if got := entity.Read(a).count; got != 1 {
		t.Fatalf("count = %d, want 1: phased listener's write did not reach entity state", got)
	}
}
