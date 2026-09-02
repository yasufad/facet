package app

import (
	"runtime"
	"sync"
)

// goroutineID returns a number identifying the calling goroutine.
//
// Go does not expose goroutine identity in the public API, so this reads the
// header that runtime.Stack writes for the current goroutine only
// ("goroutine N [..."). The buffer is pooled so the check allocates nothing on
// the steady-state path. The number is stable for the lifetime of the
// goroutine and unique among live goroutines, which is all the UI-goroutine
// check needs.
//
// The buffer is small: runtime.Stack fills it with the header line and
// truncates the rest, so the stack is not walked in full. Even so, the call
// costs about 6µs (measured 6167 ns/op on an Intel Core Ultra 5 125U). That
// is too expensive for the per-frame path, where a thousand entities may be
// touched — every UpdateEntity opens an update boundary, so a 6µs check
// there alone is most of a 60Hz frame budget at that scale. Two mechanisms
// share the work, each placed where its cost fits the build it runs in:
//
//   - App.checkUI (app_debug.go) runs at update boundaries and on exported
//     App methods, but only in a facet_debug build. It catches a direct call
//     to an App method, or a Context accessor still inside its update, from a
//     goroutine other than the UI goroutine. In a release build (app_check.go)
//     it is a no-op: the 6µs cost is acceptable in a debug or race build,
//     where it does not matter, but not on the per-frame release path.
//   - A generation counter (Context.checkGeneration, an integer compare at
//     ~1ns) runs at every Context accessor in every build. It catches a
//     context stored and used after its update has ended — cheaply enough
//     to stay on in release, where checkUI cannot.
//
// A release build therefore has no protection against a direct call to an
// App method, or a live Context, from the wrong goroutine while an update is
// still in flight; only escaping a context past its own update is caught.
// That gap is deliberate: entityMap and refCounts are already unsynchronised
// by design (see entity_map.go), so cross-goroutine misuse is undefined
// behaviour with or without this check, and the check only ever turned it
// into a clean panic instead of a race. Debug and race builds keep the full
// check.
var gidPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 128)
		return &buf
	},
}

func goroutineID() int64 {
	p := gidPool.Get().(*[]byte)
	buf := *p
	n := runtime.Stack(buf, false)
	*p = buf
	gidPool.Put(p)

	// Parse the leading "goroutine " prefix (10 bytes) followed by digits.
	s := buf[:n]
	const prefix = "goroutine "
	i := 0
	for i < len(prefix) && i < len(s) && s[i] == prefix[i] {
		i++
	}
	var id int64
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		id = id*10 + int64(s[i]-'0')
		i++
	}
	return id
}
