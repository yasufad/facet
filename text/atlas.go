package text

import (
	"github.com/go-text/typesetting/font"
	"github.com/yasufad/facet/geometry"
)

// SubpixelOffset quantises the fractional pen position that affects glyph
// rasterisation. Three buckets are enough to capture the visible subpixel
// shift without exploding the atlas: 0, 1/3 and 2/3 of a pixel.
type SubpixelOffset uint8

const (
	SubpixelZero      SubpixelOffset = 0
	SubpixelThird     SubpixelOffset = 1
	SubpixelTwoThirds SubpixelOffset = 2
)

// subpixelFor returns the bucket for a fractional pixel position.
func subpixelFor(frac float32) SubpixelOffset {
	// frac in [0, 1)
	if frac < 1.0/3.0 {
		return SubpixelZero
	}
	if frac < 2.0/3.0 {
		return SubpixelThird
	}
	return SubpixelTwoThirds
}

// AtlasEntry is one rasterised glyph in the atlas: its mask, its device-pixel
// bounds relative to the glyph's pen position, and the face it came from.
type AtlasEntry struct {
	Mask   RasterMask
	Bounds geometry.Bounds[geometry.DevicePixels]
}

// atlasKey identifies a rasterised glyph. Two glyphs with the same key share
// an atlas entry, so the same character at the same size and subpixel offset
// is rasterised once.
type atlasKey struct {
	face     *font.Face
	gid      font.GID
	size     uint32 // bits of float32
	subpixel SubpixelOffset
}

// Atlas caches rasterised glyphs keyed by face, glyph ID, size and subpixel
// offset. It is not safe for concurrent use.
//
// The atlas does not pack masks into a texture; that is the renderer's job.
// It stores each mask separately so the renderer can upload them however it
// likes, and so the atlas stays a pure data structure with no graphics
// dependencies.
type Atlas struct {
	entries map[atlasKey]AtlasEntry
}

// NewAtlas returns an empty glyph atlas.
func NewAtlas() *Atlas {
	return &Atlas{entries: make(map[atlasKey]AtlasEntry)}
}

// Entry returns the atlas entry for a glyph, rasterising on miss. The glyph's
// device-pixel bounds go through geometry.BoundsToDevicePixels so the atlas
// tiles agree with the geometry around them: both edges are snapped and the
// size is derived, rather than rounding origin and size independently.
func (a *Atlas) Entry(face Face, gid GlyphID, size geometry.Pixels, subpixel SubpixelOffset) AtlasEntry {
	if !face.valid() {
		return AtlasEntry{}
	}
	key := atlasKey{
		face:     face.face,
		gid:      font.GID(gid),
		size:     bitsOfFloat32(float32(size)),
		subpixel: subpixel,
	}
	if e, ok := a.entries[key]; ok {
		return e
	}
	e := a.rasterise(face, font.GID(gid), float32(size), subpixel)
	a.entries[key] = e
	return e
}

// rasterise produces an atlas entry for one glyph.
func (a *Atlas) rasterise(face Face, gid font.GID, size float32, subpixel SubpixelOffset) AtlasEntry {
	upem := face.face.Upem()
	if upem == 0 {
		return AtlasEntry{}
	}
	scale := size / float32(upem)

	outline := glyphOutline(face.face, gid)
	mask := rasterise(outline, scale, 0)

	// Logical bounds in pixels: the glyph's extents in font units, scaled.
	extents, ok := extentsOf(face.face, gid)
	if !ok {
		return AtlasEntry{Mask: mask}
	}
	logical := geometry.Bounds[geometry.Pixels]{
		Origin: geometry.NewPoint(
			geometry.Pixels(extents.XBearing*scale),
			geometry.Pixels(extents.YBearing*scale),
		),
		Size: geometry.NewSize(
			geometry.Pixels(extents.Width*scale),
			geometry.Pixels(-extents.Height*scale),
		),
	}
	// Convert to device pixels via the shared snapping rule so atlas tiles
	// agree with the geometry around them. The factor is 1.0 because size is
	// already in device pixels; the conversion is applied for its snapping
	// behaviour, not its scaling.
	device := geometry.BoundsToDevicePixels(logical, 1.0)
	return AtlasEntry{Mask: mask, Bounds: device}
}

// Clear empties the atlas, freeing the rasterised masks.
func (a *Atlas) Clear() {
	a.entries = make(map[atlasKey]AtlasEntry)
}
