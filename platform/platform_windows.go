//go:build windows

package platform

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/yasufad/facet/third_party/mainthread"
	"github.com/yasufad/facet/third_party/w32"
)

// windowsPlatform is the Windows implementation of [Platform]. It owns the
// main-thread dispatcher, the message loop, and the shell pieces.
type windowsPlatform struct {
	options Options

	dispatcher *mainthread.Dispatcher

	mu sync.Mutex // protects the handler fields below

	activationHandler func()
	quitHandler       func() bool
	displayHandler    func()

	clipboard windowsClipboard

	// cached displays, refreshed on display configuration change.
	displays []Display
}

// New creates a Windows platform. It must be called on the goroutine that
// will become the platform thread; that goroutine is locked to the OS thread
// by the dispatcher's hidden window.
//
// The platform is not running until [Platform.Run] is called. Methods that
// touch native handles marshal onto the platform thread through the
// dispatcher.
func New(opts Options) (Platform, error) {
	// Make this process DPI-aware so window sizes are in physical pixels.
	w32.SetProcessDpiAwarenessContext(w32.DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)

	p := &windowsPlatform{
		options:    opts,
		dispatcher: mainthread.New(opts.Name),
	}

	displays, err := enumerateDisplays()
	if err != nil {
		return nil, fmt.Errorf("initialise platform: %w", err)
	}
	p.displays = displays

	return p, nil
}

// Quit stops the event loop.
func (p *windowsPlatform) Quit() {
	p.dispatcher.Quit()
}

// Dispatch runs f on the platform thread.
func (p *windowsPlatform) Dispatch(f func()) {
	p.dispatcher.Dispatch(f)
}

// NewWindow creates a window from opts. The window is created on the platform
// thread.
func (p *windowsPlatform) NewWindow(opts WindowOptions) (Window, error) {
	var (
		w   *windowsWindow
		err error
	)
	p.dispatcher.Dispatch(func() {
		w, err = newWindowsWindow(p, opts)
	})
	if err != nil {
		return nil, err
	}
	return w, nil
}

// Displays returns the currently attached displays.
func (p *windowsPlatform) Displays() []Display {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.displays
}

// PrimaryDisplay returns the primary display.
func (p *windowsPlatform) PrimaryDisplay() Display {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, d := range p.displays {
		if d.Primary {
			return d
		}
	}
	if len(p.displays) > 0 {
		return p.displays[0]
	}
	return Display{}
}

// ActiveDisplay returns the display that contains the currently focused
// window, or the primary display if no window is focused.
func (p *windowsPlatform) ActiveDisplay() Display {
	hwnd := w32.GetForegroundWindow()
	if hwnd != 0 {
		monitor := w32.MonitorFromWindow(hwnd, w32.MONITOR_DEFAULTTONEAREST)
		p.mu.Lock()
		defer p.mu.Unlock()
		for _, d := range p.displays {
			if d.ID == fmt.Sprintf("%d", monitor) {
				return d
			}
		}
	}
	return p.PrimaryDisplay()
}

// Clipboard returns the system clipboard.
func (p *windowsPlatform) Clipboard() Clipboard {
	return p.clipboard
}

// SetCursor sets the pointer shape.
func (p *windowsPlatform) SetCursor(shape Cursor) {
	p.dispatcher.Dispatch(func() {
		setCursor(shape)
	})
}

// SetCursorVisible shows or hides the pointer.
func (p *windowsPlatform) SetCursorVisible(visible bool) {
	p.dispatcher.Dispatch(func() {
		showCursor(visible)
	})
}

// SetApplicationMenu sets the global application menu. On Windows, menus are
// per-window; this is a no-op until per-window menu support is added.
func (p *windowsPlatform) SetApplicationMenu(menu *Menu) {
	// TODO: per-window menus
}

// NewSystemTray creates a system tray icon.
func (p *windowsPlatform) NewSystemTray(opts SystemTrayOptions) (SystemTray, error) {
	return nil, fmt.Errorf("system tray: not implemented")
}

// ShowMessageDialog shows a modal message dialog.
func (p *windowsPlatform) ShowMessageDialog(dialog MessageDialog) (DialogResult, error) {
	return 0, fmt.Errorf("message dialog: not implemented")
}

// ShowOpenDialog shows a modal file-open dialog.
func (p *windowsPlatform) ShowOpenDialog(dialog OpenFileDialog) ([]string, error) {
	return nil, fmt.Errorf("open dialog: not implemented")
}

// ShowSaveDialog shows a modal file-save dialog.
func (p *windowsPlatform) ShowSaveDialog(dialog SaveFileDialog) (string, error) {
	return "", fmt.Errorf("save dialog: not implemented")
}

// SendNotification displays a system notification.
func (p *windowsPlatform) SendNotification(notification Notification) error {
	return fmt.Errorf("notification: not implemented")
}

// Activate brings the application to the foreground.
func (p *windowsPlatform) Activate() {
	p.dispatcher.Dispatch(func() {
		hwnd := w32.GetForegroundWindow()
		if hwnd != 0 {
			w32.SetForegroundWindow(hwnd)
		}
	})
}

// Hide hides all application windows.
func (p *windowsPlatform) Hide() {
	// TODO: enumerate and hide all windows
}

// Show restores windows hidden by Hide.
func (p *windowsPlatform) Show() {
	// TODO: enumerate and show all windows
}

// SetIcon sets the application icon.
func (p *windowsPlatform) SetIcon(icon []byte) {
	// TODO: load icon from bytes
}

// SetActivationHandler sets a handler called when the application is
// activated.
func (p *windowsPlatform) SetActivationHandler(handler func()) {
	p.mu.Lock()
	p.activationHandler = handler
	p.mu.Unlock()
}

// SetQuitHandler sets a handler called when the user requests the
// application to quit.
func (p *windowsPlatform) SetQuitHandler(handler func() bool) {
	p.mu.Lock()
	p.quitHandler = handler
	p.mu.Unlock()
}

// SetDisplayChangeHandler sets a handler called when the display
// configuration changes.
func (p *windowsPlatform) SetDisplayChangeHandler(handler func()) {
	p.mu.Lock()
	p.displayHandler = handler
	p.mu.Unlock()
}

// primaryScale returns the primary display's scale factor, used as the
// initial scale for new windows.
func (p *windowsPlatform) primaryScale() float32 {
	d := p.PrimaryDisplay()
	if d.ScaleFactor == 0 {
		return 1.0
	}
	return d.ScaleFactor
}

// refreshDisplays re-enumerates displays and fires the display change
// handler if the configuration changed.
func (p *windowsPlatform) refreshDisplays() {
	displays, err := enumerateDisplays()
	if err != nil {
		return
	}
	p.mu.Lock()
	p.displays = displays
	handler := p.displayHandler
	p.mu.Unlock()
	if handler != nil {
		handler()
	}
}

// _ = unsafe.Pointer(nil) ensures unsafe is imported for future use in
// platform-specific code. The window implementation uses it; this file does
// not yet, but the import is required for the build constraint to be useful.
var _ = unsafe.Pointer(nil)
