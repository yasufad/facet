// Ported from Taffy src/style/available_space.rs (MIT).
package layout

// AvailableSpace is the amount of space available to a node in a given axis:
// a definite number of pixels, or an indefinite min-content/max-content
// constraint.
type AvailableSpace struct {
	kind availableKind
	val  float32
}

type availableKind uint8

const (
	availableDefinite availableKind = iota
	availableMinContent
	availableMaxContent
)

// Definite constructs a definite available space with the given length in pixels.
func Definite(v float32) AvailableSpace { return AvailableSpace{kind: availableDefinite, val: v} }

// MinContent returns an indefinite min-content available space constraint.
func MinContent() AvailableSpace { return AvailableSpace{kind: availableMinContent} }

// MaxContent returns an indefinite max-content available space constraint.
func MaxContent() AvailableSpace { return AvailableSpace{kind: availableMaxContent} }

// definiteAvail constructs a definite available space.
func definiteAvail(v float32) AvailableSpace { return Definite(v) }

var (
	minContent = AvailableSpace{kind: availableMinContent}
	maxContent = AvailableSpace{kind: availableMaxContent}
	availZero  = AvailableSpace{kind: availableDefinite, val: 0}
)

// isDefinite reports whether the value is a definite number of pixels.
func (a AvailableSpace) isDefinite() bool { return a.kind == availableDefinite }

// intoOption returns the definite value, or none for a constraint.
func (a AvailableSpace) intoOption() optF32 {
	if a.kind == availableDefinite {
		return some(a.val)
	}
	return none()
}

// unwrapOr returns the definite value or a default.
func (a AvailableSpace) unwrapOr(def float32) float32 {
	if a.kind == availableDefinite {
		return a.val
	}
	return def
}

// unwrap returns the definite value. It panics if the value is not definite.
func (a AvailableSpace) unwrap() float32 {
	if a.kind != availableDefinite {
		panic("layout: unwrap on non-definite available space")
	}
	return a.val
}

// or returns self when definite, else the default.
func (a AvailableSpace) or(def AvailableSpace) AvailableSpace {
	if a.kind == availableDefinite {
		return a
	}
	return def
}

// orElse returns self when definite, else the result of defaultCb.
func (a AvailableSpace) orElse(defaultCb func() AvailableSpace) AvailableSpace {
	if a.kind == availableDefinite {
		return a
	}
	return defaultCb()
}

// unwrapOrElse returns the definite value or the callback's result.
func (a AvailableSpace) unwrapOrElse(defaultCb func() float32) float32 {
	if a.kind == availableDefinite {
		return a.val
	}
	return defaultCb()
}

// maybeSet returns a definite value wrapping v when v is defined, else self.
func (a AvailableSpace) maybeSet(v optF32) AvailableSpace {
	if v.isSome() {
		return definiteAvail(v.v)
	}
	return a
}

// mapDefiniteValue applies f to the definite value, leaving constraints alone.
func (a AvailableSpace) mapDefiniteValue(f func(float32) float32) AvailableSpace {
	if a.kind == availableDefinite {
		return definiteAvail(f(a.val))
	}
	return a
}

// computeFreeSpace returns the free space given the used space.
func (a AvailableSpace) computeFreeSpace(used float32) float32 {
	switch a.kind {
	case availableMaxContent:
		return float32(mathInf())
	case availableMinContent:
		return 0
	default:
		return a.val - used
	}
}

// isRoughlyEqual compares equality, treating near-equal definite values as equal.
func (a AvailableSpace) isRoughlyEqual(other AvailableSpace) bool {
	if a.kind != other.kind {
		return false
	}
	if a.kind == availableDefinite {
		return abs(a.val-other.val) < float32(1e-6)
	}
	return true
}

// fromOptF32 converts an optF32 into available space (none becomes MaxContent).
func fromOptF32(o optF32) AvailableSpace {
	if o.isSome() {
		return definiteAvail(o.v)
	}
	return maxContent
}

// maybeSubF32 subtracts a float32 from a definite available space, leaving
// constraints alone.
func (a AvailableSpace) maybeSubF32(v float32) AvailableSpace {
	if a.kind == availableDefinite {
		return definiteAvail(a.val - v)
	}
	return a
}
