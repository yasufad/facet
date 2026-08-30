package style

import (
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/layout"
)

// lengthTag distinguishes the kind of length or dimension specified.
type lengthTag uint8

const (
	lengthTagAuto lengthTag = iota
	lengthTagPixels
	lengthTagRems
	lengthTagPercent
	lengthTagMinContent
	lengthTagMaxContent
	lengthTagFitContent
	lengthTagStretch
)

// Length represents a dimension, length, percentage, or sizing keyword.
type Length struct {
	tag lengthTag
	val float32
}

// Px constructs a length in logical pixels.
func Px(p geometry.Pixels) Length {
	return Length{tag: lengthTagPixels, val: float32(p)}
}

// Rem constructs a length in rems (multiples of the root font size).
func Rem(r geometry.Rems) Length {
	return Length{tag: lengthTagRems, val: float32(r)}
}

// Pct constructs a percentage length in [0, 100].
func Pct(p float32) Length {
	return Length{tag: lengthTagPercent, val: p / 100.0}
}

// Percent constructs a percentage length as a fraction in [0, 1].
func Percent(f float32) Length {
	return Length{tag: lengthTagPercent, val: f}
}

// Auto returns the auto sizing keyword.
func Auto() Length {
	return Length{tag: lengthTagAuto}
}

// MinContent returns the min-content sizing keyword.
func MinContent() Length {
	return Length{tag: lengthTagMinContent}
}

// MaxContent returns the max-content sizing keyword.
func MaxContent() Length {
	return Length{tag: lengthTagMaxContent}
}

// FitContent constructs a fit-content sizing keyword with a pixel limit.
func FitContent(p geometry.Pixels) Length {
	return Length{tag: lengthTagFitContent, val: float32(p)}
}

// Stretch returns the stretch sizing keyword.
func Stretch() Length {
	return Length{tag: lengthTagStretch}
}

// ToPixels resolves the length into logical pixels, using remSize for rems.
// Percentages and intrinsic keywords resolve to 0 when converted to scalar pixels.
func (l Length) ToPixels(remSize geometry.Pixels) geometry.Pixels {
	switch l.tag {
	case lengthTagPixels:
		return geometry.Pixels(l.val)
	case lengthTagRems:
		return geometry.Rems(l.val).ToPixels(remSize)
	default:
		return 0
	}
}

// ToLayoutDimension converts the length into a layout.Dimension.
func (l Length) ToLayoutDimension(remSize geometry.Pixels) layout.Dimension {
	switch l.tag {
	case lengthTagPixels:
		return layout.DimLength(l.val)
	case lengthTagRems:
		return layout.DimLength(float32(geometry.Rems(l.val).ToPixels(remSize)))
	case lengthTagPercent:
		return layout.DimPercent(l.val)
	case lengthTagAuto:
		return layout.DimAuto()
	case lengthTagMinContent:
		return layout.DimMinContent()
	case lengthTagMaxContent:
		return layout.DimMaxContent()
	case lengthTagFitContent:
		return layout.DimFitContent(l.val)
	case lengthTagStretch:
		return layout.DimStretch()
	default:
		return layout.DimAuto()
	}
}

// ToLayoutLP converts the length into a layout.LengthPercentage.
func (l Length) ToLayoutLP(remSize geometry.Pixels) layout.LengthPercentage {
	switch l.tag {
	case lengthTagPixels:
		return layout.LPLength(l.val)
	case lengthTagRems:
		return layout.LPLength(float32(geometry.Rems(l.val).ToPixels(remSize)))
	case lengthTagPercent:
		return layout.LPPercent(l.val)
	default:
		return layout.LPZero()
	}
}

// ToLayoutLPA converts the length into a layout.LengthPercentageAuto.
func (l Length) ToLayoutLPA(remSize geometry.Pixels) layout.LengthPercentageAuto {
	switch l.tag {
	case lengthTagPixels:
		return layout.LPALength(l.val)
	case lengthTagRems:
		return layout.LPALength(float32(geometry.Rems(l.val).ToPixels(remSize)))
	case lengthTagPercent:
		return layout.LPAPercent(l.val)
	case lengthTagAuto:
		return layout.LPAAuto()
	default:
		return layout.LPAZero()
	}
}
