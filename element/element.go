package element

import (
	"fmt"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/layout"
)

// Element is implemented by types that participate in laying out, hit testing,
// and painting contents in a frame.
//
// Elements form an ephemeral tree rebuilt each frame. An element is a value
// built fresh and discarded after paint. State that survives the frame belongs
// in an entity in app.
//
// The three phases execute strictly in order: RequestLayout, Prepaint, and Paint.
// No phase may reach backwards or skip preceding steps.
type Element interface {
	// RequestLayout adds layout nodes to the layout tree via Frame and returns
	// the root layout identifier for this element.
	RequestLayout(f Frame) layout.NodeID

	// Prepaint commits the element's solved bounds to the frame, registers hit
	// regions, and prepares geometry for painting.
	Prepaint(f Frame, bounds geometry.Bounds[geometry.Pixels])

	// Paint emits drawing primitives (such as quads, shadows, and text) into the
	// frame's scene.
	Paint(f Frame, bounds geometry.Bounds[geometry.Pixels])
}

// Render is implemented by entity state types that produce an element tree.
//
// A view is an entity whose Render method turns retained state into an
// ephemeral element hierarchy. app owns entities and knows nothing about drawing;
// Render is the interface that bridges reactive entity state to UI elements.
type Render[T any] interface {
	Render(cx *app.Context[T]) Element
}

// AnyView is the type-erased interface representing a renderable view entity.
//
// Window uses AnyView to hold and drive arbitrary root views without needing a
// type parameter for each entity value type.
type AnyView interface {
	Render(a *app.App) Element
}

// View wraps an app.Entity[T] whose type or pointer implements Render[T].
type View[T any] struct {
	entity app.Entity[T]
}

// NewView creates a typed View wrapper around an entity.
func NewView[T any](entity app.Entity[T]) View[T] {
	return View[T]{entity: entity}
}

// Entity returns the underlying typed entity handle.
func (v View[T]) Entity() app.Entity[T] {
	return v.entity
}

// Render executes the view's Render method against its entity state and returns
// the produced Element tree.
func (v View[T]) Render(a *app.App) Element {
	var el Element
	v.entity.Update(a, func(val *T, cx *app.Context[T]) {
		if r, ok := any(val).(Render[T]); ok {
			el = r.Render(cx)
		} else if r, ok := any(*val).(Render[T]); ok {
			el = r.Render(cx)
		} else {
			panic(fmt.Sprintf("element: entity type %T does not implement Render[%T]", val, val))
		}
	})
	return el
}
