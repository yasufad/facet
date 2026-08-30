package element

import (
	"testing"

	"github.com/yasufad/facet/colour"
)

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
