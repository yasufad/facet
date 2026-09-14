//go:build windows

package platform

import (
	"fmt"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"github.com/yasufad/facet/third_party/w32"
)

// trayWindowClassName is the registered window class name for the hidden
// message-only window that receives Shell_NotifyIcon callback messages.
const trayWindowClassName = "FacetTray"

// trayClassOnce registers the tray window class once per process.
var trayClassOnce sync.Once

// wmTrayCallback is the registered window message Shell_NotifyIcon sends to
// the tray window when the user interacts with a tray icon. wParam is the
// tray ID (NOTIFYICONDATA.UID); lParam is the mouse message (WM_LBUTTONUP,
// WM_RBUTTONUP, etc.).
var wmTrayCallback = w32.RegisterWindowMessage(w32.MustStringToUTF16Ptr("Facet.TrayCallback"))

// windowsSystemTray is the Windows implementation of [SystemTray]. It owns a
// tray icon registered through Shell_NotifyIcon, identified by its HWND and
// a unique ID. Callback messages arrive at the platform's hidden tray window
// and are dispatched to the matching tray by ID.
//
// All methods are safe to call from any goroutine. Methods that touch the
// tray icon marshal onto the platform thread through the dispatcher, because
// Shell_NotifyIcon must run on the thread that owns the hidden window.
type windowsSystemTray struct {
	owner   *windowsPlatform
	id      uint32
	icon    w32.HICON
	menu    *Menu
	onClick func()
	tooltip string
}

// NewSystemTray creates a system tray icon from opts. It dispatches onto the
// platform thread because Shell_NotifyIcon and CreateWindowEx must run on
// the thread that owns the message loop, and blocks until creation completes
// — the same pattern NewWindow uses.
func (p *windowsPlatform) NewSystemTray(opts SystemTrayOptions) (SystemTray, error) {
	var (
		tray *windowsSystemTray
		err  error
		wg   sync.WaitGroup
	)
	wg.Add(1)
	p.dispatcher.Dispatch(func() {
		defer wg.Done()
		tray, err = p.newSystemTray(opts)
	})
	wg.Wait()
	if err != nil {
		return nil, err
	}
	return tray, nil
}

// newSystemTray creates the hidden tray window (if it does not already
// exist), builds the NOTIFYICONDATA, and calls Shell_NotifyIcon(NIM_ADD).
// Must be called on the platform thread.
func (p *windowsPlatform) newSystemTray(opts SystemTrayOptions) (*windowsSystemTray, error) {
	if err := p.ensureTrayWindow(); err != nil {
		return nil, err
	}

	id := p.nextTrayID
	p.nextTrayID++

	tray := &windowsSystemTray{
		owner:   p,
		id:      id,
		menu:    opts.Menu,
		onClick: opts.OnClick,
		tooltip: opts.Tooltip,
	}

	if len(opts.Icon) > 0 {
		icon, err := w32.CreateSmallHIconFromImage(opts.Icon)
		if err != nil {
			return nil, fmt.Errorf("system tray: decode icon: %w", err)
		}
		tray.icon = icon
	}

	nid := w32.NOTIFYICONDATA{
		CbSize:           uint32(unsafe.Sizeof(w32.NOTIFYICONDATA{})),
		HWnd:             p.trayHwnd,
		UID:              id,
		UFlags:           w32.NIF_MESSAGE | w32.NIF_ICON | w32.NIF_TIP,
		UCallbackMessage: wmTrayCallback,
		HIcon:            tray.icon,
	}
	copyUTF16Into(nid.SzTip[:], opts.Tooltip)

	if !w32.ShellNotifyIcon(w32.NIM_ADD, &nid) {
		if tray.icon != 0 {
			w32.DestroyIcon(tray.icon)
		}
		return nil, fmt.Errorf("system tray: Shell_NotifyIcon NIM_ADD failed")
	}

	if p.trays == nil {
		p.trays = make(map[uint32]*windowsSystemTray)
	}
	p.trays[id] = tray

	return tray, nil
}

// ensureTrayWindow creates the hidden message-only window that receives
// tray callback messages, if it does not already exist. Must be called on
// the platform thread.
func (p *windowsPlatform) ensureTrayWindow() error {
	if p.trayHwnd != 0 {
		return nil
	}

	trayClassOnce.Do(func() {
		cn := w32.MustStringToUTF16Ptr(trayWindowClassName)
		wcx := w32.WNDCLASSEX{
			Size:      uint32(unsafe.Sizeof(w32.WNDCLASSEX{})),
			WndProc:   syscall.NewCallback(w32.WindowProc(trayWndProc)),
			Instance:  w32.GetModuleHandle(""),
			ClassName: cn,
		}
		w32.RegisterClassEx(&wcx)
	})

	// HWND_MESSAGE creates a message-only window: invisible, not in the
	// taskbar, and excluded from broadcast messages — exactly what a tray
	// callback sink needs.
	hwnd := w32.CreateWindowEx(
		0,
		w32.MustStringToUTF16Ptr(trayWindowClassName),
		w32.MustStringToUTF16Ptr("__facet_tray"),
		0,
		0, 0, 0, 0,
		w32.HWND_MESSAGE,
		0,
		w32.GetModuleHandle(""),
		nil,
	)
	if hwnd == 0 {
		return fmt.Errorf("system tray: CreateWindowEx failed for tray window")
	}
	p.trayHwnd = hwnd
	return nil
}

// trayWndProc is the window procedure for the hidden tray callback window.
// Win32 gives it no user data, so it recovers the platform from
// activePlatform — the same global the window wndproc uses. The tray window
// is created before Run sets activePlatform (NewSystemTray can be called
// before Run), so messages that arrive before Run are passed to
// DefWindowProc; callback messages only arrive once the message loop is
// pumping, by which point activePlatform is set.
func trayWndProc(hwnd w32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == wmTrayCallback {
		if activePlatform != nil {
			activePlatform.dispatchTrayCallback(uint32(wParam), lParam)
		}
		return 0
	}
	return w32.DefWindowProc(hwnd, msg, wParam, lParam)
}

// dispatchTrayCallback handles a Shell_NotifyIcon callback message for the
// tray identified by id. Called on the platform thread from trayWndProc.
func (p *windowsPlatform) dispatchTrayCallback(id uint32, lParam uintptr) {
	tray, ok := p.trays[id]
	if !ok {
		return
	}
	switch uint32(lParam) {
	case w32.WM_LBUTTONUP:
		if tray.onClick != nil {
			tray.onClick()
		}
	case w32.WM_RBUTTONUP:
		tray.showMenu()
	}
}

// showMenu displays the tray's menu as a native popup at the cursor
// position. Called on the platform thread from dispatchTrayCallback, which
// is outside any entity borrow — the callback arrives from the tray
// wndproc, not from an event handler inside app.UpdateEntity — so
// TrackPopupMenu's nested modal loop is safe to run here directly, without
// the deferral that ShowContextMenu uses.
func (t *windowsSystemTray) showMenu() {
	if t.menu == nil {
		return
	}

	nm := buildPopupMenu(t.menu)
	defer w32.DestroyMenu(nm.hmenu)

	x, y, ok := w32.GetCursorPos()
	if !ok {
		return
	}

	// SetForegroundWindow is required before TrackPopupMenu on a tray
	// window: without it the menu does not dismiss when the user clicks
	// elsewhere. This is a documented Shell_NotifyIcon requirement.
	w32.SetForegroundWindow(t.owner.trayHwnd)

	const flags = w32.TPM_LEFTALIGN | w32.TPM_TOPALIGN | w32.TPM_RIGHTBUTTON | w32.TPM_RETURNCMD
	cmd := w32.TrackPopupMenuCommand(nm.hmenu, flags, int32(x), int32(y), t.owner.trayHwnd, nil)
	if cmd == 0 {
		return
	}
	if onClick, ok := nm.commands[uintptr(cmd)]; ok && onClick != nil {
		onClick()
	}
}

// SetIcon replaces the tray icon. It dispatches onto the platform thread
// because Shell_NotifyIcon must run on the thread that owns the tray window.
func (t *windowsSystemTray) SetIcon(icon []byte) {
	t.owner.dispatcher.Dispatch(func() {
		if t.icon != 0 {
			w32.DestroyIcon(t.icon)
			t.icon = 0
		}

		if len(icon) > 0 {
			hicon, err := w32.CreateSmallHIconFromImage(icon)
			if err != nil {
				return
			}
			t.icon = hicon
		}

		nid := w32.NOTIFYICONDATA{
			CbSize: uint32(unsafe.Sizeof(w32.NOTIFYICONDATA{})),
			HWnd:   t.owner.trayHwnd,
			UID:    t.id,
			UFlags: w32.NIF_ICON,
			HIcon:  t.icon,
		}
		w32.ShellNotifyIcon(w32.NIM_MODIFY, &nid)
	})
}

// SetMenu replaces the tray menu shown on right-click.
func (t *windowsSystemTray) SetMenu(menu *Menu) {
	t.owner.dispatcher.Dispatch(func() {
		t.menu = menu
	})
}

// SetTooltip replaces the tooltip text. It dispatches onto the platform
// thread because Shell_NotifyIcon must run on the thread that owns the tray
// window.
func (t *windowsSystemTray) SetTooltip(tooltip string) {
	t.owner.dispatcher.Dispatch(func() {
		t.tooltip = tooltip
		nid := w32.NOTIFYICONDATA{
			CbSize: uint32(unsafe.Sizeof(w32.NOTIFYICONDATA{})),
			HWnd:   t.owner.trayHwnd,
			UID:    t.id,
			UFlags: w32.NIF_TIP,
		}
		copyUTF16Into(nid.SzTip[:], tooltip)
		w32.ShellNotifyIcon(w32.NIM_MODIFY, &nid)
	})
}

// Remove removes the icon from the tray and destroys the icon handle. The
// SystemTray is unusable after Remove returns, so Remove blocks until the
// removal completes — the same synchronous pattern NewSystemTray uses.
func (t *windowsSystemTray) Remove() {
	var wg sync.WaitGroup
	wg.Add(1)
	t.owner.dispatcher.Dispatch(func() {
		defer wg.Done()
		nid := w32.NOTIFYICONDATA{
			CbSize: uint32(unsafe.Sizeof(w32.NOTIFYICONDATA{})),
			HWnd:   t.owner.trayHwnd,
			UID:    t.id,
		}
		w32.ShellNotifyIcon(w32.NIM_DELETE, &nid)
		if t.icon != 0 {
			w32.DestroyIcon(t.icon)
			t.icon = 0
		}
		delete(t.owner.trays, t.id)
	})
	wg.Wait()
}

// copyUTF16Into copies a Go string into a fixed-size UTF-16 array,
// truncating at the array's capacity and ensuring a NUL terminator. The
// destination is zero-initialised by Go, so unwritten elements are already
// NUL.
func copyUTF16Into(dst []uint16, src string) {
	encoded := utf16.Encode([]rune(src))
	n := len(encoded)
	if n >= len(dst) {
		n = len(dst) - 1
	}
	copy(dst[:n], encoded[:n])
}
