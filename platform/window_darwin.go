//go:build darwin

package platform

import (
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
)

// macOS modifier flag bits, from NSEvent.h. These are quoted from the API
// rather than named from memory.
const (
	modFlagShift   uint = 1 << 17
	modFlagControl uint = 1 << 18
	modFlagOption  uint = 1 << 19
	modFlagCommand uint = 1 << 20
)

// NSWindow levels. NSNormalWindowLevel is 0; floating windows use
// NSFloatingWindowLevel (1). The level for always-on-top is a small
// constant above normal.
const (
	windowLevelNormal   int = 0
	windowLevelFloating int = 3
)

// cocoaWindow is the macOS implementation of [Window]. It owns an NSWindow
// and a FacetView content view that receives mouse and key events.
//
// All methods are safe to call from any goroutine. Methods that touch the
// NSWindow marshal onto the main thread through the platform's Dispatch,
// because Cocoa window APIs must run on the main thread.
type cocoaWindow struct {
	owner   *cocoaPlatform
	options WindowOptions

	nsWindow objc.ID // the NSWindow*
	view     objc.ID // the FacetView content view

	mu sync.Mutex // protects the handler fields below

	eventHandler func(Event)
	closeHandler func() bool

	scaleFactor float32
}

// styleMaskFor derives the NSWindow style mask from the creation options.
// A decorated window gets the titled, closable, miniaturizable and
// optionally resizable flags; an undecorated window is borderless.
func styleMaskFor(opts WindowOptions) uint {
	if !opts.Decorated {
		return styleMaskBorderless
	}
	mask := styleMaskTitled | styleMaskClosable | styleMaskMiniaturizable
	if opts.Resizable {
		mask |= styleMaskResizable
	}
	return mask
}

// newCocoaWindow creates a native NSWindow from opts. It must be called on
// the main thread, because NSWindow allocation and initialisation must run
// on the thread that owns the NSApplication.
func newCocoaWindow(owner *cocoaPlatform, opts WindowOptions) (*cocoaWindow, error) {
	scale := owner.primaryScale()
	deviceWidth := float64(opts.Size.Width) * float64(scale)
	deviceHeight := float64(opts.Size.Height) * float64(scale)

	// NSRect is 32 bytes (4 × CGFloat/double). On amd64 this selector
	// returns through objc_msgSend_stret; on arm64 through the x8
	// indirect-return register. purego's objc.Send[NSRect] handles both
	// ABIs — this is the call that answers the struct-return question.
	contentRect := NSRect{
		OriginX: float64(opts.Position.X) * float64(scale),
		OriginY: float64(opts.Position.Y) * float64(scale),
		Width:   deviceWidth,
		Height:  deviceHeight,
	}

	mask := styleMaskFor(opts)

	// NSWindow alloc, then initWithContentRect:styleMask:backing:defer:.
	// The selector takes an NSRect (struct argument), a uint style mask,
	// a uint backing type, and a bool defer flag.
	nsWindow := objc.ID(class_NSWindow).Send(sel_alloc)
	if nsWindow == 0 {
		return nil, fmt.Errorf("create window: NSWindow alloc returned nil")
	}
	nsWindow = objc.Send[objc.ID](nsWindow, sel_initWithContentRect,
		contentRect, mask, backingBuffered, false)
	if nsWindow == 0 {
		return nil, fmt.Errorf("create window: initWithContentRect returned nil")
	}

	// Create the FacetView content view. alloc creates an instance of our
	// registered subclass; initWithFrame: initialises it with a frame.
	view := objc.ID(facetViewClass).Send(sel_alloc)
	if view == 0 {
		return nil, fmt.Errorf("create window: FacetView alloc returned nil")
	}
	view = objc.Send[objc.ID](view, sel_initWithContentRect,
		contentRect, styleMaskBorderless, backingBuffered, false)
	if view == 0 {
		return nil, fmt.Errorf("create window: FacetView initWithFrame returned nil")
	}

	// Accept mouse-moved events without a button held, so hover works.
	view.Send(sel_setAcceptsMouseMovedEvents, true)

	// Set the view as the window's content view.
	nsWindow.Send(sel_setContentView, view)

	// Set the title for decorated windows.
	if opts.Decorated && opts.Title != "" {
		titleStr := NSStringFromString(opts.Title)
		defer releaseID(titleStr)
		nsWindow.Send(sel_setTitle, titleStr)
	}

	// Always-on-top: set the window level above normal.
	if opts.AlwaysOnTop {
		nsWindow.Send(sel_setLevel, windowLevelFloating)
	}

	w := &cocoaWindow{
		owner:       owner,
		options:     opts,
		nsWindow:    nsWindow,
		view:        view,
		scaleFactor: scale,
	}

	// Register the view in the view-to-window map so the event callbacks
	// can recover the cocoaWindow from the view's id.
	viewWindows[view] = w

	// Show the window if requested.
	if opts.Visible {
		nsWindow.Send(sel_makeKeyAndOrderFront, objc.ID(0))
	}

	return w, nil
}

// NSStringFromString creates an autoreleased NSString from a Go string.
// The caller must call Release on the returned ID when done with it.
func NSStringFromString(s string) objc.ID {
	// stringWithUTF8String: takes a C string (null-terminated UTF-8). We
	// append a null byte and pass a pointer to it. The pointer must stay
	// valid for the duration of the call; purego keeps it alive through
	// the keepAlive mechanism in the variadic argument path.
	cstr := append([]byte(s), 0)
	return objc.ID(class_NSString).Send(sel_stringWithUTF8String,
		uintptr(unsafe.Pointer(&cstr[0])))
}

// StringFromNSString reads the UTF-8 contents of an NSString.
func StringFromNSString(id objc.ID) string {
	if id == 0 {
		return ""
	}
	ptr := id.Send(sel_UTF8String)
	if ptr == 0 {
		return ""
	}
	// The pointer is valid for the lifetime of the NSString. We read it
	// immediately, before the autorelease pool can drain.
	// unsafe.StringFromPointer reads until a null byte.
	return cStringFromPointer(uintptr(ptr))
}

// cStringFromPointer reads a null-terminated C string from a uintptr.
// The pointer is treated as *byte; the conversion is sound because the
// NSString's UTF8String returns a pointer to the string's internal
// storage, which is valid for the duration of this call.
func cStringFromPointer(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}
	var length int
	p := (*byte)(unsafe.Pointer(ptr))
	for {
		b := *(*byte)(unsafe.Add(unsafe.Pointer(p), length))
		if b == 0 {
			break
		}
		length++
	}
	buf := make([]byte, length)
	for i := 0; i < length; i++ {
		buf[i] = *(*byte)(unsafe.Add(unsafe.Pointer(p), i))
	}
	return string(buf)
}

// releaseID sends the release message to an Objective-C object,
// decrementing its retain count. The selector is registered lazily here
// rather than cached, because releasing is not on the per-frame path.
var selRelease = objc.RegisterName("release")

func releaseID(id objc.ID) {
	id.Send(selRelease)
}

// Show makes the window visible.
func (w *cocoaWindow) Show() {
	w.owner.Dispatch(func() {
		w.nsWindow.Send(sel_orderFront, objc.ID(0))
	})
}

// Hide makes the window invisible.
func (w *cocoaWindow) Hide() {
	w.owner.Dispatch(func() {
		w.nsWindow.Send(sel_orderOut, objc.ID(0))
	})
}

// Close destroys the window.
func (w *cocoaWindow) Close() {
	w.owner.Dispatch(func() {
		delete(viewWindows, w.view)
		w.nsWindow.Send(sel_close)
	})
}

// SetTitle sets the title bar text.
func (w *cocoaWindow) SetTitle(title string) {
	w.owner.Dispatch(func() {
		titleStr := NSStringFromString(title)
		defer releaseID(titleStr)
		w.nsWindow.Send(sel_setTitle, titleStr)
	})
}

// SetSize sets the client-area size in logical pixels. The platform
// converts to device pixels using the display's scale factor.
func (w *cocoaWindow) SetSize(size geometry.Size[geometry.Pixels]) {
	w.owner.Dispatch(func() {
		scale := float64(w.scaleFactor)
		// The frame includes the title bar; we need to set the content
		// view's frame, not the window's frame. Using setFrame:display:
		// on the window sets the outer frame, so we adjust by reading the
		// current frame and computing the content rect.
		frame := objc.Send[NSRect](w.nsWindow, sel_frame)
		contentFrame := objc.Send[NSRect](w.view, sel_frame)
		titleBarHeight := frame.Height - contentFrame.Height
		newFrame := NSRect{
			OriginX: frame.OriginX,
			OriginY: frame.OriginY,
			Width:   float64(size.Width) * scale,
			Height:  float64(size.Height)*scale + titleBarHeight,
		}
		w.nsWindow.Send(sel_setFrame_display_, newFrame, true)
	})
}

// Size returns the current client-area size in logical pixels.
func (w *cocoaWindow) Size() geometry.Size[geometry.Pixels] {
	// Reading the content view's frame is a struct return (NSRect, 32
	// bytes). This is the second call site that exercises the struct-
	// return path — on amd64 through objc_msgSend_stret, on arm64
	// through x8.
	var frame NSRect
	w.owner.Dispatch(func() {
		frame = objc.Send[NSRect](w.view, sel_frame)
	})
	scale := float64(w.scaleFactor)
	return geometry.Size[geometry.Pixels]{
		Width:  geometry.Pixels(frame.Width / scale),
		Height: geometry.Pixels(frame.Height / scale),
	}
}

// SetPosition sets the window's top-left corner in display coordinates,
// in logical pixels.
func (w *cocoaWindow) SetPosition(pos geometry.Point[geometry.Pixels]) {
	w.owner.Dispatch(func() {
		scale := float64(w.scaleFactor)
		w.nsWindow.Send(sel_setFrameOrigin, NSPoint{
			X: float64(pos.X) * scale,
			Y: float64(pos.Y) * scale,
		})
	})
}

// Position returns the window's top-left corner in display coordinates,
// in logical pixels.
func (w *cocoaWindow) Position() geometry.Point[geometry.Pixels] {
	var frame NSRect
	w.owner.Dispatch(func() {
		frame = objc.Send[NSRect](w.nsWindow, sel_frame)
	})
	scale := float64(w.scaleFactor)
	return geometry.Point[geometry.Pixels]{
		X: geometry.Pixels(frame.OriginX / scale),
		Y: geometry.Pixels(frame.OriginY / scale),
	}
}

// SetMinSize sets the minimum resizable size in logical pixels.
func (w *cocoaWindow) SetMinSize(size geometry.Size[geometry.Pixels]) {
	w.owner.Dispatch(func() {
		scale := float64(w.scaleFactor)
		w.nsWindow.Send(sel_setMinSize, NSSize{
			Width:  float64(size.Width) * scale,
			Height: float64(size.Height) * scale,
		})
	})
}

// SetMaxSize sets the maximum resizable size in logical pixels.
func (w *cocoaWindow) SetMaxSize(size geometry.Size[geometry.Pixels]) {
	w.owner.Dispatch(func() {
		scale := float64(w.scaleFactor)
		w.nsWindow.Send(sel_setMaxSize, NSSize{
			Width:  float64(size.Width) * scale,
			Height: float64(size.Height) * scale,
		})
	})
}

// SetResizable controls whether the user can resize the window.
func (w *cocoaWindow) SetResizable(resizable bool) {
	w.owner.Dispatch(func() {
		mask := objc.Send[uint](w.nsWindow, objc.RegisterName("styleMask"))
		if resizable {
			mask |= styleMaskResizable
		} else {
			mask &^= styleMaskResizable
		}
		w.nsWindow.Send(sel_setStyleMask, mask)
	})
}

// SetAlwaysOnTop controls whether the window stays above other windows.
func (w *cocoaWindow) SetAlwaysOnTop(onTop bool) {
	w.owner.Dispatch(func() {
		level := windowLevelNormal
		if onTop {
			level = windowLevelFloating
		}
		w.nsWindow.Send(sel_setLevel, level)
	})
}

// State reports which of WindowState's four states the window is in.
// For the first deliverable, only WindowNormal is reported; the other
// states require NSWindow miniaturize:/deminiaturize: and zoom:/isZoomed.
func (w *cocoaWindow) State() WindowState {
	// TODO: read miniaturized and zoomed state from NSWindow
	return WindowNormal
}

// SetState transitions the window to state.
func (w *cocoaWindow) SetState(state WindowState) {
	// TODO: implement minimize, maximize, fullscreen
}

// SetBackground sets the colour the client area is cleared to.
func (w *cocoaWindow) SetBackground(c colour.Rgba) {
	// TODO: NSColor colorWithCalibratedRed:green:blue:alpha:, then
	// setBackgroundColor: on the window
}

// ScaleFactor returns the display scale factor for the display this
// window is currently on.
func (w *cocoaWindow) ScaleFactor() float32 {
	return w.scaleFactor
}

// NativeHandle returns the NSWindow* as a uintptr. It is valid only while
// the window is open.
func (w *cocoaWindow) NativeHandle() uintptr {
	return uintptr(w.nsWindow)
}

// NativeSurface returns the content view's handle as a uintptr. On macOS,
// Metal draws into a CAMetalLayer on the content view, not into the
// NSWindow directly — so the surface is the view, not the window.
func (w *cocoaWindow) NativeSurface() uintptr {
	return uintptr(w.view)
}

// SetCursor sets the pointer shape over this window.
func (w *cocoaWindow) SetCursor(shape Cursor) {
	// TODO: NSCursor push/pop
}

// Focus makes this window the focused, foreground window.
func (w *cocoaWindow) Focus() {
	w.owner.Dispatch(func() {
		w.nsWindow.Send(sel_makeKeyAndOrderFront, objc.ID(0))
	})
}

// IsFocused reports whether this window currently has keyboard focus.
func (w *cocoaWindow) IsFocused() bool {
	// isKeyWindow returns a BOOL (unsigned char on macOS). Reading it as
	// an ID and checking non-zero works because objc_msgSend returns the
	// value in the integer result register.
	var focused bool
	w.owner.Dispatch(func() {
		focused = w.nsWindow.Send(sel_isKeyWindow) != 0
	})
	return focused
}

// IsVisible reports whether the window is currently shown.
func (w *cocoaWindow) IsVisible() bool {
	var visible bool
	w.owner.Dispatch(func() {
		visible = w.nsWindow.Send(sel_isVisible) != 0
	})
	return visible
}

// SetEventHandler sets the handler that receives input events.
func (w *cocoaWindow) SetEventHandler(handler func(Event)) {
	w.mu.Lock()
	w.eventHandler = handler
	w.mu.Unlock()
}

// SetCloseHandler sets a handler called when the user requests the window
// to close.
func (w *cocoaWindow) SetCloseHandler(handler func() bool) {
	w.mu.Lock()
	w.closeHandler = handler
	w.mu.Unlock()
}

// deliverPointerEvent translates an NSEvent into a PointerEvent and
// delivers it to the window's event handler. Called from the FacetView's
// mouse callback on the main thread.
func (w *cocoaWindow) deliverPointerEvent(event objc.ID, phase PointerPhase) {
	w.mu.Lock()
	handler := w.eventHandler
	w.mu.Unlock()
	if handler == nil {
		return
	}

	// locationInWindow returns NSPoint (16 bytes, two CGFloats). This
	// fits in two registers on both ABIs and does not go through stret.
	loc := objc.Send[NSPoint](event, sel_locationInWindow)

	// buttonNumber returns NSInteger. For mouseMoved (no button change),
	// the button is PointerNone.
	button := PointerNone
	buttons := PointerButtons(0)
	if phase == PointerDown || phase == PointerUp {
		btn := int(event.Send(sel_buttonNumber))
		button = macButtonNumberToPointerButton(btn)
		buttons = macButtonToButtons(button)
	}

	modifiers := macModifiersToModifiers(uint(event.Send(sel_modifierFlags)))

	// Position is in the window's coordinate system (origin at bottom-left
	// on macOS). Convert to top-left origin for the framework.
	scale := float64(w.scaleFactor)
	handler(PointerEvent{
		Phase: phase,
		Position: geometry.Point[geometry.DevicePixels]{
			X: geometry.DevicePixels(loc.X * scale),
			Y: geometry.DevicePixels(loc.Y * scale),
		},
		Button:    button,
		Buttons:   buttons,
		Modifiers: modifiers,
		Time:      time.Now(),
	})
}

// deliverKeyEvent translates an NSEvent into a KeyEvent and delivers it
// to the window's event handler. Called from the FacetView's key callback
// on the main thread.
func (w *cocoaWindow) deliverKeyEvent(event objc.ID, phase KeyPhase) {
	w.mu.Lock()
	handler := w.eventHandler
	w.mu.Unlock()
	if handler == nil {
		return
	}

	// keyCode returns unsigned short (the virtual key code).
	keyCode := int(event.Send(sel_keyCode))
	modifiers := macModifiersToModifiers(uint(event.Send(sel_modifierFlags)))

	handler(KeyEvent{
		Phase:     phase,
		Code:      macKeyCodeToKeyCode(keyCode),
		Modifiers: modifiers,
		Time:      time.Now(),
	})
}

// macButtonNumberToPointerButton maps an NSEvent buttonNumber to a
// PointerButton. macOS numbers buttons starting from 0 for the left
// button, matching the Windows backend's convention.
func macButtonNumberToPointerButton(n int) PointerButton {
	switch n {
	case 0:
		return PointerLeft
	case 1:
		return PointerRight
	case 2:
		return PointerMiddle
	case 3:
		return PointerX1
	case 4:
		return PointerX2
	default:
		return PointerNone
	}
}

// macButtonToButtons converts a single PointerButton to the PointerButtons
// bitfield held during a press.
func macButtonToButtons(b PointerButton) PointerButtons {
	switch b {
	case PointerLeft:
		return ButtonLeft
	case PointerRight:
		return ButtonRight
	case PointerMiddle:
		return ButtonMiddle
	case PointerX1:
		return ButtonX1
	case PointerX2:
		return ButtonX2
	default:
		return 0
	}
}

// macModifiersToModifiers maps NSEvent modifier flags to the framework's
// Modifiers bitfield. The NSEvent flags are at bit positions 17-20; the
// framework's are at 0-3.
func macModifiersToModifiers(flags uint) Modifiers {
	var m Modifiers
	if flags&modFlagShift != 0 {
		m |= Shift
	}
	if flags&modFlagControl != 0 {
		m |= Control
	}
	if flags&modFlagOption != 0 {
		m |= Alt
	}
	if flags&modFlagCommand != 0 {
		m |= Super
	}
	return m
}

// macKeyCodeToKeyCode maps a macOS virtual key code to the framework's
// KeyCode. macOS virtual key codes are hardware-dependent but follow a
// standard layout for USB keyboards. The mapping covers the keys in the
// KeyCode enum; unmapped keys return KeyUnknown.
func macKeyCodeToKeyCode(code int) KeyCode {
	switch code {
	// Letters (A=0x00, S=0x01, D=0x02, F=0x03, H=0x04, G=0x05, Z=0x06,
	// X=0x07, C=0x08, V=0x09, B=0x0B, Q=0x0C, W=0x0D, E=0x0E, R=0x0F,
	// Y=0x10, T=0x11, U=0x20, I=0x22, O=0x1F, P=0x23, J=0x26, K=0x28,
	// L=0x25, N=0x2D, M=0x2E)
	case 0x00:
		return KeyA
	case 0x01:
		return KeyS
	case 0x02:
		return KeyD
	case 0x03:
		return KeyF
	case 0x04:
		return KeyH
	case 0x05:
		return KeyG
	case 0x06:
		return KeyZ
	case 0x07:
		return KeyX
	case 0x08:
		return KeyC
	case 0x09:
		return KeyV
	case 0x0B:
		return KeyB
	case 0x0C:
		return KeyQ
	case 0x0D:
		return KeyW
	case 0x0E:
		return KeyE
	case 0x0F:
		return KeyR
	case 0x10:
		return KeyY
	case 0x11:
		return KeyT
	case 0x20:
		return KeyU
	case 0x22:
		return KeyI
	case 0x1F:
		return KeyO
	case 0x23:
		return KeyP
	case 0x26:
		return KeyJ
	case 0x28:
		return KeyK
	case 0x25:
		return KeyL
	case 0x2D:
		return KeyN
	case 0x2E:
		return KeyM

	// Digits (0=0x1D, 1=0x12, 2=0x13, 3=0x14, 4=0x15, 5=0x17, 6=0x16,
	// 7=0x1A, 8=0x1C, 9=0x19)
	case 0x1D:
		return Key0
	case 0x12:
		return Key1
	case 0x13:
		return Key2
	case 0x14:
		return Key3
	case 0x15:
		return Key4
	case 0x17:
		return Key5
	case 0x16:
		return Key6
	case 0x1A:
		return Key7
	case 0x1C:
		return Key8
	case 0x19:
		return Key9

	// Function keys (F1=0x7A through F12=0x6F, F13=0x69, F14=0x6B,
	// F15=0x71, F16=0x6A, F17=0x40, F18=0x4F, F19=0x50, F20=0x5A)
	case 0x7A:
		return KeyF1
	case 0x78:
		return KeyF2
	case 0x63:
		return KeyF3
	case 0x76:
		return KeyF4
	case 0x60:
		return KeyF5
	case 0x61:
		return KeyF6
	case 0x62:
		return KeyF7
	case 0x64:
		return KeyF8
	case 0x65:
		return KeyF9
	case 0x6D:
		return KeyF10
	case 0x67:
		return KeyF11
	case 0x6F:
		return KeyF12
	case 0x69:
		return KeyF13
	case 0x6B:
		return KeyF14
	case 0x71:
		return KeyF15
	case 0x6A:
		return KeyF16
	case 0x40:
		return KeyF17
	case 0x4F:
		return KeyF18
	case 0x50:
		return KeyF19
	case 0x5A:
		return KeyF20

	// Navigation
	case 0x7B:
		return KeyArrowLeft
	case 0x7C:
		return KeyArrowRight
	case 0x7D:
		return KeyArrowDown
	case 0x7E:
		return KeyArrowUp
	case 0x73:
		return KeyHome
	case 0x77:
		return KeyEnd
	case 0x74:
		return KeyPageUp
	case 0x79:
		return KeyPageDown

	// Editing
	case 0x33:
		return KeyBackspace
	case 0x24:
		return KeyEnter
	case 0x30:
		return KeyTab
	case 0x35:
		return KeyEscape
	case 0x31:
		return KeySpace
	case 0x75:
		return KeyDelete
	case 0x72:
		return KeyInsert
	case 0x57:
		return KeyCapsLock

	// Modifiers
	case 0x38:
		return KeyShiftLeft
	case 0x3C:
		return KeyShiftRight
	case 0x3B:
		return KeyControlLeft
	case 0x3E:
		return KeyControlRight
	case 0x3A:
		return KeyAltLeft
	case 0x3D:
		return KeyAltRight
	case 0x37:
		return KeySuperLeft
	case 0x36:
		return KeySuperRight

	// Punctuation
	case 0x1B:
		return KeyMinus
	case 0x18:
		return KeyEqual
	case 0x21:
		return KeyLeftBracket
	case 0x1E:
		return KeyRightBracket
	case 0x2A:
		return KeyBackslash
	case 0x29:
		return KeySemicolon
	case 0x27:
		return KeyApostrophe
	case 0x32:
		return KeyGraveAccent
	case 0x2B:
		return KeyComma
	case 0x2F:
		return KeyPeriod
	case 0x2C:
		return KeySlash

	default:
		return KeyUnknown
	}
}
