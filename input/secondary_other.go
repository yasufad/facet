//go:build !darwin

package input

import "github.com/yasufad/facet/platform"

// secondaryModifier is Control on Windows, Linux and other non-macOS platforms.
const secondaryModifier = platform.Control
