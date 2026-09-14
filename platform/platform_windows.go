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

	// menu is the application menu set by SetApplicationMenu, attached as a
	// native menu bar to every window the backend owns. It is nil until
	// SetApplicationMenu is called. Accessed only on the platform thread.
	menu *nativeMenu

	// largeIcon and smallIcon are the HICONs set by SetIcon, applied to
	// the window class (so new windows inherit them) and to every
	// existing window via WM_SETICON. They are destroyed when replaced
	// or cleared. Accessed only on the platform thread.
	largeIcon w32.HICON
	smallIcon w32.HICON

	// trayHwnd is the hidden message-only window that receives tray
	// callback messages from Shell_NotifyIcon. Created lazily on the
	// first NewSystemTray call. Accessed only on the platform thread.
	trayHwnd w32.HWND

	// trays maps each tray ID to its *windowsSystemTray so the tray
	// wndproc can recover the tray from the callback message's wParam.
	// Accessed only on the platform thread.
	trays map[uint32]*windowsSystemTray

	// nextTrayID is the next tray ID to assign. Accessed only on the
	// platform thread.
	nextTrayID uint32
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

	// Set the AppUserModelID before any windows are created. The shell
	// uses it for taskbar grouping and toast notification attribution,
	// and SetCurrentProcessExplicitAppUserModelID must be called before
	// the process creates its first window — the dispatcher's hidden
	// window is created below, so this is the last chance. An empty
	// AppUserModelID is skipped: the process keeps its default identity,
	// and SendNotification will error rather than silently showing a
	// toast attributed to a fabricated ID.
	if opts.AppUserModelID != "" {
		if hr := w32.SetCurrentProcessExplicitAppUserModelID(opts.AppUserModelID); hr != 0 {
			return nil, fmt.Errorf("initialise platform: SetCurrentProcessExplicitAppUserModelID: %#x", uint32(hr))
		}
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

// SetApplicationMenu sets the application menu. On Windows, menus are
// per-window: this builds a native HMENU from the Menu tree and attaches it
// to every window the backend currently owns, and to every window created
// afterwards through registerWindow. The previous menu, if any, is destroyed
// after it is detached, so the command map that goes with it stops
// resolving WM_COMMAND.
//
// It dispatches onto the platform thread because SetMenu, DrawMenuBar and
// DestroyMenu must run on the thread that owns the windows.
func (p *windowsPlatform) SetApplicationMenu(menu *Menu) {
	p.dispatcher.Dispatch(func() {
		// Build the new menu before detaching the old one, so a build error
		// leaves the existing menu intact.
		var nm *nativeMenu
		if menu != nil {
			nm = buildMenu(menu)
		}

		// Detach and destroy the previous menu from every window.
		if p.menu != nil {
			for hwnd := range p.windows {
				w32.SetMenu(hwnd, 0)
				w32.DrawMenuBar(hwnd)
			}
			w32.DestroyMenu(p.menu.hmenu)
			p.menu = nil
		}

		p.menu = nm

		// Attach the new menu to every existing decorated window. An
		// undecorated window asked for no OS chrome, and a menu bar is OS
		// chrome, so it is skipped.
		if nm != nil {
			for hwnd, w := range p.windows {
				if !w.options.Decorated {
					continue
				}
				w32.SetMenu(hwnd, nm.hmenu)
				w32.DrawMenuBar(hwnd)
			}
		}
	})
}

// dispatchMenuCommand looks up a WM_COMMAND command ID in the current menu's
// command map and fires the matching OnClick closure. Called from the wndProc
// on the platform thread. A command ID with no entry — from a menu item
// added by the OS or a stale ID after a menu change — is a no-op.
func (p *windowsPlatform) dispatchMenuCommand(id uintptr) {
	if p.menu == nil {
		return
	}
	if fn, ok := p.menu.commands[id]; ok && fn != nil {
		fn()
	}
}

// ShowMessageDialog shows a modal message dialog through MessageBoxW. It
// does not marshal onto the platform thread itself — MessageBoxW blocks
// for as long as the dialog is up, and the caller must not be the
// platform thread, per the threading note on
// [Platform.ShowMessageDialog]. The window package dispatches it onto a
// background goroutine and marshals the result back, the same pattern
// ShowOpenDialog and ShowSaveDialog use.
//
// No owner HWND is passed: a [MessageDialog] carries no window reference,
// and passing 0 makes the dialog application-modal rather than parented
// to a specific window — the standard behaviour for an alert with no
// nominated owner.
func (p *windowsPlatform) ShowMessageDialog(dialog MessageDialog) (DialogResult, error) {
	flags := messageDialogFlags(dialog)
	id := w32.MessageBox(0, dialog.Message, dialog.Title, flags)
	if id == 0 {
		return ResultNone, fmt.Errorf("message dialog: MessageBoxW failed")
	}
	return messageDialogResult(id), nil
}

// messageDialogFlags translates a [MessageDialog]'s button set and icon
// into the MessageBoxW uType flags. Split out from ShowMessageDialog so
// the configuration — the part a wrong button-set constant or icon
// constant would actually break — is reachable by a test that never has
// to dismiss a modal window, the same way newFileSaveDialog is split from
// ShowSaveDialog.
func messageDialogFlags(dialog MessageDialog) uint {
	var flags uint
	switch dialog.Buttons {
	case ButtonsOK:
		flags |= w32.MB_OK
	case ButtonsOKCancel:
		flags |= w32.MB_OKCANCEL
	case ButtonsYesNo:
		flags |= w32.MB_YESNO
	case ButtonsYesNoCancel:
		flags |= w32.MB_YESNOCANCEL
	}
	switch dialog.Icon {
	case IconInfo:
		flags |= w32.MB_ICONINFORMATION
	case IconWarning:
		flags |= w32.MB_ICONWARNING
	case IconError:
		flags |= w32.MB_ICONERROR
	case IconQuestion:
		flags |= w32.MB_ICONQUESTION
	}
	return flags
}

// messageDialogResult maps a MessageBoxW return value (IDOK, IDYES, etc.)
// to the platform's [DialogResult]. A value that matches no known button
// maps to ResultNone, which MessageBoxW returns 0 for on failure —
// ShowMessageDialog checks that case separately and reports an error.
func messageDialogResult(id int) DialogResult {
	switch id {
	case w32.IDOK:
		return ResultOK
	case w32.IDCANCEL:
		return ResultCancel
	case w32.IDYES:
		return ResultYes
	case w32.IDNO:
		return ResultNo
	default:
		return ResultNone
	}
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

// ShowSaveDialog shows a modal file-save dialog through the Common Item
// Dialog (IFileSaveDialog). It does not marshal onto the platform thread
// itself — Show blocks for as long as the dialog is up, and the caller is
// the one who must not be the platform thread, per the threading note on
// [Platform.ShowSaveDialog].
//
// IFileSaveDialog is a single-threaded-apartment object with the same
// threading constraints as IFileOpenDialog: CoInitializeEx(COINIT_
// APARTMENTTHREADED) followed by LockOSThread for the duration.
func (p *windowsPlatform) ShowSaveDialog(dialog SaveFileDialog) (string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hr := w32.CoInitializeEx(w32.COINIT_APARTMENTTHREADED)
	if hr != 0 && hr != 1 { // S_OK, S_FALSE (already initialised on this thread)
		return "", fmt.Errorf("save dialog: CoInitializeEx: %#x", uint32(hr))
	}
	defer w32.CoUninitialize()

	fd, err := newFileSaveDialog(dialog)
	if err != nil {
		return "", err
	}
	defer fd.Release()

	showHR := fd.Show(0)
	if w32.IsDialogCancelled(showHR) {
		return "", nil
	}
	if showHR != 0 {
		return "", fmt.Errorf("save dialog: Show: %#x", uint32(showHR))
	}

	return saveDialogResult(fd)
}

// newFileSaveDialog creates and configures an IFileSaveDialog from dialog,
// without showing it. Split out from ShowSaveDialog so the configuration —
// the part a wrong FOS flag, a wrong SetFileName call, or a bad SetDefaultExtension
// would actually break — is reachable by a test that never has to click through
// a modal window.
func newFileSaveDialog(dialog SaveFileDialog) (*w32.IFileSaveDialog, error) {
	var fd *w32.IFileSaveDialog
	hr := w32.CoCreateInstance(
		&w32.CLSID_FileSaveDialog,
		w32.CLSCTX_INPROC_SERVER,
		&w32.IID_IFileSaveDialog,
		uintptr(unsafe.Pointer(&fd)),
	)
	if hr != 0 {
		return nil, fmt.Errorf("save dialog: CoCreateInstance: %#x", uint32(hr))
	}

	// FOS_OVERWRITEPROMPT: prompt before overwriting an existing file.
	// FOS_FORCEFILESYSTEM: accept only filesystem paths, not shell items.
	// FOS_PATHMUSTEXIST: the directory component must already exist.
	fos := uint32(w32.FOS_FORCEFILESYSTEM | w32.FOS_PATHMUSTEXIST | w32.FOS_OVERWRITEPROMPT)
	if hr := fd.SetOptions(fos); hr != 0 {
		fd.Release()
		return nil, fmt.Errorf("save dialog: SetOptions: %#x", uint32(hr))
	}

	if dialog.Title != "" {
		if hr := fd.SetTitle(w32.MustStringToUTF16Ptr(dialog.Title)); hr != 0 {
			fd.Release()
			return nil, fmt.Errorf("save dialog: SetTitle: %#x", uint32(hr))
		}
	}

	if dialog.Directory != "" {
		folder, hr := w32.SHCreateItemFromParsingName(w32.MustStringToUTF16Ptr(dialog.Directory))
		if hr != 0 {
			fd.Release()
			return nil, fmt.Errorf("save dialog: resolve directory %q: %#x", dialog.Directory, uint32(hr))
		}
		setHR := fd.SetFolder(folder)
		folder.Release()
		if setHR != 0 {
			fd.Release()
			return nil, fmt.Errorf("save dialog: SetFolder: %#x", uint32(setHR))
		}
	}

	if dialog.DefaultName != "" {
		if hr := fd.SetFileName(w32.MustStringToUTF16Ptr(dialog.DefaultName)); hr != 0 {
			fd.Release()
			return nil, fmt.Errorf("save dialog: SetFileName: %#x", uint32(hr))
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
			return nil, fmt.Errorf("save dialog: SetFileTypes: %#x", uint32(hr))
		}

		// SetDefaultExtension uses the first extension from the first filter
		// as the auto-appended extension when the user types a name without
		// one. Empty extension list means "all files" (*.*) — no default.
		//
		// Unlike SetFileName (round-tripped through GetFileName above) and
		// SetOptions (round-tripped through GetOptions), IFileDialog has no
		// getter for the default extension, so this call cannot be verified
		// the same way. The vtable slot is confirmed by the same SDL/Wine
		// cross-check as the rest of the interface; what is unconfirmed is
		// that the slot holds the right function pointer at runtime.
		if len(dialog.Filters[0].Extensions) > 0 {
			ext := dialog.Filters[0].Extensions[0]
			if hr := fd.SetDefaultExtension(w32.MustStringToUTF16Ptr(ext)); hr != 0 {
				fd.Release()
				return nil, fmt.Errorf("save dialog: SetDefaultExtension: %#x", uint32(hr))
			}
		}
	}

	return fd, nil
}

// saveDialogResult reads the chosen path from a completed save dialog.
func saveDialogResult(fd *w32.IFileSaveDialog) (string, error) {
	item, hr := fd.GetResult()
	if hr != 0 {
		return "", fmt.Errorf("save dialog: GetResult: %#x", uint32(hr))
	}
	defer item.Release()

	path, hr := item.GetDisplayName(w32.SIGDN_FILESYSPATH)
	if hr != 0 {
		return "", fmt.Errorf("save dialog: GetDisplayName: %#x", uint32(hr))
	}
	return path, nil
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

// Hide hides every window the backend owns. It dispatches onto the
// platform thread because the windows map is accessed only there and
// ShowWindow runs on the thread that owns the windows. Each window's
// visibility is asked of the OS through ShowWindow(SW_HIDE), which clears
// the WS_VISIBLE style — the same state IsWindowVisible reads.
func (p *windowsPlatform) Hide() {
	p.dispatcher.Dispatch(func() {
		for hwnd := range p.windows {
			w32.ShowWindow(hwnd, w32.SW_HIDE)
		}
	})
}

// Show restores every window hidden by [Hide]. It dispatches onto the
// platform thread for the same reason, and calls ShowWindow(SW_SHOW) on
// each window, which sets the WS_VISIBLE style back. A window that was
// minimised before Hide returns shown, not restored to minimised — SW_SHOW
// makes the window visible without changing its minimised/maximised state,
// and a window that was minimised stays minimised (and therefore not
// visible on screen) until the user restores it.
func (p *windowsPlatform) Show() {
	p.dispatcher.Dispatch(func() {
		for hwnd := range p.windows {
			w32.ShowWindow(hwnd, w32.SW_SHOW)
		}
	})
}

// SetIcon sets the application icon from PNG or ICO bytes. It creates
// large and small HICONs and applies them to the window class (so new
// windows inherit the icon) and to every existing window via WM_SETICON
// (which overrides the class icon for windows already on screen). The
// previous icons, if any, are destroyed.
//
// It dispatches onto the platform thread because the windows map and
// SetClassLongPtr must run on the thread that owns the windows.
//
// SetIcon is void on the [Platform] interface, so an error decoding the
// icon bytes cannot be reported to the caller — the icon is simply not
// applied. That is an interface limitation; a future revision that needs
// to surface decode errors would change SetIcon to return an error,
// which crosses the layer boundary and is a decision for the lead.
func (p *windowsPlatform) SetIcon(icon []byte) {
	p.dispatcher.Dispatch(func() {
		// Destroy the previous icons before installing new ones or
		// clearing, so the handles do not leak across repeated calls.
		if p.largeIcon != 0 {
			w32.DestroyIcon(p.largeIcon)
			p.largeIcon = 0
		}
		if p.smallIcon != 0 {
			w32.DestroyIcon(p.smallIcon)
			p.smallIcon = 0
		}

		if len(icon) == 0 {
			// Clear the class icon and every window's override.
			for hwnd := range p.windows {
				w32.SendMessage(hwnd, w32.WM_SETICON, w32.ICON_BIG, 0)
				w32.SendMessage(hwnd, w32.WM_SETICON, w32.ICON_SMALL, 0)
			}
			return
		}

		large, err := w32.CreateLargeHIconFromImage(icon)
		if err != nil {
			return // cannot report: SetIcon is void
		}
		small, err := w32.CreateSmallHIconFromImage(icon)
		if err != nil {
			w32.DestroyIcon(large)
			return
		}
		p.largeIcon = large
		p.smallIcon = small

		// Set on the class so new windows inherit it. All Facet windows
		// share one class (windowClassName), so setting on any window's
		// class sets it for all. Apply WM_SETICON to every existing
		// window too, since the class icon does not retroactively change
		// a window that is already created.
		for hwnd := range p.windows {
			w32.SetApplicationIcon(uintptr(hwnd), large)
			w32.SendMessage(hwnd, w32.WM_SETICON, w32.ICON_BIG, uintptr(large))
			w32.SendMessage(hwnd, w32.WM_SETICON, w32.ICON_SMALL, uintptr(small))
		}
	})
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
// can find it, and attaches the current application menu if the window is
// decorated. Called on the platform thread during window creation.
func (p *windowsPlatform) registerWindow(hwnd w32.HWND, w *windowsWindow) {
	p.mu.Lock()
	p.windows[hwnd] = w
	menu := p.menu
	p.mu.Unlock()
	// An undecorated window asked for no OS chrome, and a menu bar is OS
	// chrome, so it gets no menu.
	if menu != nil && w.options.Decorated {
		w32.SetMenu(hwnd, menu.hmenu)
		w32.DrawMenuBar(hwnd)
	}
}

// unregisterWindow removes a window from the HWND map. Called on the platform
// thread from WM_DESTROY. After this returns, the wndproc will not find the
// window for its HWND and will pass messages to DefWindowProc.
//
// When the last registered window is destroyed, the platform stops its own
// run loop: a Facet application's lifetime is its windows, and a process
// pumping an empty window map never returns from Run. This is the natural
// place to detect emptiness — unregisterWindow already runs on the platform
// thread, where the map is mutated, so PostQuitMessage lands on the right
// queue without the cross-thread dispatch that [windowsPlatform.Quit] uses
// for callers off the platform thread.
//
// This is a Windows behaviour, not a macOS one: a macOS application keeps
// running with no windows (closing the last window does not quit on macOS),
// so the Darwin backend does not mirror this.
func (p *windowsPlatform) unregisterWindow(hwnd w32.HWND) {
	p.mu.Lock()
	delete(p.windows, hwnd)
	empty := len(p.windows) == 0
	p.mu.Unlock()
	if empty {
		p.dispatcher.Quit()
	}
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
