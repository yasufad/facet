package style

import "github.com/yasufad/facet/colour"

// Refinement represents a set of sparse style property overrides.
//
// Unset properties are distinguished from properties explicitly set to their
// zero values via an internal bitset mask.
type Refinement struct {
	mask mask

	display    Display
	opacity    float32
	background colour.Rgba
	flexGrow   float32
	testHigh   float32
}

// IsEmpty reports whether no properties have been set in this refinement.
func (r Refinement) IsEmpty() bool {
	return r.mask.isEmpty()
}

// Merge combines r with other, returning a new Refinement where any property
// explicitly set in other overrides the corresponding property in r.
func (r Refinement) Merge(other Refinement) Refinement {
	if other.mask.isEmpty() {
		return r
	}
	if r.mask.isEmpty() {
		return other
	}
	out := r
	out.mask = r.mask.or(other.mask)
	if other.mask.lo != 0 {
		if other.mask.has(propDisplay) {
			out.display = other.display
		}
		if other.mask.has(propOpacity) {
			out.opacity = other.opacity
		}
		if other.mask.has(propBackground) {
			out.background = other.background
		}
		if other.mask.has(propFlexGrow) {
			out.flexGrow = other.flexGrow
		}
	}
	if other.mask.hi != 0 {
		if other.mask.has(propTestHigh) {
			out.testHigh = other.testHigh
		}
	}
	return out
}

// SetDisplay sets the element's layout strategy.
func (r *Refinement) SetDisplay(d Display) {
	r.mask.set(propDisplay)
	r.display = d
}

// SetOpacity sets the element opacity in [0, 1].
func (r *Refinement) SetOpacity(opacity float32) {
	r.mask.set(propOpacity)
	r.opacity = opacity
}

// SetBackground sets the background colour from an Rgba value.
func (r *Refinement) SetBackground(c colour.Rgba) {
	r.mask.set(propBackground)
	r.background = c
}

// SetBgHsla sets the background colour from an Hsla value.
func (r *Refinement) SetBgHsla(c colour.Hsla) {
	r.mask.set(propBackground)
	r.background = c.Rgba()
}

// SetFlexGrow sets the flex grow factor.
func (r *Refinement) SetFlexGrow(grow float32) {
	r.mask.set(propFlexGrow)
	r.flexGrow = grow
}

// SetTestHigh sets the testHigh property in the high word.
func (r *Refinement) SetTestHigh(v float32) {
	r.mask.set(propTestHigh)
	r.testHigh = v
}
