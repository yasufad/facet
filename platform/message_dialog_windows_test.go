//go:build windows && facet_debug

package platform

import (
	"testing"

	"github.com/yasufad/facet/third_party/w32"
)

// TestMessageDialogFlags verifies that messageDialogFlags maps each
// MessageButtons and MessageDialogIcon combination to the correct
// MessageBoxW uType flags. Split out from ShowMessageDialog so the
// configuration — the part a wrong button-set constant or icon constant
// would actually break — is testable without dismissing a modal window,
// the same way newFileSaveDialog is split from ShowSaveDialog.
//
// The test checks the flag bits against the w32 constants rather than
// against the return value of a second call to messageDialogFlags, so a
// wrong constant (MB_OK where MB_YESNO belongs) surfaces here rather than
// as a dialog with the wrong buttons.
func TestMessageDialogFlags(t *testing.T) {
	tests := []struct {
		name   string
		dialog MessageDialog
		want   uint
	}{
		{
			name:   "OK with info icon",
			dialog: MessageDialog{Buttons: ButtonsOK, Icon: IconInfo},
			want:   w32.MB_OK | w32.MB_ICONINFORMATION,
		},
		{
			name:   "OKCancel with warning icon",
			dialog: MessageDialog{Buttons: ButtonsOKCancel, Icon: IconWarning},
			want:   w32.MB_OKCANCEL | w32.MB_ICONWARNING,
		},
		{
			name:   "YesNo with error icon",
			dialog: MessageDialog{Buttons: ButtonsYesNo, Icon: IconError},
			want:   w32.MB_YESNO | w32.MB_ICONERROR,
		},
		{
			name:   "YesNoCancel with question icon",
			dialog: MessageDialog{Buttons: ButtonsYesNoCancel, Icon: IconQuestion},
			want:   w32.MB_YESNOCANCEL | w32.MB_ICONQUESTION,
		},
		{
			name:   "zero-value dialog defaults to OK with info icon",
			dialog: MessageDialog{},
			want:   w32.MB_OK | w32.MB_ICONINFORMATION,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := messageDialogFlags(tt.dialog)
			if got != tt.want {
				t.Errorf("messageDialogFlags = %#x, want %#x", got, tt.want)
			}
		})
	}
}

// TestMessageDialogResultMapping verifies that messageDialogResult maps
// each MessageBoxW return value (IDOK, IDYES, etc.) to the correct
// DialogResult. A wrong mapping — IDYES to ResultNo, for example — would
// surface here rather than as a dialog that reports the wrong button.
func TestMessageDialogResultMapping(t *testing.T) {
	tests := []struct {
		id   int
		want DialogResult
	}{
		{w32.IDOK, ResultOK},
		{w32.IDCANCEL, ResultCancel},
		{w32.IDYES, ResultYes},
		{w32.IDNO, ResultNo},
		{0, ResultNone},
		{999, ResultNone},
	}
	for _, tt := range tests {
		if got := messageDialogResult(tt.id); got != tt.want {
			t.Errorf("messageDialogResult(%d) = %v, want %v", tt.id, got, tt.want)
		}
	}
}
