//go:build windows

// Not vendored from Wails: the Common Item Dialog (IFileOpenDialog and the
// interfaces around it) has no counterpart in Wails v3's w32 package,
// because Wails shows file pickers through its webview's own dialog
// integration. Facet has no webview, so it talks to shell32's COM dialog
// directly.
//
// The vtable layout and GUIDs below are not from memory: they were checked
// against two independent, shipping implementations that could not work in
// production if they were wrong -- SDL3's
// src/dialog/windows/SDL_windowsdialog.c, and Wine's include/shobjidl.idl
// (uuid attributes on IShellItem and IFileOpenDialog, and the
// FileOpenDialog coclass). Both agree on every field, order and GUID here.
// Vendored code follows this package's file-per-DLL layout; this file is
// analogous in shape but is Facet's own, reached the same way
// third_party/w32/taskbar.go reaches ITaskbarList3 -- CoCreateInstance and a
// vtable of uintptr, not a *_Vtbl of ComProc.

package w32

import (
	"syscall"
	"unsafe"
)

// GUIDs for the Common Item Dialog. Checked against Wine's
// include/shobjidl.idl (uuid attributes) and the Windows SDK's
// ShObjIdl_core.idl (coclass uuid attributes).
var (
	CLSID_FileOpenDialog = syscall.GUID{Data1: 0xDC1C5A9C, Data2: 0xE88A, Data3: 0x4DDE, Data4: [8]byte{0xA5, 0xA1, 0x60, 0xF8, 0x2A, 0x20, 0xAE, 0xF7}}
	IID_IFileOpenDialog  = syscall.GUID{Data1: 0xD57C7288, Data2: 0xD4AD, Data3: 0x4768, Data4: [8]byte{0xBE, 0x02, 0x9D, 0x96, 0x95, 0x32, 0xD9, 0x60}}
	IID_IShellItem       = syscall.GUID{Data1: 0x43826D1E, Data2: 0xE718, Data3: 0x42EE, Data4: [8]byte{0xBC, 0x55, 0xA1, 0xE2, 0x61, 0xC3, 0x7B, 0xFE}}

	// CLSID_FileSaveDialog and IID_IFileSaveDialog are the coclass and
	// interface IDs for the save-file Common Item Dialog. Checked against
	// Wine's include/shobjidl.idl (uuid attribute on IFileSaveDialog and the
	// FileSaveDialog coclass) and the Windows SDK ShObjIdl_core.idl — both
	// sources agree on these values.
	CLSID_FileSaveDialog = syscall.GUID{Data1: 0xC0B4E2F3, Data2: 0xBA21, Data3: 0x4773, Data4: [8]byte{0x8D, 0xBA, 0x33, 0x5E, 0xC9, 0x46, 0xEB, 0x8B}}
	IID_IFileSaveDialog  = syscall.GUID{Data1: 0x84BCCD23, Data2: 0x5FDE, Data3: 0x4CDB, Data4: [8]byte{0xAE, 0xA4, 0xAF, 0x64, 0xB8, 0x3D, 0x78, 0xAB}}
)

// FOS_* are IFileDialog::SetOptions flags. Only the ones Facet sets.
const (
	FOS_PATHMUSTEXIST    = 0x00000800
	FOS_FILEMUSTEXIST    = 0x00001000
	FOS_ALLOWMULTISELECT = 0x00000200
	FOS_FORCEFILESYSTEM  = 0x00000040
	// FOS_OVERWRITEPROMPT prompts the user before overwriting an existing
	// file. Used for save dialogs; has no effect on open dialogs.
	// Value confirmed against the FILEOPENDIALOGOPTIONS enum in the
	// Windows SDK's ShObjIdl_core.h.
	FOS_OVERWRITEPROMPT = 0x00000002
)

// SIGDN_FILESYSPATH asks IShellItem::GetDisplayName for an absolute
// filesystem path rather than a display name or shell namespace URL.
const SIGDN_FILESYSPATH = 0x80058000

// hrErrorCancelled is what IFileOpenDialog::Show returns when the user
// dismisses the dialog without choosing anything.
// HRESULT_FROM_WIN32(ERROR_CANCELLED): (1223 & 0xFFFF) | (FACILITY_WIN32<<16) | 0x80000000.
const hrErrorCancelled = 0x800704C7

// IsDialogCancelled reports whether hr is the HRESULT IFileOpenDialog::Show
// returns when the user cancels, as opposed to a real failure.
func IsDialogCancelled(hr HRESULT) bool {
	return uint32(hr) == hrErrorCancelled
}

// COMDLG_FILTERSPEC is one entry in the array IFileDialog::SetFileTypes
// takes: a human-readable label and a semicolon-separated pattern list
// ("*.txt;*.md").
type COMDLG_FILTERSPEC struct {
	PszName *uint16
	PszSpec *uint16
}

// IFileOpenDialog wraps CLSID_FileOpenDialog. Create with CoCreateInstance
// against CLSID_FileOpenDialog and IID_IFileOpenDialog.
type IFileOpenDialog struct {
	lpVtbl *iFileOpenDialogVtbl
}

// iFileOpenDialogVtbl is IFileOpenDialog's vtable, laid out in full so every
// field lands on the slot the real interface puts it in even though only
// some methods have a Go wrapper below -- a vtable field skipped in the
// middle would shift every method after it onto the wrong function pointer.
type iFileOpenDialogVtbl struct {
	QueryInterface      uintptr
	AddRef              uintptr
	Release             uintptr
	Show                uintptr
	SetFileTypes        uintptr
	SetFileTypeIndex    uintptr
	GetFileTypeIndex    uintptr
	Advise              uintptr
	Unadvise            uintptr
	SetOptions          uintptr
	GetOptions          uintptr
	SetDefaultFolder    uintptr
	SetFolder           uintptr
	GetFolder           uintptr
	GetCurrentSelection uintptr
	SetFileName         uintptr
	GetFileName         uintptr
	SetTitle            uintptr
	SetOkButtonLabel    uintptr
	SetFileNameLabel    uintptr
	GetResult           uintptr
	AddPlace            uintptr
	SetDefaultExtension uintptr
	Close               uintptr
	SetClientGuid       uintptr
	ClearClientData     uintptr
	SetFilter           uintptr
	GetResults          uintptr
	GetSelectedItems    uintptr
}

func (d *IFileOpenDialog) Release() {
	syscall.SyscallN(d.lpVtbl.Release, uintptr(unsafe.Pointer(d)))
}

// Show displays the dialog modally, blocking until the user chooses or
// cancels. hwndOwner may be 0 for no owner window.
func (d *IFileOpenDialog) Show(hwndOwner HWND) HRESULT {
	r, _, _ := syscall.SyscallN(d.lpVtbl.Show, uintptr(unsafe.Pointer(d)), uintptr(hwndOwner))
	return HRESULT(r)
}

// SetFileTypes sets the filter list. specs must outlive the call.
func (d *IFileOpenDialog) SetFileTypes(specs []COMDLG_FILTERSPEC) HRESULT {
	if len(specs) == 0 {
		return 0
	}
	r, _, _ := syscall.SyscallN(d.lpVtbl.SetFileTypes,
		uintptr(unsafe.Pointer(d)), uintptr(len(specs)), uintptr(unsafe.Pointer(&specs[0])))
	return HRESULT(r)
}

func (d *IFileOpenDialog) SetOptions(fos uint32) HRESULT {
	r, _, _ := syscall.SyscallN(d.lpVtbl.SetOptions, uintptr(unsafe.Pointer(d)), uintptr(fos))
	return HRESULT(r)
}

func (d *IFileOpenDialog) GetOptions() (uint32, HRESULT) {
	var fos uint32
	r, _, _ := syscall.SyscallN(d.lpVtbl.GetOptions, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(&fos)))
	return fos, HRESULT(r)
}

func (d *IFileOpenDialog) SetFolder(psi *IShellItem) HRESULT {
	r, _, _ := syscall.SyscallN(d.lpVtbl.SetFolder, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(psi)))
	return HRESULT(r)
}

func (d *IFileOpenDialog) SetTitle(title *uint16) HRESULT {
	r, _, _ := syscall.SyscallN(d.lpVtbl.SetTitle, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(title)))
	return HRESULT(r)
}

// GetResult returns the chosen item for a single-selection dialog.
func (d *IFileOpenDialog) GetResult() (*IShellItem, HRESULT) {
	var item *IShellItem
	r, _, _ := syscall.SyscallN(d.lpVtbl.GetResult, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(&item)))
	return item, HRESULT(r)
}

// GetResults returns every chosen item for a dialog opened with
// FOS_ALLOWMULTISELECT.
func (d *IFileOpenDialog) GetResults() (*IShellItemArray, HRESULT) {
	var arr *IShellItemArray
	r, _, _ := syscall.SyscallN(d.lpVtbl.GetResults, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(&arr)))
	return arr, HRESULT(r)
}

// IShellItem names a single filesystem or shell-namespace item.
type IShellItem struct {
	lpVtbl *iShellItemVtbl
}

type iShellItemVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	BindToHandler  uintptr
	GetParent      uintptr
	GetDisplayName uintptr
	GetAttributes  uintptr
	Compare        uintptr
}

func (s *IShellItem) Release() {
	syscall.SyscallN(s.lpVtbl.Release, uintptr(unsafe.Pointer(s)))
}

// GetDisplayName renders the item as sigdn (typically SIGDN_FILESYSPATH).
// The string returned by the OS is freed here; the caller only sees the Go
// copy.
func (s *IShellItem) GetDisplayName(sigdn uint32) (string, HRESULT) {
	var p *uint16
	r, _, _ := syscall.SyscallN(s.lpVtbl.GetDisplayName,
		uintptr(unsafe.Pointer(s)), uintptr(sigdn), uintptr(unsafe.Pointer(&p)))
	if HRESULT(r) != 0 {
		return "", HRESULT(r)
	}
	// Sound: p is a COM-allocated string the call above told us it owns;
	// CoTaskMemFree is the documented way to release it, and only after
	// copying it into a Go string do we free the OS's copy.
	defer CoTaskMemFree(unsafe.Pointer(p))
	return UTF16PtrToString(p), 0
}

// IShellItemArray holds the results of a multi-selection dialog.
type IShellItemArray struct {
	lpVtbl *iShellItemArrayVtbl
}

type iShellItemArrayVtbl struct {
	QueryInterface             uintptr
	AddRef                     uintptr
	Release                    uintptr
	BindToHandler              uintptr
	GetPropertyStore           uintptr
	GetPropertyDescriptionList uintptr
	GetAttributes              uintptr
	GetCount                   uintptr
	GetItemAt                  uintptr
	EnumItems                  uintptr
}

func (a *IShellItemArray) Release() {
	syscall.SyscallN(a.lpVtbl.Release, uintptr(unsafe.Pointer(a)))
}

func (a *IShellItemArray) GetCount() (uint32, HRESULT) {
	var n uint32
	r, _, _ := syscall.SyscallN(a.lpVtbl.GetCount, uintptr(unsafe.Pointer(a)), uintptr(unsafe.Pointer(&n)))
	return n, HRESULT(r)
}

func (a *IShellItemArray) GetItemAt(index uint32) (*IShellItem, HRESULT) {
	var item *IShellItem
	r, _, _ := syscall.SyscallN(a.lpVtbl.GetItemAt,
		uintptr(unsafe.Pointer(a)), uintptr(index), uintptr(unsafe.Pointer(&item)))
	return item, HRESULT(r)
}

// IFileSaveDialog wraps CLSID_FileSaveDialog. Create with CoCreateInstance
// against CLSID_FileSaveDialog and IID_IFileSaveDialog.
//
// The vtable layout follows the C-interface struct in the Windows SDK's
// ShObjIdl_core.h, not the alphabetised method list on Microsoft Learn —
// those two orderings differ, and a wrong slot is a legal call to the wrong
// method rather than a crash. IFileSaveDialog extends IFileDialog, so the
// 27 inherited slots come first (QueryInterface through SetFilter), followed
// by the 5 IFileSaveDialog additions in the order they appear in the SDK
// header: SetSaveAsItem, SetProperties, SetCollectedProperties, GetProperties,
// ApplyProperties. Wine's include/shobjidl.idl confirms the same order and
// the same GUIDs.
type IFileSaveDialog struct {
	lpVtbl *iFileSaveDialogVtbl
}

// iFileSaveDialogVtbl is IFileSaveDialog's vtable, laid out in full so every
// field lands on the slot the real interface puts it in even though only
// some methods have a Go wrapper — a vtable field skipped in the middle
// shifts every subsequent method onto the wrong function pointer.
type iFileSaveDialogVtbl struct {
	// Slots 0-2: IUnknown
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	// Slot 3: IModalWindow
	Show uintptr
	// Slots 4-26: IFileDialog (in declaration order from the SDK header)
	SetFileTypes        uintptr
	SetFileTypeIndex    uintptr
	GetFileTypeIndex    uintptr
	Advise              uintptr
	Unadvise            uintptr
	SetOptions          uintptr
	GetOptions          uintptr
	SetDefaultFolder    uintptr
	SetFolder           uintptr
	GetFolder           uintptr
	GetCurrentSelection uintptr
	SetFileName         uintptr
	GetFileName         uintptr
	SetTitle            uintptr
	SetOkButtonLabel    uintptr
	SetFileNameLabel    uintptr
	GetResult           uintptr
	AddPlace            uintptr
	SetDefaultExtension uintptr
	Close               uintptr
	SetClientGuid       uintptr
	ClearClientData     uintptr
	SetFilter           uintptr
	// Slots 27-31: IFileSaveDialog additions
	SetSaveAsItem          uintptr
	SetProperties          uintptr
	SetCollectedProperties uintptr
	GetProperties          uintptr
	ApplyProperties        uintptr
}

func (d *IFileSaveDialog) Release() {
	syscall.SyscallN(d.lpVtbl.Release, uintptr(unsafe.Pointer(d)))
}

// Show displays the dialog modally, blocking until the user chooses or
// cancels. hwndOwner may be 0 for no owner window.
func (d *IFileSaveDialog) Show(hwndOwner HWND) HRESULT {
	r, _, _ := syscall.SyscallN(d.lpVtbl.Show, uintptr(unsafe.Pointer(d)), uintptr(hwndOwner))
	return HRESULT(r)
}

// SetFileTypes sets the filter list. specs must outlive the call.
func (d *IFileSaveDialog) SetFileTypes(specs []COMDLG_FILTERSPEC) HRESULT {
	if len(specs) == 0 {
		return 0
	}
	r, _, _ := syscall.SyscallN(d.lpVtbl.SetFileTypes,
		uintptr(unsafe.Pointer(d)), uintptr(len(specs)), uintptr(unsafe.Pointer(&specs[0])))
	return HRESULT(r)
}

func (d *IFileSaveDialog) SetOptions(fos uint32) HRESULT {
	r, _, _ := syscall.SyscallN(d.lpVtbl.SetOptions, uintptr(unsafe.Pointer(d)), uintptr(fos))
	return HRESULT(r)
}

func (d *IFileSaveDialog) GetOptions() (uint32, HRESULT) {
	var fos uint32
	r, _, _ := syscall.SyscallN(d.lpVtbl.GetOptions, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(&fos)))
	return fos, HRESULT(r)
}

func (d *IFileSaveDialog) SetFolder(psi *IShellItem) HRESULT {
	r, _, _ := syscall.SyscallN(d.lpVtbl.SetFolder, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(psi)))
	return HRESULT(r)
}

// SetFileName sets the pre-filled filename in the dialog's file-name field.
func (d *IFileSaveDialog) SetFileName(name *uint16) HRESULT {
	r, _, _ := syscall.SyscallN(d.lpVtbl.SetFileName, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(name)))
	return HRESULT(r)
}

func (d *IFileSaveDialog) SetTitle(title *uint16) HRESULT {
	r, _, _ := syscall.SyscallN(d.lpVtbl.SetTitle, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(title)))
	return HRESULT(r)
}

// GetResult returns the chosen item after the user confirms a save path.
func (d *IFileSaveDialog) GetResult() (*IShellItem, HRESULT) {
	var item *IShellItem
	r, _, _ := syscall.SyscallN(d.lpVtbl.GetResult, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(&item)))
	return item, HRESULT(r)
}

// SetDefaultExtension sets the file extension appended when the user types a
// name without one. The extension is passed without a leading dot ("txt", not
// ".txt"). Passing an empty string clears any previously set extension.
func (d *IFileSaveDialog) SetDefaultExtension(ext *uint16) HRESULT {
	r, _, _ := syscall.SyscallN(d.lpVtbl.SetDefaultExtension, uintptr(unsafe.Pointer(d)), uintptr(unsafe.Pointer(ext)))
	return HRESULT(r)
}
