package platform

// Menu bars, context menus and shortcuts — three contracts.
//
// # 1. Native menu bars; Window gains no method
//
// SetApplicationMenu is the application-wide hook on both backends. On macOS
// it is the single screen menu bar; on Windows it attaches a native HMENU to
// every window the backend owns, present and future. The Windows backend
// walks its HWND map and calls SetMenu on each, and new windows pick up the
// current menu at creation. No method is added to the Window interface: macOS
// cannot honour a per-window menu, so Window.SetMenu would mean something on
// one backend and nothing on the other.
//
// Undecorated windows get no menu bar, and that is correct rather than a
// limitation: a window created with Decorated: false asked for no OS chrome,
// and a menu bar is OS chrome. A rendered menu bar for custom-title-bar
// applications is a ui widget, later, and needs a mechanism element does not
// yet expose.
//
// # 2. Context menus: native, and asynchronous
//
// Context menus go through the native OS path (TrackPopupMenu on Windows,
// popUpMenuPositioningItem: on macOS) because a rendered popup cannot leave
// the window, and a context menu opened near the bottom edge has nowhere to
// go. The method is
//
//	Window.ShowContextMenu(menu *Menu, at geometry.Point[geometry.Pixels])
//
// and it returns before the menu appears. TrackPopupMenu and
// popUpMenuPositioningItem: both run a nested modal message loop, so a call
// from inside an event handler would pump further messages into the frame
// loop while an entity is checked out. The backend therefore records the
// request and runs the native call on a later turn of its own message loop,
// once the current update has finished. MenuItem.OnClick fires on the
// platform thread, outside any borrow. This is the contract on both
// backends, not a Windows workaround.
//
// # 3. Shortcuts go through input's keymap; Shortcut is display text
//
// MenuItem.Shortcut is the text drawn beside the label, nothing more. The
// application binds the chord in its input.Keymap; the menu item's OnClick
// dispatches the same action. Both routes arrive at one place. RegisterHotKey
// is not used: it registers system-globally, so the chord fires whenever it
// is pressed anywhere on the desktop, whichever application has focus.
//
// platform cannot hold an input.Action — input imports platform and the
// dependency runs one way — so OnClick stays a func(). On macOS an
// NSMenuItem key equivalent fires natively once the item is installed and
// consumes the keystroke before it reaches our handler, so a chord present in
// both the menu and the keymap is handled by the menu and never reaches the
// keymap. That is harmless only because both dispatch the same action, which
// is why "same action on both routes" is a contract and not a preference.

// Menu is a tree of menu items. It is plain data: the platform builds a
// native menu from it, and the [MenuItem.OnClick] closure is called when the
// user selects that item.
//
// A menu does not own its items' lifetimes. The caller holds the Menu for as
// long as the native menu is in use; the platform does not retain it.
type Menu struct {
	Items []MenuItem
}

// MenuItem is one entry in a [Menu]. A leaf item carries an [OnClick] closure;
// a submenu item carries a Submenu and no OnClick.
type MenuItem struct {
	// Label is the text displayed for the item. An empty Label on some
	// platforms renders as a separator.
	Label string

	// Shortcut is an accelerator string such as "Ctrl+S" or "Cmd+Shift+Z".
	// It is displayed alongside the label and nothing more: the application
	// binds the chord in its input.Keymap, and the menu item's OnClick
	// dispatches the same action. The string uses the modifier names from
	// [Modifiers.String] and a key name from [KeyCode].
	Shortcut string

	// Disabled greys out the item and prevents selection.
	Disabled bool

	// Checked shows a check mark beside the item.
	Checked bool

	// Submenu, when non-nil, makes this item a submenu parent. OnClick is
	// ignored for submenu items.
	Submenu *Menu

	// OnClick is called on the platform thread when the user selects this
	// item. It is a thin closure that dispatches an input.Action, not the
	// logic itself — the action's handler lives where the keymap binds it,
	// so the menu and the keyboard arrive at one place. It is nil for
	// submenu items and separators.
	OnClick func()
}

// Separator returns a menu item that renders as a horizontal divider. It has
// no label, no shortcut and no click handler.
func Separator() MenuItem { return MenuItem{} }
