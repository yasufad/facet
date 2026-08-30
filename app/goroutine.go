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
// touched. Three mechanisms share the work, each placed where it is cheap:
//
//   - checkUI (this function) runs at update boundaries and on exported App
//     methods. A frame opens a few update boundaries, so a few 6µs checks
//     per frame is affordable. This catches direct calls to App methods from
//     a background goroutine.
//   - A generation counter (Context.checkGeneration, an integer compare at
//     ~1ns) runs at every Context accessor in a release build. It catches a
//     context stored and used after its update has ended.
//   - In a facet_debug build, checkGeneration also calls checkUI. This
//     catches a context used from another goroutine while its update is still
//     running — the case the generation compare alone cannot see, because the
//     generation still matches. The 6µs cost is acceptable in a debug or race
//     build, where it does not matter.
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
