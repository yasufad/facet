// Package layering enforces the dependency rule from docs/packages.md.
//
// Imports run one way. A package may import what its entry permits and nothing
// else, so that six people working in six packages cannot quietly couple them
// together. A failure here means the design changed and needs deciding, not that
// the table needs an extra line.
package layering

import (
	"go/build"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const module = "github.com/yasufad/facet"

// permitted mirrors the table in docs/packages.md. Keys and values are top-level
// package names; a subpackage inherits its parent's entry and may also import the
// parent itself. Standard library imports are always allowed.
var permitted = map[string][]string{
	"geometry": {},
	"colour":   {},
	"app":      {},
	"scene":    {"geometry", "colour"},
	"layout":   {},
	"platform": {"geometry", "colour", "third_party"},
	"text":     {"geometry", "colour"},
	"render":   {"geometry", "colour", "scene", "platform"},
	"style":    {"geometry", "colour", "layout", "text"},
	"input":    {"geometry", "platform"},
	"element":  {"geometry", "colour", "scene", "style", "layout", "text", "app"},
	"window": {
		"geometry", "colour", "app", "scene", "layout", "platform", "text",
		"render", "style", "input", "element",
	},
	"ui": {"geometry", "colour", "style", "element", "app"},
}

// unconstrained packages sit outside the layer stack. Vendored source keeps
// whatever shape it arrived in, and tooling is not part of the framework.
var unconstrained = []string{"third_party", "internal", "tools"}

// platforms covers every target we build for, because build constraints hide
// imports that only exist on one operating system.
var platforms = []struct{ goos, goarch string }{
	{"windows", "amd64"},
	{"darwin", "arm64"},
	{"linux", "amd64"},
}

func TestImportsFollowTheLayerStack(t *testing.T) {
	root := repoRoot(t)

	for _, p := range platforms {
		t.Run(p.goos, func(t *testing.T) {
			ctx := build.Default
			ctx.GOOS = p.goos
			ctx.GOARCH = p.goarch
			ctx.CgoEnabled = true

			for _, dir := range packageDirs(t, root) {
				pkg, err := ctx.ImportDir(dir, 0)
				if _, noGo := err.(*build.NoGoError); noGo {
					continue // nothing builds here for this platform
				}
				if err != nil {
					t.Errorf("%s: %v", rel(root, dir), err)
					continue
				}
				checkImports(t, rel(root, dir), pkg.Imports)
			}
		})
	}
}

func checkImports(t *testing.T, dir string, imports []string) {
	t.Helper()

	owner := layer(dir)
	if owner == "" || slices.Contains(unconstrained, owner) {
		return
	}
	allowed, known := permitted[owner]
	if !known {
		t.Errorf("%s: package %q is not in docs/packages.md", dir, owner)
		return
	}

	for _, imp := range imports {
		target, ok := internalImport(imp)
		if !ok || target == owner {
			continue
		}
		if !slices.Contains(allowed, target) {
			t.Errorf("%s imports %s, which its entry in docs/packages.md does not permit", dir, imp)
		}
	}
}

// internalImport reports whether an import belongs to this module, and which
// top-level package it lands in.
func internalImport(imp string) (string, bool) {
	if !strings.HasPrefix(imp, module+"/") {
		return "", false
	}
	return layer(strings.TrimPrefix(imp, module+"/")), true
}

// layer returns the top-level package a slash-separated path belongs to.
func layer(p string) string {
	p = filepath.ToSlash(p)
	if i := strings.IndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return p
}

func packageDirs(t *testing.T, root string) []string {
	t.Helper()

	var dirs []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		switch name := d.Name(); {
		case p == root:
			return nil
		case name == ".git", name == "bin", strings.HasPrefix(name, "_"):
			return filepath.SkipDir
		}
		dirs = append(dirs, p)
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}
	return dirs
}

// repoRoot walks up from the test's directory until it finds go.mod.
func repoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("no go.mod in this directory or any parent")
		}
		dir = parent
	}
}

func rel(root, dir string) string {
	r, err := filepath.Rel(root, dir)
	if err != nil {
		return dir
	}
	return path.Clean(filepath.ToSlash(r))
}
