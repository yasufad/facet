// Ported from Taffy src/style/compact_length.rs and src/style/dimension.rs (MIT).
//
// Taffy packs a length, percentage, auto or sizing keyword into a 64-bit tagged
// pointer (CompactLength). Go has no need for the packing: the same tag-plus-value
// shape lives in a plain struct, and LengthPercentage, LengthPercentageAuto and
// Dimension each wrap it exactly as they wrap CompactLength upstream. The calc
// variant is absent (Taffy's calc feature is not ported).
package layout

// clTag is the CompactLength tag, naming which kind of length a value is.
type clTag uint8

const (
	clLength            clTag = 1
	clPercent           clTag = 2
	clAuto              clTag = 3
	clMinContent        clTag = 7
	clMaxContent        clTag = 15
	clFitContentPx      clTag = 23
	clFitContentPercent clTag = 31
	clFitContentKeyword clTag = 39
	clStretch           clTag = 47
	clContent           clTag = 55
)

// compactLength is the tag-plus-value core shared by all dimension types.
type compactLength struct {
	tag clTag
	val float32
}

func clLengthVal(v float32) compactLength  { return compactLength{tag: clLength, val: v} }
func clPercentVal(v float32) compactLength { return compactLength{tag: clPercent, val: v} }
func clAutoVal() compactLength             { return compactLength{tag: clAuto} }
func clMinContentVal() compactLength       { return compactLength{tag: clMinContent} }
func clMaxContentVal() compactLength       { return compactLength{tag: clMaxContent} }
func clFitContentPxVal(v float32) compactLength {
	return compactLength{tag: clFitContentPx, val: v}
}
func clFitContentPercentVal(v float32) compactLength {
	return compactLength{tag: clFitContentPercent, val: v}
}
func clFitContentKeywordVal() compactLength { return compactLength{tag: clFitContentKeyword} }
func clStretchVal() compactLength           { return compactLength{tag: clStretch} }
func clContentVal() compactLength           { return compactLength{tag: clContent} }

func (c compactLength) isAuto() bool { return c.tag == clAuto }

func (c compactLength) isContent() bool { return c.tag == clContent }

func (c compactLength) isSizingKeyword() bool {
	switch c.tag {
	case clMinContent, clMaxContent, clFitContentKeyword, clFitContentPx, clFitContentPercent, clStretch:
		return true
	}
	return false
}

func (c compactLength) isStretch() bool { return c.tag == clStretch }

func (c compactLength) isMinContent() bool { return c.tag == clMinContent }

func (c compactLength) isMaxContent() bool { return c.tag == clMaxContent }

func (c compactLength) isFitContent() bool {
	return c.tag == clFitContentPx || c.tag == clFitContentPercent
}

func (c compactLength) isMaxOrFitContent() bool {
	return c.tag == clMaxContent || c.tag == clFitContentPx || c.tag == clFitContentPercent
}

func (c compactLength) isIntrinsic() bool {
	switch c.tag {
	case clAuto, clMinContent, clMaxContent, clFitContentPx, clFitContentPercent:
		return true
	}
	return false
}

// LengthPercentage is a length or a percentage.
type LengthPercentage struct{ cl compactLength }

// lpLength constructs an absolute length.
func lpLength(v float32) LengthPercentage { return LengthPercentage{cl: clLengthVal(v)} }

// lpPercent constructs a percentage (in [0,1], not [0,100]).
func lpPercent(v float32) LengthPercentage { return LengthPercentage{cl: clPercentVal(v)} }

// lpZero is a zero length.
var lpZero = LengthPercentage{cl: clLengthVal(0)}

// LengthPercentageAuto is a length, a percentage, or auto.
type LengthPercentageAuto struct{ cl compactLength }

// lpaLength constructs an absolute length.
func lpaLength(v float32) LengthPercentageAuto { return LengthPercentageAuto{cl: clLengthVal(v)} }

// lpaPercent constructs a percentage.
func lpaPercent(v float32) LengthPercentageAuto { return LengthPercentageAuto{cl: clPercentVal(v)} }

// lpaAuto constructs the auto keyword.
func lpaAuto() LengthPercentageAuto { return LengthPercentageAuto{cl: clAutoVal()} }

// lpaZero is a zero length.
var lpaZero = LengthPercentageAuto{cl: clLengthVal(0)}

var lpaAutoVal = LengthPercentageAuto{cl: clAutoVal()}

// fromLP lifts a LengthPercentage into a LengthPercentageAuto.
func fromLP(lp LengthPercentage) LengthPercentageAuto {
	return LengthPercentageAuto{cl: lp.cl}
}

// isAuto reports whether the value is the auto keyword.
func (l LengthPercentageAuto) isAuto() bool { return l.cl.isAuto() }

// resolveToOption resolves to an absolute value, returning none for auto.
func (l LengthPercentageAuto) resolveToOption(context optF32) optF32 {
	switch l.cl.tag {
	case clLength:
		return some(l.cl.val)
	case clPercent:
		if context.isSome() {
			return some(context.v * l.cl.val)
		}
		return none()
	case clAuto:
		return none()
	}
	return none()
}

// Dimension is a length, percentage, auto, or an intrinsic sizing keyword.
type Dimension struct{ cl compactLength }

// dimLength constructs an absolute length.
func dimLength(v float32) Dimension { return Dimension{cl: clLengthVal(v)} }

// dimPercent constructs a percentage.
func dimPercent(v float32) Dimension { return Dimension{cl: clPercentVal(v)} }

// dimAuto constructs the auto keyword.
func dimAuto() Dimension { return Dimension{cl: clAutoVal()} }

// dimMinContent constructs the min-content keyword.
func dimMinContent() Dimension { return Dimension{cl: clMinContentVal()} }

// dimMaxContent constructs the max-content keyword.
func dimMaxContent() Dimension { return Dimension{cl: clMaxContentVal()} }

// dimFitContent constructs the fit-content keyword.
func dimFitContent() Dimension { return Dimension{cl: clFitContentKeywordVal()} }

// dimFitContentPx constructs fit-content with a length limit.
func dimFitContentPx(v float32) Dimension { return Dimension{cl: clFitContentPxVal(v)} }

// dimFitContentPercent constructs fit-content with a percentage limit.
func dimFitContentPercent(v float32) Dimension { return Dimension{cl: clFitContentPercentVal(v)} }

// dimStretch constructs the stretch keyword.
func dimStretch() Dimension { return Dimension{cl: clStretchVal()} }

// dimContent constructs the content keyword (valid only for flex-basis).
func dimContent() Dimension { return Dimension{cl: clContentVal()} }

var dimAutoVal = Dimension{cl: clAutoVal()}

// dimFromLP lifts a LengthPercentage into a Dimension.
func dimFromLP(lp LengthPercentage) Dimension { return Dimension{cl: lp.cl} }

// dimFromLPA lifts a LengthPercentageAuto into a Dimension.
func dimFromLPA(lpa LengthPercentageAuto) Dimension { return Dimension{cl: lpa.cl} }

// isAuto reports whether the value is the auto keyword.
func (d Dimension) isAuto() bool { return d.cl.isAuto() }

// isContent reports whether the value is the content keyword.
func (d Dimension) isContent() bool { return d.cl.isContent() }

// isSizingKeyword reports whether the value is an intrinsic sizing keyword.
func (d Dimension) isSizingKeyword() bool { return d.cl.isSizingKeyword() }

// isStretch reports whether the value is the stretch keyword.
func (d Dimension) isStretch() bool { return d.cl.isStretch() }

// isMinContent reports whether the value is the min-content keyword.
func (d Dimension) isMinContent() bool { return d.cl.isMinContent() }

// isMaxContent reports whether the value is the max-content keyword.
func (d Dimension) isMaxContent() bool { return d.cl.isMaxContent() }

// isFitContent reports whether the value is a fit-content(...) value.
func (d Dimension) isFitContent() bool { return d.cl.isFitContent() }

// isMaxOrFitContent reports whether the value is max-content or fit-content(...).
func (d Dimension) isMaxOrFitContent() bool { return d.cl.isMaxOrFitContent() }

// isIntrinsic reports whether the value is auto or an intrinsic sizing keyword
// other than fit-content keyword and stretch.
func (d Dimension) isIntrinsic() bool { return d.cl.isIntrinsic() }

// tag returns the CompactLength tag, used by sizing-keyword resolution.
func (d Dimension) tag() clTag { return d.cl.tag }

// value returns the CompactLength value, used by sizing-keyword resolution.
func (d Dimension) value() float32 { return d.cl.val }
