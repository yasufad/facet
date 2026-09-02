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

// TestListenerAdapterTypesFitDispatchSlots is the argument for these adapters'
// existence made as a compile-time assertion rather than prose: PhasedListener
// drops into DispatchNode's handler slices and Listener into Div.OnClick with
// no conversion and no signature change anywhere else in the tree. Every other
// test in this file calls the returned closure directly, which checks the
// adapter's behaviour but never checks that it fits the slot it was built for.
func TestListenerAdapterTypesFitDispatchSlots(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	entity := app.New(a, func(cx *app.Context[listenerCounter]) listenerCounter {
		return listenerCounter{}
	})
	defer entity.Release()

	entity.Update(a, func(v *listenerCounter, cx *app.Context[listenerCounter]) {
		var _ input.KeyEventHandler = PhasedListener(cx, func(v *listenerCounter, e input.KeyEvent, phase input.DispatchPhase, cx *app.Context[listenerCounter]) bool {
			return false
		})
		var _ input.PointerEventHandler = PhasedListener(cx, func(v *listenerCounter, e input.PointerEvent, phase input.DispatchPhase, cx *app.Context[listenerCounter]) bool {
			return false
		})
		var _ input.WheelEventHandler = PhasedListener(cx, func(v *listenerCounter, e input.WheelEvent, phase input.DispatchPhase, cx *app.Context[listenerCounter]) bool {
			return false
		})
		NewDiv().OnClick(Listener(cx, func(v *listenerCounter, e ClickEvent, cx *app.Context[listenerCounter]) bool {
			return false
		}))
	})
}

// TestListenerReleasesUpgradedHandle pins the balance docs/audit.md names as
// the framework's central lifetime hazard: Listener upgrades a weak handle on
// every dispatch, and a missed Release on that upgrade leaks the entity even
// after its owner releases its own handle. Firing the listener many times and
// then dropping the owning handle must leave the entity fully dropped — a
// leaked upgrade would keep one strong count outstanding and Upgrade would
// still succeed.
func TestListenerReleasesUpgradedHandle(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	entity := app.New(a, func(cx *app.Context[listenerCounter]) listenerCounter {
		return listenerCounter{}
	})

	var weak app.WeakEntity[listenerCounter]
	var handler func(ClickEvent) bool
	entity.Update(a, func(v *listenerCounter, cx *app.Context[listenerCounter]) {
		weak = cx.WeakEntity()
		handler = Listener(cx, func(v *listenerCounter, e ClickEvent, cx *app.Context[listenerCounter]) bool {
			v.count++
			return true
		})
	})

	for range 5 {
		handler(ClickEvent{})
	}

	entity.Release()

	if _, ok := weak.Upgrade(); ok {
		t.Fatalf("entity still upgrades after its owning handle was released: a dispatch left a strong reference behind")
	}
}

// TestPhasedListenerReleasesUpgradedHandle is
// TestListenerReleasesUpgradedHandle for PhasedListener, since it upgrades
// and releases independently of Listener.
func TestPhasedListenerReleasesUpgradedHandle(t *testing.T) {
	a := app.NewApp()
	defer a.Close()

	entity := app.New(a, func(cx *app.Context[listenerCounter]) listenerCounter {
		return listenerCounter{}
	})

	var weak app.WeakEntity[listenerCounter]
	var handler input.KeyEventHandler
	entity.Update(a, func(v *listenerCounter, cx *app.Context[listenerCounter]) {
		weak = cx.WeakEntity()
		handler = PhasedListener(cx, func(v *listenerCounter, e input.KeyEvent, phase input.DispatchPhase, cx *app.Context[listenerCounter]) bool {
			v.count++
			return true
		})
	})

	for range 5 {
		handler(input.KeyEvent{}, input.Bubble)
	}

	entity.Release()

	if _, ok := weak.Upgrade(); ok {
		t.Fatalf("entity still upgrades after its owning handle was released: a dispatch left a strong reference behind")
	}
}
