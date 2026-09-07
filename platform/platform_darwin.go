//go:build darwin

package platform

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/ebitengine/purego"
	"github.com/ebitengine/purego/objc"
)

// NSRect is the Go representation of the Objective-C NSRect struct: an
// origin point and a size, both in CGFloat (double on 64-bit). It is 32
// bytes, which is larger than the two-eightbyte register limit, so on amd64
// it is returned through objc_msgSend_stret and on arm64 through the x8
// indirect-return register. purego's objc.Send[T] handles both ABIs — see
// the evidence in work/platform-02.md.
type NSRect struct {
	OriginX, OriginY float64
	Width, Height    float64
}

// NSPoint is the Go representation of NSPoint: two CGFloats, 16 bytes. It
// fits in two registers on both ABIs and does not go through stret.
type NSPoint struct {
	X, Y float64
}

// NSSize is the Go representation of NSSize: two CGFloats, 16 bytes.
type NSSize struct {
	Width, Height float64
}

// NSWindow style mask bits. These are the values AppKit defines in
// NSWindow.h; they are quoted from the API rather than named from memory.
const (
	styleMaskTitled              uint = 1 << 0
	styleMaskClosable            uint = 1 << 1
	styleMaskMiniaturizable      uint = 1 << 2
	styleMaskResizable           uint = 1 << 3
	styleMaskBorderless          uint = 0
	styleMaskFullScreen          uint = 1 << 14
	styleMaskFullSizeContentView uint = 1 << 15
)

// Backing store types for NSWindow.
const (
	backingRetained    uint = 1
	backingNonretained uint = 2
	backingBuffered    uint = 2
)

// NSApplication activation policies.
const (
	activationPolicyRegular    uint = 0
	activationPolicyAccessory  uint = 1
	activationPolicyProhibited uint = 2
)

// Cached selectors and classes, initialised once in initDarwin. Caching
// selectors is important: objc.RegisterName grabs the global Objective-C
// lock, and looking up the same selector on every event would serialise
// the event stream unnecessarily.
var (
	sel_sharedApplication         objc.SEL
	sel_setActivationPolicy       objc.SEL
	sel_activateIgnoringOtherApps objc.SEL
	sel_run                       objc.SEL
	sel_stop                      objc.SEL
	sel_terminate                 objc.SEL

	sel_alloc                      objc.SEL
	sel_initWithContentRect        objc.SEL
	sel_setTitle                   objc.SEL
	sel_setContentView             objc.SEL
	sel_contentView                objc.SEL
	sel_makeKeyAndOrderFront       objc.SEL
	sel_orderFront                 objc.SEL
	sel_orderOut                   objc.SEL
	sel_close                      objc.SEL
	sel_setFrameOrigin             objc.SEL
	sel_setFrameSize               objc.SEL
	sel_frame                      objc.SEL
	sel_setFrame_display_          objc.SEL
	sel_setMinSize                 objc.SEL
	sel_setMaxSize                 objc.SEL
	sel_setStyleMask               objc.SEL
	sel_setLevel                   objc.SEL
	sel_setBackgroundColor         objc.SEL
	sel_isVisible                  objc.SEL
	sel_isKeyWindow                objc.SEL
	sel_makeFirstResponder         objc.SEL
	sel_setAcceptsMouseMovedEvents objc.SEL

	sel_mouseDown      objc.SEL
	sel_mouseUp        objc.SEL
	sel_mouseMoved     objc.SEL
	sel_mouseDragged   objc.SEL
	sel_rightMouseDown objc.SEL
	sel_rightMouseUp   objc.SEL
	sel_otherMouseDown objc.SEL
	sel_otherMouseUp   objc.SEL
	sel_keyDown        objc.SEL
	sel_keyUp          objc.SEL

	sel_locationInWindow objc.SEL
	sel_buttonNumber     objc.SEL
	sel_modifierFlags    objc.SEL
	sel_keyCode          objc.SEL
	sel_characters       objc.SEL
	sel_timestamp        objc.SEL

	sel_UTF8String           objc.SEL
	sel_stringWithUTF8String objc.SEL

	class_NSApplication objc.Class
	class_NSWindow      objc.Class
	class_NSView        objc.Class
	class_NSEvent       objc.Class
	class_NSString      objc.Class
	class_NSColor       objc.Class

	// dispatch_async and dispatch_get_main_queue, loaded from libdispatch.
	dispatch_async          func(queue uintptr, block objc.Block)
	dispatch_get_main_queue func() uintptr

	// facetViewClass is the registered custom NSView subclass that
	// receives mouse and key events and forwards them to the owning
	// cocoaWindow's event handler.
	facetViewClass objc.Class

	darwinInitOnce sync.Once
)

// initDarwin loads the frameworks and registers the custom view class.
// It is called once, lazily, from New. purego's objc package loads
// libobjc in its own init; we load AppKit and Foundation here because the
// objc package only opens libobjc.
func initDarwin() {
	darwinInitOnce.Do(func() {
		// AppKit and Foundation are needed for NSApplication, NSWindow,
		// NSView, NSEvent and NSString. libobjc is already loaded by
		// purego/objc's init.
		if _, err := purego.Dlopen(
			"/System/Library/Frameworks/Foundation.framework/Foundation",
			purego.RTLD_GLOBAL|purego.RTLD_NOW,
		); err != nil {
			panic(fmt.Errorf("platform: load Foundation: %w", err))
		}
		if _, err := purego.Dlopen(
			"/System/Library/Frameworks/AppKit.framework/AppKit",
			purego.RTLD_GLOBAL|purego.RTLD_NOW,
		); err != nil {
			panic(fmt.Errorf("platform: load AppKit: %w", err))
		}
		if _, err := purego.Dlopen(
			"/usr/lib/libdispatch.dylib",
			purego.RTLD_GLOBAL|purego.RTLD_NOW,
		); err != nil {
			panic(fmt.Errorf("platform: load libdispatch: %w", err))
		}

		// Cache selectors. RegisterName grabs the global ObjC lock, so
		// doing this once at startup keeps the per-event path lock-free.
		sel_sharedApplication = objc.RegisterName("sharedApplication")
		sel_setActivationPolicy = objc.RegisterName("setActivationPolicy:")
		sel_activateIgnoringOtherApps = objc.RegisterName("activateIgnoringOtherApps:")
		sel_run = objc.RegisterName("run")
		sel_stop = objc.RegisterName("stop:")
		sel_terminate = objc.RegisterName("terminate:")

		sel_alloc = objc.RegisterName("alloc")
		sel_initWithContentRect = objc.RegisterName("initWithContentRect:styleMask:backing:defer:")
		sel_setTitle = objc.RegisterName("setTitle:")
		sel_setContentView = objc.RegisterName("setContentView:")
		sel_contentView = objc.RegisterName("contentView")
		sel_makeKeyAndOrderFront = objc.RegisterName("makeKeyAndOrderFront:")
		sel_orderFront = objc.RegisterName("orderFront:")
		sel_orderOut = objc.RegisterName("orderOut:")
		sel_close = objc.RegisterName("close")
		sel_setFrameOrigin = objc.RegisterName("setFrameOrigin:")
		sel_setFrameSize = objc.RegisterName("setFrameSize:")
		sel_frame = objc.RegisterName("frame")
		sel_setFrame_display_ = objc.RegisterName("setFrame:display:")
		sel_setMinSize = objc.RegisterName("setMinSize:")
		sel_setMaxSize = objc.RegisterName("setMaxSize:")
		sel_setStyleMask = objc.RegisterName("setStyleMask:")
		sel_setLevel = objc.RegisterName("setLevel:")
		sel_setBackgroundColor = objc.RegisterName("setBackgroundColor:")
		sel_isVisible = objc.RegisterName("isVisible")
		sel_isKeyWindow = objc.RegisterName("isKeyWindow")
		sel_makeFirstResponder = objc.RegisterName("makeFirstResponder:")
		sel_setAcceptsMouseMovedEvents = objc.RegisterName("setAcceptsMouseMovedEvents:")

		sel_mouseDown = objc.RegisterName("mouseDown:")
		sel_mouseUp = objc.RegisterName("mouseUp:")
		sel_mouseMoved = objc.RegisterName("mouseMoved:")
		sel_mouseDragged = objc.RegisterName("mouseDragged:")
		sel_rightMouseDown = objc.RegisterName("rightMouseDown:")
		sel_rightMouseUp = objc.RegisterName("rightMouseUp:")
		sel_otherMouseDown = objc.RegisterName("otherMouseDown:")
		sel_otherMouseUp = objc.RegisterName("otherMouseUp:")
		sel_keyDown = objc.RegisterName("keyDown:")
		sel_keyUp = objc.RegisterName("keyUp:")

		sel_locationInWindow = objc.RegisterName("locationInWindow")
		sel_buttonNumber = objc.RegisterName("buttonNumber")
		sel_modifierFlags = objc.RegisterName("modifierFlags")
		sel_keyCode = objc.RegisterName("keyCode")
		sel_characters = objc.RegisterName("characters")
		sel_timestamp = objc.RegisterName("timestamp")

		sel_UTF8String = objc.RegisterName("UTF8String")
		sel_stringWithUTF8String = objc.RegisterName("stringWithUTF8String:")

		class_NSApplication = objc.GetClass("NSApplication")
		class_NSWindow = objc.GetClass("NSWindow")
		class_NSView = objc.GetClass("NSView")
		class_NSEvent = objc.GetClass("NSEvent")
		class_NSString = objc.GetClass("NSString")
		class_NSColor = objc.GetClass("NSColor")

		// libdispatch functions for main-thread dispatch.
		purego.RegisterLibFunc(&dispatch_async, purego.RTLD_DEFAULT, "dispatch_async")
		purego.RegisterLibFunc(&dispatch_get_main_queue, purego.RTLD_DEFAULT, "dispatch_get_main_queue")

		facetViewClass = registerFacetViewClass()
	})
}

// registerFacetViewClass creates a custom NSView subclass that overrides
// mouse and key event methods. Each method looks up the cocoaWindow
// associated with the view (through a global map keyed by the view's id)
// and delivers an Event to its handler.
//
// The methods are registered through objc.RegisterClass, which creates a
// real Objective-C class in the runtime. The IMP for each method is a Go
// callback created by purego.NewCallback — no cgo is involved.
func registerFacetViewClass() objc.Class {
	mouseDownFn := func(id objc.ID, sel objc.SEL, event objc.ID) {
		if w := windowForView(id); w != nil {
			w.deliverPointerEvent(event, PointerDown)
		}
	}
	mouseUpFn := func(id objc.ID, sel objc.SEL, event objc.ID) {
		if w := windowForView(id); w != nil {
			w.deliverPointerEvent(event, PointerUp)
		}
	}
	mouseMovedFn := func(id objc.ID, sel objc.SEL, event objc.ID) {
		if w := windowForView(id); w != nil {
			w.deliverPointerEvent(event, PointerMove)
		}
	}
	mouseDraggedFn := func(id objc.ID, sel objc.SEL, event objc.ID) {
		if w := windowForView(id); w != nil {
			w.deliverPointerEvent(event, PointerMove)
		}
	}
	rightMouseDownFn := func(id objc.ID, sel objc.SEL, event objc.ID) {
		if w := windowForView(id); w != nil {
			w.deliverPointerEvent(event, PointerDown)
		}
	}
	rightMouseUpFn := func(id objc.ID, sel objc.SEL, event objc.ID) {
		if w := windowForView(id); w != nil {
			w.deliverPointerEvent(event, PointerUp)
		}
	}
	otherMouseDownFn := func(id objc.ID, sel objc.SEL, event objc.ID) {
		if w := windowForView(id); w != nil {
			w.deliverPointerEvent(event, PointerDown)
		}
	}
	otherMouseUpFn := func(id objc.ID, sel objc.SEL, event objc.ID) {
		if w := windowForView(id); w != nil {
			w.deliverPointerEvent(event, PointerUp)
		}
	}
	keyDownFn := func(id objc.ID, sel objc.SEL, event objc.ID) {
		if w := windowForView(id); w != nil {
			w.deliverKeyEvent(event, KeyDown)
		}
	}
	keyUpFn := func(id objc.ID, sel objc.SEL, event objc.ID) {
		if w := windowForView(id); w != nil {
			w.deliverKeyEvent(event, KeyUp)
		}
	}

	class, err := objc.RegisterClass("FacetView", class_NSView, nil, nil, []objc.MethodDef{
		{Cmd: sel_mouseDown, Fn: mouseDownFn},
		{Cmd: sel_mouseUp, Fn: mouseUpFn},
		{Cmd: sel_mouseMoved, Fn: mouseMovedFn},
		{Cmd: sel_mouseDragged, Fn: mouseDraggedFn},
		{Cmd: sel_rightMouseDown, Fn: rightMouseDownFn},
		{Cmd: sel_rightMouseUp, Fn: rightMouseUpFn},
		{Cmd: sel_otherMouseDown, Fn: otherMouseDownFn},
		{Cmd: sel_otherMouseUp, Fn: otherMouseUpFn},
		{Cmd: sel_keyDown, Fn: keyDownFn},
		{Cmd: sel_keyUp, Fn: keyUpFn},
	})
	if err != nil {
		panic(fmt.Errorf("platform: register FacetView class: %w", err))
	}
	return class
}

// viewWindows maps a FacetView's objc.ID to the cocoaWindow that owns it.
// The ObjC runtime holds no Go pointer — the view's id is an opaque
// Objective-C object, and this map is what lets the event callbacks
// recover the Go window. It is the same pattern the Windows backend uses
// with its HWND map, for the same reason: storing a Go pointer in OS
// storage hides it from the garbage collector.
//
// Accessed only on the main thread (the thread that runs the NSApplication
// event loop and creates/destroys windows), so no mutex is needed.
var viewWindows = map[objc.ID]*cocoaWindow{}

func windowForView(id objc.ID) *cocoaWindow {
	return viewWindows[id]
}

// cocoaPlatform is the macOS implementation of [Platform]. It owns the
// NSApplication instance and the main-thread dispatch mechanism.
type cocoaPlatform struct {
	options Options

	mu sync.Mutex // protects the fields below

	activationHandler func()
	quitHandler       func() bool
	displayHandler    func()

	displays []Display

	// app is the NSApplication instance, cached after first call to
	// sharedApplication. It is set in New and used in Run.
	app objc.ID
}

// New creates a macOS platform. It must be called on the goroutine that
// will run the platform — typically the main goroutine — because
// NSApplication must be created and run on the main thread. The goroutine's
// OS thread is locked for the duration.
func New(opts Options) (Platform, error) {
	if opts.Name == "" {
		opts.Name = "Facet"
	}

	runtime.LockOSThread()

	initDarwin()

	app := objc.ID(class_NSApplication).Send(sel_sharedApplication)
	if app == 0 {
		return nil, fmt.Errorf("initialise platform: NSApplication sharedApplication returned nil")
	}

	// Set the activation policy to Regular so the app appears in the Dock
	// and has a menu bar slot. Without this, windows appear but the app
	// does not behave like a normal application.
	app.Send(sel_setActivationPolicy, activationPolicyRegular)

	p := &cocoaPlatform{
		options: opts,
		app:     app,
		displays: []Display{
			{
				ID:          "main",
				Name:        "Main Display",
				ScaleFactor: 2.0, // Retina default; corrected at runtime when possible
				Primary:     true,
			},
		},
	}

	return p, nil
}

// Run starts the NSApplication event loop and blocks until Quit is called
// or the application is terminated. It must be called on the same goroutine
// that called New — the goroutine whose OS thread is locked to the main
// thread.
func (p *cocoaPlatform) Run() error {
	p.app.Send(sel_run)
	return nil
}

// Quit stops the event loop. It dispatches onto the main thread because
// NSApplication's stop: must be called from the main thread, and Quit may
// be called from any goroutine.
func (p *cocoaPlatform) Quit() {
	p.Dispatch(func() {
		p.app.Send(sel_stop, p.app)
	})
}

// Dispatch runs f on the main thread. It uses dispatch_async with the main
// queue, which wakes the NSApplication run loop and runs the block on the
// main thread. If called from the main thread, the block still runs
// asynchronously on a later turn of the run loop — matching the Windows
// backend's behaviour, where Dispatch from the platform thread runs on the
// next loop iteration.
func (p *cocoaPlatform) Dispatch(f func()) {
	block := objc.NewBlock(func(objc.Block) {
		f()
	})
	defer block.Release()
	dispatch_async(dispatch_get_main_queue(), block)
}

// NewWindow creates a native NSWindow from opts. The window is created on
// the main thread; this method blocks until creation completes.
func (p *cocoaPlatform) NewWindow(opts WindowOptions) (Window, error) {
	var (
		w   *cocoaWindow
		err error
		wg  sync.WaitGroup
	)
	wg.Add(1)
	p.Dispatch(func() {
		defer wg.Done()
		w, err = newCocoaWindow(p, opts)
	})
	wg.Wait()
	if err != nil {
		return nil, err
	}
	return w, nil
}

// Displays returns the currently attached displays.
func (p *cocoaPlatform) Displays() []Display {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.displays
}

// PrimaryDisplay returns the primary display.
func (p *cocoaPlatform) PrimaryDisplay() Display {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.displays) > 0 {
		return p.displays[0]
	}
	return Display{}
}

// ActiveDisplay returns the display that contains the currently focused
// window, or the primary display if no window is focused.
func (p *cocoaPlatform) ActiveDisplay() Display {
	return p.PrimaryDisplay()
}

// Clipboard returns the system clipboard.
func (p *cocoaPlatform) Clipboard() Clipboard {
	return cocoaClipboard{}
}

// SetCursorVisible shows or hides the pointer.
func (p *cocoaPlatform) SetCursorVisible(visible bool) {
	// TODO: NSCursor hide/unhide
}

// SetApplicationMenu sets the application menu. On macOS this is the
// screen menu bar. Not implemented in this round — the first macOS
// deliverable is a window with events, not menus.
func (p *cocoaPlatform) SetApplicationMenu(menu *Menu) {
	// TODO: build NSMenu from Menu tree and set as main menu
}

// NewSystemTray creates a system tray icon.
func (p *cocoaPlatform) NewSystemTray(opts SystemTrayOptions) (SystemTray, error) {
	return nil, fmt.Errorf("system tray: not implemented")
}

// ShowMessageDialog shows a modal message dialog.
func (p *cocoaPlatform) ShowMessageDialog(dialog MessageDialog) (DialogResult, error) {
	return 0, fmt.Errorf("message dialog: not implemented")
}

// ShowOpenDialog shows a modal file-open dialog.
func (p *cocoaPlatform) ShowOpenDialog(dialog OpenFileDialog) ([]string, error) {
	return nil, fmt.Errorf("open dialog: not implemented")
}

// ShowSaveDialog shows a modal file-save dialog.
func (p *cocoaPlatform) ShowSaveDialog(dialog SaveFileDialog) (string, error) {
	return "", fmt.Errorf("save dialog: not implemented")
}

// SendNotification displays a system notification.
func (p *cocoaPlatform) SendNotification(notification Notification) error {
	return fmt.Errorf("notification: not implemented")
}

// Activate brings the application to the foreground.
func (p *cocoaPlatform) Activate() {
	p.Dispatch(func() {
		p.app.Send(sel_activateIgnoringOtherApps, true)
	})
}

// Hide hides all application windows.
func (p *cocoaPlatform) Hide() {
	// TODO: NSApplication hide:
}

// Show restores windows hidden by Hide.
func (p *cocoaPlatform) Show() {
	// TODO: NSApplication unhide:
}

// SetIcon sets the application icon.
func (p *cocoaPlatform) SetIcon(icon []byte) {
	// TODO: NSImage from data, NSApplication setApplicationIconImage:
}

// SetActivationHandler sets a handler called when the application is
// activated.
func (p *cocoaPlatform) SetActivationHandler(handler func()) {
	p.mu.Lock()
	p.activationHandler = handler
	p.mu.Unlock()
}

// SetQuitHandler sets a handler called when the user requests the
// application to quit.
func (p *cocoaPlatform) SetQuitHandler(handler func() bool) {
	p.mu.Lock()
	p.quitHandler = handler
	p.mu.Unlock()
}

// SetDisplayChangeHandler sets a handler called when the display
// configuration changes.
func (p *cocoaPlatform) SetDisplayChangeHandler(handler func()) {
	p.mu.Lock()
	p.displayHandler = handler
	p.mu.Unlock()
}

// primaryScale returns the primary display's scale factor, used as the
// initial scale for new windows.
func (p *cocoaPlatform) primaryScale() float32 {
	d := p.PrimaryDisplay()
	if d.ScaleFactor == 0 {
		return 2.0 // Retina default on macOS
	}
	return d.ScaleFactor
}

// cocoaClipboard is a stub clipboard implementation. The real
// implementation will use NSPasteboard; this is a placeholder so the
// Platform interface is satisfied.
type cocoaClipboard struct{}

func (cocoaClipboard) Text() (string, error) {
	return "", fmt.Errorf("clipboard: not implemented")
}

func (cocoaClipboard) SetText(text string) error {
	return fmt.Errorf("clipboard: not implemented")
}
