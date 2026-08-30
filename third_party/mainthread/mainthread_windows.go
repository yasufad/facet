//go:build windows

// Package mainthread dispatches closures onto the Windows main thread by
// posting a registered message to a hidden window.
//
// Vendored from Wails v3: v3/pkg/application/mainthread_windows.go and
// v3/pkg/application/mainthread.go.
// https://github.com/wailsapp/wails
//
// MIT License -- see NOTICE for the full text.
//
// Modified from the original: the Wails code is a method on windowsApp that
// shares global state with the rest of the application package. This is a
// standalone package with its own Dispatcher type, so the platform backend
// owns the instance rather than reaching into a global. The technique and
// the citations are unchanged.
//
// Further modified: the hidden window is created in Run, not in New. The
// window belongs to the thread that created it, and GetMessage only
// retrieves messages for the calling thread's windows. Creating the window
// in New (on the caller's thread) and running the loop in Run (on a
// different goroutine's thread) left PostMessage posting to a thread whose
// queue GetMessage never read. New now just stores the class name; Run
// creates the window on the thread it will pump.
package mainthread

import (
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"

	"github.com/yasufad/facet/third_party/w32"
)

// wmInvokeCallback is the registered window message used to signal that
// dispatched closures are waiting. RegisterWindowMessage returns a value
// unique to the message name, guaranteed consistent across all windows in
// the system.
var wmInvokeCallback = w32.RegisterWindowMessage(w32.MustStringToUTF16Ptr("Facet.InvokeCallback"))

// Dispatcher runs closures on the thread that created its hidden window.
// The hidden window is the mechanism, not a destination: PostMessage to a
// thread queue can be swallowed by a modal inner loop, but PostMessage to a
// window is delivered when the modal loop pumps messages. That is a bug
// Wails hit in v2 and fixed with this approach.
//
// See: https://github.com/wailsapp/wails/issues/969
// See: https://devblogs.microsoft.com/oldnewthing/20050426-18/?p=35783
// See also: https://learn.microsoft.com/en-us/windows/win32/winmsg/using-messages-and-message-queues#creating-a-message-loop
// > Because the system directs messages to individual windows in an
// > application, a thread must create at least one window before starting
// > its message loop.
type Dispatcher struct {
	hwnd      w32.HWND
	threadID  w32.HANDLE
	className *uint16

	mu   sync.Mutex
	fns  map[uint32]func()
	next uint32
}

// New creates a Dispatcher. The hidden window is not created here; it is
// created in Run, on the thread that will pump the message loop, because the
// window belongs to the thread that created it and GetMessage only retrieves
// messages for the calling thread's windows. The class name should be unique
// to the application.
func New(className string) *Dispatcher {
	return &Dispatcher{
		className: w32.MustStringToUTF16Ptr(className),
		fns:       make(map[uint32]func()),
	}
}

// dispatchWndProc is the window procedure for the hidden window. It handles
// only the invoke-callback message; everything else goes to DefWindowProc.
// The Dispatcher is recovered from a global, because Win32 wndproc functions
// receive no user data — only the HWND, message, wParam and lParam.
var globalDispatcher atomic.Pointer[Dispatcher]

func dispatchWndProc(hwnd w32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if msg == wmInvokeCallback {
		if d := globalDispatcher.Load(); d != nil {
			d.invokeCallbacks()
		}
		return 0
	}
	return w32.DefWindowProc(hwnd, msg, wParam, lParam)
}

// Run starts the message loop. It blocks until Quit is called (which posts
// WM_QUIT) or the loop exits for another reason. It must be called on the
// thread that will serve as the main thread; that thread is locked to the OS
// thread and the hidden window is created here, on that thread, because the
// window belongs to the thread that created it and GetMessage only retrieves
// messages for the calling thread's windows.
func (d *Dispatcher) Run() int {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Register the window class and create the hidden window on this
	// thread, so the window's thread is the one pumping messages below.
	wcx := w32.WNDCLASSEX{
		Size:      uint32(unsafe.Sizeof(w32.WNDCLASSEX{})),
		WndProc:   syscall.NewCallback(w32.WindowProc(dispatchWndProc)),
		Instance:  w32.GetModuleHandle(""),
		ClassName: d.className,
	}
	w32.RegisterClassEx(&wcx)

	hwnd := w32.CreateWindowEx(
		0,
		d.className,
		w32.MustStringToUTF16Ptr("__facet_hidden_mainthread"),
		w32.WS_DISABLED,
		w32.CW_USEDEFAULT,
		w32.CW_USEDEFAULT,
		0,
		0,
		0,
		0,
		w32.GetModuleHandle(""),
		nil,
	)
	if hwnd == 0 {
		panic("mainthread: CreateWindowEx failed for hidden window")
	}
	d.hwnd = hwnd

	threadID, _ := w32.GetWindowThreadProcessId(hwnd)
	d.threadID = threadID

	globalDispatcher.Store(d)
	defer globalDispatcher.Store(nil)

	msg := (*w32.MSG)(unsafe.Pointer(w32.GlobalAlloc(0, uint32(unsafe.Sizeof(w32.MSG{})))))
	defer w32.GlobalFree(w32.HGLOBAL(unsafe.Pointer(msg)))

	for w32.GetMessage(msg, 0, 0, 0) != 0 {
		w32.TranslateMessage(msg)
		w32.DispatchMessage(msg)
	}

	return int(msg.WParam)
}

// Quit posts WM_QUIT, causing Run to return after the current iteration.
func (d *Dispatcher) Quit() {
	w32.PostQuitMessage(0)
}

// Dispatch queues f to run on the main thread. If called from the main
// thread, f is posted to the hidden window and runs on the next loop
// iteration. If called from another thread, f is posted and runs when the
// loop next pumps messages.
//
// The post goes to the hidden window rather than the thread queue because a
// modal inner loop (a dialog, a context menu) swallows thread-queued
// messages but pumps window messages. This is the bug Wails hit in v2.
func (d *Dispatcher) Dispatch(f func()) {
	d.mu.Lock()
	id := d.next
	d.next++
	d.fns[id] = f
	d.mu.Unlock()

	w32.PostMessage(d.hwnd, wmInvokeCallback, 0, 0)
}

// IsMainThread reports whether the caller is on the main thread.
func (d *Dispatcher) IsMainThread() bool {
	return d.threadID == w32.GetCurrentThreadId()
}

// invokeCallbacks runs all queued closures. It is called from the hidden
// window's wndproc when the invoke-callback message arrives. It must be
// called on the main thread.
func (d *Dispatcher) invokeCallbacks() {
	d.mu.Lock()
	ids := make([]uint32, 0, len(d.fns))
	for id := range d.fns {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	fns := make([]func(), len(ids))
	for i, id := range ids {
		fns[i] = d.fns[id]
		delete(d.fns, id)
	}
	d.mu.Unlock()

	for _, fn := range fns {
		fn()
	}
}
