package platform

// Menu interface proposal — three questions that need deciding before
// implementation, because the answers change whether Window gains new methods.
//
// # 1. Native vs. rendered menu bars on Windows
//
// Platform.SetApplicationMenu is already declared and is the right seam for
// macOS, where one menu bar covers the whole application. On Windows each
// window has its own menu bar, attached as a native HMENU via SetMenu(hwnd,
// hmenu). There are two options:
//
//   - Native per-window menu bars: add Window.SetMenu(*Menu). Platform.SetApplicationMenu
//     calls Window.SetMenu on each existing and future window. The OS renders the
//     bar and delivers WM_COMMAND when a user picks an item. This gives standard
//     keyboard navigation, accessibility and system theming automatically.
//
//   - Rendered menu bar: the window package renders the menu bar as ordinary
//     content in the client area. Platform.SetApplicationMenu is a no-op on
//     Windows; window receives the Menu data through a different channel (not
//     through platform at all). No new Window method. This is what a custom
//     title bar application would do, and it is the model the current TODO
//     comment implies.
//
// The decision determines whether Window is touched. A rendered menu bar keeps
// the boundary clean; a native one is richer out of the box. The answer should
// be recorded here before code is written.
//
// # 2. Context menus
//
// Right-click menus pop at a position relative to a window. If they go through
// the native OS path (TrackPopupMenu on Windows, NSMenu.popUpMenu on macOS),
// the signature is:
//
//	Window.ShowContextMenu(menu *Menu, pos geometry.Point[geometry.Pixels])
//
// If context menus are rendered popups (floating windows drawn by element or
// ui), platform never needs to know about them and no Window method is added.
//
// # 3. Shortcuts
//
// MenuItem.Shortcut is declared but not acted on. On macOS, NSMenuItem handles
// the key equivalent automatically once the item is installed in the menu bar.
// On Windows, shortcuts for a native menu bar are delivered as WM_COMMAND when
// the menu is open; for shortcuts that work while the menu is closed, RegisterHotKey
// is required per HWND, which again ties them to a window.
//
// If shortcuts should fire when no menu is visible (Ctrl+S to save, Ctrl+Z to
// undo), platform must parse MenuItem.Shortcut and register each accelerator on
// every window that holds the menu. If shortcuts only work while the menu is
// open, the OS handles them through the normal menu keyboard path and no extra
// registration is needed.
//
// ---
//
// None of these decisions are made here. The types below are the agreed-on data
// layer; Platform.SetApplicationMenu is the agreed-on application-wide hook.
// Raise the question before landing code that shapes around one answer.

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
	// It is displayed alongside the label and, where the platform supports it,
	// registered as a global shortcut for the item. The string uses the
	// modifier names from [Modifiers.String] and a key name from [KeyCode].
	Shortcut string

	// Disabled greys out the item and prevents selection.
	Disabled bool

	// Checked shows a check mark beside the item.
	Checked bool

	// Submenu, when non-nil, makes this item a submenu parent. OnClick is
	// ignored for submenu items.
	Submenu *Menu

	// OnClick is called on the platform thread when the user selects this
	// item. It is nil for submenu items and separators.
	OnClick func()
}

// Separator returns a menu item that renders as a horizontal divider. It has
// no label, no shortcut and no click handler.
func Separator() MenuItem { return MenuItem{} }
