//go:build windows

package platform

import (
	"github.com/yasufad/facet/third_party/w32"
)

// firstCommandID is the ID assigned to the first leaf menu item, so command
// IDs never collide with 0 (which Win32 reserves) and never collide with
// submenu handles, which AppendMenu stores in the same slot under MF_POPUP.
const firstCommandID uintptr = 1

// nativeMenu is a Win32 HMENU built from a [Menu], paired with the map from
// command ID to the OnClick closure each leaf item carries. The platform
// holds one of these per SetApplicationMenu call and looks up the closure in
// its WM_COMMAND handler.
type nativeMenu struct {
	hmenu    w32.HMENU
	commands map[uintptr]func()
	nextID   uintptr
}

// buildMenu constructs a native menu bar from a [Menu] tree, returning the
// HMENU and a map from command ID to each leaf item's OnClick. The caller
// owns the HMENU and must DestroyMenu it when it is no longer attached to a
// window; the command map goes with it.
//
// Submenus are built recursively and attached with MF_POPUP. Leaf items get
// MF_STRING and a unique command ID; separators get MF_SEPARATOR and no ID.
// Disabled and Checked map to MF_GRAYED and MF_CHECKED respectively.
func buildMenu(menu *Menu) *nativeMenu {
	nm := &nativeMenu{
		hmenu:    w32.CreateMenu(),
		commands: make(map[uintptr]func()),
	}
	nm.nextID = firstCommandID
	for i := range menu.Items {
		nm.appendItem(&menu.Items[i])
	}
	return nm
}

func (nm *nativeMenu) appendItem(item *MenuItem) {
	if item.Label == "" && item.Submenu == nil && item.OnClick == nil {
		// Separator: an empty item with no submenu and no click handler.
		w32.AppendMenu(nm.hmenu, w32.MF_SEPARATOR, 0, nil)
		return
	}

	if item.Submenu != nil {
		sub := buildMenu(item.Submenu)
		// AppendMenu takes the submenu handle under MF_POPUP. The submenu's
		// command map is merged into ours so WM_COMMAND on a leaf inside a
		// submenu resolves to the right closure.
		for id, fn := range sub.commands {
			nm.commands[id] = fn
		}
		w32.AppendMenu(nm.hmenu, w32.MF_POPUP, uintptr(sub.hmenu), w32.MustStringToUTF16Ptr(item.Label))
		return
	}

	id := nm.nextID
	nm.nextID++
	nm.commands[id] = item.OnClick

	flags := uint32(w32.MF_STRING)
	if item.Disabled {
		flags |= w32.MF_GRAYED
	}
	if item.Checked {
		flags |= w32.MF_CHECKED
	}
	w32.AppendMenu(nm.hmenu, flags, id, w32.MustStringToUTF16Ptr(item.Label))
}
