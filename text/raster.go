package text

import (
	"image"
	"image/color"
	"image/draw"
	"math"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"golang.org/x/image/vector"
)

// RasterMask is an antialiased coverage bitmap for a single glyph. Width and
// Height are in device pixels; the origin is the top-left of the bitmap.
// Coverage is row-major, row stride equals Width.
type RasterMask struct {
	Width    int
	Height   int
	Coverage []byte
}

// rasteriseGlyph extracts the outline for gid from face and rasterises it at
// the given scale (device pixels per font unit) using golang.org/x/image/vector,
// which computes analytic area coverage — all 256 levels — with SIMD paths
// on amd64 and arm64.
//
// The outline is in font units with Y increasing up; the mask is in device
// pixels with Y increasing down. The mask's top-left is placed at the glyph's
// bounding-box top-left so that the caller positions it using the glyph's
// bearing offsets.
func rasteriseGlyph(face *font.Face, gid font.GID, scale float32) RasterMask {
	if face == nil {
		return RasterMask{}
	}
	outline := glyphOutline(face, gid)
	if len(outline.Segments) == 0 {
		return RasterMask{}
	}

	// Compute the device-pixel bounds of the outline by scanning its points.
	minX, minY, maxX, maxY := outlineBounds(outline.Segments, scale)
	if maxX <= minX || maxY <= minY {
		return RasterMask{}
	}
	w := int(math.Ceil(float64(maxX - minX)))
	h := int(math.Ceil(float64(maxY - minY)))
	if w <= 0 || h <= 0 {
		return RasterMask{}
	}

	// Feed the outline into the vector rasteriser, scaling from font units to
	// device pixels and flipping Y. The origin is placed so that the glyph's
	// top-left bounding-box corner lands at (0, 0) in the mask.
	z := vector.NewRasterizer(w, h)
	originX := minX
	originY := maxY // top of the bitmap in Y-up font coordinates
	for _, seg := range outline.Segments {
		pts := seg.ArgsSlice()
		switch seg.Op {
		case ot.SegmentOpMoveTo:
			z.MoveTo(pts[0].X*scale-originX, originY-pts[0].Y*scale)
		case ot.SegmentOpLineTo:
			z.LineTo(pts[0].X*scale-originX, originY-pts[0].Y*scale)
		case ot.SegmentOpQuadTo:
			z.QuadTo(
				pts[0].X*scale-originX, originY-pts[0].Y*scale,
				pts[1].X*scale-originX, originY-pts[1].Y*scale,
			)
		case ot.SegmentOpCubeTo:
			z.CubeTo(
				pts[0].X*scale-originX, originY-pts[0].Y*scale,
				pts[1].X*scale-originX, originY-pts[1].Y*scale,
				pts[2].X*scale-originX, originY-pts[2].Y*scale,
			)
		}
	}
	z.ClosePath()

	// Draw the rasterised path into an alpha image. With draw.Src and an
	// opaque source the rasteriser takes its fast path, writing coverage
	// bytes directly into dst.Pix.
	dst := image.NewAlpha(image.Rect(0, 0, w, h))
	z.DrawOp = draw.Src
	z.Draw(dst, dst.Bounds(), image.NewUniform(color.Opaque), image.Point{})

	return RasterMask{
		Width:    w,
		Height:   h,
		Coverage: dst.Pix,
	}
}

// glyphOutline extracts the vector outline for a glyph, checking the embedded
// outline fallback that typesetting attaches to some SVG and bitmap glyphs.
func glyphOutline(face *font.Face, gid font.GID) font.GlyphOutline {
	data := face.GlyphData(gid)
	if data == nil {
		return font.GlyphOutline{}
	}
	outline, ok := data.(font.GlyphOutline)
	if !ok {
		switch g := data.(type) {
		case font.GlyphSVG:
			outline = g.Outline
		case font.GlyphBitmap:
			if g.Outline != nil {
				outline = *g.Outline
			}
		}
	}
	return outline
}

// outlineBounds returns the device-pixel bounding box of an outline's
// segments, scaled by scale. Y is up in the outline; the returned minY is the
// bottom and maxY is the top.
func outlineBounds(segs []ot.Segment, scale float32) (minX, minY, maxX, maxY float32) {
	minX, minY = math.MaxFloat32, math.MaxFloat32
	maxX, maxY = -math.MaxFloat32, -math.MaxFloat32
	for _, seg := range segs {
		for _, p := range seg.ArgsSlice() {
			x := p.X * scale
			y := p.Y * scale
			if x < minX {
				minX = x
			}
			if x > maxX {
				maxX = x
			}
			if y < minY {
				minY = y
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	return
}

// GlyphExtents returns the tight bounding box of a glyph in font units,
// Y-up. XBearing and YBearing are the offset from the glyph origin to the
// box's top-left; Width is horizontal and Height is negative (downward).
type GlyphExtents struct {
	XBearing float32
	YBearing float32
	Width    float32
	Height   float32
}

// extentsOf returns the extents of gid in face, in font units.
func extentsOf(face *font.Face, gid font.GID) (GlyphExtents, bool) {
	if face == nil {
		return GlyphExtents{}, false
	}
	e, ok := face.GlyphExtents(gid)
	return GlyphExtents{
		XBearing: e.XBearing,
		YBearing: e.YBearing,
		Width:    e.Width,
		Height:   e.Height,
	}, ok
}
