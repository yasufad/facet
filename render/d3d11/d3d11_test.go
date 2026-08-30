//go:build windows

package d3d11

import (
	"testing"

	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/render"
)

func TestShelfPacker(t *testing.T) {
	packer := newShelfPacker(100, 100)

	// First tile fits at (0, 0)
	x, y, ok := packer.allocate(30, 20)
	if !ok || x != 0 || y != 0 {
		t.Fatalf("expected (0, 0), got (%d, %d), ok=%v", x, y, ok)
	}

	// Second tile fits on same shelf at (30, 0)
	x, y, ok = packer.allocate(40, 25)
	if !ok || x != 30 || y != 0 {
		t.Fatalf("expected (30, 0), got (%d, %d), ok=%v", x, y, ok)
	}

	// Third tile doesn't fit on first shelf (30 + 40 + 50 > 100), starts new shelf at y=25
	x, y, ok = packer.allocate(50, 30)
	if !ok || x != 0 || y != 25 {
		t.Fatalf("expected (0, 25), got (%d, %d), ok=%v", x, y, ok)
	}

	// Oversized tile fails
	_, _, ok = packer.allocate(150, 10)
	if ok {
		t.Fatalf("expected oversized tile to fail")
	}

	// Zero or negative size fails
	_, _, ok = packer.allocate(0, 10)
	if ok {
		t.Fatalf("expected zero width to fail")
	}
	_, _, ok = packer.allocate(10, 0)
	if ok {
		t.Fatalf("expected zero height to fail")
	}

	// Reset resets cursor
	packer.reset()
	x, y, ok = packer.allocate(10, 10)
	if !ok || x != 0 || y != 0 {
		t.Fatalf("expected (0, 0) after reset, got (%d, %d)", x, y)
	}
}

func TestNewInvalidSurface(t *testing.T) {
	_, err := New(0, geometry.NewSize[geometry.DevicePixels](100, 100), render.Options{})
	if err == nil {
		t.Fatalf("expected error for surface handle 0, got nil")
	}
}
