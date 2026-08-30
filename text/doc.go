// Package text turns strings of runes into shaped, measured and rasterised
// glyphs, and stops there. It does not draw: the renderer takes the glyph runs
// and alpha masks this package produces and puts them on screen.
//
// The package wraps github.com/go-text/typesetting, which supplies font loading
// and matching, script and bidi segmentation, HarfBuzz-equivalent shaping and
// line wrapping in pure Go. typesetting is the only third-party dependency
// permitted here. Nothing above this package knows it exists: shaped lines,
// glyph runs, metrics and rasterised glyphs are exposed in this package's own
// types, so that swapping the shaping engine would be a change confined to
// text.
//
// One auxiliary import is forced by typesetting's own API: golang.org/x/image/
// math/fixed. typesetting expresses font sizes and advances as fixed.Int26_6,
// so constructing a shaping.Input requires naming that type. It carries no
// behaviour of its own and travels with typesetting; it is not an independent
// dependency decision.
//
// # Rasterisation
//
// typesetting stops at glyph outlines. This package rasterises them itself,
// with a pure-Go scanline rasteriser using supersampled antialiasing. The
// alternatives — golang.org/x/image/vector and a GPU compute pass — were
// rejected: the first is a third-party import the dependency rule forbids in
// this package, and the second belongs to render, which sits above text. The
// choice and the measurement behind it are recorded in docs/architecture.md.
//
// # Invariants
//
//   - Shaped output is cached by run, not by string. The same word in the same
//     face at the same size is shaped once; the cache key is derived from the
//     run's text, face, size, direction, script, language and font features.
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
