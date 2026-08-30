//go:build facet_debug

package app

// debugChecks is true in a facet_debug build, where checkGeneration also calls
// checkUI. Tests that require the full goroutine check at Context accessors
// run when this is true.
const debugChecks = true
