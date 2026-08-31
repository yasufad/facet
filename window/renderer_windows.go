//go:build windows

package window

import (
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/render"
	"github.com/yasufad/facet/render/d3d11"
)

func newDefaultRenderer(surface uintptr, size geometry.Size[geometry.DevicePixels], opts render.Options) (render.Renderer, error) {
	return d3d11.New(surface, size, opts)
}
