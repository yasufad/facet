// Package text turns strings of runes into shaped, measured and rasterised
// glyphs, and stops there. It does not draw: the renderer takes the glyph runs
// and alpha masks this package produces and puts them on screen.
//
// The package wraps github.com/go-text/typesetting, which supplies font loading
// and matching, script and bidi segmentation, HarfBuzz-equivalent shaping and
// line wrapping in pure Go. Nothing above this package knows it exists: shaped
// lines, glyph runs, metrics and rasterised glyphs are exposed in this
// package's own types, so that swapping the shaping engine would be a change
// confined to text.
//
// Two imports from golang.org/x/image are unavoidable. math/fixed is the
// fixed-point type typesetting's API speaks in: font sizes and advances are
// fixed.Int26_6, so constructing a shaping.Input requires naming it. vector
// is the glyph rasteriser: it computes analytic area coverage with SIMD paths
// on amd64 and arm64, and is faster than a hand-written scanline rasteriser
// at every size measured. The comparison is recorded in
// docs/architecture.md.
//
// # Rasterisation
//
// typesetting stops at glyph outlines. This package rasterises them through
// golang.org/x/image/vector, which computes analytic area coverage — all 256
// levels — rather than the 17 levels a 4×4 supersample ceiling allows. GPU
// compute, as GPUI does, belongs to render, which sits above text; the text
// package produces data and does not draw.
//
// # Invariants
//
//   - Caching happens at two levels. lineCache holds the finished ShapedLine
//     for a whole ShapeLine or WrapText call, keyed on the input text, the
//     style runs (font, size, direction, language, features) and the wrap
//     width; a hit skips segmentation, shaping and line wrapping entirely.
//     Underneath it, shapeCache holds shaped output per homogeneous sub-run
//     after segmentation, keyed on the sub-run's text, resolved face, size,
//     direction, script, language and font features; a hit here still pays
//     for segmentation and wrapping, but skips the shaper. Both are bounded by
//     total byte size with least-recently-used eviction, not by entry count,
//     because a shaped word and a shaped heading differ by orders of
//     magnitude in weight. Nothing invalidates an entry early: a face, a
//     resolved glyph and a wrap width are all immutable for as long as they
//     are valid cache inputs, so an entry is correct until it is evicted.
//   - A returned []ShapedLine is the caller's own copy and never aliases
//     lineCache's own copy. ShapedRun.Glyphs is exported and its elements are
//     mutable, so a caller free to mutate a glyph in what it was handed could
//     otherwise corrupt every other caller sharing that cache entry.
//     ShapeLine and WrapText pay a slice copy on every call, hit or miss, to
//     keep that impossible.
//   - A request is for text, not for a font. When a face lacks a glyph for a
//     rune, a fallback face is selected per rune so that something always
//     draws; the shaped run records which face each glyph came from.
//   - Metrics speak in geometry.Pixels. Rasterised glyph bounds speak in
//     geometry.DevicePixels and go through geometry.BoundsToDevicePixels so
//     that atlas tiles agree with the geometry around them.
//   - The UI runs on one goroutine and the typesetting types this package
//     holds (font.Face, fontscan.FontMap, the shaper) are not safe for
//     concurrent use. Callers do not share a System across goroutines.
package text
