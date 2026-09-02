package platform

// WindowState is the mutually exclusive states a window can be in. A window
// is in exactly one at a time — Minimized, Maximized and Fullscreen never
// combine, which four independent setters (Minimize, Maximize, Restore,
// SetFullscreen) would let a caller represent regardless of what the OS
// actually allows, and would force every caller to sequence correctly:
// restore-then-maximise and maximise-then-restore are different operations
// with nothing to say so.
type WindowState int

const (
	// WindowNormal is the restored state: neither minimized, maximized nor
	// fullscreen.
	WindowNormal WindowState = iota

	// WindowMinimized is iconified to the taskbar.
	WindowMinimized

	// WindowMaximized fills the work area of the display it is on.
	WindowMaximized

	// WindowFullscreen fills the entire display it is on, edge to edge,
	// with no border or title bar. Unlike Minimized and Maximized this is
	// not a state the operating system tracks on Windows: a backend
	// implements it as a borderless window resized to the display's full
	// bounds, and must remember what to restore when leaving it.
	WindowFullscreen
)
