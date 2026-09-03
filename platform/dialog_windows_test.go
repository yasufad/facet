//go:build windows

package platform

import (
	"testing"

	"github.com/yasufad/facet/third_party/w32"
)

func TestFilterSpecPattern(t *testing.T) {
	tests := []struct {
		name       string
		extensions []string
		want       string
	}{
		{"no extensions matches everything", nil, "*.*"},
		{"single extension", []string{"txt"}, "*.txt"},
		{"multiple extensions join with semicolons", []string{"txt", "md"}, "*.txt;*.md"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := filterSpecPattern(tt.extensions); got != tt.want {
				t.Errorf("filterSpecPattern(%v) = %q, want %q", tt.extensions, got, tt.want)
			}
		})
	}
}

// hresultFromUint32 converts the bit pattern of a Win32 error code (which,
// as HRESULT_FROM_WIN32 constructs it, sets the sign bit and so does not fit
// as a positive int32 constant) into an HRESULT. Written through a runtime
// variable rather than a direct conversion so Go performs the truncating
// reinterpretation instead of rejecting the literal as a constant overflow.
func hresultFromUint32(v uint32) w32.HRESULT {
	return w32.HRESULT(v)
}

func TestIsDialogCancelled(t *testing.T) {
	if !w32.IsDialogCancelled(hresultFromUint32(0x800704C7)) {
		t.Error("IsDialogCancelled(0x800704C7) = false, want true (HRESULT_FROM_WIN32(ERROR_CANCELLED))")
	}
	if w32.IsDialogCancelled(0) {
		t.Error("IsDialogCancelled(S_OK) = true, want false")
	}
	if w32.IsDialogCancelled(hresultFromUint32(0x80070005)) { // E_ACCESSDENIED, a real failure
		t.Error("IsDialogCancelled(E_ACCESSDENIED) = true, want false")
	}
}
