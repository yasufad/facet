//go:build windows

// Not vendored from Wails: IMM32 has no counterpart in Wails v3's w32
// package because Wails' webview owns IME composition itself, so its w32
// bindings never needed to read a composition string. Facet draws
// composition text inline (see platform.IMECompositionEvent), which needs
// the Input Method Manager API directly. Added here, in the one-file-per-DLL
// layout the rest of this package follows, and recorded in
// third_party/README alongside the package's other additions.

package w32

import (
	"syscall"
	"unsafe"
)

// HIMC is a handle to an input context, returned by ImmGetContext.
type HIMC = HANDLE

// GCS_COMPSTR and GCS_CURSORPOS select which part of the composition
// ImmGetCompositionString reports: the in-progress composition string, and
// the cursor's position within it.
const (
	GCS_COMPSTR   = 0x0008
	GCS_CURSORPOS = 0x0080
)

var (
	modimm32 = syscall.NewLazyDLL("imm32.dll")

	procImmGetContext            = modimm32.NewProc("ImmGetContext")
	procImmReleaseContext        = modimm32.NewProc("ImmReleaseContext")
	procImmGetCompositionStringW = modimm32.NewProc("ImmGetCompositionStringW")
)

// ImmGetContext returns the input context associated with hwnd. The caller
// must release it with ImmReleaseContext once done.
func ImmGetContext(hwnd HWND) HIMC {
	ret, _, _ := procImmGetContext.Call(uintptr(hwnd))
	return HIMC(ret)
}

// ImmReleaseContext releases an input context obtained from ImmGetContext.
func ImmReleaseContext(hwnd HWND, himc HIMC) {
	procImmReleaseContext.Call(uintptr(hwnd), uintptr(himc))
}

// ImmGetCompositionString returns the UTF-16 code units of the composition
// string, or cursor position when gcs is GCS_CURSORPOS, without decoding it:
// the caller is closer to knowing which of those two it asked for and what
// unit it wants back.
//
// Called twice, as ImmGetCompositionStringW itself requires: once with a nil
// buffer to learn the required size, then again with a buffer of that size.
// The two calls happen under the same composition, on the platform thread,
// so the size cannot change between them.
func ImmGetCompositionString(himc HIMC, gcs uint32) []uint16 {
	n, _, _ := procImmGetCompositionStringW.Call(uintptr(himc), uintptr(gcs), 0, 0)
	byteLen := int32(n)
	if byteLen <= 0 {
		return nil
	}
	units := make([]uint16, byteLen/2)
	// Sound: units is a Go-owned buffer sized from the byte length
	// ImmGetCompositionStringW itself just reported, and the call writes at
	// most that many bytes into it before returning on the same thread.
	procImmGetCompositionStringW.Call(
		uintptr(himc), uintptr(gcs),
		uintptr(unsafe.Pointer(&units[0])), uintptr(byteLen),
	)
	return units
}

// ImmGetCompositionCursor returns the cursor position within the composition
// string, as a UTF-16 code-unit offset, or -1 if the IME reports none.
func ImmGetCompositionCursor(himc HIMC) int {
	ret, _, _ := procImmGetCompositionStringW.Call(uintptr(himc), GCS_CURSORPOS, 0, 0)
	cursor := int32(ret)
	if cursor < 0 {
		return -1
	}
	return int(cursor)
}
