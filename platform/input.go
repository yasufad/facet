package platform

// KeyCode identifies a physical key, independent of keyboard layout. It is
// what keybindings are written against: Ctrl+Z means the key in the Z
// position, whether the layout produces 'z', 'y' or something else.
//
// The set is not exhaustive. Keys not listed here are reported as
// [KeyUnknown] by a backend; adding a code is a backend change, not an
// interface change.
type KeyCode int

const (
	KeyUnknown KeyCode = iota

	// Letters.
	KeyA
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ

	// Digits across the top row.
	Key0
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9

	// Function keys.
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
	KeyF13
	KeyF14
	KeyF15
	KeyF16
	KeyF17
	KeyF18
	KeyF19
	KeyF20
	KeyF21
	KeyF22
	KeyF23
	KeyF24

	// Navigation.
	KeyArrowLeft
	KeyArrowRight
	KeyArrowUp
	KeyArrowDown
	KeyHome
	KeyEnd
	KeyPageUp
	KeyPageDown

	// Editing.
	KeyBackspace
	KeyEnter
	KeyTab
	KeyEscape
	KeySpace
	KeyDelete
	KeyInsert
	KeyCapsLock

	// Modifier keys. Left and right are distinguished where the platform
	// reports them; a backend that cannot tell them apart reports both as the
	// left variant.
	KeyShiftLeft
	KeyShiftRight
	KeyControlLeft
	KeyControlRight
	KeyAltLeft
	KeyAltRight
	KeySuperLeft
	KeySuperRight

	// Punctuation and symbol keys on a US layout. A backend maps by physical
	// position, not by the label printed on the key.
	KeyMinus
	KeyEqual
	KeyLeftBracket
	KeyRightBracket
	KeyBackslash
	KeySemicolon
	KeyApostrophe
	KeyGraveAccent
	KeyComma
	KeyPeriod
	KeySlash
)

// Modifiers is a bitfield of the modifier keys held during an event.
type Modifiers uint8

const (
	// Shift is the Shift key on all platforms.
	Shift Modifiers = 1 << iota
	// Control is the Control key on all platforms. On macOS this is the
	// leftmost modifier, not the Command key.
	Control
	// Alt is Option on macOS and Alt on Windows and Linux.
	Alt
	// Super is the Command key on macOS, the Windows key on Windows, and the
	// Super key on Linux.
	Super
)

// Has reports whether all of other is set in m.
func (m Modifiers) Has(other Modifiers) bool { return m&other == other }

// IsEmpty reports whether no modifiers are set.
func (m Modifiers) IsEmpty() bool { return m == 0 }

// String returns a platform-neutral description such as "Shift+Ctrl".
func (m Modifiers) String() string {
	var parts []string
	if m.Has(Shift) {
		parts = append(parts, "Shift")
	}
	if m.Has(Control) {
		parts = append(parts, "Ctrl")
	}
	if m.Has(Alt) {
		parts = append(parts, "Alt")
	}
	if m.Has(Super) {
		parts = append(parts, "Super")
	}
	s := ""
	for i, p := range parts {
		if i > 0 {
			s += "+"
		}
		s += p
	}
	return s
}

// PointerButton identifies a mouse or trackpad button.
type PointerButton int

const (
	PointerNone PointerButton = iota
	PointerLeft
	PointerRight
	PointerMiddle
	PointerX1
	PointerX2
)

// PointerButtons is a bitfield of the buttons held during a pointer event.
type PointerButtons uint8

const (
	ButtonLeft   PointerButtons = 1 << (PointerLeft - 1)
	ButtonRight  PointerButtons = 1 << (PointerRight - 1)
	ButtonMiddle PointerButtons = 1 << (PointerMiddle - 1)
	ButtonX1     PointerButtons = 1 << (PointerX1 - 1)
	ButtonX2     PointerButtons = 1 << (PointerX2 - 1)
)

// Has reports whether the given button is held.
func (b PointerButtons) Has(button PointerButton) bool {
	if button <= PointerNone {
		return false
	}
	return b&(1<<(button-1)) != 0
}

// PointerPhase distinguishes movement from button press and release.
type PointerPhase int

const (
	PointerMove PointerPhase = iota
	PointerDown
	PointerUp
)

// KeyPhase distinguishes a key press from a release or an auto-repeat.
type KeyPhase int

const (
	KeyDown KeyPhase = iota
	KeyUp
	KeyRepeat
)
