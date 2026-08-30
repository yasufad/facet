//go:build facet_debug

package app

// checkGeneration panics if the context has escaped the update it was created
// in, or if it is used from a goroutine other than the UI goroutine.
//
// In a facet_debug build, checkGeneration also calls checkUI (~6µs). This
// catches a context used from another goroutine while its update is still
// running — the case the generation compare alone cannot see, because the
// generation still matches. The cost is acceptable in a debug or race build,
// where a 6µs check per accessor does not matter. Release builds keep the
// integer compare only; see context_check.go.
func (c *Context[T]) checkGeneration() {
	c.app.checkUI()
	if c.app.generation != c.generation {
		panic("app: context used after its update has ended")
	}
}
