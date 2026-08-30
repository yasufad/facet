package scene

import "github.com/yasufad/facet/geometry"

// AtlasTextureKind labels the content of an atlas texture, so the renderer
// picks the right sampler and shader path. The Scene carries the kind as data
// only; it does not decide which kind a glyph or image lands in — that is the
// atlas owner's job, in text and render.
type AtlasTextureKind uint8

const (
	// TextureMonochrome holds single-channel coverage maps: glyphs drawn with a
	// caller-supplied colour through a MonochromeSprite.
	TextureMonochrome AtlasTextureKind = iota
	// TexturePolychrome holds full-colour images and emoji, drawn through a
	// PolychromeSprite.
	TexturePolychrome
	// TextureSubpixel holds subpixel-antialiased glyph coverage. No primitive
	// in the current set consumes it; the kind is defined so the atlas can
	// reserve a texture for it without the Scene gaining a new primitive.
	TextureSubpixel
)

// AtlasTextureID identifies one texture within an atlas. The index is a u32 for
// shader compatibility across Metal, D3D11 and Vulkan.
type AtlasTextureID struct {
	Index uint32
	Kind  AtlasTextureKind
}

// TileID identifies one tile within its texture. It is the serialised allocator
// handle from the atlas packer, opaque to the Scene.
type TileID uint32

// AtlasTile is the reference a sprite carries to a region of an atlas texture.
// It names the texture and tile by identifier and gives the tile's rectangle
// within the texture, in device pixels. The Scene does not own an atlas,
// allocate from one, or know how glyphs get into it; it only records the
// reference so the renderer can look the tile up.
type AtlasTile struct {
	TextureID AtlasTextureID
	TileID    TileID
	Padding   uint32
	Bounds    geometry.Bounds[geometry.DevicePixels]
}
