//go:build windows

// Vendored from Wails v3: v3/pkg/w32/dialogs.go.
// https://github.com/wailsapp/wails
//
// MIT License -- see NOTICE for the full text.

package w32

import (
	"unsafe"
)

func MessageBoxWithIcon(hwnd HWND, text *uint16, caption *uint16, iconID int, flags uint32) (int32, error) {

	params := MSGBOXPARAMS{
		cbSize:      uint32(unsafe.Sizeof(MSGBOXPARAMS{})),
		hwndOwner:   hwnd,
		hInstance:   GetApplicationHandle(),
		lpszText:    text,
		lpszCaption: caption,
		dwStyle:     flags,
		lpszIcon:    (*uint16)(unsafe.Pointer(uintptr(iconID))),
	}

	r, _, err := procMessageBoxIndirect.Call(
		uintptr(unsafe.Pointer(&params)),
	)
	if r == 0 {
		return 0, err
	}
	return int32(r), nil
}
