// Package app is the reactive core of Facet.
//
// It owns the entity map, the contexts that mediate access to entity state, the
// effect queue that batches side effects, and the foreground and background
// executors. Everything above this package assumes this one is correct.
//
// # Entities
//
// State lives in entities. An Entity[T] is a handle — an identifier into a map
// the App owns — not a pointer to the value. Reads and writes go through a
// context, which is what makes mutation observable. Handles are cheap to copy
// and compare.
//
// A handle is reference-counted. New returns an owning handle (count one).
// Clone produces an additional owning handle for a long-lived holder; Release
// drops one ownership. When the strong count reaches zero the entity is queued
// for release and dropped during the next effect flush, after its release
// observers have run. WeakEntity[T] observes without keeping the value alive:
// observers must not resurrect what they watch, so the framework holds weak
// handles to observers.
//
// Go has no RAII, so dropping is explicit. The framework Releases the handles
// it owns — subscriptions are cancelled when their observer is dropped, and
// release callbacks cascade drops. Code that stores a strong handle in a
// long-lived structure is responsible for Releasing it, typically in an
// OnRelease callback.
//
// Cycles between entities holding strong handles leak: neither count reaches
// zero, so neither is dropped. Break such cycles with WeakEntity. There is no
// cycle collector; the cost of one is not worth paying on the per-frame path.
//
// # Contexts
//
// App is the global surface: it owns every entity's data and reaches the
// effect queue and executors. Context[T] is for working on one entity; it
// dereferences to App and adds Notify, Emit, Observe and Subscribe scoped to
// that entity. AsyncApp is the only context that may cross an await point: it
// holds a weak reference to the App and marshals every entity operation back to
// the UI goroutine, so calls become fallible.
//
// # Reactivity
//
// Notify marks an entity dirty and schedules its observers. Observe runs when
// another entity notifies. Subscribe receives typed events an entity emits via
// Emit. Effects queue during an update and flush once at its end, so a burst of
// mutations produces one frame rather than one each. Notify is deduplicated per
// entity per flush: a hundred notifications collapse to a single observer run.
// Effects raised while flushing are appended to the queue and processed in
// order, so an effect may cause another and the flush only ends when the queue
// is empty.
//
// # Threading
//
// The UI runs on one goroutine. A context used from another goroutine panics
// with a message naming the mistake rather than corrupting state quietly.
// Two mechanisms enforce the invariant, each placed where it is cheap:
//
//   - checkUI (goroutine identity, ~6µs) runs at update boundaries and on
//     every exported App method. A frame opens a few update boundaries, so a
//     few 6µs checks per frame is affordable. Direct calls to App methods
//     from a background goroutine are caught here.
//   - A generation counter (integer compare, ~1ns) runs at every Context
//     accessor. A Context records the generation at creation; after its
//     update ends, the generation has moved on and any escaped context
//     panics. This catches the realistic mistake — a context stored and used
//     after its update — at every accessor, cheaply.
//
// The entity map is single-goroutine by design and has no mutex; adding one
// to make it safe from other goroutines would remove the reason the rest of
// the design works. Background work returns to the foreground to touch state
// through AsyncApp, which dispatches onto the UI goroutine.
//
// # Scope
//
// app knows nothing about drawing. No geometry, no colours, no elements. A
// view is an entity whose notification schedules a repaint, but the repaint
// belongs to window and the render method belongs to element. app provides the
// mechanism and stops there.
package app
