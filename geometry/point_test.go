package geometry

import "testing"

// A point converted on its own must land where the same point converted as a
// Bounds origin lands. If the two rounding rules ever diverge, a cursor
// position or glyph origin converted bare will be a pixel off from the same
// position converted inside a Bounds — an intermittent gap that looks like a
// renderer bug. The values are fractional at a fractional scale so the
// rounding is exercised, not bypassed.
func TestPointToDevicePixelsMatchesBoundsOrigin(t *testing.T) {
	p := NewPoint(Pixels(10.4), Pixels(7.6))
	const factor = 1.5

	standalone := PointToDevicePixels(p, factor)
	asOrigin := BoundsToDevicePixels(NewBounds(p, NewSize(Pixels(20), Pixels(20))), factor).Origin

	if standalone != asOrigin {
		t.Fatalf("point = %v, bounds origin = %v; a bare point must round as a Bounds origin does",
			standalone, asOrigin)
	}
}

func TestPointToDevicePixelsRoundsNearest(t *testing.T) {
	// 10.4 px * 2 = 20.8 -> 21; 7.2 px * 2 = 14.4 -> 14.
	got := PointToDevicePixels(NewPoint(Pixels(10.4), Pixels(7.2)), 2)
	want := NewPoint(DevicePixels(21), DevicePixels(14))
	if got != want {
		t.Fatalf("PointToDevicePixels = %v, want %v", got, want)
	}
}

func TestDevicePointToPixels(t *testing.T) {
	got := DevicePointToPixels(NewPoint(DevicePixels(21), DevicePixels(14)), 2)
	want := NewPoint(Pixels(10.5), Pixels(7))
	if got != want {
		t.Fatalf("DevicePointToPixels = %v, want %v", got, want)
	}
}
