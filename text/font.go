package text

import (
	"fmt"
	"math"

	"github.com/go-text/typesetting/font"
)

// bitsOfFloat32 returns the bit pattern of f, so a float32 can be used as a
// deterministic map key without going through the unsafe package.
func bitsOfFloat32(f float32) uint32 { return math.Float32bits(f) }

// Weight is the stroke thickness of a font, on the OpenType scale from 100
// (thin) to 900 (black). 400 is normal.
type Weight float32

// Common weights, named after their OpenType values.
const (
	WeightThin       Weight = 100
	WeightExtraLight Weight = 200
	WeightLight      Weight = 300
	WeightNormal     Weight = 400
	WeightMedium     Weight = 500
	WeightSemibold   Weight = 600
	WeightBold       Weight = 700
	WeightExtraBold  Weight = 800
	WeightBlack      Weight = 900
)

// Style selects between upright and italic faces.
type Style uint8

const (
	// StyleNormal is an upright face.
	StyleNormal Style = iota
	// StyleItalic is an italic or oblique face. typesetting collapses oblique
	// into italic for matching purposes, so no separate oblique value is
	// exposed.
	StyleItalic
)

// Stretch is the width of a font as a fraction of normal: 1.0 is normal, 0.5
// ultra-condensed, 2.0 ultra-expanded.
type Stretch float32

// Common stretches, named after their OpenType fractions.
const (
	StretchUltraCondensed Stretch = 0.5
	StretchExtraCondensed Stretch = 0.625
	StretchCondensed      Stretch = 0.75
	StretchSemiCondensed  Stretch = 0.875
	StretchNormal         Stretch = 1.0
	StretchSemiExpanded   Stretch = 1.125
	StretchExpanded       Stretch = 1.25
	StretchExtraExpanded  Stretch = 1.5
	StretchUltraExpanded  Stretch = 2.0
)

// FontRequest describes a font to resolve: a primary family, optional
// fallback families tried in order, and the aspect that selects a particular
// face within a family.
type FontRequest struct {
	// Family is the primary family name, as shipped by the font (for example
	// "Segoe UI" or "Arial").
	Family string
	// Families are additional family names tried, in priority order, when
	// Family does not match. They are consulted before the system fallback
	// stack.
	Families []string
	// Weight selects the stroke thickness.
	Weight Weight
	// Style selects upright or italic.
	Style Style
	// Stretch selects the width. The zero value is treated as StretchNormal.
	Stretch Stretch
}

// withDefaults returns a copy of r with the default aspect values filled in,
// matching the CSS font-matching defaults typesetting expects.
func (r FontRequest) withDefaults() FontRequest {
	out := r
	if out.Weight == 0 {
		out.Weight = WeightNormal
	}
	if out.Stretch == 0 {
		out.Stretch = StretchNormal
	}
	return out
}

// aspect converts the request's aspect into the typesetting font.Aspect used
// for matching.
func (r FontRequest) aspect() font.Aspect {
	return font.Aspect{
		Style:   font.Style(r.Style),
		Weight:  font.Weight(r.Weight),
		Stretch: font.Stretch(r.Stretch),
	}
}

// families returns the family list in priority order, with the primary family
// first and empty names dropped.
func (r FontRequest) families() []string {
	out := make([]string, 0, 1+len(r.Families))
	if r.Family != "" {
		out = append(out, r.Family)
	}
	for _, f := range r.Families {
		if f != "" {
			out = append(out, f)
		}
	}
	if len(out) == 0 {
		out = append(out, "")
	}
	return out
}

// Face is an opaque, comparable handle to a resolved font face. It carries no
// public state: callers pass it back to the System that issued it. Two Faces
// are equal only if they refer to the same underlying face, so a Face may be
// used as a map key, which is what the glyph atlas does.
//
// The zero Face is not valid; methods on it panic or return false.
type Face struct {
	// face holds the typesetting face. It is unexported so that nothing above
	// this package can depend on the typesetting API.
	face *font.Face
}

// valid reports whether f refers to a loaded face.
func (f Face) valid() bool { return f.face != nil }

// GlyphID identifies a glyph within a face, as returned by the shaper. It is
// opaque to callers above this package.
type GlyphID uint32

// FontFeature activates or deactivates an OpenType feature for a run. Tag is
// the four-character feature tag, such as "liga" or "kern"; Value is 1 to
// enable or 0 to disable.
type FontFeature struct {
	Tag   string
	Value uint32
}

func (f FontFeature) String() string {
	return fmt.Sprintf("%s=%d", f.Tag, f.Value)
}
