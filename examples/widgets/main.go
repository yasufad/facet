package main

import (
	"fmt"
	"log"

	"github.com/yasufad/facet/app"
	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/element"
	"github.com/yasufad/facet/geometry"
	"github.com/yasufad/facet/platform"
	"github.com/yasufad/facet/style"
	"github.com/yasufad/facet/ui"
	"github.com/yasufad/facet/window"
)

// WidgetsView is the reactive root view. It holds the list of items and the
// index of the selected item, plus handles to the retained widget state
// entities (text field, list, scroll view) so Render can wire them together.
type WidgetsView struct {
	items    []string
	selected int

	input  app.Entity[ui.TextFieldState]
	list   app.Entity[ui.ListState]
	scroll app.Entity[ui.ScrollState]
}

// Render builds the element tree driving four widgets in one window: a
// TextField next to a Button in a header row, a virtual List on the left,
// and a ScrollView with long content on the right.
func (v *WidgetsView) Render(cx *app.Context[WidgetsView]) element.Element {
	header := element.NewDiv().
		Flex().
		FlexRow().
		WFull().
		AlignItems(style.AlignItemsCentre).
		GapCol(style.Px(8)).
		Padding(style.Px(8)).
		Child(
			ui.NewTextField(cx.App(), v.input).
				Placeholder("Type an item, then press Add"),
		).
		Child(
			ui.NewButton("Add").
				OnClick(element.Listener(cx, func(v *WidgetsView, e element.ClickEvent, cx *app.Context[WidgetsView]) bool {
					text := v.input.Read(cx.App()).Text()
					if text == "" {
						return false
					}
					v.items = append(v.items, text)
					v.selected = len(v.items) - 1
					v.input.Update(cx.App(), func(st *ui.TextFieldState, cx *app.Context[ui.TextFieldState]) {
						st.SetText("")
						cx.Notify()
					})
					cx.Notify()
					return true
				})),
		).
		Child(
			ui.NewButton("Clear").
				OnClick(element.Listener(cx, func(v *WidgetsView, e element.ClickEvent, cx *app.Context[WidgetsView]) bool {
					v.items = nil
					v.selected = -1
					cx.Notify()
					return true
				})),
		)

	list := ui.NewList(cx.App(), v.list).
		Count(len(v.items)).
		ItemHeight(geometry.Pixels(28)).
		Render(func(index int) element.Element {
			bg := colour.Rgba{R: 0.16, G: 0.18, B: 0.22, A: 1.0}
			textColour := colour.Rgba{R: 0.9, G: 0.9, B: 0.95, A: 1.0}
			if index == v.selected {
				bg = colour.Rgba{R: 0.30, G: 0.55, B: 0.90, A: 1.0}
			}
			return element.NewDiv().
				Flex().
				FlexRow().
				AlignItems(style.AlignItemsCentre).
				PaddingX(style.Px(8)).
				WFull().
				HFull().
				Bg(bg).
				Child(
					element.NewText(fmt.Sprintf("%d. %s", index, v.items[index])).
						FontSize(geometry.Pixels(14)).
						TextColour(textColour),
				).
				OnClick(element.Listener(cx, func(v *WidgetsView, e element.ClickEvent, cx *app.Context[WidgetsView]) bool {
					v.selected = index
					cx.Notify()
					return true
				}))
		})

	// Long content for the scroll view: a column of labelled paragraphs.
	scrollContent := element.NewDiv().Flex().FlexCol().WFull().GapRow(style.Px(12))
	for i := 0; i < 40; i++ {
		scrollContent.Child(
			element.NewDiv().
				Flex().
				FlexCol().
				WFull().
				Padding(style.Px(12)).
				Bg(colour.Rgba{R: 0.18, G: 0.20, B: 0.24, A: 1.0}).
				Rounded(geometry.Pixels(6)).
				Child(
					element.NewText(fmt.Sprintf("Section %d", i)).
						FontSize(geometry.Pixels(16)).
						TextColour(colour.Rgba{R: 0.95, G: 0.95, B: 0.95, A: 1.0}),
				).
				Child(
					element.NewText(fmt.Sprintf("This is paragraph %d in the scroll view. Resize the window while scrolling, scroll to the very end and past it, and leave it idle.", i)).
						FontSize(geometry.Pixels(13)).
						TextColour(colour.Rgba{R: 0.7, G: 0.72, B: 0.78, A: 1.0}),
				),
		)
	}

	scroll := ui.NewScrollView(cx.App(), v.scroll).
		Child(scrollContent)

	body := element.NewDiv().
		Flex().
		FlexRow().
		WFull().
		FlexGrow(1).
		GapCol(style.Px(8)).
		PaddingX(style.Px(8)).
		PaddingBottom(style.Px(8)).
		Child(
			element.NewDiv().
				Flex().
				FlexCol().
				WFull().
				FlexGrow(1).
				Bg(colour.Rgba{R: 0.12, G: 0.14, B: 0.18, A: 1.0}).
				Rounded(geometry.Pixels(6)).
				OverflowHidden().
				Child(list),
		).
		Child(
			element.NewDiv().
				Flex().
				FlexCol().
				WFull().
				FlexGrow(1).
				Bg(colour.Rgba{R: 0.12, G: 0.14, B: 0.18, A: 1.0}).
				Rounded(geometry.Pixels(6)).
				OverflowHidden().
				Child(scroll),
		)

	return element.NewDiv().
		Flex().
		FlexCol().
		WFull().
		HFull().
		Bg(colour.Rgba{R: 0.10, G: 0.12, B: 0.16, A: 1.0}).
		Child(header).
		Child(body)
}

func main() {
	p, err := platform.New(platform.Options{Name: "Facet Widgets"})
	if err != nil {
		log.Fatal(err)
	}

	a := app.NewApp()
	defer a.Close()

	w, err := window.New(p, a, window.WindowOptions{
		Title:     "Facet Widgets",
		Size:      geometry.NewSize[geometry.Pixels](900, 600),
		Resizable: true,
		Decorated: true,
		Visible:   true,
		VSync:     true,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer w.Close()

	inputEnt := app.New(a, func(cx *app.Context[ui.TextFieldState]) ui.TextFieldState {
		return ui.NewTextFieldState("")
	})
	listEnt := app.New(a, func(cx *app.Context[ui.ListState]) ui.ListState {
		return ui.NewListState()
	})
	scrollEnt := app.New(a, func(cx *app.Context[ui.ScrollState]) ui.ScrollState {
		return ui.NewScrollState()
	})

	// Seed a handful of items so the list is non-empty on first launch.
	seed := []string{"alpha", "bravo", "charlie", "delta", "echo", "foxtrot", "golf", "hotel", "india", "juliet"}

	rootEnt := app.New(a, func(cx *app.Context[WidgetsView]) WidgetsView {
		return WidgetsView{
			items:    seed,
			selected: -1,
			input:    inputEnt,
			list:     listEnt,
			scroll:   scrollEnt,
		}
	})

	w.SetRootView(element.NewView(rootEnt))

	p.Dispatch(w.Draw)
	if err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
