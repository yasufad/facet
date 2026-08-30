package platform

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
)

// WindowOptions configures a window at creation. The zero value is a hidden,
// decorated, opaque, non-resizable 800×600 window with a black background and
// no title — the least surprising starting point that a caller then overrides.
//
// Sizes and positions are in logical pixels — the same unit layout and
// styling speak in. The platform converts to device pixels using the display's
// scale factor; the caller does not do scale arithmetic, which is what typed
// units exist to prevent.
type WindowOptions struct {
	// Title is the window title shown in the title bar and the taskbar.
	Title string

	// Size is the initial client-area size. The client area excludes the
	// title bar and borders; the full window is larger.
	Size geometry.Size[geometry.Pixels]

	// Position is the initial window position in display coordinates. The
	// zero value lets the platform choose — typically cascading from the last
	// created window. A non-zero value pins the top-left corner.
	Position geometry.Point[geometry.Pixels]

	// MinSize is the smallest the user can resize the window to. The zero
	// value means no minimum.
	MinSize geometry.Size[geometry.Pixels]

	// MaxSize is the largest the user can resize the window to. The zero
	// value means no maximum.
	MaxSize geometry.Size[geometry.Pixels]

	// Background is the colour the client area is cleared to before the
	// renderer draws. It is the colour seen through transparent content or
	// during the first frame.
	Background colour.Rgba

	// Resizable controls whether the user can drag the window borders to
	// resize.
	Resizable bool

	// Decorated controls whether the window has a platform title bar and
	// border. An undecorated window is a borderless client area; the
	// framework is responsible for any custom title bar.
	Decorated bool

	// Transparent makes the window's background composited with the desktop
	// rather than opaque. Support and restrictions are platform-dependent;
	// a backend documents what it permits.
	Transparent bool

	// AlwaysOnTop keeps the window above non-always-on-top windows.
	AlwaysOnTop bool

	// Visible controls whether the window is shown on creation. false (the
	// zero value) creates a hidden window; the caller shows it with
	// [Window.Show]. This lets the caller set up event handlers and content
	// before the window appears.
	Visible bool
}

// Window is a native window whose client area belongs to Facet. The platform
// creates it, owns its native handle, and delivers input events to it; the
// renderer takes the handle and draws into the client area.
//
// Window is a layer boundary. Methods change by explicit decision, never as a
// side effect of a backend. A new backend implements this interface; it does
// not alter it.
type Window interface {
	// Show makes the window visible if it was hidden.
	Show()

	// Hide makes the window invisible without closing it.
	Hide()

	// Close destroys the window. The [Window.SetCloseHandler] handler, if set,
	// is called first and may veto the close. After Close returns the window
	// is unusable.
	Close()

	// SetTitle sets the title bar and taskbar text.
	SetTitle(title string)

	// SetSize sets the client-area size in logical pixels. The full window is
	// larger by the title bar and border; the platform converts to device
	// pixels using the display's scale factor.
	SetSize(size geometry.Size[geometry.Pixels])

	// Size returns the current client-area size in logical pixels.
	Size() geometry.Size[geometry.Pixels]

	// SetPosition sets the window's top-left corner in display coordinates,
	// in logical pixels.
	SetPosition(pos geometry.Point[geometry.Pixels])

	// Position returns the window's top-left corner in display coordinates,
	// in logical pixels.
	Position() geometry.Point[geometry.Pixels]

	// SetMinSize sets the minimum resizable size in logical pixels. The zero
	// value removes the minimum.
	SetMinSize(size geometry.Size[geometry.Pixels])

	// SetMaxSize sets the maximum resizable size in logical pixels. The zero
	// value removes the maximum.
	SetMaxSize(size geometry.Size[geometry.Pixels])

	// SetResizable controls whether the user can resize the window by
	// dragging its borders.
	SetResizable(resizable bool)

	// SetAlwaysOnTop controls whether the window stays above other windows.
	SetAlwaysOnTop(onTop bool)

	// SetBackground sets the colour the client area is cleared to.
	SetBackground(c colour.Rgba)

	// ScaleFactor returns the display scale factor for the display this
	// window is currently on. It changes when the window moves between
	// displays or the user changes the DPI setting; a [ScaleChangedEvent]
	// is delivered when it does.
	ScaleFactor() float32

	// NativeHandle returns the platform's native window handle — an HWND on
	// Windows, an NSWindow* on macOS, a GtkWidget* on Linux — as a uintptr.
	// It is the handle for window operations: positioning, parenting,
	// activation. platform does not know D3D, Metal or Vulkan exist; it hands
	// out the handle and stops.
	//
	// The value is valid only while the window is open. A caller that stores
	// it must release the window before using the handle.
	NativeHandle() uintptr

	// NativeSurface returns the platform's drawing surface — the handle a
	// graphics API binds its swapchain to — as a uintptr. On Windows this is
	// the same HWND as NativeHandle, because Direct3D creates its swapchain
	// against the window. On macOS it is the layer-backed content view (or its
	// CAMetalLayer), not the NSWindow, because Metal draws into the layer. On
	// Linux it is the GtkWidget or its underlying window.
	//
	// render takes this handle and owns the device, swapchain and shaders.
	// platform must not know which graphics API will be used. Returning the
	// window handle alone is not enough on macOS: Metal cannot draw into an
	// NSWindow, and render would have to do Cocoa work to find its own
	// surface — which is the seam this boundary exists to prevent.
	//
	// The value is valid only while the window is open.
	NativeSurface() uintptr

	// Focus makes this window the focused, foreground window.
	Focus()

	// IsFocused reports whether this window currently has keyboard focus.
	IsFocused() bool

	// IsVisible reports whether the window is currently shown.
	IsVisible() bool

	// SetEventHandler sets the handler that receives input events — pointer,
	// wheel, key, text, IME, focus, resize and scale-change. Events are
	// delivered synchronously on the platform thread, in arrival order.
	// Setting a handler replaces any previous one.
	SetEventHandler(handler func(Event))

	// SetCloseHandler sets a handler called when the user requests the window
	// to close — by clicking the close button, pressing Alt+F4, or choosing
	// Close from the taskbar. Returning false prevents the close; returning
	// true or nil allows it. The handler is called on the platform thread.
	SetCloseHandler(handler func() bool)
}
