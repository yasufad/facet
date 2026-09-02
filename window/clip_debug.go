//go:build facet_debug

package window

import "fmt"

func (w *Window) checkClipStackEmpty() {
	if w.clipDepth != 0 {
		panic(fmt.Sprintf("window: clip stack not empty at end of paint (depth %d)", w.clipDepth))
	}
}

func (w *Window) checkPrepaintClipStackEmpty() {
	if len(w.prepaintClipStack) != 0 {
		panic(fmt.Sprintf("window: prepaint clip stack not empty at end of prepaint (depth %d)", len(w.prepaintClipStack)))
	}
}
