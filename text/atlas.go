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

// SubpixelFor returns the bucket for a fractional pixel position in [0, 1).
func SubpixelFor(frac float32) SubpixelOffset {
	if frac < 1.0/3.0 {
		return SubpixelZero
	}
	if frac < 2.0/3.0 {
		return SubpixelThird
	}
	return SubpixelTwoThirds
}

func subpixelFor(frac float32) SubpixelOffset {
	return SubpixelFor(frac)
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
//
// ScaleFactor is the display's device pixels per logical pixel. Glyphs are
// rasterised at device-pixel resolution and their bounds are converted
// through geometry.BoundsToDevicePixels with this factor, so atlas tiles
// agree with the geometry around them at any scale.
type Atlas struct {
	entries     map[atlasKey]AtlasEntry
	ScaleFactor float32
}

// NewAtlas returns an empty glyph atlas for a display with the given scale
// factor (device pixels per logical pixel). Pass 1.0 for a standard display,
// 2.0 for a HiDPI display.
func NewAtlas(scaleFactor float32) *Atlas {
	return &Atlas{entries: make(map[atlasKey]AtlasEntry), ScaleFactor: scaleFactor}
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
	// Rasterise at device-pixel resolution: the logical size scaled by the
	// display factor, divided by units-per-em.
	deviceScale := size * a.ScaleFactor / float32(upem)
	mask := rasteriseGlyph(face.face, gid, deviceScale)

	// Logical bounds in pixels: the glyph's extents in font units, scaled by
	// the logical size over units-per-em.
	logicalScale := size / float32(upem)
	extents, ok := extentsOf(face.face, gid)
	if !ok {
		return AtlasEntry{Mask: mask}
	}
	logical := geometry.Bounds[geometry.Pixels]{
		Origin: geometry.NewPoint(
			geometry.Pixels(extents.XBearing*logicalScale),
			geometry.Pixels(extents.YBearing*logicalScale),
		),
		Size: geometry.NewSize(
			geometry.Pixels(extents.Width*logicalScale),
			geometry.Pixels(-extents.Height*logicalScale),
		),
	}
	// Convert to device pixels via the shared snapping rule with the real
	// display scale, so atlas tiles agree with the geometry around them.
	device := geometry.BoundsToDevicePixels(logical, a.ScaleFactor)
	return AtlasEntry{Mask: mask, Bounds: device}
}

// Clear empties the atlas, freeing the rasterised masks.
func (a *Atlas) Clear() {
	a.entries = make(map[atlasKey]AtlasEntry)
}
