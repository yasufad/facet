//go:build windows

package platform

import (
	"fmt"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/third_party/w32"
)

// enumerateDisplays returns all attached monitors in device-pixel
// coordinates. The primary display is first, matching the order
// [Platform.Displays] guarantees.
func enumerateDisplays() ([]Display, error) {
	screens, err := w32.GetAllScreens()
	if err != nil {
		return nil, fmt.Errorf("enumerate displays: %w", err)
	}

	displays := make([]Display, 0, len(screens))
	var primary Display

	for _, s := range screens {
		d := Display{
			ID:          fmt.Sprintf("%d", s.HMonitor),
			Name:        s.Name,
			ScaleFactor: s.ScaleFactor,
			Primary:     s.IsPrimary,
			Bounds: geometry.Bounds[geometry.DevicePixels]{
				Origin: geometry.Point[geometry.DevicePixels]{
					X: geometry.DevicePixels(s.RcMonitor.Left),
					Y: geometry.DevicePixels(s.RcMonitor.Top),
				},
				Size: geometry.Size[geometry.DevicePixels]{
					Width:  geometry.DevicePixels(s.RcMonitor.Right - s.RcMonitor.Left),
					Height: geometry.DevicePixels(s.RcMonitor.Bottom - s.RcMonitor.Top),
				},
			},
			WorkArea: geometry.Bounds[geometry.DevicePixels]{
				Origin: geometry.Point[geometry.DevicePixels]{
					X: geometry.DevicePixels(s.RcWork.Left),
					Y: geometry.DevicePixels(s.RcWork.Top),
				},
				Size: geometry.Size[geometry.DevicePixels]{
					Width:  geometry.DevicePixels(s.RcWork.Right - s.RcWork.Left),
					Height: geometry.DevicePixels(s.RcWork.Bottom - s.RcWork.Top),
				},
			},
		}
		if d.Primary {
			primary = d
		} else {
			displays = append(displays, d)
		}
	}

	// Primary display first.
	if primary.ID != "" {
		displays = append([]Display{primary}, displays...)
	}

	return displays, nil
}
