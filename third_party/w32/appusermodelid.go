//go:build windows

// Not vendored from Wails: SetCurrentProcessExplicitAppUserModelID has no
// counterpart in Wails v3's w32 package. Facet needs it for Windows toast
// notifications, which require an AppUserModelID registered for the
// process before any UI is shown.

package w32

import "unsafe"

// procSetCurrentProcessExplicitAppUserModelID is the shell32 procedure for
// SetCurrentProcessExplicitAppUserModelID. modshell32 is defined in
// shell32.go (vendored from Wails); this proc is Facet's own addition.
var procSetCurrentProcessExplicitAppUserModelID = modshell32.NewProc("SetCurrentProcessExplicitAppUserModelID")

// SetCurrentProcessExplicitAppUserModelID sets the AppUserModelID for the
// current process. The shell uses the AppUserModelID for taskbar grouping,
// jump lists, and toast notification attribution. It must be called early
// in the process lifetime, before any windows are created, and the ID
// should correspond to a Start Menu shortcut for notifications to display
// properly.
//
// Returns S_OK (0) on success, or an error HRESULT otherwise.
func SetCurrentProcessExplicitAppUserModelID(appID string) HRESULT {
	ptr := MustStringToUTF16Ptr(appID)
	ret, _, _ := procSetCurrentProcessExplicitAppUserModelID.Call(
		uintptr(unsafe.Pointer(ptr)))
	return HRESULT(ret)
}
