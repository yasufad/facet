//go:build windows

package platform

import (
	"fmt"
	"runtime"
	"strings"
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

	mu sync.Mutex // protects the fields below

	activationHandler func()
	quitHandler       func() bool
	displayHandler    func()

	clipboard windowsClipboard

	// cached displays, refreshed on display configuration change.
	displays []Display

	// windows maps each HWND to its *windowsWindow so the wndproc can
	// recover the window from the handle the OS passes it. The map keeps
	// the Go pointer visible to the garbage collector — storing it in
	// GWLP_USERDATA instead would hide it from the collector, and a caller
	// dropping its Window handle would leave the wndproc dereferencing
	// freed memory on the next WM_* message.
	//
	// Accessed only on the platform thread (the thread that runs the
	// message loop), so the mutex is shared with the handler fields above
	// rather than needing its own.
	windows map[w32.HWND]*windowsWindow
}

// New creates a Windows platform. It must be called on the goroutine that
// will run the platform — typically the main goroutine. That goroutine's OS
// thread is locked for the duration of the call so the dispatcher's hidden
// window belongs to the right thread, and [Platform.Run] must later be
// called from the same goroutine. Calling Run from a different goroutine
// panics.
//
// The platform is not running until [Platform.Run] is called, but windows
// can be created before Run: NewWindow dispatches onto the platform thread
// through the hidden window, which already exists. Methods that touch
// native handles marshal onto the platform thread through the dispatcher.
func New(opts Options) (Platform, error) {
	// Default the application name so New(Options{}) works. The name
	// becomes the Win32 window class name, which must be non-empty.
	if opts.Name == "" {
		opts.Name = "Facet"
	}

	// Make this process DPI-aware so window sizes are in physical pixels.
	w32.SetProcessDpiAwarenessContext(w32.DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2)

	dispatcher, err := mainthread.New(opts.Name)
	if err != nil {
		return nil, fmt.Errorf("initialise platform: %w", err)
	}

	p := &windowsPlatform{
		options:    opts,
		dispatcher: dispatcher,
		windows:    make(map[w32.HWND]*windowsWindow),
	}

	displays, err := enumerateDisplays()
	if err != nil {
		return nil, fmt.Errorf("initialise platform: %w", err)
	}
	p.displays = displays

	return p, nil
}

// Quit stops the event loop. It dispatches onto the platform thread because
// PostQuitMessage posts to the calling thread's queue, and Quit may be called
// from any goroutine.
func (p *windowsPlatform) Quit() {
	p.dispatcher.Dispatch(func() {
		p.dispatcher.Quit()
	})
}

// Dispatch runs f on the platform thread.
func (p *windowsPlatform) Dispatch(f func()) {
	p.dispatcher.Dispatch(f)
}

// NewWindow creates a window from opts. The window is created on the platform
// thread; this method blocks until creation completes.
func (p *windowsPlatform) NewWindow(opts WindowOptions) (Window, error) {
	var (
		w   *windowsWindow
		err error
		wg  sync.WaitGroup
	)
	wg.Add(1)
	p.dispatcher.Dispatch(func() {
		defer wg.Done()
		w, err = newWindowsWindow(p, opts)
	})
	wg.Wait()
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

// ShowOpenDialog shows a modal file-open dialog through the Common Item
// Dialog (IFileOpenDialog). It does not marshal onto the platform thread
// itself — Show blocks for as long as the dialog is up, and the caller is
// the one who must not be the platform thread, per the threading note on
// [Platform.ShowOpenDialog].
//
// IFileOpenDialog is a single-threaded-apartment object: it must be created
// and driven from the same OS thread, which CoInitializeEx(COINIT_
// APARTMENTTHREADED) establishes. LockOSThread pins the calling goroutine to
// that thread for the duration; nothing here assumes it is the platform
// thread's OS thread, only that it stays put.
func (p *windowsPlatform) ShowOpenDialog(dialog OpenFileDialog) ([]string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hr := w32.CoInitializeEx(w32.COINIT_APARTMENTTHREADED)
	if hr != 0 && hr != 1 { // S_OK, S_FALSE (already initialised on this thread)
		return nil, fmt.Errorf("open dialog: CoInitializeEx: %#x", uint32(hr))
	}
	defer w32.CoUninitialize()

	fd, err := newFileOpenDialog(dialog)
	if err != nil {
		return nil, err
	}
	defer fd.Release()

	showHR := fd.Show(0)
	if w32.IsDialogCancelled(showHR) {
		return nil, nil
	}
	if showHR != 0 {
		return nil, fmt.Errorf("open dialog: Show: %#x", uint32(showHR))
	}

	if dialog.Multiple {
		return openDialogResults(fd)
	}
	return openDialogResult(fd)
}

// newFileOpenDialog creates and configures a Common Item Dialog from dialog,
// without showing it. Split out from ShowOpenDialog so the configuration —
// the part a wrong FOS flag or a malformed filter pattern would actually
// break — is reachable by a test that never has to click through a modal
// window.
func newFileOpenDialog(dialog OpenFileDialog) (*w32.IFileOpenDialog, error) {
	var fd *w32.IFileOpenDialog
	hr := w32.CoCreateInstance(
		&w32.CLSID_FileOpenDialog,
		w32.CLSCTX_INPROC_SERVER,
		&w32.IID_IFileOpenDialog,
		uintptr(unsafe.Pointer(&fd)),
	)
	if hr != 0 {
		return nil, fmt.Errorf("open dialog: CoCreateInstance: %#x", uint32(hr))
	}

	fos := uint32(w32.FOS_FORCEFILESYSTEM | w32.FOS_PATHMUSTEXIST | w32.FOS_FILEMUSTEXIST)
	if dialog.Multiple {
		fos |= w32.FOS_ALLOWMULTISELECT
	}
	if hr := fd.SetOptions(fos); hr != 0 {
		fd.Release()
		return nil, fmt.Errorf("open dialog: SetOptions: %#x", uint32(hr))
	}

	if dialog.Title != "" {
		if hr := fd.SetTitle(w32.MustStringToUTF16Ptr(dialog.Title)); hr != 0 {
			fd.Release()
			return nil, fmt.Errorf("open dialog: SetTitle: %#x", uint32(hr))
		}
	}

	if dialog.Directory != "" {
		folder, hr := w32.SHCreateItemFromParsingName(w32.MustStringToUTF16Ptr(dialog.Directory))
		if hr != 0 {
			fd.Release()
			return nil, fmt.Errorf("open dialog: resolve directory %q: %#x", dialog.Directory, uint32(hr))
		}
		setHR := fd.SetFolder(folder)
		folder.Release()
		if setHR != 0 {
			fd.Release()
			return nil, fmt.Errorf("open dialog: SetFolder: %#x", uint32(setHR))
		}
	}

	if len(dialog.Filters) > 0 {
		specs := make([]w32.COMDLG_FILTERSPEC, len(dialog.Filters))
		for i, f := range dialog.Filters {
			specs[i] = w32.COMDLG_FILTERSPEC{
				PszName: w32.MustStringToUTF16Ptr(f.Name),
				PszSpec: w32.MustStringToUTF16Ptr(filterSpecPattern(f.Extensions)),
			}
		}
		if hr := fd.SetFileTypes(specs); hr != 0 {
			fd.Release()
			return nil, fmt.Errorf("open dialog: SetFileTypes: %#x", uint32(hr))
		}
	}

	return fd, nil
}

// filterSpecPattern joins extensions ("txt", "md") into the
// semicolon-separated pattern list ("*.txt;*.md") IFileDialog::SetFileTypes
// takes. No extensions means no restriction.
func filterSpecPattern(extensions []string) string {
	if len(extensions) == 0 {
		return "*.*"
	}
	patterns := make([]string, len(extensions))
	for i, ext := range extensions {
		patterns[i] = "*." + ext
	}
	return strings.Join(patterns, ";")
}

// openDialogResult reads the single chosen item from a dialog shown without
// FOS_ALLOWMULTISELECT.
func openDialogResult(fd *w32.IFileOpenDialog) ([]string, error) {
	item, hr := fd.GetResult()
	if hr != 0 {
		return nil, fmt.Errorf("open dialog: GetResult: %#x", uint32(hr))
	}
	defer item.Release()

	path, hr := item.GetDisplayName(w32.SIGDN_FILESYSPATH)
	if hr != 0 {
		return nil, fmt.Errorf("open dialog: GetDisplayName: %#x", uint32(hr))
	}
	return []string{path}, nil
}

// openDialogResults reads every chosen item from a dialog shown with
// FOS_ALLOWMULTISELECT.
func openDialogResults(fd *w32.IFileOpenDialog) ([]string, error) {
	items, hr := fd.GetResults()
	if hr != 0 {
		return nil, fmt.Errorf("open dialog: GetResults: %#x", uint32(hr))
	}
	defer items.Release()

	count, hr := items.GetCount()
	if hr != 0 {
		return nil, fmt.Errorf("open dialog: GetCount: %#x", uint32(hr))
	}

	paths := make([]string, 0, count)
	for i := uint32(0); i < count; i++ {
		item, hr := items.GetItemAt(i)
		if hr != 0 {
			return nil, fmt.Errorf("open dialog: GetItemAt(%d): %#x", i, uint32(hr))
		}
		path, hr := item.GetDisplayName(w32.SIGDN_FILESYSPATH)
		item.Release()
		if hr != 0 {
			return nil, fmt.Errorf("open dialog: GetDisplayName(%d): %#x", i, uint32(hr))
		}
		paths = append(paths, path)
	}
	return paths, nil
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

// registerWindow records a window in the platform's HWND map so the wndproc
// can find it. Called on the platform thread during window creation.
func (p *windowsPlatform) registerWindow(hwnd w32.HWND, w *windowsWindow) {
	p.mu.Lock()
	p.windows[hwnd] = w
	p.mu.Unlock()
}

// unregisterWindow removes a window from the HWND map. Called on the platform
// thread from WM_DESTROY. After this returns, the wndproc will not find the
// window for its HWND and will pass messages to DefWindowProc.
func (p *windowsPlatform) unregisterWindow(hwnd w32.HWND) {
	p.mu.Lock()
	delete(p.windows, hwnd)
	p.mu.Unlock()
}

// windowByHWND looks up a window by its HWND. Called from the wndproc on the
// platform thread. Returns nil if the HWND is not a Facet window (or has
// been destroyed).
func (p *windowsPlatform) windowByHWND(hwnd w32.HWND) *windowsWindow {
	p.mu.Lock()
	w := p.windows[hwnd]
	p.mu.Unlock()
	return w
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
