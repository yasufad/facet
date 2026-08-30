package layout

import (
	"path/filepath"
	"testing"
)

// TestXMLFlexFixtures runs the ported Taffy flexbox XML fixtures.
func TestXMLFlexFixtures(t *testing.T) {
	dir := filepath.Join("testdata", "flex")
	runXMLTestDir(t, dir)
}
