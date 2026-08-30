package platform

// Options configures the platform at construction. The constructor itself is
// provided by the backend — a build-tagged file in this package — because it
// returns a concrete type implementing [Platform]. The interface is defined
// here; the constructor arrives with the first backend.
type Options struct {
	// Name is the application name. It is used for the window class name on
	// Windows, the bundle name on macOS, and the application ID on Linux.
	Name string

	// Icon is the application icon data, in the same platform-dependent
	// format as [SystemTrayOptions.Icon].
	Icon []byte

	// SingleInstance prevents more than one running process. When a second
	// instance starts it activates the first and exits. The mechanism is
	// platform-dependent: a named mutex on Windows, a lock file or launch
	// service on macOS, a DBus name on Linux.
	SingleInstance bool
}

// Platform is the operating-system layer: the native event loop, windows,
// displays, main-thread dispatch, and the shell — clipboard, cursor, menus,
// tray, dialogs and notifications.
//
// It is a layer boundary. Three packages — render, window and input — are
// written against this interface, so a change to it is a change to all three.
// Plan a change before writing it.
//
// All methods are safe to call from any goroutine unless the method's
// documentation says otherwise. Methods that touch native handles marshal
// onto the platform thread internally; the caller does not need to dispatch.
type Platform interface {
	// Run starts the native event loop and blocks until [Platform.Quit] is
	// called or the platform's natural quit condition is met (the last window
	// closing on macOS, for example). It must be called on the goroutine that
	// will serve as the UI goroutine; the implementation locks that goroutine
	// to the OS thread.
	//
	// The event loop is where input events are translated and delivered and
	// where dispatched closures run. It does not return until the application
	// is done.
	Run() error

	// Quit stops the event loop. [Platform.Run] returns after the current
	// iteration completes. Safe to call from any goroutine, including the
	// platform thread itself.
	Quit()

	// Dispatch runs f on the platform thread — the thread [Platform.Run] is
	// running on. If called from the platform thread, f runs on the next loop
	// iteration (or immediately, at the backend's discretion); if called from
	// another goroutine, f is queued and run when the loop next drains
	// dispatched work.
	//
	// This is the seam the foreground executor in package app sits on: the
	// window package wires the executor's wake channel to Dispatch so that
	// background work returns to the UI goroutine. platform itself never
	// imports app.
	Dispatch(f func())

	// NewWindow creates a window from opts. The window is shown or hidden
	// according to [WindowOptions.Visible]. Returns an error if the window
	// cannot be created.
	NewWindow(opts WindowOptions) (Window, error)

	// Displays returns the currently attached displays. The slice is ordered
	// with the primary display first. The caller must not retain the slice
	// across display-configuration changes; call again when
	// [Platform.SetDisplayChangeHandler] fires.
	Displays() []Display

	// PrimaryDisplay returns the primary display — the one that holds the
	// taskbar or dock and receives new windows by default. It is the first
	// element of [Platform.Displays].
	PrimaryDisplay() Display

	// ActiveDisplay returns the display that contains the currently focused
	// window, or the primary display if no window is focused.
	ActiveDisplay() Display

	// Clipboard returns the system clipboard.
	Clipboard() Clipboard

	// SetCursor sets the pointer shape over the active window. See [Cursor]
	// for the available shapes.
	SetCursor(shape Cursor)

	// SetCursorVisible shows or hides the pointer. Hiding is per-application;
	// the pointer returns when the application exits.
	SetCursorVisible(visible bool)

	// SetApplicationMenu sets the global application menu. On platforms with a
	// global menu bar (macOS) this is the screen menu bar; on platforms
	// without one (Windows, Linux) it sets the menu bar on each window, or is
	// a no-op where window menus are set per-window.
	SetApplicationMenu(menu *Menu)

	// NewSystemTray creates a system tray icon from opts.
	NewSystemTray(opts SystemTrayOptions) (SystemTray, error)

	// ShowMessageDialog shows a modal message dialog and blocks until the
	// user dismisses it. It must not be called on the platform thread from
	// within an event handler, as that would deadlock the loop; the window
	// package dispatches it onto a background goroutine and marshals the
	// result back.
	ShowMessageDialog(dialog MessageDialog) (DialogResult, error)

	// ShowOpenDialog shows a modal file-open dialog and blocks until the user
	// dismisses it. It returns the selected paths, or an empty slice if the
	// user cancelled. The same threading note as
	// [Platform.ShowMessageDialog] applies.
	ShowOpenDialog(dialog OpenFileDialog) ([]string, error)

	// ShowSaveDialog shows a modal file-save dialog and blocks until the user
	// dismisses it. It returns the chosen path, or an empty string if the
	// user cancelled. The same threading note as
	// [Platform.ShowMessageDialog] applies.
	ShowSaveDialog(dialog SaveFileDialog) (string, error)

	// SendNotification displays a system notification. It returns an error
	// if notifications are disabled or the platform does not support them.
	SendNotification(notification Notification) error

	// Activate brings the application to the foreground. On macOS this
	// un-hides and activates the app through NSApplication; on Windows it
	// brings the foreground window to the front.
	Activate()

	// Hide hides all application windows without quitting. [Platform.Show]
	// restores them.
	Hide()

	// Show restores windows hidden by [Platform.Hide].
	Show()

	// SetIcon sets the application icon.
	SetIcon(icon []byte)

	// SetActivationHandler sets a handler called when the application is
	// activated — brought to the foreground by the user or the system. It is
	// called on the platform thread.
	SetActivationHandler(handler func())

	// SetQuitHandler sets a handler called when the user requests the
	// application to quit — through the menu, Cmd+Q on macOS, or the system
	// asking it to terminate. Returning false prevents the quit; returning
	// true allows it. The handler is called on the platform thread. It is the
	// application's chance to save documents or confirm the quit.
	SetQuitHandler(handler func() bool)

	// SetDisplayChangeHandler sets a handler called when the global display
	// configuration changes — a monitor is attached or removed, the primary
	// display changes, or a scale factor changes system-wide. After it fires,
	// the values returned by [Platform.Displays] and
	// [Platform.PrimaryDisplay] reflect the new configuration. It is called
	// on the platform thread.
	//
	// This is the global signal for display-set changes. The per-window
	// signal for a scale-factor change — including when the window moves to a
	// different display — is [ScaleChangedEvent], delivered through the
	// window's event handler. A consumer that needs both subscribes to each
	// once; they do not duplicate each other.
	SetDisplayChangeHandler(handler func())
}
