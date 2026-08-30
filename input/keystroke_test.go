package input

import (
	"testing"

	"github.com/yasufad/facet/platform"
)

func TestParseKeystroke(t *testing.T) {
	tests := []struct {
		input string
		want  Keystroke
	}{
		{
			input: "a",
			want:  Keystroke{Code: platform.KeyA, Modifiers: 0},
		},
		{
			input: "A",
			want:  Keystroke{Code: platform.KeyA, Modifiers: platform.Shift},
		},
		{
			input: "ctrl-z",
			want:  Keystroke{Code: platform.KeyZ, Modifiers: platform.Control},
		},
		{
			input: "cmd-shift-p",
			want:  Keystroke{Code: platform.KeyP, Modifiers: platform.Super | platform.Shift},
		},
		{
			input: "alt-x",
			want:  Keystroke{Code: platform.KeyX, Modifiers: platform.Alt},
		},
		{
			input: "ctrl-alt-down",
			want:  Keystroke{Code: platform.KeyArrowDown, Modifiers: platform.Control | platform.Alt},
		},
		{
			input: "space",
			want:  Keystroke{Code: platform.KeySpace, Modifiers: 0},
		},
		{
			input: "f12",
			want:  Keystroke{Code: platform.KeyF12, Modifiers: 0},
		},
		{
			input: "ctrl--",
			want:  Keystroke{Code: platform.KeyMinus, Modifiers: platform.Control},
		},
		{
			input: "-",
			want:  Keystroke{Code: platform.KeyMinus, Modifiers: 0},
		},
		{
			input: "secondary-s",
			want:  Keystroke{Code: platform.KeyS, Modifiers: secondaryModifier},
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseKeystroke(tt.input)
			if err != nil {
				t.Fatalf("ParseKeystroke(%q) error = %v", tt.input, err)
			}
			if !got.Matches(tt.want) {
				t.Fatalf("ParseKeystroke(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseKeySequence(t *testing.T) {
	seq, err := ParseKeySequence("ctrl-w left")
	if err != nil {
		t.Fatalf("ParseKeySequence error = %v", err)
	}
	if len(seq) != 2 {
		t.Fatalf("expected 2 keystrokes, got %d", len(seq))
	}
	if seq[0].Code != platform.KeyW || seq[0].Modifiers != platform.Control {
		t.Fatalf("first key: got %+v, want ctrl-w", seq[0])
	}
	if seq[1].Code != platform.KeyArrowLeft || seq[1].Modifiers != 0 {
		t.Fatalf("second key: got %+v, want left", seq[1])
	}
}

func TestKeystrokeFromEventAndString(t *testing.T) {
	event := platform.KeyEvent{
		Code:      platform.KeyEscape,
		Modifiers: platform.Control | platform.Shift,
		Phase:     platform.KeyDown,
	}
	ks := KeystrokeFromEvent(event)
	if ks.Code != platform.KeyEscape || ks.Modifiers != (platform.Control|platform.Shift) {
		t.Fatalf("unexpected Keystroke: %+v", ks)
	}

	str := ks.String()
	if str != "ctrl-shift-escape" {
		t.Fatalf("unexpected String(): got %q, want %q", str, "ctrl-shift-escape")
	}

	// Roundtrip
	parsed, err := ParseKeystroke(str)
	if err != nil {
		t.Fatalf("roundtrip parse error: %v", err)
	}
	if !parsed.Matches(ks) {
		t.Fatalf("roundtrip mismatch: got %+v, want %+v", parsed, ks)
	}
}

func TestParseKeystrokeErrors(t *testing.T) {
	badInputs := []string{
		"",
		"   ",
		"ctrl-",
		"unknownkey",
		"ctrl-a-b",
	}

	for _, input := range badInputs {
		t.Run(input, func(t *testing.T) {
			_, err := ParseKeystroke(input)
			if err == nil {
				t.Fatalf("expected error for %q, got nil", input)
			}
		})
	}
}

func TestEmptyKeystroke(t *testing.T) {
	var empty Keystroke
	if empty.String() != "unknown" {
		t.Fatalf("empty keystroke string: got %q, want %q", empty.String(), "unknown")
	}
}
