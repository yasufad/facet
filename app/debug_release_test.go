//go:build !facet_debug

package app

// debugChecks is false in a release build, where checkGeneration does only the
// integer compare. Tests that require the full goroutine check at Context
// accessors skip when this is false.
const debugChecks = false
