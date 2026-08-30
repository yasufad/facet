package text

import (
	"github.com/yasufad/facet/geometry"
)

// supersample is the per-axis oversampling factor. Each pixel is split into a
// supersample×supersample grid; a glyph is rasterised at that resolution and
// averaged down to one byte of coverage per pixel. 4 is the sweet spot: it
// gives 16 coverage levels per pixel at a cost of 16× the work, and matches
// what FreeType's light autohinter produces visually.
const supersample = 4

// maxCoverage is the largest value a coverage byte can take. A pixel fully
// inside the glyph is maxCoverage; one fully outside is 0.
const maxCoverage = 255

// RasterMask is an antialiased coverage bitmap for a single glyph. Width and
// Height are in device pixels; the origin is the top-left of the bitmap.
// Coverage is row-major, row stride equals Width.
type RasterMask struct {
	Width    int
	Height   int
	Coverage []byte
}

// rasterise converts a glyph outline into a coverage mask. The outline is in
// font units, Y-up; scale converts font units to device pixels. The mask's
// origin is placed so that the glyph's origin (its pen position) lands at
// mask coordinate (0, baseline). The caller positions the mask in the atlas
// using the returned bounds.
//
// The rasteriser is a supersampled scanline converter: it tessellates each
// contour into a coverage accumulator at supersample resolution, then boxes
// the accumulator down to one coverage byte per device pixel. It handles
// move, line, quadratic and cubic segments, the four operations OpenType
// outlines use.
func rasterise(outline Outline, scale float32, baseline int) RasterMask {
	if len(outline.Segments) == 0 {
		return RasterMask{}
	}

	// Compute the device-pixel bounds of the glyph by tracing the outline.
	bounds := deviceOutlineBounds(outline, scale)
	if bounds.Width <= 0 || bounds.Height <= 0 {
		return RasterMask{Width: bounds.Width, Height: bounds.Height}
	}

	w := bounds.Width
	h := bounds.Height
	sw := w * supersample
	sh := h * supersample

	// The accumulator holds coverage samples in [0, supersample] per cell.
	// A coverage of supersample means the cell's centre is inside the glyph.
	acc := newAccumulator(sw, sh)

	// Transform: outline point (fx, fy) in font units -> accumulator cell.
	// The glyph origin maps to (0, baseline) in device pixels, which is
	// (0, baseline*supersample) in accumulator cells. Outline Y is up;
	// accumulator Y is down. So cellY = (baseline - fy*scale) * supersample,
	// shifted so the top of the bounds is 0.
	originX := float32(bounds.OriginX)
	originY := float32(bounds.OriginY)
	toCell := func(fx, fy float32) (float32, float32) {
		x := (fx*scale - originX) * supersample
		y := (originY - fy*scale) * supersample
		return x, y
	}

	// Trace each contour, filling the accumulator's winding buffer.
	for i := 0; i < len(outline.Segments); {
		i = traceContour(outline.Segments, i, acc, toCell)
	}

	// Convert winding deltas to coverage per supersample row using the
	// non-zero rule: walk each row left to right, accumulating winding; a
	// cell is inside when the running winding is non-zero.
	acc.scanFill()

	// Box down to device pixels.
	cov := make([]byte, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			sum := 0
			for sy := 0; sy < supersample; sy++ {
				row := (y*supersample + sy) * sw
				for sx := 0; sx < supersample; sx++ {
					if acc.cover[row+x*supersample+sx] != 0 {
						sum++
					}
				}
			}
			cov[y*w+x] = byte((sum*maxCoverage + (supersample*supersample)/2) / (supersample * supersample))
		}
	}
	return RasterMask{Width: w, Height: h, Coverage: cov}
}

// deviceBounds describes a glyph's device-pixel placement. OriginX and
// OriginY are the top-left of the bitmap in device pixels relative to the
// glyph's pen position; Width and Height are the bitmap dimensions.
type deviceBounds struct {
	OriginX int
	OriginY int
	Width   int
	Height  int
}

// deviceOutlineBounds computes the device-pixel bounds of an outline by
// scanning its points. The bounds are floored at the top-left and ceiled at
// the bottom-right so the bitmap always contains the whole glyph.
func deviceOutlineBounds(outline Outline, scale float32) deviceBounds {
	minX, minY := float32(1e30), float32(1e30)
	maxX, maxY := float32(-1e30), float32(-1e30)
	for _, seg := range outline.Segments {
		for _, p := range seg.Args {
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
	if maxX < minX {
		return deviceBounds{}
	}
	// Y is up in the outline; the bitmap's top is the glyph's maximum Y.
	return deviceBounds{
		OriginX: floorInt(minX),
		OriginY: floorInt(maxY),
		Width:   ceilInt(maxX) - floorInt(minX),
		Height:  ceilInt(maxY) - floorInt(minY),
	}
}

func floorInt(v float32) int {
	if v >= 0 {
		return int(v)
	}
	n := int(v)
	if float32(n) != v {
		n--
	}
	return n
}

func ceilInt(v float32) int {
	if v <= 0 {
		return 0
	}
	n := int(v)
	if float32(n) != v {
		n++
	}
	return n
}

// traceContour traces one contour starting at segment i, filling acc, and
// returns the index of the first segment of the next contour. A contour is a
// sequence of segments starting with SegMoveTo and running until the next
// SegMoveTo or the end of the outline.
func traceContour(segs []Segment, start int, acc *accumulator, toCell func(float32, float32) (float32, float32)) int {
	if start >= len(segs) || segs[start].Op != SegMoveTo {
		return start + 1
	}
	cur := segs[start].Args[0]
	cx, cy := toCell(cur.X, cur.Y)
	i := start + 1
	for i < len(segs) && segs[i].Op != SegMoveTo {
		seg := segs[i]
		switch seg.Op {
		case SegLineTo:
			nx, ny := toCell(seg.Args[0].X, seg.Args[0].Y)
			acc.line(cx, cy, nx, ny)
			cx, cy = nx, ny
		case SegQuadTo:
			cx, cy = acc.quad(cx, cy, toCell, seg.Args[0], seg.Args[1])
		case SegCubeTo:
			cx, cy = acc.cubic(cx, cy, toCell, seg.Args[0], seg.Args[1], seg.Args[2])
		}
		i++
	}
	// Close the contour back to the move-to point.
	startX, startY := toCell(segs[start].Args[0].X, segs[start].Args[0].Y)
	acc.line(cx, cy, startX, startY)
	return i
}

// accumulator is a supersampled coverage grid. It uses the scanline
// intersection / winding-number approach: for each scanline it records the
// x-intersections of contour edges, then fills between them with the non-zero
// winding rule.
//
// This implementation uses a direct edge-walk that adds +winding or -winding
// to each cell an edge passes through, then a per-row prefix sum converts the
// winding into coverage. It is the classic "active edge" approach simplified
// to integer cells.
type accumulator struct {
	w, h  int
	wind  []int16 // per-cell winding delta
	cover []int16 // per-cell coverage (1 inside, 0 outside) after scanFill
}

func newAccumulator(w, h int) *accumulator {
	return &accumulator{
		w:    w,
		h:    h,
		wind: make([]int16, w*h),
	}
}

// scanFill converts the per-cell winding deltas into per-cell coverage using
// the non-zero winding rule. A cell is inside the glyph when the running
// winding number across its row is non-zero.
func (a *accumulator) scanFill() {
	a.cover = make([]int16, len(a.wind))
	for y := 0; y < a.h; y++ {
		row := y * a.w
		winding := int16(0)
		for x := 0; x < a.w; x++ {
			winding += a.wind[row+x]
			if winding != 0 {
				a.cover[row+x] = 1
			}
		}
	}
}

// line adds a straight edge from (x0,y0) to (x1,y1) to the winding buffer.
func (a *accumulator) line(x0, y0, x1, y1 float32) {
	// We use a DDA over the major axis. For each scanline row crossed, we
	// record the x at which the edge enters and the winding direction.
	if y0 == y1 {
		return // horizontal edges contribute no winding
	}
	dir := int16(1)
	if y1 < y0 {
		x0, x1 = x1, x0
		y0, y1 = y1, y0
		dir = -1
	}
	// Clamp to the accumulator.
	if y1 <= 0 || y0 >= float32(a.h) {
		return
	}
	startY := int(y0)
	if startY < 0 {
		startY = 0
	}
	endY := int(y1)
	if endY > a.h {
		endY = a.h
	}
	slope := (x1 - x0) / (y1 - y0)
	for y := startY; y < endY; y++ {
		yc := float32(y) + 0.5
		x := x0 + (yc-y0)*slope
		xi := int(x)
		if xi < 0 {
			xi = 0
		} else if xi >= a.w {
			xi = a.w - 1
		}
		a.wind[y*a.w+xi] += dir
	}
}

// quad flattens a quadratic Bézier into line segments and adds them.
func (a *accumulator) quad(x0, y0 float32, toCell func(float32, float32) (float32, float32), c, end geometry.Point[float32]) (float32, float32) {
	cx, cy := toCell(c.X, c.Y)
	ex, ey := toCell(end.X, end.Y)
	steps := quadSteps(x0, y0, cx, cy, ex, ey)
	for i := 1; i <= steps; i++ {
		t := float32(i) / float32(steps)
		mt := 1 - t
		nx := mt*mt*x0 + 2*mt*t*cx + t*t*ex
		ny := mt*mt*y0 + 2*mt*t*cy + t*t*ey
		a.line(x0, y0, nx, ny)
		x0, y0 = nx, ny
	}
	return ex, ey
}

// cubic flattens a cubic Bézier into line segments and adds them.
func (a *accumulator) cubic(x0, y0 float32, toCell func(float32, float32) (float32, float32), c1, c2, end geometry.Point[float32]) (float32, float32) {
	c1x, c1y := toCell(c1.X, c1.Y)
	c2x, c2y := toCell(c2.X, c2.Y)
	ex, ey := toCell(end.X, end.Y)
	steps := cubicSteps(x0, y0, c1x, c1y, c2x, c2y, ex, ey)
	for i := 1; i <= steps; i++ {
		t := float32(i) / float32(steps)
		mt := 1 - t
		nx := mt*mt*mt*x0 + 3*mt*mt*t*c1x + 3*mt*t*t*c2x + t*t*t*ex
		ny := mt*mt*mt*y0 + 3*mt*mt*t*c1y + 3*mt*t*t*c2y + t*t*t*ey
		a.line(x0, y0, nx, ny)
		x0, y0 = nx, ny
	}
	return ex, ey
}

// quadSteps picks a flattening step count for a quadratic. The count grows
// with the control polygon's size so curves stay smooth at any scale.
func quadSteps(x0, y0, cx, cy, ex, ey float32) int {
	d := absF(cx-x0) + absF(cy-y0) + absF(ex-cx) + absF(ey-cy)
	n := int(d)
	if n < 2 {
		n = 2
	}
	if n > 64 {
		n = 64
	}
	return n
}

// cubicSteps picks a flattening step count for a cubic.
func cubicSteps(x0, y0, c1x, c1y, c2x, c2y, ex, ey float32) int {
	d := absF(c1x-x0) + absF(c1y-y0) + absF(c2x-c1x) + absF(c2y-c1y) + absF(ex-c2x) + absF(ey-c2y)
	n := int(d)
	if n < 2 {
		n = 2
	}
	if n > 96 {
		n = 96
	}
	return n
}

func absF(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
