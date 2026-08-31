package element

import (
	"testing"

	"github.com/yasufad/facet/colour"
	"github.com/yasufad/facet/style"
)

func BenchmarkBuildTreeUnstyled(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root := NewDiv().
			Children(
				NewDiv(),
				NewDiv(),
				NewDiv(),
				NewDiv(),
				NewDiv(),
				NewDiv(),
				NewDiv(),
				NewDiv(),
				NewDiv(),
				NewDiv(),
			)
		_ = root
	}
}

func BenchmarkBuildTree10Children(b *testing.B) {
	c := colour.Rgba{R: 0.2, G: 0.4, B: 0.8, A: 1.0}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root := NewDiv().
			Flex().
			Bg(c).
			Children(
				NewDiv().Flex().Bg(c),
				NewDiv().Flex().Bg(c),
				NewDiv().Flex().Bg(c),
				NewDiv().Flex().Bg(c),
				NewDiv().Flex().Bg(c),
				NewDiv().Flex().Bg(c),
				NewDiv().Flex().Bg(c),
				NewDiv().Flex().Bg(c),
				NewDiv().Flex().Bg(c),
				NewDiv().Flex().Bg(c),
			)
		_ = root
	}
}

func BenchmarkBuildTreeFullyStyled(b *testing.B) {
	c := colour.Rgba{R: 0.2, G: 0.4, B: 0.8, A: 1.0}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root := NewDiv().
			Flex().
			FlexRow().
			Gap(style.Px(8), style.Px(8)).
			Padding(style.Px(16)).
			Margin(style.Px(8)).
			Width(style.Px(800)).
			Height(style.Px(600)).
			Bg(c).
			Border(1).
			BorderColour(c).
			Rounded(4).
			Children(
				NewDiv().Flex().Width(style.Px(60)).Height(style.Px(40)).Bg(c).Padding(style.Px(4)).Rounded(2),
				NewDiv().Flex().Width(style.Px(60)).Height(style.Px(40)).Bg(c).Padding(style.Px(4)).Rounded(2),
				NewDiv().Flex().Width(style.Px(60)).Height(style.Px(40)).Bg(c).Padding(style.Px(4)).Rounded(2),
				NewDiv().Flex().Width(style.Px(60)).Height(style.Px(40)).Bg(c).Padding(style.Px(4)).Rounded(2),
				NewDiv().Flex().Width(style.Px(60)).Height(style.Px(40)).Bg(c).Padding(style.Px(4)).Rounded(2),
				NewDiv().Flex().Width(style.Px(60)).Height(style.Px(40)).Bg(c).Padding(style.Px(4)).Rounded(2),
				NewDiv().Flex().Width(style.Px(60)).Height(style.Px(40)).Bg(c).Padding(style.Px(4)).Rounded(2),
				NewDiv().Flex().Width(style.Px(60)).Height(style.Px(40)).Bg(c).Padding(style.Px(4)).Rounded(2),
				NewDiv().Flex().Width(style.Px(60)).Height(style.Px(40)).Bg(c).Padding(style.Px(4)).Rounded(2),
				NewDiv().Flex().Width(style.Px(60)).Height(style.Px(40)).Bg(c).Padding(style.Px(4)).Rounded(2),
			)
		_ = root
	}
}
