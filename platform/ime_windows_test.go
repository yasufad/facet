//go:build windows

package platform

import "testing"

// TestRuneOffsetFromUTF16 exercises the conversion from a UTF-16 code-unit
// offset to a rune offset. The two units agree everywhere in the basic
// multilingual plane; they diverge only where a composed character sits
// outside it, which is why the surrogate-pair cases below matter more than
// the ASCII ones.
func TestRuneOffsetFromUTF16(t *testing.T) {
	// "a\U0001F600b" — 'a', U+1F600 (a surrogate pair, 2 UTF-16 units),
	// 'b'. Rune offsets: a=0, 😀=1, b=2. UTF-16 unit offsets: a=0, 😀=1
	// (high surrogate) or 2 (low surrogate), b=3.
	units := []uint16{'a', 0xD83D, 0xDE00, 'b'}

	tests := []struct {
		name       string
		unitOffset int
		wantRune   int
	}{
		{"start", 0, 0},
		{"before surrogate pair", 1, 1},
		{"inside surrogate pair", 2, 2}, // degenerate: the high surrogate already started this rune
		{"after surrogate pair", 3, 2},
		{"end", 4, 3},
		{"past end clamps to length", 10, 3},
		{"no cursor passes through", -1, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runeOffsetFromUTF16(units, tt.unitOffset)
			if got != tt.wantRune {
				t.Errorf("runeOffsetFromUTF16(%v, %d) = %d, want %d", units, tt.unitOffset, got, tt.wantRune)
			}
		})
	}
}

// TestRuneOffsetFromUTF16AllBasicPlane verifies the common case, where every
// character is in the basic multilingual plane and the two offsets are
// numerically identical.
func TestRuneOffsetFromUTF16AllBasicPlane(t *testing.T) {
	// "café" — every character is one UTF-16 unit.
	units := []uint16{'c', 'a', 'f', 0x00E9}
	for i := 0; i <= len(units); i++ {
		if got := runeOffsetFromUTF16(units, i); got != i {
			t.Errorf("runeOffsetFromUTF16(%v, %d) = %d, want %d", units, i, got, i)
		}
	}
}
