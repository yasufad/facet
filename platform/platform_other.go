//go:build !windows

package platform

import (
	"fmt"
	"runtime"
)

// New reports that no backend exists for the current OS yet. It is not a
// stub for a backend: Run, NewWindow and the rest of [Platform] have no
// implementation here and are not meant to grow one behind this file. Its
// only job is to keep New declared on every OS, so `go build ./...`,
// `go vet ./...` and `go test ./...` — the three commands every change is
// checked against — catch a break in code that calls it (examples, tools)
// on Linux and macOS too, rather than only on a Windows machine weeks later.
//
// When a real backend lands for an OS, that OS is removed from this file's
// build constraint; the day every OS has one, the file is deleted.
func New(opts Options) (Platform, error) {
	return nil, fmt.Errorf("platform: no backend for %s yet", runtime.GOOS)
}
