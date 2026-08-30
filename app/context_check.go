//go:build !facet_debug

package app

// checkGeneration panics if the context has escaped the update it was created
// in. The generation is an integer compare (~1ns), so it is cheap enough to
// run at every accessor on the per-frame path.
//
// In a release build this is the only check at Context accessors. It catches a
// context used after its update has ended, but not one used from another
// goroutine while the update is still running (the generation still matches).
// That case is caught in a facet_debug build, where checkGeneration also calls
// checkUI. See context_debug.go.
func (c *Context[T]) checkGeneration() {
	if c.app.generation != c.generation {
		panic("app: context used after its update has ended")
	}
}
