package style

// Property indices for the parallel bitset mask.
//
// Inset, margin, padding, border widths, and corner radii use 4 bits each
// (one per edge or corner). Size and gap use 2 bits each (width/height or
// row/column). Scalar and enum properties use 1 bit each.
const (
	propDisplay uint8 = iota
	propOpacity
	propBackground

	// propFlexGrow is placed in the high word (bit 64) so high-word mask
	// operations are exercised by a genuine property on the main path.
	propFlexGrow uint8 = 64
)

// mask is a 128-bit bitset indicating which properties have been explicitly
// configured on a Refinement.
type mask struct {
	lo uint64
	hi uint64
}

// has reports whether the property bit is set.
func (m mask) has(bit uint8) bool {
	if bit < 64 {
		return (m.lo & (uint64(1) << bit)) != 0
	}
	return (m.hi & (uint64(1) << (bit - 64))) != 0
}

// set marks the property bit as set.
func (m *mask) set(bit uint8) {
	if bit < 64 {
		m.lo |= uint64(1) << bit
	} else {
		m.hi |= uint64(1) << (bit - 64)
	}
}

// or returns the union of m and other.
func (m mask) or(other mask) mask {
	return mask{
		lo: m.lo | other.lo,
		hi: m.hi | other.hi,
	}
}

// isEmpty reports whether no property bits are set.
func (m mask) isEmpty() bool {
	return m.lo == 0 && m.hi == 0
}
