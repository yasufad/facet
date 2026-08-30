package style

// Property indices for the parallel bitset mask.
const (
	propDisplay uint8 = iota
	propOpacity
	propBackground
	propFlexGrow

	propTestHigh uint8 = 64
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
