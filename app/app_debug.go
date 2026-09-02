//go:build facet_debug

package app

// checkUI panics if the calling goroutine is not the UI goroutine. It is
// called from every exported entry point that touches entity state, the
// effect queue or the subscriber sets. The check costs about 6µs
// (runtime.Stack to read the goroutine header); see goroutine.go for the
// measurement. It runs only in a facet_debug build: a release build relies on
// Context.checkGeneration's integer compare instead, which catches a context
// used after its update has ended but not a direct call to an App method from
// the wrong goroutine while an update is in flight. See app_check.go.
func (app *App) checkUI() {
	if goroutineID() != app.uiGoroutine {
		panic("app: context used from a goroutine other than the UI goroutine")
	}
}
