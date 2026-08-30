package geometry

// Number is the constraint on the coordinate and dimension types of the
// shapes. It admits the units this package defines — Pixels, ScaledPixels and
// Rems are all float32, DevicePixels is int32 — and the plain integer type
// used in tests. Every type in the set supports the arithmetic, comparison
// and unary negation the shapes rely on, so a single constraint serves them
// all.
type Number interface {
	~float32 | ~int32 | ~int
}
