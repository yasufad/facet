package platform

// SystemTrayOptions configures a system tray icon at creation.
type SystemTrayOptions struct {
	// Icon is the icon image data. The format is platform-dependent: PNG on
	// Windows and Linux, a multi-image ICNS or PNG on macOS. A backend
	// documents which it accepts.
	Icon []byte

	// Tooltip is the text shown when the pointer hovers over the icon.
	Tooltip string

	// Menu is the menu shown when the icon is clicked. On platforms that
	// distinguish left and right click (Windows, Linux), this is the
	// right-click menu; on macOS it is the menu shown on any click.
	Menu *Menu

	// OnClick is called when the icon is left-clicked and no menu is set, or
	// on platforms where left-click is the menu trigger. It is nil-safe.
	OnClick func()
}

// SystemTray is a live tray icon. It is created by [Platform.NewSystemTray]
// and removed by calling [SystemTray.Remove].
type SystemTray interface {
	// SetIcon replaces the tray icon.
	SetIcon(icon []byte)

	// SetMenu replaces the tray menu.
	SetMenu(menu *Menu)

	// SetTooltip replaces the tooltip text.
	SetTooltip(tooltip string)

	// Remove removes the icon from the tray. The SystemTray is unusable
	// after Remove returns.
	Remove()
}
