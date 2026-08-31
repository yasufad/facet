package ui

import (
	"testing"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/element/elementtest"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/input"
	"github.com/yasufad/facet/style"
)

func TestButtonLifecycleAndScene(t *testing.T) {
	frame := elementtest.NewFrame()
	btn := NewButton("Save Changes")

	rootID := btn.RequestLayout(frame)
	frame.Solve(rootID, 200, 100)

	btnBounds := frame.LayoutBounds(rootID)
	if btnBounds.Size.Width <= 0 || btnBounds.Size.Height <= 0 {
		t.Fatalf("expected positive solved button dimensions, got %v", btnBounds.Size)
	}

	frame.SetPhase(elementtest.PhasePrepaint)
	btn.Prepaint(frame, btnBounds)

	if len(frame.HitRegions()) == 0 {
		t.Fatalf("expected hit region registered during prepaint")
	}

	frame.SetPhase(elementtest.PhasePaint)
	btn.Paint(frame, btnBounds)

	quads := frame.Quads()
	if len(quads) != 1 {
		t.Fatalf("expected exactly 1 background quad, got %d", len(quads))
	}

	q := quads[0]
	if q.Background != defaultButtonBg {
		t.Errorf("expected default background %v, got %v", defaultButtonBg, q.Background)
	}
	if q.BorderColour != defaultButtonBorder {
		t.Errorf("expected default border colour %v, got %v", defaultButtonBorder, q.BorderColour)
	}
	expectedRadius := geometry.ScaledPixels(4.0 * frame.ScaleFactor())
	if q.CornerRadii.TopLeft != expectedRadius {
		t.Errorf("expected corner radius %v, got %v", expectedRadius, q.CornerRadii.TopLeft)
	}

	if len(frame.MonochromeSprites()) == 0 {
		t.Fatalf("expected monochrome text sprites emitted into scene")
	}
}

func TestButtonClickDispatch(t *testing.T) {
	frame := elementtest.NewFrame()
	clicked := false

	btn := NewButton("Click Me").
		OnClick(func(event element.ClickEvent) bool {
			clicked = true
			return true
		})

	rootID := btn.RequestLayout(frame)
	frame.Solve(rootID, 200, 100)

	bounds := frame.LayoutBounds(rootID)
	frame.SetPhase(elementtest.PhasePrepaint)
	btn.Prepaint(frame, bounds)

	centerPt := geometry.Point[geometry.Pixels]{
		X: bounds.Origin.X + bounds.Size.Width/2,
		Y: bounds.Origin.Y + bounds.Size.Height/2,
	}

	frame.DispatchPointer(elementtest.PointerDown, centerPt, element.MouseButtonLeft, 0)
	if clicked {
		t.Fatalf("click handler fired prematurely on PointerDown")
	}

	frame.DispatchPointer(elementtest.PointerUp, centerPt, element.MouseButtonLeft, 0)
	if !clicked {
		t.Fatalf("expected click handler to fire on PointerUp")
	}
}

func TestButtonHoverStyling(t *testing.T) {
	frame := elementtest.NewFrame()
	btn := NewButton("Hover Test")

	rootID := btn.RequestLayout(frame)
	frame.Solve(rootID, 200, 100)

	bounds := frame.LayoutBounds(rootID)
	frame.SetPhase(elementtest.PhasePrepaint)
	btn.Prepaint(frame, bounds)

	hitID := frame.HitRegions()[0].ID
	frame.SetHovered(hitID, true)

	frame.SetPhase(elementtest.PhasePaint)
	btn.Paint(frame, bounds)

	quads := frame.Quads()
	if len(quads) != 1 {
		t.Fatalf("expected 1 quad, got %d", len(quads))
	}
	if quads[0].Background != defaultButtonHoverBg {
		t.Errorf("expected hover background %v, got %v", defaultButtonHoverBg, quads[0].Background)
	}
}

func TestButtonHoverTextColour(t *testing.T) {
	frame := elementtest.NewFrame()
	hoverTextColour := colour.Rgba{R: 1.0, G: 0.84, B: 0.0, A: 1.0} // Gold

	btn := NewButton("Hover Text Test").
		Hover(func(r *style.Refinement) {
			r.SetTextColour(hoverTextColour)
		})

	rootID := btn.RequestLayout(frame)
	frame.Solve(rootID, 200, 100)

	bounds := frame.LayoutBounds(rootID)
	frame.SetPhase(elementtest.PhasePrepaint)
	btn.Prepaint(frame, bounds)

	hitID := frame.HitRegions()[0].ID
	frame.SetHovered(hitID, true)

	frame.SetPhase(elementtest.PhasePaint)
	btn.Paint(frame, bounds)

	sprites := frame.MonochromeSprites()
	if len(sprites) == 0 {
		t.Fatalf("expected monochrome text sprites emitted for button label")
	}
	for _, sp := range sprites {
		if sp.Colour != hoverTextColour {
			t.Errorf("expected glyph sprite colour %v on hover, got %v", hoverTextColour, sp.Colour)
		}
	}
}

func TestButtonActiveStyling(t *testing.T) {
	frame := elementtest.NewFrame()
	btn := NewButton("Active Test")

	rootID := btn.RequestLayout(frame)
	frame.Solve(rootID, 200, 100)

	bounds := frame.LayoutBounds(rootID)
	frame.SetPhase(elementtest.PhasePrepaint)
	btn.Prepaint(frame, bounds)

	hitID := frame.HitRegions()[0].ID
	frame.SetActive(hitID, true)

	frame.SetPhase(elementtest.PhasePaint)
	btn.Paint(frame, bounds)

	quads := frame.Quads()
	if len(quads) != 1 {
		t.Fatalf("expected 1 quad, got %d", len(quads))
	}
	if quads[0].Background != defaultButtonActiveBg {
		t.Errorf("expected active background %v, got %v", defaultButtonActiveBg, quads[0].Background)
	}
}

func TestButtonFocusStyling(t *testing.T) {
	frame := elementtest.NewFrame()
	focusID := input.NewFocusID()
	btn := NewButton("Focus Test").TrackFocus(focusID)

	rootID := btn.RequestLayout(frame)
	frame.Solve(rootID, 200, 100)

	bounds := frame.LayoutBounds(rootID)
	frame.SetPhase(elementtest.PhasePrepaint)
	btn.Prepaint(frame, bounds)

	frame.SetFocused(focusID, true)

	frame.SetPhase(elementtest.PhasePaint)
	btn.Paint(frame, bounds)

	quads := frame.Quads()
	if len(quads) != 1 {
		t.Fatalf("expected 1 quad, got %d", len(quads))
	}
	q := quads[0]
	if q.BorderColour != defaultButtonFocus {
		t.Errorf("expected focus border colour %v, got %v", defaultButtonFocus, q.BorderColour)
	}
	expectedBorderWidth := geometry.ScaledPixels(2.0 * frame.ScaleFactor())
	if q.BorderWidths.Top != expectedBorderWidth {
		t.Errorf("expected focus border width %v, got %v", expectedBorderWidth, q.BorderWidths.Top)
	}
}

func TestButtonDisabled(t *testing.T) {
	frame := elementtest.NewFrame()
	clicked := false

	btn := NewButton("Disabled Button").
		Disabled(true).
		OnClick(func(event element.ClickEvent) bool {
			clicked = true
			return true
		})

	rootID := btn.RequestLayout(frame)
	frame.Solve(rootID, 200, 100)

	bounds := frame.LayoutBounds(rootID)
	frame.SetPhase(elementtest.PhasePrepaint)
	btn.Prepaint(frame, bounds)

	centerPt := geometry.Point[geometry.Pixels]{
		X: bounds.Origin.X + bounds.Size.Width/2,
		Y: bounds.Origin.Y + bounds.Size.Height/2,
	}

	frame.DispatchPointer(elementtest.PointerDown, centerPt, element.MouseButtonLeft, 0)
	frame.DispatchPointer(elementtest.PointerUp, centerPt, element.MouseButtonLeft, 0)

	if clicked {
		t.Fatalf("disabled button must not fire click events")
	}
}

func TestButtonEmptyLabel(t *testing.T) {
	frame := elementtest.NewFrame()
	btn := NewButton("")

	rootID := btn.RequestLayout(frame)
	frame.Solve(rootID, 100, 50)

	bounds := frame.LayoutBounds(rootID)
	frame.SetPhase(elementtest.PhasePrepaint)
	btn.Prepaint(frame, bounds)

	frame.SetPhase(elementtest.PhasePaint)
	btn.Paint(frame, bounds)

	if len(frame.Quads()) != 1 {
		t.Fatalf("expected 1 quad for empty button, got %d", len(frame.Quads()))
	}
	if len(frame.MonochromeSprites()) != 0 {
		t.Fatalf("expected 0 text sprites for empty button, got %d", len(frame.MonochromeSprites()))
	}
}

func TestButtonCustomRefinement(t *testing.T) {
	frame := elementtest.NewFrame()
	customBg := colour.Rgba{R: 0.9, G: 0.1, B: 0.1, A: 1.0}
	customHover := colour.Rgba{R: 1.0, G: 0.2, B: 0.2, A: 1.0}

	var customRefine style.Refinement
	customRefine.SetBackground(customBg)

	btn := NewButton("Custom").
		Refine(customRefine).
		Hover(func(r *style.Refinement) {
			r.SetBackground(customHover)
		})

	rootID := btn.RequestLayout(frame)
	frame.Solve(rootID, 150, 60)

	bounds := frame.LayoutBounds(rootID)
	frame.SetPhase(elementtest.PhasePrepaint)
	btn.Prepaint(frame, bounds)

	// Normal paint
	frame.SetPhase(elementtest.PhasePaint)
	btn.Paint(frame, bounds)
	quads := frame.Quads()
	if len(quads) != 1 || quads[0].Background != customBg {
		t.Fatalf("expected custom base background %v, got %v", customBg, quads[0].Background)
	}

	// Hover frame pass with fresh element
	hoverFrame := elementtest.NewFrame()
	hoverBtn := NewButton("Custom").
		Refine(customRefine).
		Hover(func(r *style.Refinement) {
			r.SetBackground(customHover)
		})

	hoverRootID := hoverBtn.RequestLayout(hoverFrame)
	hoverFrame.Solve(hoverRootID, 150, 60)
	hoverBounds := hoverFrame.LayoutBounds(hoverRootID)

	hoverFrame.SetPhase(elementtest.PhasePrepaint)
	hoverBtn.Prepaint(hoverFrame, hoverBounds)

	hoverFrame.SetHovered(hoverFrame.HitRegions()[0].ID, true)
	hoverFrame.SetPhase(elementtest.PhasePaint)
	hoverBtn.Paint(hoverFrame, hoverBounds)

	hoverQuads := hoverFrame.Quads()
	if len(hoverQuads) != 1 || hoverQuads[0].Background != customHover {
		t.Fatalf("expected custom hover background %v, got %v", customHover, hoverQuads[0].Background)
	}
}
