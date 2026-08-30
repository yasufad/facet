package platform

import "github.com/yasufad/facet/geometry"

// Display describes one attached monitor. It is plain data: the platform
// enumerates displays and returns Display values; nothing above this layer
// holds a long-lived reference to one that the platform does not also hold.
//
// All coordinates are in device pixels, relative to the virtual desktop
// origin. On a single-monitor system the origin is (0, 0); on a multi-monitor
// system the primary display's top-left is the origin and others are placed
// relative to it, which may give negative coordinates.
type Display struct {
	// ID is a platform-specific display identifier. It is stable across
	// enumerations within a session but not across sessions, and is not
	// human-readable.
	ID string

	// Name is the display's name where the platform provides one. It may be
	// empty.
	Name string

	// Bounds is the full display rectangle, including any areas occupied by
	// the taskbar or dock.
	Bounds geometry.Bounds[geometry.DevicePixels]

	// WorkArea is the usable rectangle, excluding the taskbar on Windows and
	// the dock and menu bar on macOS.
	WorkArea geometry.Bounds[geometry.DevicePixels]

	// ScaleFactor is the display's DPI scale factor: 1.0 at 96 DPI, 2.0 at
	// 192 DPI. It is the number geometry.Pixels are multiplied by to reach
	// device pixels.
	ScaleFactor float32

	// Primary reports whether this is the primary display — the one that
	// receives the taskbar and new windows by default.
	Primary bool
}
