package geometry

import "testing"

func TestPixelsScaleAndToDevicePixels(t *testing.T) {
	p := Pixels(10)

	if got := p.Scale(2); got != ScaledPixels(20) {
		t.Fatalf("Scale = %v, want 20", got)
	}
	if got := p.ToDevicePixels(2); got != DevicePixels(20) {
		t.Fatalf("ToDevicePixels = %v, want 20", got)
	}
}

// ToDevicePixels rounds to the nearest device pixel, so 10.4 px at scale 2
// becomes 21 device pixels (20.8 rounded).
func TestPixelsToDevicePixelsRounds(t *testing.T) {
	got := Pixels(10.4).ToDevicePixels(2)
	if got != DevicePixels(21) {
		t.Fatalf("ToDevicePixels = %v, want 21", got)
	}
}

func TestScaledPixelsToDevicePixelsCeils(t *testing.T) {
	// Ceil, not round, so a scaled region never shrinks below its extent.
	if got := ScaledPixels(20.1).ToDevicePixels(); got != DevicePixels(21) {
		t.Fatalf("ToDevicePixels = %v, want 21", got)
	}
}

func TestDevicePixelsToPixels(t *testing.T) {
	if got := DevicePixels(20).ToPixels(2); got != Pixels(10) {
		t.Fatalf("ToPixels = %v, want 10", got)
	}
}

func TestRemsToPixels(t *testing.T) {
	if got := Rems(2).ToPixels(Pixels(16)); got != Pixels(32) {
		t.Fatalf("ToPixels = %v, want 32", got)
	}
}

func TestDevicePixelsToBytes(t *testing.T) {
	if got := DevicePixels(10).ToBytes(4); got != 40 {
		t.Fatalf("ToBytes = %d, want 40", got)
	}
}
