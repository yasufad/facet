// Command upstream syncs the reference checkouts named in upstream.pins into
// _upstream/, at exactly the commits pinned there.
//
// Checkouts are shallow, blobless and sparse, so a large upstream such as Zed costs
// only the subtree we actually read. The leading underscore keeps _upstream/ out of
// sight of the Go toolchain.
//
//	go run ./tools/upstream            sync every checkout to its pinned commit
//	go run ./tools/upstream -update    move the pins to their branch heads
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	pinsFile  = "upstream.pins"
	targetDir = "_upstream"
)

// pin is one line of upstream.pins: a project, where it lives, and the commit we
// read it at.
type pin struct {
	name   string
	url    string
	branch string
	commit string
	sparse []string
}

func main() {
	log.SetFlags(0)
	log.SetPrefix("upstream: ")

	update := flag.Bool("update", false, "resolve each branch head and rewrite upstream.pins")
	flag.Parse()

	root, err := findRoot()
	if err != nil {
		log.Fatal(err)
	}
	path := filepath.Join(root, pinsFile)

	pins, err := parsePins(path)
	if err != nil {
		log.Fatal(err)
	}

	if *update {
		if err := updatePins(path, pins); err != nil {
			log.Fatal(err)
		}
		return
	}

	for _, p := range pins {
		if err := sync(root, p); err != nil {
			log.Fatalf("sync %s: %v", p.name, err)
		}
	}
}

// findRoot walks up from the working directory looking for upstream.pins, so the
// command works from anywhere in the tree.
func findRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, pinsFile)); err == nil {
			return dir, nil
		} else if !errors.Is(err, fs.ErrNotExist) {
			return "", fmt.Errorf("stat %s: %w", filepath.Join(dir, pinsFile), err)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no %s in this directory or any parent", pinsFile)
		}
		dir = parent
	}
}

func parsePins(path string) ([]pin, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open pins: %w", err)
	}
	defer f.Close()

	var pins []pin
	scan := bufio.NewScanner(f)
	for line := 1; scan.Scan(); line++ {
		text := strings.TrimSpace(scan.Text())
		if text == "" || strings.HasPrefix(text, "#") {
			continue
		}
		fields := strings.Fields(text)
		if len(fields) < 4 {
			return nil, fmt.Errorf("%s:%d: want at least name, url, branch and commit", path, line)
		}
		if len(fields[3]) != 40 {
			return nil, fmt.Errorf("%s:%d: %q is not a full commit hash", path, line, fields[3])
		}
		pins = append(pins, pin{
			name:   fields[0],
			url:    fields[1],
			branch: fields[2],
			commit: fields[3],
			sparse: fields[4:],
		})
	}
	if err := scan.Err(); err != nil {
		return nil, fmt.Errorf("read pins: %w", err)
	}
	if len(pins) == 0 {
		return nil, fmt.Errorf("%s lists no projects", path)
	}
	return pins, nil
}

// sync brings one checkout to its pinned commit, creating it if it is not there.
func sync(root string, p pin) error {
	dir := filepath.Join(root, targetDir, p.name)

	if _, err := os.Stat(filepath.Join(dir, ".git")); errors.Is(err, fs.ErrNotExist) {
		if err := initRepo(dir, p); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("stat checkout: %w", err)
	} else if err := git(dir, "remote", "set-url", "origin", p.url); err != nil {
		return err
	}

	if err := setSparse(dir, p); err != nil {
		return err
	}

	if head, err := output(dir, "rev-parse", "HEAD"); err == nil && head == p.commit {
		fmt.Printf("%-8s up to date at %s\n", p.name, short(p.commit))
		return nil
	}

	fmt.Printf("%-8s fetching %s\n", p.name, short(p.commit))
	if err := git(dir, "fetch", "--depth", "1", "origin", p.commit); err != nil {
		return err
	}
	return git(dir, "checkout", "--quiet", "--detach", "FETCH_HEAD")
}

// initRepo prepares an empty repository configured for a blobless partial clone.
// Fetching with a filter requires the promisor remote to be declared up front.
func initRepo(dir string, p pin) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create checkout directory: %w", err)
	}
	if err := git(dir, "init", "--quiet"); err != nil {
		return err
	}
	if err := git(dir, "remote", "add", "origin", p.url); err != nil {
		return err
	}
	for _, kv := range [][2]string{
		{"extensions.partialClone", "origin"},
		{"remote.origin.promisor", "true"},
		{"remote.origin.partialclonefilter", "blob:none"},
	} {
		if err := git(dir, "config", kv[0], kv[1]); err != nil {
			return err
		}
	}
	return nil
}

func setSparse(dir string, p pin) error {
	if len(p.sparse) == 0 {
		return git(dir, "sparse-checkout", "disable")
	}
	return git(dir, append([]string{"sparse-checkout", "set", "--cone"}, p.sparse...)...)
}

// updatePins resolves each branch head and rewrites the commit in place. Hashes are
// a fixed width, so substituting one for another leaves the file's alignment and
// comments untouched.
func updatePins(path string, pins []pin) error {
	body, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read pins: %w", err)
	}
	text := string(body)

	changed := false
	for _, p := range pins {
		head, err := remoteHead(p)
		if err != nil {
			return err
		}
		if head == p.commit {
			fmt.Printf("%-8s unchanged at %s\n", p.name, short(head))
			continue
		}
		fmt.Printf("%-8s %s -> %s\n", p.name, short(p.commit), short(head))
		text = strings.Replace(text, p.commit, head, 1)
		changed = true
	}
	if !changed {
		return nil
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return fmt.Errorf("write pins: %w", err)
	}
	return nil
}

func remoteHead(p pin) (string, error) {
	out, err := output("", "ls-remote", p.url, "refs/heads/"+p.branch)
	if err != nil {
		return "", err
	}
	name, _, ok := strings.Cut(out, "\t")
	if !ok || len(name) != 40 {
		return "", fmt.Errorf("%s: no branch %q at %s", p.name, p.branch, p.url)
	}
	return name, nil
}

func git(dir string, args ...string) error {
	_, err := output(dir, args...)
	return err
}

// output runs git and returns its trimmed standard output. On failure the error
// carries everything git printed, since that is the only useful diagnostic.
func output(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out)), nil
}

func short(commit string) string { return commit[:12] }
