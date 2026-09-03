package text

import (
	"hash/maphash"
	"slices"
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

// lineCacheKey identifies a wrapped result: the text, a fingerprint of its
// style runs, and the width it was wrapped to. ShapeLine always wraps at
// noWrapWidth, so its entries are independent of any caller-chosen width;
// WrapText entries are keyed per width, since a different width can produce a
// different set of lines for the same text and runs.
//
// runsHash is a 64-bit digest, not the runs' serialised bytes: ShapeLine is
// called per text element per frame, so building and converting a []byte
// into a string on every lookup — a hit included — is exactly the allocation
// the shape cache's own key was fixed to stop paying, one level up. The hash
// keeps map lookup allocation-free; to eliminate the risk of a digest collision
// silently returning wrong glyphs, each cache entry stores a copy of the runs
// it was built from and confirms them field by field on a hit before returning.
type lineCacheKey struct {
	text     string
	runsHash uint64
	maxWidth fixed.Int26_6
}

// lineCacheEntry stores the shaped lines along with a cloned copy of the style
// runs used to build them, used to confirm a cache hit against hash collision.
type lineCacheEntry struct {
	runs  []StyleRun
	lines []ShapedLine
}

// lineCache memoises the fully wrapped result of wrap, so a repeated
// ShapeLine or WrapText call for the same text, runs and width is a map
// lookup and a slice copy rather than a re-segmentation and re-wrap. It sits
// above shapeCache: a hit here skips shapeCache entirely, a miss still uses
// it for any sub-run shaping.
//
// hash is reused across hashRuns calls rather than constructed fresh each
// time, and is why lineCache is embedded in System by value rather than
// held behind a pointer to a fresh cache per call.
type lineCache struct {
	lru  *byteLRU[lineCacheKey, lineCacheEntry]
	hash maphash.Hash
}

func newLineCache() lineCache {
	return lineCache{lru: newByteLRU[lineCacheKey, lineCacheEntry](defaultLineCacheBytes, sizeOfLineCacheEntry)}
}

// sizeOfLineCacheEntry estimates the byte weight of a wrapped result and its
// stored style runs for the cache's eviction accounting.
func sizeOfLineCacheEntry(e lineCacheEntry) int64 {
	const runSize = int64(unsafe.Sizeof(StyleRun{}))
	return sizeOfShapedLines(e.lines) + int64(len(e.runs))*runSize
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

// hashRuns computes runsHash: every field that reaches shaping or wrapping
// for each run, in order, fed a byte or a string at a time into the reused
// hash. WriteByte and WriteString copy straight into the hash's internal
// state, so this builds no intermediate []byte and converts no string —
// unlike the equivalent string key it replaces, this allocates nothing on a
// hit. Field boundaries are marked the same way featuresKey marks them: not
// a general-purpose serialisation, just enough to make two different run
// sets land on different byte sequences before they are hashed.
func (c *lineCache) hashRuns(runs []StyleRun) uint64 {
	h := &c.hash
	h.Reset()
	for _, r := range runs {
		writeUint32(h, uint32(r.ByteLen))
		h.WriteByte(0)
		h.WriteString(r.Font.Family)
		h.WriteByte(0)
		for _, f := range r.Font.Families {
			h.WriteString(f)
			h.WriteByte(',')
		}
		h.WriteByte(0)
		writeUint32(h, bitsOfFloat32(float32(r.Font.Weight)))
		h.WriteByte(byte(r.Font.Style))
		writeUint32(h, bitsOfFloat32(float32(r.Font.Stretch)))
		writeUint32(h, bitsOfFloat32(float32(r.Size)))
		h.WriteByte(byte(r.Direction))
		h.WriteString(r.Language)
		h.WriteByte(0)
		for _, f := range r.Features {
			h.WriteString(f.Tag)
			writeUint32(h, f.Value)
			h.WriteByte(';')
		}
		h.WriteByte('|')
	}
	return h.Sum64()
}

// writeUint32 feeds v's bytes into h one at a time, so encoding a numeric
// field never needs a []byte the way appendUint32 does for the string-keyed
// caches elsewhere in this package.
func writeUint32(h *maphash.Hash, v uint32) {
	h.WriteByte(byte(v >> 24))
	h.WriteByte(byte(v >> 16))
	h.WriteByte(byte(v >> 8))
	h.WriteByte(byte(v))
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

// get returns the cached shaped lines for key if present and confirmed by an
// exact field-by-field comparison against the entry's stored style runs. If the
// digest matched but the runs differ (a hash collision), get returns nil,
// false so the caller treats it as a miss and shapes afresh.
func (c *lineCache) get(key lineCacheKey, runs []StyleRun) ([]ShapedLine, bool) {
	if entry, ok := c.lru.Get(key); ok && styleRunsEqual(runs, entry.runs) {
		return entry.lines, true
	}
	return nil, false
}

// put stores lines and a cloned copy of runs under key. runs is cloned so a
// caller that mutates its own slice afterwards cannot corrupt the confirmation
// data of an entry already in the cache.
func (c *lineCache) put(key lineCacheKey, runs []StyleRun, lines []ShapedLine) {
	c.lru.Put(key, lineCacheEntry{
		runs:  cloneStyleRuns(runs),
		lines: lines,
	})
}

// cloneStyleRuns returns a copy of runs whose slice fields (Font.Families and
// Features) do not alias runs itself. Stored in lineCacheEntry to confirm a hit
// against hash collision; must not alias caller-owned memory because a caller
// mutating a run after calling ShapeLine or WrapText would otherwise corrupt
// the cache entry's confirmation data.
func cloneStyleRuns(runs []StyleRun) []StyleRun {
	if runs == nil {
		return nil
	}
	out := make([]StyleRun, len(runs))
	for i, r := range runs {
		out[i] = r
		if len(r.Font.Families) > 0 {
			out[i].Font.Families = slices.Clone(r.Font.Families)
		}
		if len(r.Features) > 0 {
			out[i].Features = slices.Clone(r.Features)
		}
	}
	return out
}

// styleRunsEqual reports whether a and b describe identical styling across all
// runs. StyleRun and FontRequest both carry slices (Families and Features), so
// neither is comparable with ==; this is a manual field-by-field comparison
// with slices.Equal on the slice fields.
func styleRunsEqual(a, b []StyleRun) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].ByteLen != b[i].ByteLen ||
			a[i].Size != b[i].Size ||
			a[i].Direction != b[i].Direction ||
			a[i].Language != b[i].Language ||
			a[i].Font.Family != b[i].Font.Family ||
			a[i].Font.Weight != b[i].Font.Weight ||
			a[i].Font.Style != b[i].Font.Style ||
			a[i].Font.Stretch != b[i].Font.Stretch ||
			!slices.Equal(a[i].Font.Families, b[i].Font.Families) ||
			!slices.Equal(a[i].Features, b[i].Features) {
			return false
		}
	}
	return true
}

// SetLineCacheBytes sets the wrapped-line cache's byte ceiling, evicting
// immediately if the cache is already over it. Zero means the cache holds
// nothing, not that it is unbounded. The default is defaultLineCacheBytes.
func (s *System) SetLineCacheBytes(n int64) { s.lineCache.lru.SetMaxBytes(n) }
