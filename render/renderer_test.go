package render_test

import (
	"testing"

	"github.com/yasufad/facet/render"
)

func TestOptionsZeroValue(t *testing.T) {
	opts := render.Options{}
	if opts.VSync {
		t.Errorf("expected VSync to default to false, got %v", opts.VSync)
	}
}
