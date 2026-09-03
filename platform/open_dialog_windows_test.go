//go:build windows && facet_debug

package platform

import (
	"os"
	"runtime"
	"testing"

	"github.com/yasufad/facet/third_party/w32"
)

// initCOMForTest establishes the single-threaded apartment newFileOpenDialog
// needs (CoCreateInstance fails with CO_E_NOTINITIALIZED otherwise) and
// pins the test goroutine to the OS thread it initialised, mirroring what
// ShowOpenDialog itself does before calling newFileOpenDialog.
func initCOMForTest(t *testing.T) {
	t.Helper()
	runtime.LockOSThread()
	t.Cleanup(runtime.UnlockOSThread)

	hr := w32.CoInitializeEx(w32.COINIT_APARTMENTTHREADED)
	if hr != 0 && hr != 1 {
		t.Fatalf("CoInitializeEx: %#x", uint32(hr))
	}
	t.Cleanup(w32.CoUninitialize)
}

// TestNewFileOpenDialogConfigures exercises the part of ShowOpenDialog that
// a wrong FOS flag, GUID or vtable slot would actually break: creating the
// Common Item Dialog and applying options, title, initial folder and
// filters. It never calls Show, so it needs no human to click through a
// modal window — that last step (a real dialog opening and returning a real
// path) still needs to be observed by a person on a real desktop session,
// the same gap examples/button has.
//
// It is behind facet_debug because CoCreateInstance against the shell's
// dialog CLSID is UI infrastructure that may behave differently without an
// interactive window station, matching this package's existing convention
// for anything that touches real desktop resources.
func TestNewFileOpenDialogConfigures(t *testing.T) {
	initCOMForTest(t)

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	fd, err := newFileOpenDialog(OpenFileDialog{
		Title:     "Facet Test Open Dialog",
		Directory: dir,
		Filters: []FileFilter{
			{Name: "Go files", Extensions: []string{"go"}},
			{Name: "All files", Extensions: nil},
		},
		Multiple: true,
	})
	if err != nil {
		t.Fatalf("newFileOpenDialog: %v", err)
	}
	defer fd.Release()

	fos, hr := fd.GetOptions()
	if hr != 0 {
		t.Fatalf("GetOptions: %#x", uint32(hr))
	}
	if fos&w32.FOS_ALLOWMULTISELECT == 0 {
		t.Error("FOS_ALLOWMULTISELECT not set for Multiple: true")
	}
	if fos&w32.FOS_FILEMUSTEXIST == 0 {
		t.Error("FOS_FILEMUSTEXIST not set")
	}
}

// TestNewFileOpenDialogSingleSelect covers the other side of the flag: a
// dialog configured without Multiple must not carry FOS_ALLOWMULTISELECT.
// Both this and the test above set one field and check a different one,
// rather than checking back the field each just set — the actual value at
// the interaction, not just that the call didn't error.
func TestNewFileOpenDialogSingleSelect(t *testing.T) {
	initCOMForTest(t)

	fd, err := newFileOpenDialog(OpenFileDialog{})
	if err != nil {
		t.Fatalf("newFileOpenDialog: %v", err)
	}
	defer fd.Release()

	fos, hr := fd.GetOptions()
	if hr != 0 {
		t.Fatalf("GetOptions: %#x", uint32(hr))
	}
	if fos&w32.FOS_ALLOWMULTISELECT != 0 {
		t.Error("FOS_ALLOWMULTISELECT set without Multiple: true")
	}
}

// TestNewFileOpenDialogRejectsMissingDirectory confirms a bad initial
// directory is reported as an error at configuration time rather than
// silently ignored or deferred to Show.
func TestNewFileOpenDialogRejectsMissingDirectory(t *testing.T) {
	initCOMForTest(t)

	_, err := newFileOpenDialog(OpenFileDialog{
		Directory: `Z:\this\path\does\not\exist\facet-test`,
	})
	if err == nil {
		t.Error("expected an error for a nonexistent directory, got nil")
	}
}
