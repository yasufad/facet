// Ported from Taffy src/geometry.rs (MIT).
//
// Geometric primitives the layout algorithm and its test suite are written
// against. Taffy models Size, Rect, Point and Line as generic structs; Go keeps
// the same generic shape. Methods that introduce a new type parameter (Rust's
// map/zip_map) or that apply only to a specific instantiation become free
// functions here, because Go does not permit methods with extra type parameters
// or methods on a single instantiation of a generic type.
package layout

// optF32 is the Go stand-in for Taffy's Option<f32>: a value that may or may not
// be defined. It is a value type so that structs containing it copy cleanly.
// nil would do the same job with less storage, but pointer aliasing under copy
// would silently share state between sizes that ought to be independent.
type optF32 struct {
	v   float32
	set bool
}

// some wraps a defined float32.
func some(v float32) optF32 { return optF32{v: v, set: true} }

// none is an undefined float32.
func none() optF32 { return optF32{} }

// get returns the value and whether it is defined.
func (o optF32) get() (float32, bool) { return o.v, o.set }

// isSome reports whether the value is defined.
func (o optF32) isSome() bool { return o.set }

// toOpt lifts a definite float32 into an optF32, leaving nil as none.
func toOpt(v *float32) optF32 {
	if v == nil {
		return none()
	}
	return some(*v)
}

// ptr returns a pointer to the value, or nil when undefined.
func (o optF32) ptr() *float32 {
	if !o.set {
		return nil
	}
	v := o.v
	return &v
}

// absoluteAxis is the simple horizontal/vertical axis.
type absoluteAxis uint8

const (
	absoluteHorizontal absoluteAxis = iota
	absoluteVertical
)

// otherAxis returns the opposite axis.
func (a absoluteAxis) otherAxis() absoluteAxis {
	if a == absoluteHorizontal {
		return absoluteVertical
	}
	return absoluteHorizontal
}

// abstractAxis is the CSS abstract (inline/block) axis.
type abstractAxis uint8

const (
	abstractInline abstractAxis = iota
	abstractBlock
)

func (a abstractAxis) other() abstractAxis {
	if a == abstractInline {
		return abstractBlock
	}
	return abstractInline
}

// asAbsNaive converts an abstract axis to an absolute one, assuming inline is
// horizontal (true under horizontal-tb, the only writing mode Taffy supports).
func (a abstractAxis) asAbsNaive() absoluteAxis {
	if a == abstractInline {
		return absoluteHorizontal
	}
	return absoluteVertical
}

// Line is an abstract line with a start and an end.
type Line[T any] struct {
	Start T
	End   T
}

// mapLine applies f to both ends of the line.
func mapLine[T, R any](l Line[T], f func(T) R) Line[R] {
	return Line[R]{Start: f(l.Start), End: f(l.End)}
}

// lineBoolTrue is a Line[bool] with both ends true.
var lineBoolTrue = Line[bool]{Start: true, End: true}

// lineBoolFalse is a Line[bool] with both ends false.
var lineBoolFalse = Line[bool]{Start: false, End: false}

// Size holds a width and a height.
type Size[T any] struct {
	Width  T
	Height T
}

// sizeMap applies f to both components.
func sizeMap[T, R any](s Size[T], f func(T) R) Size[R] {
	return Size[R]{Width: f(s.Width), Height: f(s.Height)}
}

// sizeZipMap applies f component-wise across two sizes.
func sizeZipMap[T, U, R any](a Size[T], b Size[U], f func(T, U) R) Size[R] {
	return Size[R]{Width: f(a.Width, b.Width), Height: f(a.Height, b.Height)}
}

// sizeMain returns the extent along the flexbox main axis.
func sizeMain[T any](s Size[T], dir flexDirection) T {
	if dir.isRow() {
		return s.Width
	}
	return s.Height
}

// sizeCross returns the extent along the flexbox cross axis.
func sizeCross[T any](s Size[T], dir flexDirection) T {
	if dir.isRow() {
		return s.Height
	}
	return s.Width
}

// sizeSetMain sets the extent along the flexbox main axis.
func sizeSetMain[T any](s *Size[T], dir flexDirection, v T) {
	if dir.isRow() {
		s.Width = v
	} else {
		s.Height = v
	}
}

// sizeSetCross sets the extent along the flexbox cross axis.
func sizeSetCross[T any](s *Size[T], dir flexDirection, v T) {
	if dir.isRow() {
		s.Height = v
	} else {
		s.Width = v
	}
}

// sizeWithMain returns a copy with the main-axis extent set.
func sizeWithMain[T any](s Size[T], dir flexDirection, v T) Size[T] {
	if dir.isRow() {
		return Size[T]{Width: v, Height: s.Height}
	}
	return Size[T]{Width: s.Width, Height: v}
}

// sizeWithCross returns a copy with the cross-axis extent set.
func sizeWithCross[T any](s Size[T], dir flexDirection, v T) Size[T] {
	if dir.isRow() {
		return Size[T]{Width: s.Width, Height: v}
	}
	return Size[T]{Width: v, Height: s.Height}
}

// sizeMapMain returns a copy with the main axis mapped through f.
func sizeMapMain[T any](s Size[T], dir flexDirection, f func(T) T) Size[T] {
	if dir.isRow() {
		return Size[T]{Width: f(s.Width), Height: s.Height}
	}
	return Size[T]{Width: s.Width, Height: f(s.Height)}
}

// sizeMapCross returns a copy with the cross axis mapped through f.
func sizeMapCross[T any](s Size[T], dir flexDirection, f func(T) T) Size[T] {
	if dir.isRow() {
		return Size[T]{Width: s.Width, Height: f(s.Height)}
	}
	return Size[T]{Width: f(s.Width), Height: s.Height}
}

// sizeGetAbs returns the component along an absolute axis.
func sizeGetAbs[T any](s Size[T], axis absoluteAxis) T {
	if axis == absoluteHorizontal {
		return s.Width
	}
	return s.Height
}

// sizeSetAbs sets the component along an absolute axis.
func sizeSetAbs[T any](s *Size[T], axis absoluteAxis, v T) {
	if axis == absoluteHorizontal {
		s.Width = v
	} else {
		s.Height = v
	}
}

// sizeZeroF32 is a Size[float32] with zero width and height.
var sizeZeroF32 = Size[float32]{Width: 0, Height: 0}

// sizeNone is a Size[optF32] with both components undefined.
var sizeNone = Size[optF32]{Width: none(), Height: none()}

// sizeMaxContent is a Size[AvailableSpace] filled with MaxContent.
var sizeMaxContent = Size[AvailableSpace]{Width: maxContent, Height: maxContent}

// sizeF32Max applies max component-wise.
func sizeF32Max(a, b Size[float32]) Size[float32] {
	return Size[float32]{Width: f32Max(a.Width, b.Width), Height: f32Max(a.Height, b.Height)}
}

// sizeF32Min applies min component-wise.
func sizeF32Min(a, b Size[float32]) Size[float32] {
	return Size[float32]{Width: f32Min(a.Width, b.Width), Height: f32Min(a.Height, b.Height)}
}

// sizeF32HasNonZeroArea reports whether both width and height are positive.
func sizeF32HasNonZeroArea(s Size[float32]) bool {
	return s.Width > 0 && s.Height > 0
}

// newSizeOpt returns a Size[optF32] with both components defined.
func newSizeOpt(width, height float32) Size[optF32] {
	return Size[optF32]{Width: some(width), Height: some(height)}
}

// sizeFromCross returns a Size[optF32] with only the cross axis defined.
func sizeFromCross(dir flexDirection, v optF32) Size[optF32] {
	s := sizeNone
	if dir.isRow() {
		s.Height = v
	} else {
		s.Width = v
	}
	return s
}

// sizeOptMaybeApplyAspectRatio fills in the missing dimension from the defined
// one and the aspect ratio (width/height). If ratio is nil or both/neither are
// defined, the size is returned unchanged.
func sizeOptMaybeApplyAspectRatio(s Size[optF32], ratio *float32) Size[optF32] {
	if ratio == nil {
		return s
	}
	r := *ratio
	if s.Width.isSome() && !s.Height.isSome() {
		w := s.Width.v
		return Size[optF32]{Width: s.Width, Height: some(w / r)}
	}
	if !s.Width.isSome() && s.Height.isSome() {
		h := s.Height.v
		return Size[optF32]{Width: some(h * r), Height: s.Height}
	}
	return s
}

// sizeOptUnwrapOr replaces undefined components with the corresponding alt values.
func sizeOptUnwrapOr(s Size[optF32], alt Size[float32]) Size[float32] {
	w := alt.Width
	if s.Width.isSome() {
		w = s.Width.v
	}
	h := alt.Height
	if s.Height.isSome() {
		h = s.Height.v
	}
	return Size[float32]{Width: w, Height: h}
}

// sizeOptOr returns a size with each component taken from self when defined, else alt.
func sizeOptOr(s, alt Size[optF32]) Size[optF32] {
	w := alt.Width
	if s.Width.isSome() {
		w = s.Width
	}
	h := alt.Height
	if s.Height.isSome() {
		h = s.Height
	}
	return Size[optF32]{Width: w, Height: h}
}

// sizeOptBothAxisDefined reports whether both components are defined.
func sizeOptBothAxisDefined(s Size[optF32]) bool {
	return s.Width.isSome() && s.Height.isSome()
}

// sizeAvailIntoOptions converts a Size[AvailableSpace] into a Size[optF32].
func sizeAvailIntoOptions(s Size[AvailableSpace]) Size[optF32] {
	return Size[optF32]{Width: s.Width.intoOption(), Height: s.Height.intoOption()}
}

// sizeAvailMaybeSet returns a definite value when v is defined, else self.
func sizeAvailMaybeSet(s Size[AvailableSpace], v Size[optF32]) Size[AvailableSpace] {
	return Size[AvailableSpace]{
		Width:  s.Width.maybeSet(v.Width),
		Height: s.Height.maybeSet(v.Height),
	}
}

// Rect is an axis-aligned rectangle of four edge values.
type Rect[T any] struct {
	Left   T
	Right  T
	Top    T
	Bottom T
}

// rectMap applies f to all four sides.
func rectMap[T, R any](r Rect[T], f func(T) R) Rect[R] {
	return Rect[R]{Left: f(r.Left), Right: f(r.Right), Top: f(r.Top), Bottom: f(r.Bottom)}
}

// rectZipSize applies f to each side paired with the matching size component
// (left/right with width, top/bottom with height).
func rectZipSize[T, U, R any](r Rect[T], s Size[U], f func(T, U) R) Rect[R] {
	return Rect[R]{
		Left:   f(r.Left, s.Width),
		Right:  f(r.Right, s.Width),
		Top:    f(r.Top, s.Height),
		Bottom: f(r.Bottom, s.Height),
	}
}

// rectF32HorizontalAxisSum returns Left + Right.
func rectF32HorizontalAxisSum(r Rect[float32]) float32 { return r.Left + r.Right }

// rectF32VerticalAxisSum returns Top + Bottom.
func rectF32VerticalAxisSum(r Rect[float32]) float32 { return r.Top + r.Bottom }

// rectF32SumAxes returns both axis sums as a Size.
func rectF32SumAxes(r Rect[float32]) Size[float32] {
	return Size[float32]{Width: rectF32HorizontalAxisSum(r), Height: rectF32VerticalAxisSum(r)}
}

// rectF32MainAxisSum returns the sum of the two fields on the main axis.
func rectF32MainAxisSum(r Rect[float32], dir flexDirection) float32 {
	if dir.isRow() {
		return rectF32HorizontalAxisSum(r)
	}
	return rectF32VerticalAxisSum(r)
}

// rectF32CrossAxisSum returns the sum of the two fields on the cross axis.
func rectF32CrossAxisSum(r Rect[float32], dir flexDirection) float32 {
	if dir.isRow() {
		return rectF32VerticalAxisSum(r)
	}
	return rectF32HorizontalAxisSum(r)
}

// rectMainStart returns the start/top value from the main-axis perspective.
func rectMainStart[T any](r Rect[T], dir flexDirection) T {
	if dir.isRow() {
		return r.Left
	}
	return r.Top
}

// rectMainEnd returns the end/bottom value from the main-axis perspective.
func rectMainEnd[T any](r Rect[T], dir flexDirection) T {
	if dir.isRow() {
		return r.Right
	}
	return r.Bottom
}

// rectCrossStart returns the start/top value from the cross-axis perspective.
func rectCrossStart[T any](r Rect[T], dir flexDirection) T {
	if dir.isRow() {
		return r.Top
	}
	return r.Left
}

// rectCrossEnd returns the end/bottom value from the cross-axis perspective.
func rectCrossEnd[T any](r Rect[T], dir flexDirection) T {
	if dir.isRow() {
		return r.Bottom
	}
	return r.Right
}

// rectF32GridAxisSum returns the sum of the two fields on an absolute axis.
func rectF32GridAxisSum(r Rect[float32], axis absoluteAxis) float32 {
	if axis == absoluteHorizontal {
		return r.Left + r.Right
	}
	return r.Top + r.Bottom
}

// rectHorizontalComponents returns left and right as a Line.
func rectHorizontalComponents[T any](r Rect[T]) Line[T] {
	return Line[T]{Start: r.Left, End: r.Right}
}

// rectVerticalComponents returns top and bottom as a Line.
func rectVerticalComponents[T any](r Rect[T]) Line[T] {
	return Line[T]{Start: r.Top, End: r.Bottom}
}

// rectZeroF32 is a Rect[float32] with all sides zero.
var rectZeroF32 = Rect[float32]{}

// newRectF32 constructs a Rect[float32].
func newRectF32(left, right, top, bottom float32) Rect[float32] {
	return Rect[float32]{Left: left, Right: right, Top: top, Bottom: bottom}
}

// rectF32Union returns the smallest rectangle (in edge-coordinate form)
// containing both.
func rectF32Union(r, other Rect[float32]) Rect[float32] {
	return Rect[float32]{
		Left:   f32Min(r.Left, other.Left),
		Right:  f32Max(r.Right, other.Right),
		Top:    f32Min(r.Top, other.Top),
		Bottom: f32Max(r.Bottom, other.Bottom),
	}
}

// rectF32Add adds two Rect[float32] component-wise.
func rectF32Add(a, b Rect[float32]) Rect[float32] {
	return Rect[float32]{
		Left:   a.Left + b.Left,
		Right:  a.Right + b.Right,
		Top:    a.Top + b.Top,
		Bottom: a.Bottom + b.Bottom,
	}
}

// pointTranspose swaps the x and y components of a Point.
func pointTransposeOverflow(p Point[overflow]) Point[overflow] {
	return Point[overflow]{X: p.Y, Y: p.X}
}

// Point is a 2D coordinate.
type Point[T any] struct {
	X T
	Y T
}

// pointZeroF32 is the origin.
var pointZeroF32 = Point[float32]{X: 0, Y: 0}

// pointNone is a Point[optF32] with both components undefined.
var pointNone = Point[optF32]{X: none(), Y: none()}

// pointMap applies f to both components.
func pointMap[T, R any](p Point[T], f func(T) R) Point[R] {
	return Point[R]{X: f(p.X), Y: f(p.Y)}
}

// pointMain returns the component along the main axis.
func pointMain[T any](p Point[T], dir flexDirection) T {
	if dir.isRow() {
		return p.X
	}
	return p.Y
}

// pointCross returns the component along the cross axis.
func pointCross[T any](p Point[T], dir flexDirection) T {
	if dir.isRow() {
		return p.Y
	}
	return p.X
}

// pointTranspose swaps the x and y components.
func pointTranspose[T any](p Point[T]) Point[T] {
	return Point[T]{X: p.Y, Y: p.X}
}

// rectBoolMainStart returns the main-start field of a Rect[bool].
func rectBoolMainStart(r Rect[bool], dir flexDirection) bool {
	if dir.isRow() {
		return r.Left
	}
	return r.Top
}

// rectBoolMainEnd returns the main-end field of a Rect[bool].
func rectBoolMainEnd(r Rect[bool], dir flexDirection) bool {
	if dir.isRow() {
		return r.Right
	}
	return r.Bottom
}

// rectBoolCrossStart returns the cross-start field of a Rect[bool].
func rectBoolCrossStart(r Rect[bool], dir flexDirection) bool {
	if dir.isRow() {
		return r.Top
	}
	return r.Left
}

// rectBoolCrossEnd returns the cross-end field of a Rect[bool].
func rectBoolCrossEnd(r Rect[bool], dir flexDirection) bool {
	if dir.isRow() {
		return r.Bottom
	}
	return r.Right
}

// MinMax holds a minimum and a maximum.
type MinMax[Min, Max any] struct {
	Min Min
	Max Max
}
