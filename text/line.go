package text

import (
	"sort"

	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/shaping"
	"github.com/yasufad/facet/geometry"
)

// ShapedLine is a laid-out line of text: its runs in visual order, its
// metrics, and the mapping between byte offsets within the line and x
// positions. It is the unit a caller paints and hit-tests against.
type ShapedLine struct {
	runs    []ShapedRun
	width   geometry.Pixels
	ascent  geometry.Pixels
	descent geometry.Pixels
	len     int // byte length of the line's text

	// clusters are the line's glyph clusters sorted by x position, supporting
	// IndexForX. byByte is the same clusters sorted by start byte, supporting
	// XForIndex. A cluster may span several bytes (a ligature) or several
	// glyphs (a decomposition); the mapping treats it as one caret boundary.
	clusters []cluster
	byByte   []cluster
}

// cluster is one caret boundary: the x of its left edge and the byte range it
// covers within the line.
type cluster struct {
	x         geometry.Pixels
	startByte int
	endByte   int
}

// Runs returns the shaped runs in visual order, left to right.
func (l ShapedLine) Runs() []ShapedRun { return l.runs }

// Width returns the line's advance width.
func (l ShapedLine) Width() geometry.Pixels { return l.width }

// Ascent returns the distance from the baseline to the top of the line.
func (l ShapedLine) Ascent() geometry.Pixels { return l.ascent }

// Descent returns the distance from the baseline to the bottom of the line, as
// a positive number.
func (l ShapedLine) Descent() geometry.Pixels { return l.descent }

// Height returns the line's total height: ascent plus descent.
func (l ShapedLine) Height() geometry.Pixels { return l.ascent + l.descent }

// Len returns the byte length of the line's text.
func (l ShapedLine) Len() int { return l.len }

// XForIndex returns the x position of the caret boundary at the given byte
// offset within the line. An offset of 0 is the line's left edge; an offset of
// Len is its right edge. Offsets inside a multi-byte cluster round to the
// cluster's boundary.
func (l ShapedLine) XForIndex(byteIndex int) geometry.Pixels {
	if byteIndex <= 0 {
		if len(l.byByte) != 0 {
			return l.byByte[0].x
		}
		return 0
	}
	if byteIndex >= l.len {
		return l.width
	}
	// Find the cluster whose [startByte, endByte) contains byteIndex.
	i := sort.Search(len(l.byByte), func(i int) bool {
		return l.byByte[i].endByte > byteIndex
	})
	if i >= len(l.byByte) {
		return l.width
	}
	return l.byByte[i].x
}

// IndexForX returns the byte offset of the caret boundary at the given x
// position, or (lineLen, false) when x lies at or beyond the line's right edge.
// Hit testing uses this to turn a click into a text position.
func (l ShapedLine) IndexForX(x geometry.Pixels) (int, bool) {
	if len(l.clusters) == 0 || x < l.clusters[0].x {
		return 0, true
	}
	if x >= l.width {
		return l.len, false
	}
	// Last cluster whose left edge is at or before x.
	i := sort.Search(len(l.clusters), func(i int) bool {
		return l.clusters[i].x > x
	})
	if i == 0 {
		return 0, true
	}
	return l.clusters[i-1].startByte, true
}

// ClosestIndexForX returns the byte offset of the boundary nearest to x,
// rounding to whichever side of the cluster midpoint x falls on. It is the
// query arrow keys and mouse selection want when the exact boundary is
// ambiguous.
func (l ShapedLine) ClosestIndexForX(x geometry.Pixels) int {
	if len(l.clusters) == 0 {
		return 0
	}
	if x <= l.clusters[0].x {
		return 0
	}
	if x >= l.width {
		return l.len
	}
	for i := 0; i < len(l.clusters); i++ {
		next := l.width
		if i+1 < len(l.clusters) {
			next = l.clusters[i+1].x
		}
		mid := (l.clusters[i].x + next) / 2
		if x <= mid {
			return l.clusters[i].startByte
		}
	}
	return l.len
}

// buildLine assembles a ShapedLine from a wrapped shaping.Line (runs in visual
// order) and the paragraph coordinate tables. lineStartRune is the rune index
// where the line begins; runeToByte maps paragraph rune indices to byte
// offsets within the paragraph.
func buildLine(
	line shaping.Line,
	paragraph []rune,
	runeToByte []int,
	lineStartRune int,
) ShapedLine {
	shaped := ShapedLine{}

	var ascent, descent float32
	penX := float32(0)

	// First pass: compute metrics and the line's rune range.
	lineEndRune := lineStartRune
	for _, run := range line {
		if a := fromFixed(run.LineBounds.Ascent); a > ascent {
			ascent = a
		}
		if d := -fromFixed(run.LineBounds.Descent); d > descent {
			descent = d
		}
		end := run.Runes.Offset + run.Runes.Count
		if end > lineEndRune {
			lineEndRune = end
		}
	}
	shaped.ascent = geometry.Pixels(ascent)
	shaped.descent = geometry.Pixels(descent)
	baseline := ascent

	lineBaseByte := runeToByte[lineStartRune]
	shaped.len = runeToByte[lineEndRune] - lineBaseByte

	// Cluster boundaries keyed by paragraph rune index, to deduplicate
	// multi-glyph clusters and pick the leftmost x.
	clusterX := map[int]geometry.Pixels{}
	clusterRange := map[int][2]int{}

	shaped.runs = make([]ShapedRun, 0, len(line))
	for _, run := range line {
		glyphs := make([]Glyph, len(run.Glyphs))
		for i, g := range run.Glyphs {
			x := penX + fromFixed(g.XOffset)
			glyphs[i] = Glyph{
				ID:       GlyphID(g.GlyphID),
				Position: geometry.NewPoint(geometry.Pixels(x), geometry.Pixels(baseline+fromFixed(g.YOffset))),
				Cluster:  runeToByte[g.ClusterIndex] - lineBaseByte,
				Face:     Face{face: run.Face},
			}
			penX += fromFixed(g.Advance)

			// Record the cluster boundary. The first glyph encountered for a
			// cluster is the leftmost, since runs are in visual order.
			if _, set := clusterX[g.ClusterIndex]; !set {
				clusterX[g.ClusterIndex] = geometry.Pixels(x)
				runesNext := g.ClusterIndex + g.RuneCount
				if runesNext > g.ClusterIndex {
					clusterRange[g.ClusterIndex] = [2]int{
						runeToByte[g.ClusterIndex] - lineBaseByte,
						runeToByte[runesNext] - lineBaseByte,
					}
				} else {
					clusterRange[g.ClusterIndex] = [2]int{
						runeToByte[g.ClusterIndex] - lineBaseByte,
						runeToByte[g.ClusterIndex] - lineBaseByte,
					}
				}
			}
		}
		shaped.runs = append(shaped.runs, ShapedRun{
			Face:      Face{face: run.Face},
			Direction: fromDi(run.Direction),
			Glyphs:    glyphs,
		})
	}
	shaped.width = geometry.Pixels(penX)

	// Build the cluster slices and sort them.
	shaped.clusters = make([]cluster, 0, len(clusterX))
	for runeIdx, x := range clusterX {
		r := clusterRange[runeIdx]
		shaped.clusters = append(shaped.clusters, cluster{x: x, startByte: r[0], endByte: r[1]})
	}
	sort.Slice(shaped.clusters, func(i, j int) bool {
		return shaped.clusters[i].x < shaped.clusters[j].x
	})
	shaped.byByte = append([]cluster(nil), shaped.clusters...)
	sort.Slice(shaped.byByte, func(i, j int) bool {
		return shaped.byByte[i].startByte < shaped.byByte[j].startByte
	})
	return shaped
}

// fromDi converts a typesetting direction to our Direction.
func fromDi(d di.Direction) Direction {
	if d.Progression() == di.FromTopLeft {
		return LTR
	}
	return RTL
}
