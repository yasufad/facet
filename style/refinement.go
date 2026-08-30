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
	background colour.Hsla
	flexGrow   float32
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
	return out
}

// Display sets the element's layout strategy.
func (r Refinement) Display(d Display) Refinement {
	r.mask.set(propDisplay)
	r.display = d
	return r
}

// Flex sets the element layout strategy to flexbox.
func (r Refinement) Flex() Refinement {
	return r.Display(DisplayFlex)
}

// Block sets the element layout strategy to block.
func (r Refinement) Block() Refinement {
	return r.Display(DisplayBlock)
}

// Hidden removes the element from layout entirely.
func (r Refinement) Hidden() Refinement {
	return r.Display(DisplayNone)
}

// Opacity sets the element opacity in [0, 1].
func (r Refinement) Opacity(opacity float32) Refinement {
	r.mask.set(propOpacity)
	r.opacity = opacity
	return r
}

// Bg sets the background colour from an Rgba value.
func (r Refinement) Bg(c colour.Rgba) Refinement {
	r.mask.set(propBackground)
	r.background = c.Hsla()
	return r
}

// BgHsla sets the background colour from an Hsla value.
func (r Refinement) BgHsla(c colour.Hsla) Refinement {
	r.mask.set(propBackground)
	r.background = c
	return r
}

// FlexGrow sets the flex grow factor.
func (r Refinement) FlexGrow(grow float32) Refinement {
	r.mask.set(propFlexGrow)
	r.flexGrow = grow
	return r
}
