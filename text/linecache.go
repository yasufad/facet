package text

import (
	"unsafe"

	"golang.org/x/image/math/fixed"
)

// defaultLineCacheBytes bounds lineCache. Its entries are the finished
// ShapedLine, which carries the same glyphs as the shape cache plus a
// cluster table roughly twice their count in size (clusters and byByte). At
// the same working set as defaultShapeCacheBytes describes, that runs to
// roughly double the weight per glyph, so the line cache gets the same 4 MiB
// ceiling: it holds fewer entries for the same budget, which is the right
// trade since a line-cache hit skips segmentation and wrapping entirely
// rather than just the HarfBuzz call.
const defaultLineCacheBytes = 4 << 20

// lineCacheKey identifies a wrapped result: the text, its style runs and the
// width it was wrapped to. ShapeLine always wraps at noWrapWidth, so its
// entries are independent of any caller-chosen width; WrapText entries are
// keyed per width, since a different width can produce a different set of
// lines for the same text and runs.
type lineCacheKey struct {
	text     string
	runs     string
	maxWidth fixed.Int26_6
}

// lineCache memoises the fully wrapped result of wrap, so a repeated
// ShapeLine or WrapText call for the same text, runs and width is a map
// lookup and a slice copy rather than a re-segmentation and re-wrap. It sits
// above shapeCache: a hit here skips shapeCache entirely, a miss still uses
// it for any sub-run shaping.
type lineCache struct {
	lru *byteLRU[lineCacheKey, []ShapedLine]
}

func newLineCache() lineCache {
	return lineCache{lru: newByteLRU[lineCacheKey, []ShapedLine](defaultLineCacheBytes, sizeOfShapedLines)}
}

// sizeOfShapedLines estimates the byte weight of a wrapped result for the
// cache's eviction accounting.
func sizeOfShapedLines(lines []ShapedLine) int64 {
	const (
		glyphSize   = int64(unsafe.Sizeof(Glyph{}))
		clusterSize = int64(unsafe.Sizeof(cluster{}))
	)
	var n int64
	for _, l := range lines {
		for _, run := range l.runs {
			n += int64(len(run.Glyphs)) * glyphSize
		}
		// clusters and byByte both hold one cluster value per caret boundary.
		n += int64(len(l.clusters)) * clusterSize * 2
	}
	return n + 32 // +32 for an empty line's own header
}

// styleRunsKey renders a style run slice into a string that changes exactly
// when wrap's result would: every field that reaches shaping or wrapping for
// each run, in order. It is not a general-purpose serialisation — separators
// are chosen to make accidental collisions unlikely, not impossible, which
// matches featuresKey's existing standard for a cache key.
func styleRunsKey(runs []StyleRun) string {
	if len(runs) == 0 {
		return ""
	}
	var b []byte
	for _, r := range runs {
		b = appendUint32(b, uint32(r.ByteLen))
		b = append(b, 0)
		b = append(b, r.Font.Family...)
		b = append(b, 0)
		for _, f := range r.Font.Families {
			b = append(b, f...)
			b = append(b, ',')
		}
		b = append(b, 0)
		b = appendUint32(b, bitsOfFloat32(float32(r.Font.Weight)))
		b = append(b, byte(r.Font.Style))
		b = appendUint32(b, bitsOfFloat32(float32(r.Font.Stretch)))
		b = appendUint32(b, bitsOfFloat32(float32(r.Size)))
		b = append(b, byte(r.Direction))
		b = append(b, r.Language...)
		b = append(b, 0)
		for _, f := range r.Features {
			b = append(b, f.Tag...)
			b = appendUint32(b, f.Value)
			b = append(b, ';')
		}
		b = append(b, '|')
	}
	return string(b)
}

// cloneShapedLines returns a copy of lines whose glyph slices do not alias
// lines itself. ShapedRun.Glyphs is exported, and Glyph's fields (Position in
// particular) are plain mutable values, so a caller that mutates a glyph in
// a line handed back from a cache hit would otherwise corrupt every other
// caller sharing that hit — and the cache's own stored copy along with them,
// since Get and Put would be handing out and taking in the same backing
// array. wrap calls this on every return, hit or miss, so the slice stored
// in the cache is never the same one a caller holds.
//
// clusters and byByte are not cloned: nothing outside this package can reach
// them to mutate. XForIndex, IndexForX and ClosestIndexForX only ever return
// copied geometry.Pixels and int values, never the cluster slice itself.
func cloneShapedLines(lines []ShapedLine) []ShapedLine {
	out := make([]ShapedLine, len(lines))
	for i, l := range lines {
		out[i] = l
		if len(l.runs) == 0 {
			continue
		}
		runs := make([]ShapedRun, len(l.runs))
		for j, r := range l.runs {
			runs[j] = r
			runs[j].Glyphs = append([]Glyph(nil), r.Glyphs...)
		}
		out[i].runs = runs
	}
	return out
}

// SetLineCacheBytes sets the wrapped-line cache's byte ceiling, evicting
// immediately if the cache is already over it. Zero means the cache holds
// nothing, not that it is unbounded. The default is defaultLineCacheBytes.
func (s *System) SetLineCacheBytes(n int64) { s.lineCache.lru.SetMaxBytes(n) }
