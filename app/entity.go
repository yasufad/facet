package app

import (
	"fmt"
	"sync"
)

// entityID identifies one entity within an App. It is an opaque, comparable
// value: handles carry it, the entity map keys by it, and observers and
// subscribers are registered against it. It carries no type information.
type entityID uint64

// refCounts is the shared reference-count table for one App. Both the App and
// every handle to its entities hold the same *refCounts, so a handle can
// increment and decrement its entity's strong count without reaching back into
// the App.
//
// It is touched only on the UI goroutine. Strong handles are never held across
// goroutines; background work that needs to touch an entity goes through
// AsyncApp, which marshals onto the UI goroutine. The mutex below guards only
// the shutdown transition, which may be observed from a finaliser-dispatched
// release; every other access is single-goroutine and unsynchronised.
type refCounts struct {
	counts  map[entityID]int
	dropped []entityID // entities whose strong count reached zero, awaiting a flush

	mu       sync.Mutex
	shutdown bool
}

func newRefCounts() *refCounts {
	return &refCounts{counts: make(map[entityID]int)}
}

// strong increments the strong count for id and returns the new count. It is a
// programmer error (a double release) to increment from zero after the entity
// was queued for drop.
func (r *refCounts) strong(id entityID) {
	r.counts[id]++
}

// release decrements the strong count for id. When it reaches zero the id is
// queued on dropped for the next effect flush to reap. After shutdown the
// count is still tracked but no drop is queued: the App and its entity map are
// gone, so there is nothing to reap.
func (r *refCounts) release(id entityID) {
	c, ok := r.counts[id]
	if !ok || c <= 0 {
		panic(fmt.Sprintf("app: over-release of entity %d", id))
	}
	c--
	r.counts[id] = c
	if c == 0 && !r.isShutdown() {
		r.dropped = append(r.dropped, id)
	}
}

// upgrade attempts to turn a weak reference into a strong one. It returns false
// if the entity has no live strong handles. The increment is a single
// check-then-set under the single-goroutine invariant.
func (r *refCounts) upgrade(id entityID) bool {
	if r.counts[id] <= 0 {
		return false
	}
	r.counts[id]++
	return true
}

// markShutdown prevents further drop queuing. Called once when the App is
// destroyed, so any handle released afterwards (for example by a finaliser
// running late) does not touch a reaped entity map.
func (r *refCounts) markShutdown() {
	r.mu.Lock()
	r.shutdown = true
	r.mu.Unlock()
}

// isShutdown reports whether the App has been closed. Unlike every other
// field on refCounts, shutdown is set from a background goroutine (a
// finaliser-dispatched release can run after Close) and read from one too
// (AsyncApp.run and AsyncApp.Update, checking whether to touch state at all)
// — the one field on this type actually reached from off the UI goroutine, so
// it is the one that needs the mutex on every access rather than none.
func (r *refCounts) isShutdown() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.shutdown
}

// Entity is a strong, typed handle to state owned by an App.
//
// It is an identifier into the App's entity map, not a pointer: reads and
// writes go through a context. The handle is cheap to copy and compare. A copy
// is a transient borrow — it does not add a strong reference. Call Clone to
// create an additional owning handle for a long-lived holder, and pair every
// Clone (and the owning handle returned by New) with a Release.
//
// The zero value is not a valid handle; methods on it panic.
type Entity[T any] struct {
	id entityID
	rc *refCounts
}

// EntityID returns the identifier backing this handle. It is stable for the
// lifetime of the entity and equal across all strong and weak handles to it.
func (e Entity[T]) EntityID() entityID { return e.id }

// AnyEntity returns the type-erased handle for this entity.
func (e Entity[T]) AnyEntity() AnyEntity {
	e.mustValid()
	return AnyEntity{id: e.id}
}

// Clone returns an owning handle to the same entity, incrementing its strong
// count. Use it whenever a handle is stored in a long-lived structure; pair it
// with a Release when that structure is done.
func (e Entity[T]) Clone() Entity[T] {
	e.mustValid()
	e.rc.strong(e.id)
	return e
}

// Release drops one ownership of the entity. When the last strong handle is
// released the entity is queued for release and dropped during the next effect
// flush, after its release observers have run.
func (e Entity[T]) Release() {
	e.mustValid()
	e.rc.release(e.id)
}

// Downgrade returns a weak handle to the same entity. A weak handle does not
// keep the entity alive and is what observers hold, so that watching an entity
// does not resurrect it.
func (e Entity[T]) Downgrade() WeakEntity[T] {
	e.mustValid()
	return WeakEntity[T]{id: e.id, rc: e.rc}
}

// Read returns a pointer to the entity's value through the given App. It panics
// if the entity is being updated (a re-entrant update) or has been dropped.
func (e Entity[T]) Read(cx *App) *T {
	e.mustValid()
	return ReadEntity(cx, e)
}

// Update runs f against the entity's value, holding a Context for it. Effects
// raised inside f (and any it causes) flush once f returns. It panics on a
// re-entrant update of the same entity.
func (e Entity[T]) Update(cx *App, f func(v *T, cx *Context[T])) {
	e.mustValid()
	UpdateEntity(cx, e, f)
}

func (e Entity[T]) mustValid() {
	if e.rc == nil {
		panic("app: use of a zero-value Entity")
	}
}

// WeakEntity is a weak, typed handle to an entity. It does not keep the entity
// alive: Upgrade returns false once the last strong handle has been released
// and the entity has been dropped. Observers hold weak handles so that
// watching an entity cannot resurrect it.
type WeakEntity[T any] struct {
	id entityID
	rc *refCounts
}

// EntityID returns the identifier backing this weak handle.
func (w WeakEntity[T]) EntityID() entityID { return w.id }

// AnyEntity returns the type-erased handle for this weak entity.
func (w WeakEntity[T]) AnyEntity() AnyEntity {
	return AnyEntity{id: w.id}
}

// Upgrade attempts to turn this weak handle into a strong one. It returns the
// strong handle and true if the entity is still alive, or the zero WeakEntity
// and false if it has been dropped.
//
// Upgrade touches the reference-count table and so must run on the UI
// goroutine. Code on another goroutine must reach the entity through AsyncApp,
// which marshals onto the UI goroutine.
func (w WeakEntity[T]) Upgrade() (Entity[T], bool) {
	if w.rc == nil {
		return Entity[T]{}, false
	}
	if !w.rc.upgrade(w.id) {
		return Entity[T]{}, false
	}
	return Entity[T]{id: w.id, rc: w.rc}, true
}
