// Package platformtest provides exported test doubles for the platform
// package, so packages above platform (window, ui, internal/integration) can
// test against a platform.Window without each holding a private stub that
// breaks every time platform.Window gains a method.
//
// It is the platform counterpart to element/elementtest: one exported double
// turns a new interface method from a three-party handshake across agents who
// do not own each other's files into one package's commit.
package platformtest

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/platform"
)

// Window is an exported test double implementing platform.Window. The zero
// value is a usable window with a 1.0 scale factor; fields are exported so a
// test can set the values it wants to read back, and read what a method under
// test wrote.
//
// Methods that set a value store it in the corresponding field; methods that
// read return the field. SetEventHandler and SetCloseHandler store the
// closures so a test can call them directly.
type Window struct {
	SizeVal      geometry.Size[geometry.Pixels]
	PosVal       geometry.Point[geometry.Pixels]
	Scale        float32
	StateVal     platform.WindowState
	Cursors      []platform.Cursor
	Visible      bool
	Focused      bool
	NativeHnd    uintptr
	NativeSurf   uintptr
	EventHandler func(platform.Event)
	CloseHandler func() bool
}

// Ensure Window implements platform.Window.
var _ platform.Window = (*Window)(nil)

// NewWindow constructs a test Window with the given client-area size and
// scale factor. IsVisible and IsFocused default to true, matching what a
// real window reports once shown and focused.
func NewWindow(size geometry.Size[geometry.Pixels], scale float32) *Window {
	return &Window{
		SizeVal: size,
		Scale:   scale,
		Visible: true,
		Focused: true,
	}
}

func (w *Window) Show()                                           {}
func (w *Window) Hide()                                           {}
func (w *Window) Close()                                          {}
func (w *Window) SetTitle(title string)                           {}
func (w *Window) SetSize(size geometry.Size[geometry.Pixels])     { w.SizeVal = size }
func (w *Window) Size() geometry.Size[geometry.Pixels]            { return w.SizeVal }
func (w *Window) SetPosition(pos geometry.Point[geometry.Pixels]) { w.PosVal = pos }
func (w *Window) Position() geometry.Point[geometry.Pixels]       { return w.PosVal }
func (w *Window) SetMinSize(size geometry.Size[geometry.Pixels])  {}
func (w *Window) SetMaxSize(size geometry.Size[geometry.Pixels])  {}
func (w *Window) SetResizable(resizable bool)                     {}
func (w *Window) SetAlwaysOnTop(onTop bool)                       {}
func (w *Window) State() platform.WindowState                     { return w.StateVal }
func (w *Window) SetState(state platform.WindowState)             { w.StateVal = state }
func (w *Window) SetBackground(c colour.Rgba)                     {}
func (w *Window) ScaleFactor() float32                            { return w.Scale }
func (w *Window) NativeHandle() uintptr                           { return w.NativeHnd }
func (w *Window) NativeSurface() uintptr                          { return w.NativeSurf }
func (w *Window) Focus()                                          {}
func (w *Window) IsFocused() bool                                 { return w.Focused }
func (w *Window) IsVisible() bool                                 { return w.Visible }
func (w *Window) SetCursor(shape platform.Cursor) {
	w.Cursors = append(w.Cursors, shape)
}
func (w *Window) SetEventHandler(h func(platform.Event)) {
	w.EventHandler = h
}
func (w *Window) SetCloseHandler(h func() bool) {
	w.CloseHandler = h
}
