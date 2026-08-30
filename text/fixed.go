package text

import "golang.org/x/image/math/fixed"

// toFixed converts a pixel measurement to the fixed.Int26_6 (1/64 pixel) units
// typesetting's shaping API speaks in.
func toFixed(px float32) fixed.Int26_6 { return fixed.Int26_6(px * 64) }

// fromFixed converts a fixed.Int26_6 value back to pixels.
func fromFixed(v fixed.Int26_6) float32 { return float32(v) / 64 }
