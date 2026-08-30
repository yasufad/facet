//go:build windows

// Vendored from Wails v3: v3/pkg/w32/shlwapi.go.
// https://github.com/wailsapp/wails
//
// MIT License -- see NOTICE for the full text.

package w32

import (
	"syscall"
	"unsafe"
)

var (
	modshlwapi = syscall.NewLazyDLL("shlwapi.dll")

	procSHCreateMemStream = modshlwapi.NewProc("SHCreateMemStream")
)

func SHCreateMemStream(data []byte) (uintptr, error) {
	ret, _, err := procSHCreateMemStream.Call(
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
	)
	if ret == 0 {
		return 0, err
	}

	return ret, nil
}
