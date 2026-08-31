package window

import (
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/geometry"
)

const (
	defaultWindowWidth  = 640
	defaultWindowHeight = 480
	defaultRemSize      = 16
)

// WindowOptions configures a window at creation.
//
// The zero value provides a valid, usable window configuration with default
// dimensions (640x480 logical pixels) and root rem font size (16px).
type WindowOptions struct {
	// Title is the text shown in the platform window title bar.
	Title string

	// Size is the initial client area dimensions in logical pixels.
	Size geometry.Size[geometry.Pixels]

	// Position is the initial top-left corner in display coordinates.
	Position geometry.Point[geometry.Pixels]

	// MinSize is the minimum resizable client area in logical pixels.
	MinSize geometry.Size[geometry.Pixels]

	// MaxSize is the maximum resizable client area in logical pixels.
	MaxSize geometry.Size[geometry.Pixels]

	// Background is the clear colour of the client area.
	Background colour.Rgba

	// Resizable controls whether the window borders can be resized by the user.
	Resizable bool

	// Decorated controls whether native platform window decorations are displayed.
	Decorated bool

	// Transparent enables desktop composition for transparent client backgrounds.
	Transparent bool

	// AlwaysOnTop keeps the window floating above regular windows.
	AlwaysOnTop bool

	// Visible controls whether the window is shown immediately upon creation.
	Visible bool

	// VSync enables vertical synchronisation on presentation.
	VSync bool

	// RemSize is the root font size in logical pixels used to resolve rem units.
	RemSize geometry.Pixels
}

func (o WindowOptions) withDefaults() WindowOptions {
	if o.Size.Width <= 0 || o.Size.Height <= 0 {
		o.Size = geometry.NewSize[geometry.Pixels](defaultWindowWidth, defaultWindowHeight)
	}
	if o.RemSize <= 0 {
		o.RemSize = defaultRemSize
	}
	return o
}
