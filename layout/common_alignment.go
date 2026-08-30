// Ported from Taffy src/compute/common/alignment.rs (MIT).
package layout

// resolveSelfAlignmentSafety resolves the safe/unsafe overflow-position
// fallback for a self-level alignment value. If the alignment subject
// overflows its alignment container and the requested alignment is safe, fall
// back to logical Start per CSS Box Alignment. Otherwise drop the safety
// modifier and return the bare keyword.
func resolveSelfAlignmentSafety(a AlignItems, overflows bool) alignItemsKeyword {
	if a.Safety == alignmentSafe && overflows {
		return alignItemsStart
	}
	return a.Keyword
}

// applyAlignmentFallback resolves any spec-defined fallbacks for the given
// AlignContent value, returning the bare position keyword the alignment math
// should use.
func applyAlignmentFallback(freeSpace float32, numItems int, mode AlignContent) alignContentKeyword {
	keyword := mode.Keyword
	isSafe := mode.Safety == alignmentSafe

	// 1. If there is only a single item being aligned or the items overflow the
	//    container, the distributed alignment keywords fall back to a positional
	//    keyword and gain implicit safe semantics so step 2 can flip them to
	//    Start on overflow.
	if numItems <= 1 || freeSpace <= 0 {
		switch keyword {
		case alignContentStretch, alignContentSpaceBetween:
			keyword = alignContentFlexStart
			isSafe = true
		case alignContentSpaceAround, alignContentSpaceEvenly:
			keyword = alignContentCentre
			isSafe = true
		}
	}

	// 2. Safe alignment falls back to Start whenever the alignment subject would
	//    overflow the alignment container.
	if freeSpace <= 0 && isSafe {
		keyword = alignContentStart
	}

	return keyword
}

// computeAlignmentOffset is the generic alignment function used for both
// align-content and justify-content, in both Flexbox and CSS Grid.
func computeAlignmentOffset(
	freeSpace float32,
	numItems int,
	gap float32,
	mode alignContentKeyword,
	layoutIsFlexReversed bool,
	isFirst bool,
) float32 {
	if isFirst {
		switch mode {
		case alignContentStart:
			return 0
		case alignContentFlexStart:
			if layoutIsFlexReversed {
				return freeSpace
			}
			return 0
		case alignContentEnd:
			return freeSpace
		case alignContentFlexEnd:
			if layoutIsFlexReversed {
				return 0
			}
			return freeSpace
		case alignContentCentre:
			return freeSpace / 2
		case alignContentStretch:
			return 0
		case alignContentSpaceBetween:
			return 0
		case alignContentSpaceAround:
			if freeSpace >= 0 {
				return (freeSpace / float32(numItems)) / 2
			}
			return freeSpace / 2
		case alignContentSpaceEvenly:
			if freeSpace >= 0 {
				return freeSpace / float32(numItems+1)
			}
			return freeSpace / 2
		}
		return 0
	}
	freeSpace = f32Max(freeSpace, 0)
	switch mode {
	case alignContentStart, alignContentFlexStart, alignContentEnd,
		alignContentFlexEnd, alignContentCentre, alignContentStretch:
		return gap
	case alignContentSpaceBetween:
		return gap + freeSpace/float32(numItems-1)
	case alignContentSpaceAround:
		return gap + freeSpace/float32(numItems)
	case alignContentSpaceEvenly:
		return gap + freeSpace/float32(numItems+1)
	}
	return gap
}
