//go:build !facet_debug

package app

// checkUI is a no-op in a release build. It is called from every exported
// entry point that touches entity state, the effect queue or the subscriber
// sets; a context used from another goroutine while its update is still
// running is not caught here in release builds. It is still caught if the
// context is used after its update has ended, by Context.checkGeneration's
// integer compare. See context_check.go and app_debug.go.
func (app *App) checkUI() {}
