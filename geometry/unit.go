package geometry

import "math"

// Pixels are logical pixels: the unit layout and styling speak in. They are
// unaffected by the display scale until something converts them.
type Pixels float32

// Zero pixels.
const ZeroPixels Pixels = 0

// Float32 returns the raw value of p.
func (p Pixels) Float32() float32 { return float32(p) }

// Floor rounds p down to the nearest whole pixel.
func (p Pixels) Floor() Pixels { return Pixels(math.Floor(float64(p))) }

// Round rounds p to the nearest whole pixel.
func (p Pixels) Round() Pixels { return Pixels(math.Round(float64(p))) }

// Ceil rounds p up to the nearest whole pixel.
func (p Pixels) Ceil() Pixels { return Pixels(math.Ceil(float64(p))) }

// Abs returns the absolute value of p.
func (p Pixels) Abs() Pixels { return Pixels(math.Abs(float64(p))) }

// Scale multiplies p by the display scale factor, producing ScaledPixels
// before rounding to device pixels.
func (p Pixels) Scale(factor float32) ScaledPixels { return ScaledPixels(float32(p) * factor) }

// ToDevicePixels converts logical pixels to physical pixels, rounding to the
// nearest device pixel.
func (p Pixels) ToDevicePixels(factor float32) DevicePixels {
	return DevicePixels(int32(math.Round(float64(float32(p) * factor))))
}

// DevicePixels are physical pixels on the display: the unit the renderer
// speaks in.
type DevicePixels int32

// Zero device pixels.
const ZeroDevicePixels DevicePixels = 0

// Int32 returns the raw value of d.
func (d DevicePixels) Int32() int32 { return int32(d) }

// ToBytes returns the number of bytes needed to store d pixels at the given
// bytes-per-pixel stride, useful for framebuffer and image buffers.
func (d DevicePixels) ToBytes(bytesPerPixel uint8) uint32 {
	return uint32(int32(d)) * uint32(bytesPerPixel)
}

// ToScaledPixels converts device pixels to scaled pixels at one-to-one; no
// scale factor is needed because both share the device coordinate space.
func (d DevicePixels) ToScaledPixels() ScaledPixels { return ScaledPixels(float32(d)) }

// ToPixels converts physical pixels to logical pixels by dividing by the
// display scale factor.
func (d DevicePixels) ToPixels(factor float32) Pixels { return Pixels(float32(d) / factor) }

// ScaledPixels are logical pixels multiplied by the display scale, before
// rounding to device pixels. They are the intermediate unit between layout
// and the renderer.
type ScaledPixels float32

// Zero scaled pixels.
const ZeroScaledPixels ScaledPixels = 0

// Float32 returns the raw value of s.
func (s ScaledPixels) Float32() float32 { return float32(s) }

// Floor rounds s down to the nearest whole scaled pixel.
func (s ScaledPixels) Floor() ScaledPixels { return ScaledPixels(math.Floor(float64(s))) }

// Round rounds s to the nearest whole scaled pixel.
func (s ScaledPixels) Round() ScaledPixels { return ScaledPixels(math.Round(float64(s))) }

// Ceil rounds s up to the nearest whole scaled pixel.
func (s ScaledPixels) Ceil() ScaledPixels { return ScaledPixels(math.Ceil(float64(s))) }

// ToPixels converts scaled pixels back to logical pixels by dividing by the
// display scale factor.
func (s ScaledPixels) ToPixels(factor float32) Pixels { return Pixels(float32(s) / factor) }

// ToDevicePixels rounds s up to the nearest device pixel. Ceil rather than
// round so a scaled region never shrinks below its logical extent.
func (s ScaledPixels) ToDevicePixels() DevicePixels {
	return DevicePixels(int32(math.Ceil(float64(s))))
}

// Rems are multiples of the root font size. They convert to pixels only when
// the rem size is known, which is why conversion takes it as an argument.
type Rems float32

// Zero rems.
const ZeroRems Rems = 0

// Float32 returns the raw value of r.
func (r Rems) Float32() float32 { return float32(r) }

// ToPixels converts rems to pixels by multiplying by the current rem size.
func (r Rems) ToPixels(remSize Pixels) Pixels {
	return Pixels(float32(r) * float32(remSize))
}
