//go:build windows

package platform

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/third_party/w32"
)

// windowsWindow is the Windows implementation of [Window]. It owns an HWND
// and a window procedure that translates WM_* messages into [Event] values
// delivered through the event handler.
//
// All methods are safe to call from any goroutine. Methods that touch the
// HWND marshal onto the platform thread by dispatching through the platform,
// because Win32 window APIs are not safe to call from arbitrary threads.
type windowsWindow struct {
	hwnd    w32.HWND
	owner   *windowsPlatform
	options WindowOptions

	mu sync.Mutex // protects the handler fields below

	eventHandler func(Event)
	closeHandler func() bool

	// scaleFactor is the scale factor of the display the window is on. It is
	// updated when the window moves between displays or the DPI setting
	// changes.
	scaleFactor float32
}

// windowClassName is the registered window class name for Facet windows.
const windowClassName = "FacetWindow"

// windowClass is registered once per process.
var windowClassOnce sync.Once

func registerWindowClass() {
	windowClassOnce.Do(func() {
		cn := w32.MustStringToUTF16Ptr(windowClassName)
		wcx := w32.WNDCLASSEX{
			Size:       uint32(unsafe.Sizeof(w32.WNDCLASSEX{})),
			WndProc:    syscall.NewCallback(w32.WindowProc(windowWndProc)),
			Instance:   w32.GetModuleHandle(""),
			ClassName:  cn,
			Background: w32.COLOR_BTNFACE + 1,
		}
		w32.RegisterClassEx(&wcx)
	})
}

// activePlatform is the platform whose message loop is running. The wndproc
// is a package-level function (Win32 gives it no user data), so it recovers
// the platform from this global to look up windows by HWND. Set in Run,
// cleared when Run returns. Only one platform runs at a time.
var activePlatform *windowsPlatform

// windowWndProc is the window procedure for all Facet windows. It looks up
// the window by HWND in the platform's window map and dispatches to the
// window's handler method.
func windowWndProc(hwnd w32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if activePlatform != nil {
		w := activePlatform.windowByHWND(hwnd)
		if w != nil {
			return w.wndProc(hwnd, msg, wParam, lParam)
		}
	}
	return w32.DefWindowProc(hwnd, msg, wParam, lParam)
}

// newWindowsWindow creates a native window from opts. It must be called on
// the platform thread, because CreateWindowEx must run on the thread that
// registered the window class and will run the message loop.
func newWindowsWindow(owner *windowsPlatform, opts WindowOptions) (*windowsWindow, error) {
	registerWindowClass()

	// Convert logical pixels to device pixels using the primary display's
	// scale factor. The window corrects itself in WM_DPICHANGED if it lands
	// on a different display.
	scale := owner.primaryScale()
	dpi := uint(scale * 96.0)

	style := uint(w32.WS_OVERLAPPEDWINDOW)
	if !opts.Resizable {
		style &^= w32.WS_THICKFRAME
	}
	if !opts.Decorated {
		style &^= w32.WS_CAPTION | w32.WS_THICKFRAME | w32.WS_SYSMENU | w32.WS_MINIMIZEBOX | w32.WS_MAXIMIZEBOX
	}
	if opts.Visible {
		style |= w32.WS_VISIBLE
	}

	var exStyle uint
	if opts.Transparent {
		exStyle |= w32.WS_EX_NOREDIRECTIONBITMAP
	}
	if opts.AlwaysOnTop {
		exStyle |= w32.WS_EX_TOPMOST
	}

	// WindowOptions.Size is the client area, but CreateWindowEx takes the
	// outer size. Adjust the wanted client size to the outer size using
	// AdjustWindowRectExForDpi, which accounts for the frame at the
	// window's DPI rather than the primary display's. AdjustWindowRectEx
	// assumes the primary DPI, which is wrong on a secondary monitor with
	// a different scale factor.
	var width, height int
	if opts.Size.Width != 0 && opts.Size.Height != 0 {
		clientW := int(opts.Size.Width.ToDevicePixels(scale))
		clientH := int(opts.Size.Height.ToDevicePixels(scale))
		rect := w32.RECT{Right: int32(clientW), Bottom: int32(clientH)}
		w32.AdjustWindowRectExForDpi(&rect, style, false, exStyle, dpi)
		width = int(rect.Right - rect.Left)
		height = int(rect.Bottom - rect.Top)
	}

	x := int(opts.Position.X.ToDevicePixels(scale))
	y := int(opts.Position.Y.ToDevicePixels(scale))
	if opts.Position.X == 0 && opts.Position.Y == 0 {
		x = w32.CW_USEDEFAULT
		y = w32.CW_USEDEFAULT
	}

	hwnd := w32.CreateWindowEx(
		exStyle,
		w32.MustStringToUTF16Ptr(windowClassName),
		w32.MustStringToUTF16Ptr(opts.Title),
		style,
		x, y, width, height,
		0, 0, w32.GetModuleHandle(""), nil,
	)
	if hwnd == 0 {
		return nil, fmt.Errorf("create window: CreateWindowEx failed")
	}

	w := &windowsWindow{
		hwnd:        hwnd,
		owner:       owner,
		options:     opts,
		scaleFactor: scale,
	}

	// Register the window in the platform's HWND map so the wndproc can
	// find it. The map keeps the Go pointer visible to the garbage
	// collector; storing it in GWLP_USERDATA would hide it, and a caller
	// dropping its Window handle would leave the wndproc dereferencing
	// freed memory on the next WM_* message.
	owner.registerWindow(hwnd, w)

	// Apply min/max size constraints if set.
	if opts.MinSize.Width != 0 || opts.MinSize.Height != 0 {
		w.SetMinSize(opts.MinSize)
	}
	if opts.MaxSize.Width != 0 || opts.MaxSize.Height != 0 {
		w.SetMaxSize(opts.MaxSize)
	}

	return w, nil
}

// wndProc handles messages for this window. It returns 0 to indicate the
// message was handled; otherwise it calls DefWindowProc.
func (w *windowsWindow) wndProc(hwnd w32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case w32.WM_CLOSE:
		w.mu.Lock()
		handler := w.closeHandler
		w.mu.Unlock()
		if handler != nil {
			if !handler() {
				return 0 // veto
			}
		}
		w32.DestroyWindow(hwnd)
		return 0

	case w32.WM_DISPLAYCHANGE:
		// Display configuration changed (monitor attached/removed, DPI
		// changed, resolution changed). Refresh the platform's display
		// cache and fire the display change handler. WM_DISPLAYCHANGE is
		// broadcast to all top-level windows; each window triggers the
		// refresh, which is idempotent.
		w.owner.refreshDisplays()

	case w32.WM_DESTROY:
		// Remove the window from the platform's HWND map so the wndproc
		// stops dispatching messages to it. Without this the map leaks
		// one entry per closed window.
		w.owner.unregisterWindow(hwnd)
		w.emit(FocusEvent{Focused: false, Time: time.Now()})

	case w32.WM_SIZE:
		if wParam == w32.SIZE_MINIMIZED {
			break
		}
		width := int32(lParam & 0xFFFF)
		height := int32((lParam >> 16) & 0xFFFF)
		deviceSize := geometry.Size[geometry.DevicePixels]{
			Width:  geometry.DevicePixels(width),
			Height: geometry.DevicePixels(height),
		}
		logicalSize := geometry.Size[geometry.Pixels]{
			Width:  deviceSize.Width.ToPixels(w.scaleFactor),
			Height: deviceSize.Height.ToPixels(w.scaleFactor),
		}
		w.emit(ResizeEvent{Size: logicalSize, Time: time.Now()})

	case w32.WM_DPICHANGED:
		// wParam low word is the new DPI for the window.
		dpi := uint32(wParam & 0xFFFF)
		newScale := float32(dpi) / 96.0
		if newScale != w.scaleFactor {
			w.scaleFactor = newScale
			w.emit(ScaleChangedEvent{ScaleFactor: newScale, Time: time.Now()})
		}
		// lParam points to the suggested window rect for the new DPI.
		// This is sound: lParam holds a pointer to a RECT owned by the
		// OS, valid only for the duration of this message. We read it
		// here and do not retain the pointer beyond the call.
		rect := (*w32.RECT)(unsafe.Pointer(lParam))
		w32.SetWindowPos(hwnd, 0,
			int(rect.Left), int(rect.Top),
			int(rect.Right-rect.Left), int(rect.Bottom-rect.Top),
			w32.SWP_NOZORDER|w32.SWP_NOACTIVATE,
		)

	case w32.WM_MOUSEMOVE:
		w.emit(PointerEvent{
			Phase:     PointerMove,
			Position:  lParamToClientPoint(lParam),
			Buttons:   wParamToPointerButtons(wParam),
			Modifiers: keyStateToModifiers(),
			Time:      time.Now(),
		})

	case w32.WM_LBUTTONDOWN:
		w.emitPointerDownUp(lParam, wParam, PointerLeft, PointerDown)
	case w32.WM_LBUTTONUP:
		w.emitPointerDownUp(lParam, wParam, PointerLeft, PointerUp)
	case w32.WM_RBUTTONDOWN:
		w.emitPointerDownUp(lParam, wParam, PointerRight, PointerDown)
	case w32.WM_RBUTTONUP:
		w.emitPointerDownUp(lParam, wParam, PointerRight, PointerUp)
	case w32.WM_MBUTTONDOWN:
		w.emitPointerDownUp(lParam, wParam, PointerMiddle, PointerDown)
	case w32.WM_MBUTTONUP:
		w.emitPointerDownUp(lParam, wParam, PointerMiddle, PointerUp)
	case w32.WM_XBUTTONDOWN:
		button := PointerX1
		if int(wParam>>16)&w32.MK_XBUTTON2 != 0 {
			button = PointerX2
		}
		w.emitPointerDownUp(lParam, wParam, button, PointerDown)
	case w32.WM_XBUTTONUP:
		button := PointerX1
		if int(wParam>>16)&w32.MK_XBUTTON2 != 0 {
			button = PointerX2
		}
		w.emitPointerDownUp(lParam, wParam, button, PointerUp)

	case w32.WM_MOUSEWHEEL:
		// WM_MOUSEWHEEL delivers wheel delta in wParam high word, in
		// multiples of WHEEL_DELTA (120). Position is in screen coordinates,
		// so convert to client coordinates.
		delta := int32(int16(wParam>>16)) / w32.WHEEL_DELTA
		sx := int(int16(lParam & 0xFFFF))
		sy := int(int16((lParam >> 16) & 0xFFFF))
		cx, cy, _ := w32.ScreenToClient(hwnd, sx, sy)
		w.emit(WheelEvent{
			Position: geometry.Point[geometry.DevicePixels]{
				X: geometry.DevicePixels(cx),
				Y: geometry.DevicePixels(cy),
			},
			Delta: ScrollDelta{
				Unit:   ScrollLines,
				DeltaY: float32(delta),
			},
			Phase:     ScrollMoved,
			Modifiers: keyStateToModifiers(),
			Time:      time.Now(),
		})

	case w32.WM_MOUSEHWHEEL:
		// Horizontal mouse wheel. Delta in wParam high word.
		delta := int32(int16(wParam>>16)) / w32.WHEEL_DELTA
		sx := int(int16(lParam & 0xFFFF))
		sy := int(int16((lParam >> 16) & 0xFFFF))
		cx, cy, _ := w32.ScreenToClient(hwnd, sx, sy)
		w.emit(WheelEvent{
			Position: geometry.Point[geometry.DevicePixels]{
				X: geometry.DevicePixels(cx),
				Y: geometry.DevicePixels(cy),
			},
			Delta: ScrollDelta{
				Unit:   ScrollLines,
				DeltaX: float32(delta),
			},
			Phase:     ScrollMoved,
			Modifiers: keyStateToModifiers(),
			Time:      time.Now(),
		})

	case w32.WM_KEYDOWN, w32.WM_SYSKEYDOWN:
		vk := uint32(wParam)
		w.emit(KeyEvent{
			Phase:     KeyDown,
			Code:      vkToKeyCode(vk),
			Modifiers: keyStateToModifiers(),
			Time:      time.Now(),
		})

	case w32.WM_KEYUP, w32.WM_SYSKEYUP:
		vk := uint32(wParam)
		w.emit(KeyEvent{
			Phase:     KeyUp,
			Code:      vkToKeyCode(vk),
			Modifiers: keyStateToModifiers(),
			Time:      time.Now(),
		})

	case w32.WM_CHAR:
		r := rune(wParam)
		if r >= 0x20 || r == 0x09 || r == 0x0D || r == 0x08 {
			w.emit(TextEvent{Text: string(r), Time: time.Now()})
		}

	case w32.WM_SETFOCUS:
		w.emit(FocusEvent{Focused: true, Time: time.Now()})
	case w32.WM_KILLFOCUS:
		w.emit(FocusEvent{Focused: false, Time: time.Now()})

	case w32.WM_ERASEBKGND:
		// Paint the background with the configured colour so areas not yet
		// covered by the renderer show the right colour during a resize.
		hdc := w32.HDC(wParam)
		if rc := w32.GetClientRect(hwnd); rc != nil {
			c := w.options.Background
			colorRef := w32.COLORREF(uint32(c.R*255) | uint32(c.G*255)<<8 | uint32(c.B*255)<<16)
			hbrush := w32.CreateSolidBrush(colorRef)
			w32.FillRect(hdc, rc, hbrush)
			w32.DeleteObject(w32.HGDIOBJ(hbrush))
		}
		return 1
	}

	return w32.DefWindowProc(hwnd, msg, wParam, lParam)
}

// emit delivers an event to the handler, if one is set. It is called on the
// platform thread from the wndproc.
func (w *windowsWindow) emit(e Event) {
	w.mu.Lock()
	handler := w.eventHandler
	w.mu.Unlock()
	if handler != nil {
		handler(e)
	}
}

// emitPointerDownUp emits a pointer event for a button press or release.
func (w *windowsWindow) emitPointerDownUp(lParam, wParam uintptr, button PointerButton, phase PointerPhase) {
	w.emit(PointerEvent{
		Phase:     phase,
		Position:  lParamToClientPoint(lParam),
		Button:    button,
		Buttons:   wParamToPointerButtons(wParam),
		Modifiers: keyStateToModifiers(),
		Time:      time.Now(),
	})
}

// Show makes the window visible.
func (w *windowsWindow) Show() {
	w32.ShowWindow(w.hwnd, w32.SW_SHOW)
}

// Hide makes the window invisible.
func (w *windowsWindow) Hide() {
	w32.ShowWindow(w.hwnd, w32.SW_HIDE)
}

// Close destroys the window. The close handler, if set, is called first and
// may veto.
func (w *windowsWindow) Close() {
	w32.SendMessage(w.hwnd, w32.WM_CLOSE, 0, 0)
}

// SetTitle sets the title bar text.
func (w *windowsWindow) SetTitle(title string) {
	w32.SetWindowText(w.hwnd, title)
}

// SetSize sets the client-area size in logical pixels. The window stays
// where it is; only the size changes.
func (w *windowsWindow) SetSize(size geometry.Size[geometry.Pixels]) {
	deviceW := int(size.Width.ToDevicePixels(w.scaleFactor))
	deviceH := int(size.Height.ToDevicePixels(w.scaleFactor))
	// Adjust for the non-client area to get the client size we asked for,
	// at the window's DPI rather than the primary display's.
	rect := w32.RECT{Right: int32(deviceW), Bottom: int32(deviceH)}
	style := w32.GetWindowLong(w.hwnd, w32.GWL_STYLE)
	exStyle := w32.GetWindowLong(w.hwnd, w32.GWL_EXSTYLE)
	dpi := uint(w.scaleFactor * 96.0)
	w32.AdjustWindowRectExForDpi(&rect, uint(style), false, uint(exStyle), dpi)
	// SetWindowPos with SWP_NOMOVE leaves the position alone. MoveWindow
	// sets position as well as size, and there is no sentinel meaning
	// "leave it" — -1 is a coordinate, and using it moves the window to
	// the top-left corner of the screen.
	w32.SetWindowPos(w.hwnd, 0, 0, 0,
		int(rect.Right-rect.Left), int(rect.Bottom-rect.Top),
		w32.SWP_NOMOVE|w32.SWP_NOZORDER|w32.SWP_NOACTIVATE)
}

// Size returns the current client-area size in logical pixels.
func (w *windowsWindow) Size() geometry.Size[geometry.Pixels] {
	rc := w32.GetClientRect(w.hwnd)
	if rc == nil {
		return geometry.Size[geometry.Pixels]{}
	}
	return geometry.Size[geometry.Pixels]{
		Width:  geometry.DevicePixels(rc.Right).ToPixels(w.scaleFactor),
		Height: geometry.DevicePixels(rc.Bottom).ToPixels(w.scaleFactor),
	}
}

// SetPosition sets the window's top-left corner in logical pixels.
func (w *windowsWindow) SetPosition(pos geometry.Point[geometry.Pixels]) {
	x := int(pos.X.ToDevicePixels(w.scaleFactor))
	y := int(pos.Y.ToDevicePixels(w.scaleFactor))
	w32.SetWindowPos(w.hwnd, 0, x, y, 0, 0, w32.SWP_NOSIZE|w32.SWP_NOZORDER|w32.SWP_NOACTIVATE)
}

// Position returns the window's top-left corner in logical pixels.
func (w *windowsWindow) Position() geometry.Point[geometry.Pixels] {
	rc := w32.GetWindowRect(w.hwnd)
	if rc == nil {
		return geometry.Point[geometry.Pixels]{}
	}
	return geometry.Point[geometry.Pixels]{
		X: geometry.DevicePixels(rc.Left).ToPixels(w.scaleFactor),
		Y: geometry.DevicePixels(rc.Top).ToPixels(w.scaleFactor),
	}
}

// SetMinSize sets the minimum resizable size in logical pixels.
func (w *windowsWindow) SetMinSize(size geometry.Size[geometry.Pixels]) {
	w.options.MinSize = size
	// Enforce by clamping the current size if needed.
	cur := w.Size()
	if size.Width != 0 && cur.Width < size.Width {
		cur.Width = size.Width
	}
	if size.Height != 0 && cur.Height < size.Height {
		cur.Height = size.Height
	}
	if cur.Width != w.Size().Width || cur.Height != w.Size().Height {
		w.SetSize(cur)
	}
}

// SetMaxSize sets the maximum resizable size in logical pixels.
func (w *windowsWindow) SetMaxSize(size geometry.Size[geometry.Pixels]) {
	w.options.MaxSize = size
	cur := w.Size()
	if size.Width != 0 && cur.Width > size.Width {
		cur.Width = size.Width
	}
	if size.Height != 0 && cur.Height > size.Height {
		cur.Height = size.Height
	}
	if cur.Width != w.Size().Width || cur.Height != w.Size().Height {
		w.SetSize(cur)
	}
}

// SetResizable controls whether the user can resize the window.
func (w *windowsWindow) SetResizable(resizable bool) {
	style := w32.GetWindowLong(w.hwnd, w32.GWL_STYLE)
	if resizable {
		style |= w32.WS_THICKFRAME
	} else {
		style &^= w32.WS_THICKFRAME
	}
	w32.SetWindowLong(w.hwnd, w32.GWL_STYLE, uint32(style))
}

// SetAlwaysOnTop controls whether the window stays above other windows.
func (w *windowsWindow) SetAlwaysOnTop(onTop bool) {
	hwndInsertAfter := w32.HWND_NOTOPMOST
	if onTop {
		hwndInsertAfter = w32.HWND_TOPMOST
	}
	w32.SetWindowPos(w.hwnd, hwndInsertAfter, 0, 0, 0, 0,
		w32.SWP_NOMOVE|w32.SWP_NOSIZE|w32.SWP_NOACTIVATE)
}

// SetBackground sets the colour the client area is cleared to.
func (w *windowsWindow) SetBackground(c colour.Rgba) {
	w.options.Background = c
	w32.InvalidateRect(w.hwnd, nil, true)
}

// ScaleFactor returns the display scale factor for the display this window
// is on.
func (w *windowsWindow) ScaleFactor() float32 {
	return w.scaleFactor
}

// NativeHandle returns the HWND as a uintptr.
func (w *windowsWindow) NativeHandle() uintptr {
	return uintptr(w.hwnd)
}

// NativeSurface returns the drawing surface. On Windows, Direct3D creates its
// swapchain against the window itself, so this is the same as NativeHandle.
func (w *windowsWindow) NativeSurface() uintptr {
	return uintptr(w.hwnd)
}

// Focus makes this window the focused, foreground window.
func (w *windowsWindow) Focus() {
	w32.SetForegroundWindow(w.hwnd)
}

// IsFocused reports whether this window currently has keyboard focus.
func (w *windowsWindow) IsFocused() bool {
	return w32.GetFocus() == w.hwnd
}

// IsVisible reports whether the window is currently shown.
func (w *windowsWindow) IsVisible() bool {
	return w32.IsWindowVisible(w.hwnd)
}

// SetEventHandler sets the handler that receives input events.
func (w *windowsWindow) SetEventHandler(handler func(Event)) {
	w.mu.Lock()
	w.eventHandler = handler
	w.mu.Unlock()
}

// SetCloseHandler sets a handler called when the user requests the window to
// close.
func (w *windowsWindow) SetCloseHandler(handler func() bool) {
	w.mu.Lock()
	w.closeHandler = handler
	w.mu.Unlock()
}
