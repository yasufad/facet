package input

import (
	"fmt"
	"strings"

	"github.com/yasufad/facet/platform"
)

// Keystroke represents a physical key press along with held modifier keys.
type Keystroke struct {
	Code      platform.KeyCode
	Modifiers platform.Modifiers
}

// KeystrokeFromEvent constructs a Keystroke from a platform.KeyEvent.
func KeystrokeFromEvent(e platform.KeyEvent) Keystroke {
	return Keystroke{
		Code:      e.Code,
		Modifiers: e.Modifiers,
	}
}

// Matches reports whether this keystroke matches other in key code and modifiers.
func (k Keystroke) Matches(other Keystroke) bool {
	return k.Code == other.Code && k.Modifiers == other.Modifiers
}

// String returns a canonical hyphen-separated representation such as
// "ctrl-shift-a", "up", or "space".
func (k Keystroke) String() string {
	var parts []string
	if k.Modifiers.Has(platform.Control) {
		parts = append(parts, "ctrl")
	}
	if k.Modifiers.Has(platform.Alt) {
		parts = append(parts, "alt")
	}
	if k.Modifiers.Has(platform.Shift) {
		parts = append(parts, "shift")
	}
	if k.Modifiers.Has(platform.Super) {
		parts = append(parts, "super")
	}

	keyName := keyCodeToString(k.Code)
	parts = append(parts, keyName)
	return strings.Join(parts, "-")
}

// ParseKeystroke parses a single key chord string such as "ctrl-shift-a", "cmd-z",
// "space", or "ctrl--".
func ParseKeystroke(source string) (Keystroke, error) {
	s := strings.TrimSpace(source)
	if s == "" {
		return Keystroke{}, fmt.Errorf("parse keystroke: empty string")
	}

	var modifiers platform.Modifiers
	var keyStr string

	// Handle trailing hyphen for minus key (e.g. "ctrl--", "alt-shift--", "-")
	hasTrailingMinus := false
	if strings.HasSuffix(s, "--") {
		s = strings.TrimSuffix(s, "-")
		hasTrailingMinus = true
	} else if s == "-" {
		return Keystroke{Code: platform.KeyMinus}, nil
	}

	parts := strings.Split(s, "-")
	if hasTrailingMinus {
		parts[len(parts)-1] = "-"
	}

	for i, part := range parts {
		lower := strings.ToLower(part)
		switch lower {
		case "ctrl", "control":
			if i == len(parts)-1 && len(parts) == 1 {
				keyStr = lower
			} else {
				modifiers |= platform.Control
			}
		case "alt", "option":
			if i == len(parts)-1 && len(parts) == 1 {
				keyStr = lower
			} else {
				modifiers |= platform.Alt
			}
		case "shift":
			if i == len(parts)-1 && len(parts) == 1 {
				keyStr = lower
			} else {
				modifiers |= platform.Shift
			}
		case "cmd", "command", "super", "win", "windows":
			if i == len(parts)-1 && len(parts) == 1 {
				keyStr = lower
			} else {
				modifiers |= platform.Super
			}
		case "secondary":
			if i == len(parts)-1 && len(parts) == 1 {
				keyStr = lower
			} else {
				modifiers |= secondaryModifier
			}
		default:
			if keyStr != "" {
				return Keystroke{}, fmt.Errorf("parse keystroke %q: multiple key codes specified", source)
			}
			keyStr = part
		}
	}

	if keyStr == "" {
		return Keystroke{}, fmt.Errorf("parse keystroke %q: no key specified", source)
	}

	// Single uppercase letter implicitly adds Shift modifier
	if len(keyStr) == 1 && keyStr[0] >= 'A' && keyStr[0] <= 'Z' {
		modifiers |= platform.Shift
		keyStr = strings.ToLower(keyStr)
	}

	code, ok := parseKeyCode(strings.ToLower(keyStr))
	if !ok {
		return Keystroke{}, fmt.Errorf("parse keystroke %q: unknown key %q", source, keyStr)
	}

	return Keystroke{
		Code:      code,
		Modifiers: modifiers,
	}, nil
}

// ParseKeySequence parses a whitespace-separated sequence of keystrokes such
// as "ctrl-w left" or "ctrl-x 0".
func ParseKeySequence(source string) ([]Keystroke, error) {
	fields := strings.Fields(source)
	if len(fields) == 0 {
		return nil, fmt.Errorf("parse key sequence: empty string")
	}

	seq := make([]Keystroke, len(fields))
	for i, f := range fields {
		k, err := ParseKeystroke(f)
		if err != nil {
			return nil, fmt.Errorf("parse key sequence %q at %d: %w", source, i, err)
		}
		seq[i] = k
	}
	return seq, nil
}

func parseKeyCode(s string) (platform.KeyCode, bool) {
	if len(s) == 1 {
		b := s[0]
		if b >= 'a' && b <= 'z' {
			return platform.KeyCode(int(platform.KeyA) + int(b-'a')), true
		}
		if b >= '0' && b <= '9' {
			return platform.KeyCode(int(platform.Key0) + int(b-'0')), true
		}
	}

	switch s {
	case "-":
		return platform.KeyMinus, true
	case "=":
		return platform.KeyEqual, true
	case "[":
		return platform.KeyLeftBracket, true
	case "]":
		return platform.KeyRightBracket, true
	case "\\":
		return platform.KeyBackslash, true
	case ";":
		return platform.KeySemicolon, true
	case "'":
		return platform.KeyApostrophe, true
	case "`":
		return platform.KeyGraveAccent, true
	case ",":
		return platform.KeyComma, true
	case ".":
		return platform.KeyPeriod, true
	case "/":
		return platform.KeySlash, true
	case "left", "arrowleft":
		return platform.KeyArrowLeft, true
	case "right", "arrowright":
		return platform.KeyArrowRight, true
	case "up", "arrowup":
		return platform.KeyArrowUp, true
	case "down", "arrowdown":
		return platform.KeyArrowDown, true
	case "home":
		return platform.KeyHome, true
	case "end":
		return platform.KeyEnd, true
	case "pageup", "pgup":
		return platform.KeyPageUp, true
	case "pagedown", "pgdn":
		return platform.KeyPageDown, true
	case "backspace":
		return platform.KeyBackspace, true
	case "enter", "return":
		return platform.KeyEnter, true
	case "tab":
		return platform.KeyTab, true
	case "escape", "esc":
		return platform.KeyEscape, true
	case "space":
		return platform.KeySpace, true
	case "delete", "del":
		return platform.KeyDelete, true
	case "insert", "ins":
		return platform.KeyInsert, true
	case "capslock":
		return platform.KeyCapsLock, true
	case "minus":
		return platform.KeyMinus, true
	case "equal", "equals":
		return platform.KeyEqual, true
	case "leftbracket", "bracketleft":
		return platform.KeyLeftBracket, true
	case "rightbracket", "bracketright":
		return platform.KeyRightBracket, true
	case "backslash":
		return platform.KeyBackslash, true
	case "semicolon":
		return platform.KeySemicolon, true
	case "apostrophe", "quote":
		return platform.KeyApostrophe, true
	case "grave", "backquote", "backtick":
		return platform.KeyGraveAccent, true
	case "comma":
		return platform.KeyComma, true
	case "period", "dot":
		return platform.KeyPeriod, true
	case "slash":
		return platform.KeySlash, true
	case "ctrl", "control":
		return platform.KeyControlLeft, true
	case "shift":
		return platform.KeyShiftLeft, true
	case "alt", "option":
		return platform.KeyAltLeft, true
	case "super", "cmd", "win":
		return platform.KeySuperLeft, true
	}

	if strings.HasPrefix(s, "f") && len(s) >= 2 {
		num := 0
		for _, r := range s[1:] {
			if r < '0' || r > '9' {
				return platform.KeyUnknown, false
			}
			num = num*10 + int(r-'0')
		}
		if num >= 1 && num <= 24 {
			return platform.KeyCode(int(platform.KeyF1) + (num - 1)), true
		}
	}

	return platform.KeyUnknown, false
}

func keyCodeToString(code platform.KeyCode) string {
	if code >= platform.KeyA && code <= platform.KeyZ {
		return string(rune('a' + (code - platform.KeyA)))
	}
	if code >= platform.Key0 && code <= platform.Key9 {
		return string(rune('0' + (code - platform.Key0)))
	}
	if code >= platform.KeyF1 && code <= platform.KeyF24 {
		return fmt.Sprintf("f%d", int(code-platform.KeyF1)+1)
	}

	switch code {
	case platform.KeyArrowLeft:
		return "left"
	case platform.KeyArrowRight:
		return "right"
	case platform.KeyArrowUp:
		return "up"
	case platform.KeyArrowDown:
		return "down"
	case platform.KeyHome:
		return "home"
	case platform.KeyEnd:
		return "end"
	case platform.KeyPageUp:
		return "pageup"
	case platform.KeyPageDown:
		return "pagedown"
	case platform.KeyBackspace:
		return "backspace"
	case platform.KeyEnter:
		return "enter"
	case platform.KeyTab:
		return "tab"
	case platform.KeyEscape:
		return "escape"
	case platform.KeySpace:
		return "space"
	case platform.KeyDelete:
		return "delete"
	case platform.KeyInsert:
		return "insert"
	case platform.KeyCapsLock:
		return "capslock"
	case platform.KeyShiftLeft, platform.KeyShiftRight:
		return "shift"
	case platform.KeyControlLeft, platform.KeyControlRight:
		return "ctrl"
	case platform.KeyAltLeft, platform.KeyAltRight:
		return "alt"
	case platform.KeySuperLeft, platform.KeySuperRight:
		return "super"
	case platform.KeyMinus:
		return "-"
	case platform.KeyEqual:
		return "="
	case platform.KeyLeftBracket:
		return "["
	case platform.KeyRightBracket:
		return "]"
	case platform.KeyBackslash:
		return "\\"
	case platform.KeySemicolon:
		return ";"
	case platform.KeyApostrophe:
		return "'"
	case platform.KeyGraveAccent:
		return "`"
	case platform.KeyComma:
		return ","
	case platform.KeyPeriod:
		return "."
	case platform.KeySlash:
		return "/"
	default:
		return "unknown"
	}
}
