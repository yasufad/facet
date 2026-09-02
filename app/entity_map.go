package app

import (
	"fmt"
)

// entityMap is the App's storage of entity values, keyed by entityID. Each
// stored value is a *T rather than a T: New boxes the value once at
// construction, and every read or update aliases that same pointer rather
// than copying in or out of it, so a write through a leased or read pointer
// is visible everywhere else the entity is reached. It is single-goroutine by
// design and carries no locking.
//
// The lease mechanism moves a value out of the map for the duration of a
// mutable update, so a re-entrant update of the same entity finds an empty
// slot and panics rather than aliasing the value. Reads find the value in
// place; a read of an entity currently on lease panics for the same reason.
type entityMap struct {
	values map[entityID]any
	nextID entityID
}

func newEntityMap() *entityMap {
	return &entityMap{values: make(map[entityID]any)}
}

// reserveID allocates an entityID and seeds a strong count of one for the
// eventual owning handle. The value is inserted separately by insert, so a
// constructor that creates further entities cannot observe a half-built one.
func (m *entityMap) reserveID(rc *refCounts) entityID {
	m.nextID++
	id := m.nextID
	rc.counts[id] = 1
	return id
}

// insert stores the value for id. Called after the constructor has run, so the
// value is fully formed before any other handle can reach it.
func (m *entityMap) insert(id entityID, value any) {
	m.values[id] = value
}

// read returns the value for id as T. It panics if the entity is currently on
// lease (being updated) or has been dropped.
func (m *entityMap) read(id entityID) any {
	v, ok := m.values[id]
	if !ok {
		panic(fmt.Sprintf("app: entity %d is being updated or has been dropped", id))
	}
	return v
}

// lease removes the value for id from the map and returns it, so a re-entrant
// update or read of the same entity finds an empty slot and panics. endLease
// restores it.
func (m *entityMap) lease(id entityID) any {
	v, ok := m.values[id]
	if !ok {
		panic(fmt.Sprintf("app: cannot update entity %d while it is already being updated", id))
	}
	delete(m.values, id)
	return v
}

// endLease restores a value leased out by lease.
func (m *entityMap) endLease(id entityID, value any) {
	m.values[id] = value
}

// takeDropped returns the ids queued for drop and removes their counts. The
// values themselves are returned by remove.
func (m *entityMap) takeDropped(rc *refCounts) []entityID {
	if len(rc.dropped) == 0 {
		return nil
	}
	dropped := rc.dropped
	rc.dropped = nil
	return dropped
}

// remove drops the value for id from the map. Called during the effect flush
// for each id whose strong count reached zero, after release observers have
// run. It returns the value so the caller can run release callbacks against it.
func (m *entityMap) remove(id entityID) any {
	v := m.values[id]
	delete(m.values, id)
	return v
}
