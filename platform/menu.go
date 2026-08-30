package platform

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
