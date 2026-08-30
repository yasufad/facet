package layout

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// TestXMLFlexFixtures runs the ported Taffy flexbox XML fixtures.
func TestXMLFlexFixtures(t *testing.T) {
	dir := filepath.Join("testdata", "flex")
	runXMLTestDir(t, dir)
}

// flexFixtureExclusions lists the upstream fixtures that are deliberately not
// carried in layout/testdata/flex/. Each entry is a glob matched against the
// upstream filename. The list and its rationale are recorded in
// layout/testdata/flex/README.
var flexFixtureExclusions = []string{
	"balance_*",
	"bevy_issue_10343_grid__*",
	"bevy_issue_21240__*",
	"overflow_start_edge_absolute_child__*",
}

// TestFlexFixtureDrift compares the fixtures in layout/testdata/flex/ against
// the upstream Taffy checkout in _upstream/taffy/tests/xml/flex/. It fails if
// either directory contains a file the other does not, after removing the
// documented exclusions. The test skips when _upstream/taffy is not present,
// so it does not break a checkout that has not run `go run ./tools/upstream`.
func TestFlexFixtureDrift(t *testing.T) {
	upstreamDir := filepath.Join("..", "_upstream", "taffy", "tests", "xml", "flex")
	if _, err := os.Stat(upstreamDir); os.IsNotExist(err) {
		t.Skip("_upstream/taffy not present; run `go run ./tools/upstream` to sync")
	}

	upstream := listXMLFiles(t, upstreamDir)
	ported := listXMLFiles(t, filepath.Join("testdata", "flex"))

	excluded := make(map[string]bool)
	for _, glob := range flexFixtureExclusions {
		for name := range upstream {
			if matchGlob(name, glob) {
				excluded[name] = true
			}
		}
	}

	var missing []string
	for name := range upstream {
		if excluded[name] {
			continue
		}
		if !ported[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("upstream fixtures not in layout/testdata/flex/ and not excluded:\n  %s",
			strings.Join(missing, "\n  "))
	}

	var extra []string
	for name := range ported {
		if !upstream[name] {
			extra = append(extra, name)
		}
	}
	if len(extra) > 0 {
		sort.Strings(extra)
		t.Errorf("fixtures in layout/testdata/flex/ with no upstream counterpart:\n  %s",
			strings.Join(extra, "\n  "))
	}
}

// listXMLFiles returns a set of *.xml filenames (without directory) in dir.
func listXMLFiles(t *testing.T, dir string) map[string]bool {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	set := make(map[string]bool)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".xml") {
			continue
		}
		set[e.Name()] = true
	}
	return set
}

// matchGlob is a simple glob matcher supporting only '*' wildcards, which is
// all the exclusion patterns use.
func matchGlob(name, pattern string) bool {
	if !strings.Contains(pattern, "*") {
		return name == pattern
	}
	parts := strings.Split(pattern, "*")
	if !strings.HasPrefix(name, parts[0]) {
		return false
	}
	name = name[len(parts[0]):]
	for i := 1; i < len(parts)-1; i++ {
		idx := strings.Index(name, parts[i])
		if idx < 0 {
			return false
		}
		name = name[idx+len(parts[i]):]
	}
	return strings.HasSuffix(name, parts[len(parts)-1])
}
