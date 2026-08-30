// Ported from Taffy tests/common/src/lib.rs (MIT).
//
// The test measure function and node context used by the XML fixture harness.
// Supports zero-sized, fixed-size, aspect-ratio and Ahem-text nodes.
package layout

import "strings"

// writingMode is horizontal or vertical text.
type writingMode uint8

const (
	writingHorizontal writingMode = iota
	writingVertical
)

// testMeasureData is the measurement data for a test node.
type testMeasureData struct {
	kind        testMeasureKind
	fixedSize   Size[float32]
	aspectW     float32
	aspectRatio float32
	text        string
	writingMode writingMode
}

type testMeasureKind uint8

const (
	testMeasureZero testMeasureKind = iota
	testMeasureFixed
	testMeasureAspectRatio
	testMeasureAhemText
)

// testNodeContext is the per-node context for test measure functions.
type testNodeContext struct {
	count       int
	measureData testMeasureData
}

// newTestContextZero creates a zero-sized test context.
func newTestContextZero() testNodeContext {
	return testNodeContext{measureData: testMeasureData{kind: testMeasureZero}}
}

// newTestContextFixed creates a fixed-size test context.
func newTestContextFixed(w, h float32) testNodeContext {
	return testNodeContext{measureData: testMeasureData{kind: testMeasureFixed, fixedSize: Size[float32]{Width: w, Height: h}}}
}

// newTestContextAspectRatio creates an aspect-ratio test context.
func newTestContextAspectRatio(w, ratio float32) testNodeContext {
	return testNodeContext{measureData: testMeasureData{kind: testMeasureAspectRatio, aspectW: w, aspectRatio: ratio}}
}

// newTestContextAhemText creates an Ahem-text test context.
func newTestContextAhemText(text string, wm writingMode) testNodeContext {
	return testNodeContext{measureData: testMeasureData{kind: testMeasureAhemText, text: text, writingMode: wm}}
}

// testMeasureFunction is the measure function used by the XML fixture tests.
func testMeasureFunction(in LayoutInput, id NodeID, ctx any, style *Style) LayoutOutput {
	var tctx *testNodeContext
	if ctx != nil {
		if c, ok := ctx.(*testNodeContext); ok {
			tctx = c
		}
	}
	return computeLeafLayout(in, style, nil, func(known Size[optF32], avail Size[availableSpace]) Size[float32] {
		if known.Width.isSome() && known.Height.isSome() {
			return Size[float32]{Width: known.Width.v, Height: known.Height.v}
		}
		if tctx == nil {
			return Size[float32]{Width: known.Width.unwrapOr(0), Height: known.Height.unwrapOr(0)}
		}
		tctx.count++
		var compute Size[float32]
		switch tctx.measureData.kind {
		case testMeasureZero:
			compute = sizeZeroF32
		case testMeasureFixed:
			compute = tctx.measureData.fixedSize
		case testMeasureAspectRatio:
			compute = measureAspectRatio(tctx.measureData, known)
		case testMeasureAhemText:
			compute = measureAhemText(tctx.measureData, known, avail)
		}
		return Size[float32]{
			Width:  known.Width.unwrapOr(compute.Width),
			Height: known.Height.unwrapOr(compute.Height),
		}
	})
}

func measureAspectRatio(d testMeasureData, known Size[optF32]) Size[float32] {
	w := known.Width.unwrapOr(d.aspectW)
	h := known.Height.unwrapOr(w * d.aspectRatio)
	return Size[float32]{Width: w, Height: h}
}

const (
	ahemZWS     = '\u200B'
	ahemHWidth  = 10.0
	ahemHHeight = 10.0
)

func measureAhemText(d testMeasureData, known Size[optF32], avail Size[availableSpace]) Size[float32] {
	var inlineAxis absoluteAxis
	if d.writingMode == writingHorizontal {
		inlineAxis = absoluteHorizontal
	} else {
		inlineAxis = absoluteVertical
	}
	blockAxis := inlineAxis.otherAxis()

	lines := strings.Split(d.text, string(ahemZWS))
	if len(lines) == 0 {
		return sizeZeroF32
	}

	minLineLen := 0
	maxLineLen := 0
	for _, line := range lines {
		if len(line) > minLineLen {
			minLineLen = len(line)
		}
		maxLineLen += len(line)
	}

	inlineSize := sizeGetAbs(known, inlineAxis)
	if !inlineSize.isSome() {
		availInline := sizeGetAbsAvail(avail, inlineAxis)
		switch availInline.kind {
		case availableMinContent:
			inlineSize = some(float32(minLineLen) * ahemHWidth)
		case availableMaxContent:
			inlineSize = some(float32(maxLineLen) * ahemHWidth)
		case availableDefinite:
			inlineSize = some(f32Min(availInline.val, float32(maxLineLen)*ahemHWidth))
		}
	}
	inlineSize = some(f32Max(inlineSize.v, float32(minLineLen)*ahemHWidth))

	blockSize := sizeGetAbs(known, blockAxis)
	if !blockSize.isSome() {
		inlineLineLen := int(inlineSize.v / ahemHWidth)
		lineCount := 1
		currentLineLen := 0
		for _, line := range lines {
			if currentLineLen+len(line) > inlineLineLen {
				if currentLineLen > 0 {
					lineCount++
				}
				currentLineLen = len(line)
			} else {
				currentLineLen += len(line)
			}
		}
		blockSize = some(float32(lineCount) * ahemHHeight)
	}

	if d.writingMode == writingHorizontal {
		return Size[float32]{Width: inlineSize.v, Height: blockSize.v}
	}
	return Size[float32]{Width: blockSize.v, Height: inlineSize.v}
}

// sizeGetAbsAvail extracts the available space along an absolute axis.
func sizeGetAbsAvail(s Size[availableSpace], axis absoluteAxis) availableSpace {
	if axis == absoluteHorizontal {
		return s.Width
	}
	return s.Height
}
