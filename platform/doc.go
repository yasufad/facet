// Package platform is the operating-system layer of Facet.
//
// It owns the native event loop, windows, main-thread dispatch, displays,
// cursors, clipboard, menus, tray, dialogs, notifications, and the raw input
// stream. It is the only package permitted [unsafe], and only for platform
// calls; everything above it is ordinary Go.
//
// # Two constraints shape the interface
//
// First: a native handle, never a graphics API. [Platform] hands out a window
// handle — an HWND on Windows, an NSWindow* on macOS, a GtkWidget* on Linux —
// and stops there. [Window.NativeHandle] returns it as a uintptr so no
// unsafe.Pointer crosses the layer boundary. The render package takes that
// handle and owns the device, the swapchain and the shaders. platform must not
// import render and must not know D3D, Metal or Vulkan exist. That is what
// keeps a second graphics backend a package rather than a rewrite.
//
// Second: input is a stream, not callbacks into the framework. platform reads
// the native event stream and surfaces events as typed [Event] values; the
// input and window packages decide what they mean. Nothing above platform has
// to know that Windows reports wheel deltas in multiples of 120, or that macOS
// delivers key events through an NSEvent monitor. The [Window] event handler
// receives events synchronously on the platform thread, in arrival order; the
// platform translates and forgets.
//
// # Threading
//
// The platform event loop runs on a single OS thread, locked with
// [runtime.LockOSThread]. That thread is the UI goroutine: the entity map is
// single-goroutine by design, and the event loop is where it lives. [Run]
// blocks on that thread until [Quit] is called.
//
// [Dispatch] runs a closure on the platform thread from any goroutine. The
// foreground executor in package app dispatches its results through it: the
// window package — which imports both app and platform — wires the executor's
// wake channel to Dispatch so background work returns to the UI goroutine.
// platform itself never imports app; the connection is made above, not below.
//
// # Backends
//
// The interface lives in this package; each backend is a set of files behind
// build tags, following the layout Wails uses: <feature>_{windows,darwin,linux}.go.
// The Windows backend reaches Win32 through syscall and
// golang.org/x/sys/windows, with no cgo. macOS and Linux will go through
// github.com/ebitengine/purego. CGO_ENABLED=0 must build on every target.
//
// # Invariants
//
//   - Imports only geometry, colour and third_party, enforced by the layering
//     test.
//   - No cgo, here or anywhere. unsafe is permitted only for platform calls
//     and must not leak upward through the interface.
//   - Surfaces a window handle and an input stream, never a rendering API.
//   - Platform and Window are layer boundaries; they change by explicit
//     decision, never as a side effect of an implementation.
package platform
