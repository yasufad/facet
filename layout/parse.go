// Ported from Taffy src/util/parse.rs (MIT).
//
// Taffy uses the cssparser crate to parse CSS values. The Go port uses a
// small hand-written parser that handles the subset of CSS values the test
// fixtures use: keywords, lengths (Npx), and percentages (N%).
package layout

import (
	"fmt"
	"strconv"
	"strings"
)

// parseError is returned for unparseable input.
type parseError struct{ msg string }

func (e *parseError) Error() string { return e.msg }

// parseLengthPercentage parses "Npx" or "N%" into a LengthPercentage.
func parseLengthPercentage(s string) (LengthPercentage, error) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		v, err := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32)
		if err != nil {
			return LengthPercentage{}, &parseError{msg: fmt.Sprintf("parse percentage %q: %v", s, err)}
		}
		return lpPercent(float32(v) / 100), nil
	}
	if strings.HasSuffix(s, "px") {
		s = strings.TrimSuffix(s, "px")
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 32)
	if err != nil {
		return LengthPercentage{}, &parseError{msg: fmt.Sprintf("parse length %q: %v", s, err)}
	}
	return lpLength(float32(v)), nil
}

// parseLengthPercentageAuto parses "auto", "Npx", or "N%".
func parseLengthPercentageAuto(s string) (LengthPercentageAuto, error) {
	s = strings.TrimSpace(s)
	if strings.EqualFold(s, "auto") {
		return lpaAuto(), nil
	}
	lp, err := parseLengthPercentage(s)
	if err != nil {
		return LengthPercentageAuto{}, err
	}
	return fromLP(lp), nil
}

// parseDimension parses "auto", "min-content", "max-content", "fit-content",
// "fit-content(Npx)", "fit-content(N%)", "stretch", "content", "Npx", or "N%".
func parseDimension(s string) (Dimension, error) {
	s = strings.TrimSpace(s)
	if strings.EqualFold(s, "auto") {
		return dimAuto(), nil
	}
	if strings.EqualFold(s, "min-content") {
		return dimMinContent(), nil
	}
	if strings.EqualFold(s, "max-content") {
		return dimMaxContent(), nil
	}
	if strings.EqualFold(s, "fit-content") {
		return dimFitContent(), nil
	}
	if strings.EqualFold(s, "stretch") {
		return dimStretch(), nil
	}
	if strings.EqualFold(s, "content") {
		return dimContent(), nil
	}
	if strings.HasPrefix(strings.ToLower(s), "fit-content(") && strings.HasSuffix(s, ")") {
		inner := s[len("fit-content(") : len(s)-1]
		lp, err := parseLengthPercentage(inner)
		if err != nil {
			return Dimension{}, err
		}
		switch lp.cl.tag {
		case clLength:
			return dimFitContentPx(lp.cl.val), nil
		case clPercent:
			return dimFitContentPercent(lp.cl.val), nil
		}
		return Dimension{}, &parseError{msg: fmt.Sprintf("invalid fit-content argument %q", inner)}
	}
	lp, err := parseLengthPercentage(s)
	if err != nil {
		return Dimension{}, err
	}
	return dimFromLP(lp), nil
}

// parseDisplay parses a display keyword.
func parseDisplay(s string) (display, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "flex":
		return displayFlex, nil
	case "block":
		return displayBlock, nil
	case "flow-root":
		return displayFlowRoot, nil
	case "none":
		return displayNone, nil
	}
	return 0, &parseError{msg: fmt.Sprintf("unknown display %q", s)}
}

// parseDirection parses a direction keyword.
func parseDirection(s string) (direction, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "ltr":
		return directionLtr, nil
	case "rtl":
		return directionRtl, nil
	}
	return 0, &parseError{msg: fmt.Sprintf("unknown direction %q", s)}
}

// parseBoxSizing parses a box-sizing keyword.
func parseBoxSizing(s string) (boxSizing, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "border-box":
		return boxSizingBorderBox, nil
	case "content-box":
		return boxSizingContentBox, nil
	}
	return 0, &parseError{msg: fmt.Sprintf("unknown box-sizing %q", s)}
}

// parseOverflow parses an overflow keyword.
func parseOverflow(s string) (overflow, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "visible":
		return overflowVisible, nil
	case "hidden":
		return overflowHidden, nil
	case "scroll":
		return overflowScroll, nil
	case "clip":
		return overflowClip, nil
	}
	return 0, &parseError{msg: fmt.Sprintf("unknown overflow %q", s)}
}

// parsePosition parses a position keyword.
func parsePosition(s string) (position, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "relative":
		return positionRelative, nil
	case "absolute":
		return positionAbsolute, nil
	}
	return 0, &parseError{msg: fmt.Sprintf("unknown position %q", s)}
}

// parseFlexDirection parses a flex-direction keyword.
func parseFlexDirection(s string) (flexDirection, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "row":
		return FlexRow, nil
	case "column":
		return FlexColumn, nil
	case "row-reverse":
		return FlexRowReverse, nil
	case "column-reverse":
		return FlexColumnReverse, nil
	}
	return 0, &parseError{msg: fmt.Sprintf("unknown flex-direction %q", s)}
}

// parseFlexWrap parses a flex-wrap keyword.
func parseFlexWrap(s string) (flexWrap, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "nowrap":
		return FlexNoWrap, nil
	case "wrap":
		return FlexWrap, nil
	case "wrap-reverse":
		return FlexWrapReverse, nil
	case "balance", "balance-all":
		// Balance is not implemented; treat as wrap.
		return FlexWrap, nil
	}
	return 0, &parseError{msg: fmt.Sprintf("unknown flex-wrap %q", s)}
}

// parseAlignItems parses an align-items keyword, optionally prefixed with
// "safe" or "unsafe".
func parseAlignItems(s string) (AlignItems, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	safety := alignmentUnsafe
	if strings.HasPrefix(s, "safe ") {
		safety = alignmentSafe
		s = strings.TrimSpace(s[5:])
	} else if strings.HasPrefix(s, "unsafe ") {
		safety = alignmentUnsafe
		s = strings.TrimSpace(s[7:])
	}
	var kw alignItemsKeyword
	switch s {
	case "start":
		kw = alignItemsStart
	case "end":
		kw = alignItemsEnd
	case "flex-start":
		kw = alignItemsFlexStart
	case "flex-end":
		kw = alignItemsFlexEnd
	case "self-start":
		kw = alignItemsSelfStart
	case "self-end":
		kw = alignItemsSelfEnd
	case "center":
		kw = alignItemsCenter
	case "baseline":
		kw = alignItemsBaseline
	case "stretch":
		kw = alignItemsStretch
	default:
		return AlignItems{}, &parseError{msg: fmt.Sprintf("unknown align-items %q", s)}
	}
	return AlignItems{Keyword: kw, Safety: safety}, nil
}

// parseAlignContent parses an align-content keyword, optionally prefixed with
// "safe" or "unsafe".
func parseAlignContent(s string) (AlignContent, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	safety := alignmentUnsafe
	if strings.HasPrefix(s, "safe ") {
		safety = alignmentSafe
		s = strings.TrimSpace(s[5:])
	} else if strings.HasPrefix(s, "unsafe ") {
		safety = alignmentUnsafe
		s = strings.TrimSpace(s[7:])
	}
	var kw alignContentKeyword
	switch s {
	case "start":
		kw = alignContentStart
	case "end":
		kw = alignContentEnd
	case "flex-start":
		kw = alignContentFlexStart
	case "flex-end":
		kw = alignContentFlexEnd
	case "center":
		kw = alignContentCenter
	case "stretch":
		kw = alignContentStretch
	case "space-between":
		kw = alignContentSpaceBetween
	case "space-evenly":
		kw = alignContentSpaceEvenly
	case "space-around":
		kw = alignContentSpaceAround
	default:
		return AlignContent{}, &parseError{msg: fmt.Sprintf("unknown align-content %q", s)}
	}
	return AlignContent{Keyword: kw, Safety: safety}, nil
}

// parseFloat32 parses a float32.
func parseFloat32(s string) (float32, error) {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 32)
	if err != nil {
		return 0, err
	}
	return float32(v), nil
}
