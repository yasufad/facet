//go:build windows

package platform

// Run starts the native event loop and blocks until Quit is called. It must
// be called on the same goroutine that called New, because the dispatcher's
// hidden window was created on that thread.
//
// The loop is owned by the main-thread dispatcher, which runs a standard
// Win32 GetMessage/TranslateMessage/DispatchMessage loop. The dispatcher's
// hidden window receives posted closures and runs them; Facet windows
// receive input and paint messages and dispatch them through their wndproc.
//
// WM_DISPLAYCHANGE is broadcast to all top-level windows, so it arrives at
// each Facet window's wndproc, which calls refreshDisplays on the platform.
// The loop itself does not need to intercept it.
func (p *windowsPlatform) Run() error {
	p.dispatcher.Run()
	return nil
}
