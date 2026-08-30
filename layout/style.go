// Ported from Taffy src/style/mod.rs (MIT).
//
// The typed CSS style for a single node. Grid fields are absent; the generic
// CustomIdent parameter is dropped (it only names grid lines/areas). Taffy
// exposes the fields through the CoreStyle, FlexboxContainerStyle,
// FlexboxItemStyle and BlockContainerStyle traits; here those traits are methods
// on *Style so the algorithm reads against the same vocabulary.
package layout

// Style is the typed CSS style information for a single node.
type Style struct {
	Display        display
	ItemIsTable    bool
	ItemIsReplaced bool
	BoxSizing      boxSizing
	Direction      direction

	Overflow       Point[overflow]
	ScrollbarWidth float32
	Contain        contain

	Position position
	Inset    Rect[LengthPercentageAuto]

	Size        Size[Dimension]
	MinSize     Size[LengthPercentageAuto]
	MaxSize     Size[LengthPercentageAuto]
	AspectRatio *float32

	Margin  Rect[LengthPercentageAuto]
	Padding Rect[LengthPercentage]
	Border  Rect[LengthPercentage]

	Gap            Size[LengthPercentage]
	AlignItems     *AlignItems
	AlignSelf      *AlignItems
	AlignContent   *AlignContent
	JustifyContent *AlignContent

	TextAlign textAlign

	FlexDirection flexDirection
	FlexWrap      flexWrap
	FlexBasis     Dimension
	FlexGrow      float32
	FlexShrink    float32
}

// defaultStyle is Taffy's Style::DEFAULT for the flexbox+block feature set.
var defaultStyle = Style{
	Display:        displayDefault,
	BoxSizing:      boxSizingBorderBox,
	Direction:      directionLtr,
	Overflow:       Point[overflow]{X: overflowVisible, Y: overflowVisible},
	ScrollbarWidth: 0,
	Contain:        containNone,
	Position:       positionRelative,
	Inset:          rectAutoLPA(),
	Margin:         rectZeroLPA(),
	Padding:        rectZeroLP(),
	Border:         rectZeroLP(),
	Size:           sizeAutoDim(),
	MinSize:        sizeAutoLPA(),
	MaxSize:        sizeAutoLPA(),
	Gap:            sizeZeroLP(),
	AlignItems:     nil,
	AlignSelf:      nil,
	AlignContent:   nil,
	JustifyContent: nil,
	TextAlign:      textAlignAuto,
	FlexDirection:  FlexRow,
	FlexWrap:       FlexNoWrap,
	FlexGrow:       0,
	FlexShrink:     1,
	FlexBasis:      dimAutoVal,
}

// newStyle returns a copy of the default style.
func newStyle() Style { return defaultStyle }

// --- CoreStyle ---

func (s *Style) boxGenerationMode() boxGenerationMode {
	if s.Display == displayNone {
		return boxGenNone
	}
	return boxGenNormal
}

func (s *Style) isBlock() bool { return s.Display == displayBlock }

func (s *Style) isCompressibleReplaced() bool { return s.ItemIsReplaced }

func (s *Style) boxSizingVal() boxSizing { return s.BoxSizing }

func (s *Style) directionVal() direction { return s.Direction }

func (s *Style) overflowVal() Point[overflow] { return s.Overflow }

func (s *Style) scrollbarWidthVal() float32 { return s.ScrollbarWidth }

func (s *Style) positionVal() position { return s.Position }

func (s *Style) insetVal() Rect[LengthPercentageAuto] { return s.Inset }

func (s *Style) sizeVal() Size[Dimension] { return s.Size }

func (s *Style) minSizeVal() Size[LengthPercentageAuto] { return s.MinSize }

func (s *Style) maxSizeVal() Size[LengthPercentageAuto] { return s.MaxSize }

func (s *Style) aspectRatioVal() *float32 { return s.AspectRatio }

func (s *Style) marginVal() Rect[LengthPercentageAuto] { return s.Margin }

func (s *Style) paddingVal() Rect[LengthPercentage] { return s.Padding }

func (s *Style) borderVal() Rect[LengthPercentage] { return s.Border }

func (s *Style) containVal() contain { return s.Contain }

// --- FlexboxContainerStyle ---

func (s *Style) flexDirectionVal() flexDirection { return s.FlexDirection }

func (s *Style) flexWrapVal() flexWrap { return s.FlexWrap }

func (s *Style) gapVal() Size[LengthPercentage] { return s.Gap }

func (s *Style) alignContentVal() *AlignContent { return s.AlignContent }

func (s *Style) alignItemsVal() *AlignItems { return s.AlignItems }

func (s *Style) justifyContentVal() *AlignContent { return s.JustifyContent }

// --- FlexboxItemStyle ---

func (s *Style) flexBasisVal() Dimension { return s.FlexBasis }

func (s *Style) flexGrowVal() float32 { return s.FlexGrow }

func (s *Style) flexShrinkVal() float32 { return s.FlexShrink }

func (s *Style) alignSelfVal() *AlignItems { return s.AlignSelf }

// --- BlockContainerStyle ---

func (s *Style) textAlignVal() textAlign { return s.TextAlign }

// --- style helpers (from style_helpers.rs) ---

func rectAutoLPA() Rect[LengthPercentageAuto] {
	return Rect[LengthPercentageAuto]{
		Left: lpaAuto(), Right: lpaAuto(), Top: lpaAuto(), Bottom: lpaAuto(),
	}
}

func rectZeroLPA() Rect[LengthPercentageAuto] {
	return Rect[LengthPercentageAuto]{Left: lpaZero, Right: lpaZero, Top: lpaZero, Bottom: lpaZero}
}

func rectZeroLP() Rect[LengthPercentage] {
	return Rect[LengthPercentage]{Left: lpZero, Right: lpZero, Top: lpZero, Bottom: lpZero}
}

func sizeAutoDim() Size[Dimension] { return Size[Dimension]{Width: dimAuto(), Height: dimAuto()} }
func sizeAutoLPA() Size[LengthPercentageAuto] {
	return Size[LengthPercentageAuto]{Width: lpaAuto(), Height: lpaAuto()}
}
func sizeZeroLP() Size[LengthPercentage] {
	return Size[LengthPercentage]{Width: lpZero, Height: lpZero}
}
