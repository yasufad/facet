// Ported from Taffy src/util/sys.rs (MIT).
//
// Taffy gates its float helpers behind an allocator feature flag and provides a
// no_std polyfill. Go's math package is always available, so only the named
// helpers the algorithm references come across.
package layout

import "math"

// f32Max returns the larger of two float32 values.
func f32Max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// f32Min returns the smaller of two float32 values.
func f32Min(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// round rounds to the nearest whole number (half up), matching Taffy's round.
func round(v float32) float32 {
	return float32(math.Floor(float64(v) + 0.5))
}

// ceil rounds up to the nearest whole number.
func ceil(v float32) float32 {
	return float32(math.Ceil(float64(v)))
}

// floor rounds down to the nearest whole number.
func floor(v float32) float32 {
	return float32(math.Floor(float64(v)))
}

// abs returns the absolute value.
func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// mathInf returns positive infinity.
func mathInf() float32 { return float32(math.Inf(1)) }

// isNaN reports whether v is a NaN.
func isNaN(v float32) bool { return math.IsNaN(float64(v)) }
