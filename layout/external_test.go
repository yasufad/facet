package layout_test

import (
	"testing"

	"github.com/yasufad/facet/layout"
)

func TestFlexRowTwoChildren(t *testing.T) {
	tree := layout.NewTaffyTree()

	// Child 1: 100x50
	s1 := layout.NewStyle()
	s1.Size = layout.Size[layout.Dimension]{
		Width:  layout.DimLength(100),
		Height: layout.DimLength(50),
	}
	c1 := tree.NewLeaf(s1)

	// Child 2: 200x50
	s2 := layout.NewStyle()
	s2.Size = layout.Size[layout.Dimension]{
		Width:  layout.DimLength(200),
		Height: layout.DimLength(50),
	}
	c2 := tree.NewLeaf(s2)

	// Root container: 400x100, flex row with 10px gap
	rootStyle := layout.NewStyle()
	rootStyle.Display = layout.DisplayFlex
	rootStyle.FlexDirection = layout.FlexRow
	rootStyle.Gap = layout.Size[layout.LengthPercentage]{
		Width:  layout.LPLength(10),
		Height: layout.LPZero(),
	}
	rootStyle.Size = layout.Size[layout.Dimension]{
		Width:  layout.DimLength(400),
		Height: layout.DimLength(100),
	}

	root := tree.NewWithChildren(rootStyle, []layout.NodeID{c1, c2})

	tree.ComputeLayout(root, layout.Size[layout.AvailableSpace]{
		Width:  layout.Definite(400),
		Height: layout.Definite(100),
	})

	// Assert root layout
	rootLayout := tree.Layout(root)
	if rootLayout.Location.X != 0 || rootLayout.Location.Y != 0 {
		t.Errorf("root location = (%v, %v), want (0, 0)", rootLayout.Location.X, rootLayout.Location.Y)
	}
	if rootLayout.Size.Width != 400 || rootLayout.Size.Height != 100 {
		t.Errorf("root size = (%v, %v), want (400, 100)", rootLayout.Size.Width, rootLayout.Size.Height)
	}

	// Assert Child 1 layout: (0, 0, 100, 50)
	c1Layout := tree.Layout(c1)
	if c1Layout.Location.X != 0 || c1Layout.Location.Y != 0 {
		t.Errorf("child 1 location = (%v, %v), want (0, 0)", c1Layout.Location.X, c1Layout.Location.Y)
	}
	if c1Layout.Size.Width != 100 || c1Layout.Size.Height != 50 {
		t.Errorf("child 1 size = (%v, %v), want (100, 50)", c1Layout.Size.Width, c1Layout.Size.Height)
	}

	// Assert Child 2 layout: (110, 0, 200, 50)
	c2Layout := tree.Layout(c2)
	if c2Layout.Location.X != 110 || c2Layout.Location.Y != 0 {
		t.Errorf("child 2 location = (%v, %v), want (110, 0)", c2Layout.Location.X, c2Layout.Location.Y)
	}
	if c2Layout.Size.Width != 200 || c2Layout.Size.Height != 50 {
		t.Errorf("child 2 size = (%v, %v), want (200, 50)", c2Layout.Size.Width, c2Layout.Size.Height)
	}
}

func TestStyleAllNonDefaultFields(t *testing.T) {
	aspectRatio := float32(16.0 / 9.0)
	alignItems := layout.AlignItemsCentre
	alignSelf := layout.AlignItemsFlexEnd
	alignContent := layout.AlignContentSpaceBetween
	justifyContent := layout.AlignContentCentre

	s := layout.Style{
		Display:        layout.DisplayBlock,
		ItemIsTable:    true,
		ItemIsReplaced: true,
		BoxSizing:      layout.BoxSizingContentBox,
		Direction:      layout.DirectionRtl,
		Overflow: layout.Point[layout.Overflow]{
			X: layout.OverflowScroll,
			Y: layout.OverflowHidden,
		},
		ScrollbarWidth: 15.0,
		Contain:        layout.ContainLayout | layout.ContainPaint,
		Position:       layout.PositionAbsolute,
		Inset: layout.Rect[layout.LengthPercentageAuto]{
			Top:    layout.LPALength(10),
			Right:  layout.LPALength(10),
			Bottom: layout.LPALength(10),
			Left:   layout.LPALength(10),
		},
		Size: layout.Size[layout.Dimension]{
			Width:  layout.DimLength(200),
			Height: layout.DimLength(100),
		},
		MinSize: layout.Size[layout.LengthPercentageAuto]{
			Width:  layout.LPALength(50),
			Height: layout.LPALength(50),
		},
		MaxSize: layout.Size[layout.LengthPercentageAuto]{
			Width:  layout.LPALength(500),
			Height: layout.LPALength(500),
		},
		AspectRatio: &aspectRatio,
		Margin: layout.Rect[layout.LengthPercentageAuto]{
			Top:    layout.LPALength(5),
			Right:  layout.LPALength(5),
			Bottom: layout.LPALength(5),
			Left:   layout.LPALength(5),
		},
		Padding: layout.Rect[layout.LengthPercentage]{
			Top:    layout.LPLength(8),
			Right:  layout.LPLength(8),
			Bottom: layout.LPLength(8),
			Left:   layout.LPLength(8),
		},
		Border: layout.Rect[layout.LengthPercentage]{
			Top:    layout.LPLength(2),
			Right:  layout.LPLength(2),
			Bottom: layout.LPLength(2),
			Left:   layout.LPLength(2),
		},
		Gap: layout.Size[layout.LengthPercentage]{
			Width:  layout.LPLength(12),
			Height: layout.LPLength(12),
		},
		AlignItems:     &alignItems,
		AlignSelf:      &alignSelf,
		AlignContent:   &alignContent,
		JustifyContent: &justifyContent,
		TextAlign:      layout.TextAlignCentre,
		FlexDirection:  layout.FlexColumn,
		FlexWrap:       layout.FlexWrapWrap,
		FlexBasis:      layout.DimLength(150),
		FlexGrow:       2.0,
		FlexShrink:     0.5,
	}

	tree := layout.NewTaffyTree()
	node := tree.NewLeaf(s)
	retrieved := tree.Style(node)

	if retrieved.Display != layout.DisplayBlock {
		t.Errorf("Display = %v, want DisplayBlock", retrieved.Display)
	}
	if retrieved.BoxSizing != layout.BoxSizingContentBox {
		t.Errorf("BoxSizing = %v, want BoxSizingContentBox", retrieved.BoxSizing)
	}
	if retrieved.Direction != layout.DirectionRtl {
		t.Errorf("Direction = %v, want DirectionRtl", retrieved.Direction)
	}
	if retrieved.Overflow.X != layout.OverflowScroll || retrieved.Overflow.Y != layout.OverflowHidden {
		t.Errorf("Overflow = (%v, %v), want (Scroll, Hidden)", retrieved.Overflow.X, retrieved.Overflow.Y)
	}
	if retrieved.ScrollbarWidth != 15.0 {
		t.Errorf("ScrollbarWidth = %v, want 15.0", retrieved.ScrollbarWidth)
	}
	if retrieved.Contain != (layout.ContainLayout | layout.ContainPaint) {
		t.Errorf("Contain = %v, want ContainLayout|ContainPaint", retrieved.Contain)
	}
	if retrieved.Position != layout.PositionAbsolute {
		t.Errorf("Position = %v, want PositionAbsolute", retrieved.Position)
	}
	if retrieved.TextAlign != layout.TextAlignCentre {
		t.Errorf("TextAlign = %v, want TextAlignCentre", retrieved.TextAlign)
	}
	if retrieved.FlexDirection != layout.FlexColumn {
		t.Errorf("FlexDirection = %v, want FlexColumn", retrieved.FlexDirection)
	}
	if retrieved.FlexWrap != layout.FlexWrapWrap {
		t.Errorf("FlexWrap = %v, want FlexWrapWrap", retrieved.FlexWrap)
	}
	if retrieved.FlexGrow != 2.0 || retrieved.FlexShrink != 0.5 {
		t.Errorf("FlexGrow/Shrink = (%v, %v), want (2.0, 0.5)", retrieved.FlexGrow, retrieved.FlexShrink)
	}
}

func TestComputeLayoutWithMeasure(t *testing.T) {
	tree := layout.NewTaffyTree()

	type textContext struct {
		content string
	}

	leafStyle := layout.NewStyle()
	leaf := tree.NewLeafWithContext(leafStyle, &textContext{content: "hello world"})

	measureCalled := false
	measure := func(inputs layout.LayoutInput, id layout.NodeID, ctx any, style *layout.Style) layout.LayoutOutput {
		measureCalled = true
		tc, ok := ctx.(*textContext)
		if !ok || tc.content != "hello world" {
			t.Errorf("unexpected context: %v", ctx)
		}
		return layout.LayoutOutput{
			Size: layout.Size[float32]{
				Width:  float32(len(tc.content) * 10),
				Height: 20,
			},
		}
	}

	tree.ComputeLayoutWithMeasure(leaf, layout.Size[layout.AvailableSpace]{
		Width:  layout.MaxContent(),
		Height: layout.MinContent(),
	}, measure)

	if !measureCalled {
		t.Fatal("measure function was not called")
	}

	leafLayout := tree.Layout(leaf)
	if leafLayout.Size.Width != 110 || leafLayout.Size.Height != 20 {
		t.Errorf("leaf size = (%v, %v), want (110, 20)", leafLayout.Size.Width, leafLayout.Size.Height)
	}
}
