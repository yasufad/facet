package text

import (
	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
	"github.com/yasufad/facet/geometry"
)

// SegmentOp identifies the kind of an outline segment.
type SegmentOp uint8

const (
	// SegMoveTo starts a new contour at the given point.
	SegMoveTo SegmentOp = iota
	// SegLineTo draws a straight line to the given point.
	SegLineTo
	// SegQuadTo draws a quadratic Bézier to the end point, using the single
	// control point.
	SegQuadTo
	// SegCubeTo draws a cubic Bézier to the end point, using two control
	// points.
	SegCubeTo
)

// Segment is one piece of a glyph outline. Coordinates are in font units,
// with Y increasing up. A glyph outline is a sequence of closed contours,
// each beginning with a SegMoveTo.
type Segment struct {
	Op   SegmentOp
	Args [3]geometry.Point[float32]
}

// Outline is a glyph's vector outline in font units, Y-up. Glyphs without an
// outline (whitespace, bitmaps the rasteriser does not handle) return a zero
// Outline with no segments.
type Outline struct {
	Segments []Segment
}

// glyphOutline fetches the outline for a glyph in the given face, converting
// the typesetting segment representation into our own. Bitmap, SVG and COLR
// glyphs are not handled; their outlines, when present as a fallback, are.
func glyphOutline(face *font.Face, gid font.GID) Outline {
	if face == nil {
		return Outline{}
	}
	data := face.GlyphData(gid)
	if data == nil {
		return Outline{}
	}
	outline, ok := data.(font.GlyphOutline)
	if !ok {
		// Bitmap, SVG or colour glyphs: try the embedded outline fallback
		// that typesetting attaches to some of them.
		switch g := data.(type) {
		case font.GlyphSVG:
			outline = g.Outline
		case font.GlyphBitmap:
			if g.Outline != nil {
				outline = *g.Outline
			}
		}
	}
	out := Outline{Segments: make([]Segment, len(outline.Segments))}
	for i, seg := range outline.Segments {
		out.Segments[i] = convertSegment(seg)
	}
	return out
}

// convertSegment translates a typesetting segment into our own representation.
func convertSegment(seg ot.Segment) Segment {
	out := Segment{Op: SegmentOp(seg.Op)}
	for i, p := range seg.ArgsSlice() {
		out.Args[i] = geometry.NewPoint(float32(p.X), float32(p.Y))
	}
	return out
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
