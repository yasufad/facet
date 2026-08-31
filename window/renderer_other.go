//go:build !windows

package window

import (
	"errors"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/render"
)

func newDefaultRenderer(surface uintptr, size geometry.Size[geometry.DevicePixels], opts render.Options) (render.Renderer, error) {
	return nil, errors.New("window: no graphics renderer available on this platform")
}
