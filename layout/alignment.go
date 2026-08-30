// Ported from Taffy src/style/alignment.rs (MIT).
//
// The public alignment types are structs of two orthogonal fields: a position
// keyword and an overflow-position (safe/unsafe) modifier. The CSS spellings
// (Start, End, FlexStart, ..., SafeStart, ...) are exposed as constructors so
// call sites read identically to the upstream constants.
package layout

// alignItemsKeyword is the position-keyword half of AlignItems.
type alignItemsKeyword uint8

const (
	alignItemsStart alignItemsKeyword = iota
	alignItemsEnd
	alignItemsFlexStart
	alignItemsFlexEnd
	alignItemsSelfStart
	alignItemsSelfEnd
	alignItemsCentre
	alignItemsBaseline
	alignItemsStretch
)

// alignContentKeyword is the position-keyword half of AlignContent.
type alignContentKeyword uint8

const (
	alignContentStart alignContentKeyword = iota
	alignContentEnd
	alignContentFlexStart
	alignContentFlexEnd
	alignContentCentre
	alignContentStretch
	alignContentSpaceBetween
	alignContentSpaceEvenly
	alignContentSpaceAround
)

// reversed returns the RTL-reversed keyword: Start<->End, FlexStart<->FlexEnd.
// Stretch maps to End to preserve the layout algorithms' historical handling.
// Centre and the distribution keywords are direction-symmetric.
func (k alignContentKeyword) reversed() alignContentKeyword {
	switch k {
	case alignContentStart:
		return alignContentEnd
	case alignContentEnd:
		return alignContentStart
	case alignContentFlexStart:
		return alignContentFlexEnd
	case alignContentFlexEnd:
		return alignContentFlexStart
	case alignContentStretch:
		return alignContentEnd
	default:
		return k
	}
}

// alignmentSafety is the safe/unsafe overflow-position modifier.
type alignmentSafety uint8

const (
	alignmentUnsafe alignmentSafety = iota
	alignmentSafe
)

// AlignItems controls cross-axis alignment of children (and aliases AlignSelf,
// JustifyItems, JustifySelf).
type AlignItems struct {
	Keyword alignItemsKeyword
	Safety  alignmentSafety
}

// AlignSelf is an alias for AlignItems.
type AlignSelf = AlignItems

// JustifyItems is an alias for AlignItems.
type JustifyItems = AlignItems

// JustifySelf is an alias for AlignItems.
type JustifySelf = AlignItems

// AlignItems values matching Taffy's associated constants. These are exported
// package-level variables rather than constants because Go does not support
// struct constants.
var (
	AlignItemsStart         = AlignItems{Keyword: alignItemsStart, Safety: alignmentUnsafe}
	AlignItemsEnd           = AlignItems{Keyword: alignItemsEnd, Safety: alignmentUnsafe}
	AlignItemsFlexStart     = AlignItems{Keyword: alignItemsFlexStart, Safety: alignmentUnsafe}
	AlignItemsFlexEnd       = AlignItems{Keyword: alignItemsFlexEnd, Safety: alignmentUnsafe}
	AlignItemsSelfStart     = AlignItems{Keyword: alignItemsSelfStart, Safety: alignmentUnsafe}
	AlignItemsSelfEnd       = AlignItems{Keyword: alignItemsSelfEnd, Safety: alignmentUnsafe}
	AlignItemsCentre        = AlignItems{Keyword: alignItemsCentre, Safety: alignmentUnsafe}
	AlignItemsBaseline      = AlignItems{Keyword: alignItemsBaseline, Safety: alignmentUnsafe}
	AlignItemsStretch       = AlignItems{Keyword: alignItemsStretch, Safety: alignmentUnsafe}
	AlignItemsSafeStart     = AlignItems{Keyword: alignItemsStart, Safety: alignmentSafe}
	AlignItemsSafeEnd       = AlignItems{Keyword: alignItemsEnd, Safety: alignmentSafe}
	AlignItemsSafeFlexStart = AlignItems{Keyword: alignItemsFlexStart, Safety: alignmentSafe}
	AlignItemsSafeFlexEnd   = AlignItems{Keyword: alignItemsFlexEnd, Safety: alignmentSafe}
	AlignItemsSafeSelfStart = AlignItems{Keyword: alignItemsSelfStart, Safety: alignmentSafe}
	AlignItemsSafeSelfEnd   = AlignItems{Keyword: alignItemsSelfEnd, Safety: alignmentSafe}
	AlignItemsSafeCentre    = AlignItems{Keyword: alignItemsCentre, Safety: alignmentSafe}
)

// isSafe reports whether the safe overflow-position modifier is set.
func (a AlignItems) isSafe() bool { return a.Safety == alignmentSafe }

// keyword returns the underlying position keyword.
func (a AlignItems) keyword() alignItemsKeyword { return a.Keyword }

// resolveSelfRelative resolves SelfStart/SelfEnd to Start/End based on the
// item's own direction. In the inline axis the keyword flips when the item's
// direction differs from the container's; in the block axis (horizontal-tb
// only) it never flips.
func (a AlignItems) resolveSelfRelative(itemDir, containerDir direction, axisIsInline bool) AlignItems {
	flip := axisIsInline && itemDir != containerDir
	var kw alignItemsKeyword
	switch a.Keyword {
	case alignItemsSelfStart:
		if flip {
			kw = alignItemsEnd
		} else {
			kw = alignItemsStart
		}
	case alignItemsSelfEnd:
		if flip {
			kw = alignItemsStart
		} else {
			kw = alignItemsEnd
		}
	default:
		kw = a.Keyword
	}
	return AlignItems{Keyword: kw, Safety: a.Safety}
}

// AlignContent controls distribution of content (and aliases JustifyContent).
type AlignContent struct {
	Keyword alignContentKeyword
	Safety  alignmentSafety
}

// JustifyContent is an alias for AlignContent.
type JustifyContent = AlignContent

// AlignContent values matching Taffy's associated constants. These are exported
// package-level variables rather than constants because Go does not support
// struct constants.
var (
	AlignContentStart         = AlignContent{Keyword: alignContentStart, Safety: alignmentUnsafe}
	AlignContentEnd           = AlignContent{Keyword: alignContentEnd, Safety: alignmentUnsafe}
	AlignContentFlexStart     = AlignContent{Keyword: alignContentFlexStart, Safety: alignmentUnsafe}
	AlignContentFlexEnd       = AlignContent{Keyword: alignContentFlexEnd, Safety: alignmentUnsafe}
	AlignContentCentre        = AlignContent{Keyword: alignContentCentre, Safety: alignmentUnsafe}
	AlignContentStretch       = AlignContent{Keyword: alignContentStretch, Safety: alignmentUnsafe}
	AlignContentSpaceBetween  = AlignContent{Keyword: alignContentSpaceBetween, Safety: alignmentUnsafe}
	AlignContentSpaceEvenly   = AlignContent{Keyword: alignContentSpaceEvenly, Safety: alignmentUnsafe}
	AlignContentSpaceAround   = AlignContent{Keyword: alignContentSpaceAround, Safety: alignmentUnsafe}
	AlignContentSafeStart     = AlignContent{Keyword: alignContentStart, Safety: alignmentSafe}
	AlignContentSafeEnd       = AlignContent{Keyword: alignContentEnd, Safety: alignmentSafe}
	AlignContentSafeFlexStart = AlignContent{Keyword: alignContentFlexStart, Safety: alignmentSafe}
	AlignContentSafeFlexEnd   = AlignContent{Keyword: alignContentFlexEnd, Safety: alignmentSafe}
	AlignContentSafeCentre    = AlignContent{Keyword: alignContentCentre, Safety: alignmentSafe}
)

// isSafe reports whether the safe overflow-position modifier is set.
func (a AlignContent) isSafe() bool { return a.Safety == alignmentSafe }

// keyword returns the underlying position keyword.
func (a AlignContent) keyword() alignContentKeyword { return a.Keyword }
