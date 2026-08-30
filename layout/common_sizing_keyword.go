// Ported from Taffy src/compute/common/sizing_keyword.rs (MIT).
package layout

// sizingKeywordResolution is how a sizing keyword resolves to a used size.
type sizingKeywordResolution struct {
	kind  sizingKeywordKind
	value availableSpace // for Measure
	exact float32        // for Exact
}

type sizingKeywordKind uint8

const (
	sizingMeasure sizingKeywordKind = iota
	sizingExact
)

// resolveSizingKeyword resolves an item's size style in one axis if it is a
// sizing keyword. Returns nil if the style is not a sizing keyword or cannot
// be resolved in the current context.
func resolveSizingKeyword(style Dimension, stretchSize optF32, percentBasis optF32) *sizingKeywordResolution {
	switch style.tag() {
	case clMinContent:
		return &sizingKeywordResolution{kind: sizingMeasure, value: minContent}
	case clMaxContent:
		return &sizingKeywordResolution{kind: sizingMeasure, value: maxContent}
	case clFitContentPx:
		return &sizingKeywordResolution{kind: sizingMeasure, value: definiteAvail(style.value())}
	case clFitContentPercent:
		if percentBasis.isSome() {
			return &sizingKeywordResolution{kind: sizingMeasure, value: definiteAvail(percentBasis.v * style.value())}
		}
		return nil
	case clFitContentKeyword:
		if stretchSize.isSome() {
			return &sizingKeywordResolution{kind: sizingMeasure, value: definiteAvail(stretchSize.v)}
		}
		return nil
	case clStretch:
		if stretchSize.isSome() {
			return &sizingKeywordResolution{kind: sizingExact, exact: stretchSize.v}
		}
		return nil
	}
	return nil
}

// resolveAbsoluteSizingKeywords resolves the sizing keywords on the size styles
// of an absolutely positioned item, filling in the corresponding known
// dimensions axes.
func resolveAbsoluteSizingKeywords(
	t LayoutTree,
	node NodeID,
	known *Size[optF32],
	sizeStyle Size[Dimension],
	areaSize Size[float32],
	inset Rect[optF32],
	margin Rect[optF32],
	sizing sizingMode,
) {
	stretchSize := Size[float32]{
		Width: f32Max(areaSize.Width-
			inset.Left.unwrapOr(0)-inset.Right.unwrapOr(0)-
			margin.Left.unwrapOr(0)-margin.Right.unwrapOr(0), 0),
		Height: f32Max(areaSize.Height-
			inset.Top.unwrapOr(0)-inset.Bottom.unwrapOr(0)-
			margin.Top.unwrapOr(0)-margin.Bottom.unwrapOr(0), 0),
	}

	var kwW, kwH *sizingKeywordResolution
	if !known.Width.isSome() {
		kwW = resolveSizingKeyword(sizeStyle.Width, some(stretchSize.Width), some(areaSize.Width))
	}
	if !known.Height.isSome() {
		kwH = resolveSizingKeyword(sizeStyle.Height, some(stretchSize.Height), some(areaSize.Height))
	}

	if kwW != nil && kwW.kind == sizingMeasure && kwH != nil && kwH.kind == sizingMeasure {
		measured := measureChildSizeBoth(t, node, sizeNone,
			Size[optF32]{Width: some(areaSize.Width), Height: some(areaSize.Height)},
			Size[availableSpace]{Width: kwW.value, Height: kwH.value},
			sizing, lineBoolFalse)
		*known = Size[optF32]{Width: some(measured.Width), Height: some(measured.Height)}
		return
	}

	if kwW != nil {
		if kwW.kind == sizingExact {
			known.Width = some(kwW.exact)
		} else {
			w := measureChildSize(t, node, *known,
				Size[optF32]{Width: some(areaSize.Width), Height: some(areaSize.Height)},
				Size[availableSpace]{Width: kwW.value, Height: definiteAvail(stretchSize.Height)},
				sizing, absoluteHorizontal, lineBoolFalse)
			known.Width = some(w)
		}
	}
	if kwH != nil {
		if kwH.kind == sizingExact {
			known.Height = some(kwH.exact)
		} else {
			availW := definiteAvail(stretchSize.Width)
			if known.Width.isSome() {
				availW = definiteAvail(known.Width.v)
			}
			h := measureChildSize(t, node, *known,
				Size[optF32]{Width: some(areaSize.Width), Height: some(areaSize.Height)},
				Size[availableSpace]{Width: availW, Height: kwH.value},
				sizing, absoluteVertical, lineBoolFalse)
			known.Height = some(h)
		}
	}
}

// unwrapOr returns the definite value or the default.
func (o optF32) unwrapOr(def float32) float32 {
	if o.isSome() {
		return o.v
	}
	return def
}
