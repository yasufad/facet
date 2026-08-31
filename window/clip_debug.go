//go:build facet_debug

package window

import "fmt"

func (w *Window) checkClipStackEmpty() {
	if w.clipDepth != 0 {
		panic(fmt.Sprintf("window: clip stack not empty at end of paint (depth %d)", w.clipDepth))
	}
}
