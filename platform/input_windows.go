//go:build windows

package platform

import (
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/third_party/w32"
)

// vkToKeyCode maps a Windows virtual-key code to a [KeyCode]. Keys not in the
// map return [KeyUnknown]; adding a mapping is a backend change, not an
// interface change.
func vkToKeyCode(vk uint32) KeyCode {
	switch vk {
	case w32.VK_LEFT:
		return KeyArrowLeft
	case w32.VK_RIGHT:
		return KeyArrowRight
	case w32.VK_UP:
		return KeyArrowUp
	case w32.VK_DOWN:
		return KeyArrowDown
	case w32.VK_HOME:
		return KeyHome
	case w32.VK_END:
		return KeyEnd
	case w32.VK_PRIOR:
		return KeyPageUp
	case w32.VK_NEXT:
		return KeyPageDown
	case w32.VK_BACK:
		return KeyBackspace
	case w32.VK_RETURN:
		return KeyEnter
	case w32.VK_TAB:
		return KeyTab
	case w32.VK_ESCAPE:
		return KeyEscape
	case w32.VK_SPACE:
		return KeySpace
	case w32.VK_DELETE:
		return KeyDelete
	case w32.VK_INSERT:
		return KeyInsert
	case w32.VK_CAPITAL:
		return KeyCapsLock
	case w32.VK_SHIFT:
		// Windows does not distinguish left/right in WM_KEYDOWN; it sends
		// VK_SHIFT. Querying GetKeyState for VK_LSHIFT/VK_RSHIFT would work
		// but adds a syscall per event. Report the left variant; the
		// modifier state in the event is what matters, not which shift.
		return KeyShiftLeft
	case w32.VK_CONTROL:
		return KeyControlLeft
	case w32.VK_MENU:
		return KeyAltLeft
	case w32.VK_LWIN:
		return KeySuperLeft
	case w32.VK_RWIN:
		return KeySuperRight
	case w32.VK_OEM_MINUS:
		return KeyMinus
	case w32.VK_OEM_PLUS:
		return KeyEqual
	case w32.VK_OEM_4:
		return KeyLeftBracket
	case w32.VK_OEM_6:
		return KeyRightBracket
	case w32.VK_OEM_5:
		return KeyBackslash
	case w32.VK_OEM_1:
		return KeySemicolon
	case w32.VK_OEM_7:
		return KeyApostrophe
	case w32.VK_OEM_3:
		return KeyGraveAccent
	case w32.VK_OEM_COMMA:
		return KeyComma
	case w32.VK_OEM_PERIOD:
		return KeyPeriod
	case w32.VK_OEM_2:
		return KeySlash
	}

	if vk >= '0' && vk <= '9' {
		return Key0 + KeyCode(int(vk-'0'))
	}
	if vk >= 'A' && vk <= 'Z' {
		return KeyA + KeyCode(int(vk-'A'))
	}
	if vk >= w32.VK_F1 && vk <= w32.VK_F24 {
		return KeyF1 + KeyCode(int(vk-w32.VK_F1))
	}

	return KeyUnknown
}

// keyStateToModifiers reads the current modifier state from the keyboard
// asynchronous key state. It is called from the WndProc on the platform
// thread, so GetKeyState (which reflects the state at the last input
// message) is the correct query.
func keyStateToModifiers() Modifiers {
	var m Modifiers
	if uint16(w32.GetKeyState(w32.VK_SHIFT))&0x8000 != 0 {
		m |= Shift
	}
	if uint16(w32.GetKeyState(w32.VK_CONTROL))&0x8000 != 0 {
		m |= Control
	}
	if uint16(w32.GetKeyState(w32.VK_MENU))&0x8000 != 0 {
		m |= Alt
	}
	if uint16(w32.GetKeyState(w32.VK_LWIN))&0x8000 != 0 || uint16(w32.GetKeyState(w32.VK_RWIN))&0x8000 != 0 {
		m |= Super
	}
	return m
}

// lParamToClientPoint extracts the client-area position from an lParam packed
// as a 16-bit x and 16-bit y, as WM_MOUSE* messages use. The position is in
// device pixels.
func lParamToClientPoint(lParam uintptr) geometry.Point[geometry.DevicePixels] {
	x := int32(int16(lParam & 0xFFFF))
	y := int32(int16((lParam >> 16) & 0xFFFF))
	return geometry.Point[geometry.DevicePixels]{X: geometry.DevicePixels(x), Y: geometry.DevicePixels(y)}
}

// wParamToPointerButtons extracts the held mouse buttons from the wParam of
// WM_MOUSEMOVE, WM_LBUTTONDOWN and similar messages.
func wParamToPointerButtons(wParam uintptr) PointerButtons {
	var b PointerButtons
	if wParam&w32.MK_LBUTTON != 0 {
		b |= ButtonLeft
	}
	if wParam&w32.MK_RBUTTON != 0 {
		b |= ButtonRight
	}
	if wParam&w32.MK_MBUTTON != 0 {
		b |= ButtonMiddle
	}
	if wParam&w32.MK_XBUTTON1 != 0 {
		b |= ButtonX1
	}
	if wParam&w32.MK_XBUTTON2 != 0 {
		b |= ButtonX2
	}
	return b
}
