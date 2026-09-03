//go:build windows && facet_debug

package platform

import (
	"os"
	"testing"

	"github.com/yasufad/facet/third_party/w32"
)

// TestNewFileSaveDialogConfigures creates a real IFileSaveDialog via
// CoCreateInstance, applies options, title, initial folder, default filename
// and filters, then round-trips GetOptions to check the actual FOS bits rather
// than just that the calls didn't error -- independent evidence, alongside the
// SDK/Wine cross-check recorded in third_party/README, that the vtable layout
// is right: a wrong slot here reads as a legal call to the wrong method, not
// a crash.
//
// It is behind facet_debug because CoCreateInstance against the shell's dialog
// CLSID is UI infrastructure that may behave differently without an interactive
// window station, matching this package's existing convention for anything that
// touches real desktop resources.
func TestNewFileSaveDialogConfigures(t *testing.T) {
	initCOMForTest(t)

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	fd, err := newFileSaveDialog(SaveFileDialog{
		Title:       "Facet Test Save Dialog",
		Directory:   dir,
		DefaultName: "untitled.txt",
		Filters: []FileFilter{
			{Name: "Text files", Extensions: []string{"txt"}},
			{Name: "All files", Extensions: nil},
		},
	})
	if err != nil {
		t.Fatalf("newFileSaveDialog: %v", err)
	}
	defer fd.Release()

	fos, hr := fd.GetOptions()
	if hr != 0 {
		t.Fatalf("GetOptions: %#x", uint32(hr))
	}
	// FOS_OVERWRITEPROMPT is the save-specific flag the open dialog does not
	// set. Checking it here rather than the field we just set (Title, etc.)
	// means a bug in SetOptions would surface here, independent of SetTitle.
	if fos&w32.FOS_OVERWRITEPROMPT == 0 {
		t.Error("FOS_OVERWRITEPROMPT not set")
	}
	if fos&w32.FOS_FORCEFILESYSTEM == 0 {
		t.Error("FOS_FORCEFILESYSTEM not set")
	}
	if fos&w32.FOS_PATHMUSTEXIST == 0 {
		t.Error("FOS_PATHMUSTEXIST not set")
	}
	// FOS_FILEMUSTEXIST is an open-dialog flag and must not be set on a save
	// dialog -- a save dialog creates a file, not selects an existing one.
	if fos&w32.FOS_FILEMUSTEXIST != 0 {
		t.Error("FOS_FILEMUSTEXIST set on a save dialog (open-dialog flag)")
	}
}

// TestNewFileSaveDialogNoFilters covers the minimal case -- an empty dialog
// with no filters, no title, no default name. FOS_OVERWRITEPROMPT must still
// be set because it is unconditional; FOS_FILEMUSTEXIST must not be.
// Setting one field and checking a different one (not the one we just set) is
// the pattern from AGENTS.md: the assertion must be independent of the action.
func TestNewFileSaveDialogNoFilters(t *testing.T) {
	initCOMForTest(t)

	fd, err := newFileSaveDialog(SaveFileDialog{})
	if err != nil {
		t.Fatalf("newFileSaveDialog: %v", err)
	}
	defer fd.Release()

	fos, hr := fd.GetOptions()
	if hr != 0 {
		t.Fatalf("GetOptions: %#x", uint32(hr))
	}
	if fos&w32.FOS_OVERWRITEPROMPT == 0 {
		t.Error("FOS_OVERWRITEPROMPT not set for empty SaveFileDialog")
	}
	if fos&w32.FOS_FILEMUSTEXIST != 0 {
		t.Error("FOS_FILEMUSTEXIST set on a save dialog")
	}
}

// TestNewFileSaveDialogRejectsMissingDirectory confirms a bad initial
// directory is reported as an error at configuration time rather than silently
// ignored or deferred to Show. Mirrors TestNewFileOpenDialogRejectsMissingDirectory.
func TestNewFileSaveDialogRejectsMissingDirectory(t *testing.T) {
	initCOMForTest(t)

	_, err := newFileSaveDialog(SaveFileDialog{
		Directory: `Z:\this\path\does\not\exist\facet-test`,
	})
	if err == nil {
		t.Error("expected an error for a nonexistent directory, got nil")
	}
}
