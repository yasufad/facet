package elementtest

import (
	"testing"

	"github.com/yasufad/facet/element"
)

// TestTextLayoutQueryableAfterFrame drives a Text element through Frame's
// real Taffy solve, then queries the resulting TextLayout with no Frame in
// scope — the shape ui needs for caret arithmetic during event dispatch,
// where no Frame exists.
func TestTextLayoutQueryableAfterFrame(t *testing.T) {
	frame := NewFrame()

	txt := element.NewText("Hello").FontSize(16)
	nodeID := txt.RequestLayout(frame)
	frame.Solve(nodeID, 200, 40)

	tl := txt.Layout()

	if x := tl.XForIndex(0); x != 0 {
		t.Fatalf("XForIndex(0) = %v, want 0", x)
	}
	fullWidth := tl.XForIndex(len("Hello"))
	if fullWidth <= 0 {
		t.Fatalf("XForIndex(len(\"Hello\")) = %v, want > 0", fullWidth)
	}

	idx, ok := tl.IndexForX(0)
	if !ok || idx != 0 {
		t.Fatalf("IndexForX(0) = (%d, %v), want (0, true)", idx, ok)
	}

	if closest := tl.ClosestIndexForX(fullWidth); closest != len("Hello") {
		t.Fatalf("ClosestIndexForX(fullWidth) = %d, want %d", closest, len("Hello"))
	}
}

// TestTextLayoutZeroValueIsSafe confirms a TextLayout taken before the Text
// element has shaped anything reports the empty position instead of
// panicking — a text field may query its layout before the first frame that
// shapes its content.
func TestTextLayoutZeroValueIsSafe(t *testing.T) {
	txt := element.NewText("unshaped")
	tl := txt.Layout()

	if x := tl.XForIndex(3); x != 0 {
		t.Fatalf("XForIndex on an unshaped Text = %v, want 0", x)
	}
	if idx, ok := tl.IndexForX(10); idx != 0 || ok {
		t.Fatalf("IndexForX on an unshaped Text = (%d, %v), want (0, false)", idx, ok)
	}
	if idx := tl.ClosestIndexForX(10); idx != 0 {
		t.Fatalf("ClosestIndexForX on an unshaped Text = %d, want 0", idx)
	}
}
