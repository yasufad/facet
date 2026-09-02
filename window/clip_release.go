//go:build !facet_debug

package window

func (w *Window) checkClipStackEmpty() {}

func (w *Window) checkPrepaintClipStackEmpty() {}
