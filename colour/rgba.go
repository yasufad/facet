package colour

// Rgba is a colour in the sRGB space with straight alpha.
//
// Each component is a float32 in [0, 1]. The zero value is transparent black.
type Rgba struct {
	R float32 // Red, in [0, 1].
	G float32 // Green, in [0, 1].
	B float32 // Blue, in [0, 1].
	A float32 // Alpha, in [0, 1]; 1 is fully opaque.
}

// Rgb constructs an opaque Rgba from a packed 0xRRGGBB integer.
func Rgb(hex uint32) Rgba {
	return Rgba{
		R: float32((hex>>16)&0xff) / 255,
		G: float32((hex>>8)&0xff) / 255,
		B: float32(hex&0xff) / 255,
		A: 1,
	}
}

// Rgba32 constructs an Rgba from a packed 0xRRGGBBAA integer.
func Rgba32(hex uint32) Rgba {
	return Rgba{
		R: float32((hex>>24)&0xff) / 255,
		G: float32((hex>>16)&0xff) / 255,
		B: float32((hex>>8)&0xff) / 255,
		A: float32(hex&0xff) / 255,
	}
}

// IsOpaque reports whether the colour is fully opaque. The renderer uses this
// to skip blending.
func (c Rgba) IsOpaque() bool {
	return c.A >= 1
}

// Opacity returns a copy with the alpha scaled by factor, clamped to [0, 1].
func (c Rgba) Opacity(factor float32) Rgba {
	return Rgba{R: c.R, G: c.G, B: c.B, A: c.A * clamp01(factor)}
}

// Premultiply returns the colour with its red, green and blue components
// multiplied by its alpha, leaving alpha unchanged. The renderer wants
// premultiplied components; colours are stored straight so that blending and
// interpolation remain accurate, and this is the explicit conversion.
func (c Rgba) Premultiply() Rgba {
	return Rgba{R: c.R * c.A, G: c.G * c.A, B: c.B * c.A, A: c.A}
}

// Lighten returns the colour with its lightness increased by amount, computed
// through HSL. amount is added to lightness and the result clamped to [0, 1].
func (c Rgba) Lighten(amount float32) Rgba {
	return c.Hsla().Lighten(amount).Rgba()
}

// Darken returns the colour with its lightness decreased by amount, computed
// through HSL. amount is subtracted from lightness and the result clamped to
// [0, 1].
func (c Rgba) Darken(amount float32) Rgba {
	return c.Hsla().Darken(amount).Rgba()
}
