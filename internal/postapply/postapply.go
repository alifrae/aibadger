package postapply

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/PVRLabs/aibadger/internal/writer"
)

// Result is the exact filesystem delta introduced across the files targeted by
// one Badger apply operation. It is independent of unrelated pre-existing Git
// changes elsewhere in the worktree.
type Result struct {
	Files     []string
	Diff      string
	Additions int
	Deletions int
}

// Capture stores pre-apply copies of every targeted regular file. Call Finish
// after the apply to render an exact before/after unified diff, then Cleanup.
type Capture struct {
	root      string
	tempDir   string
	paths     []string
	beforeSet map[string]bool
}

// Begin captures the pre-apply state for all paths touched by updates.
func Begin(root string, updates []writer.FileUpdate) (*Capture, error) {
	paths, err := updatePaths(updates)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, errors.New("post-apply capture has no target paths")
	}
	tempDir, err := os.MkdirTemp("", "badger-postapply-")
	if err != nil {
		return nil, fmt.Errorf("create post-apply capture directory: %w", err)
	}
	capture := &Capture{root: root, tempDir: tempDir, paths: paths, beforeSet: make(map[string]bool, len(paths))}
	for _, path := range paths {
		exists, err := capture.copyCurrent("before", path)
		if err != nil {
			capture.Cleanup()
			return nil, fmt.Errorf("capture pre-apply %s: %w", path, err)
		}
		capture.beforeSet[path] = exists
	}
	return capture, nil
}

// Finish captures the post-apply state and returns the exact delta introduced
// by the apply operation for the targeted paths.
func (c *Capture) Finish() (Result, error) {
	if c == nil || c.tempDir == "" {
		return Result{}, errors.New("post-apply capture is not initialized")
	}
	var result Result
	var diffs []string
	for _, path := range c.paths {
		afterExists, err := c.copyCurrent("after", path)
		if err != nil {
			return Result{}, fmt.Errorf("capture post-apply %s: %w", path, err)
		}
		beforeExists := c.beforeSet[path]
		if !beforeExists && !afterExists {
			continue
		}
		diff, changed, err := c.diffPath(path, beforeExists, afterExists)
		if err != nil {
			return Result{}, err
		}
		if !changed {
			continue
		}
		result.Files = append(result.Files, path)
		diffs = append(diffs, strings.TrimRight(diff, "\n"))
	}
	result.Diff = strings.Join(diffs, "\n")
	result.Additions, result.Deletions = countChangedLines(result.Diff)
	return result, nil
}

// Cleanup removes temporary pre/post-apply file copies.
func (c *Capture) Cleanup() {
	if c == nil || c.tempDir == "" {
		return
	}
	_ = os.RemoveAll(c.tempDir)
	c.tempDir = ""
}

func updatePaths(updates []writer.FileUpdate) ([]string, error) {
	seen := make(map[string]struct{})
	var paths []string
	for _, update := range updates {
		var candidates []string
		if update.Kind == writer.UpdateKindPatch {
			var err error
			candidates, err = writer.PatchPaths(update.Content)
			if err != nil {
				return nil, err
			}
		} else {
			candidates = []string{update.Path}
		}
		for _, path := range candidates {
			path = filepath.ToSlash(filepath.Clean(path))
			if path == "." || path == ".." || filepath.IsAbs(path) || strings.HasPrefix(path, "../") {
				return nil, fmt.Errorf("unsafe post-apply path %q", path)
			}
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			paths = append(paths, path)
		}
	}
	sort.Strings(paths)
	return paths, nil
}

func (c *Capture) copyCurrent(stage, rel string) (bool, error) {
	source := filepath.Join(c.root, filepath.FromSlash(rel))
	// Engine write validation is authoritative for symlink containment. Stat
	// follows an allowed in-repository symlink so the capture reflects the same
	// bytes that a whole-file write would actually modify.
	info, err := os.Stat(source)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !info.Mode().IsRegular() {
		return false, fmt.Errorf("target is not a regular file")
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return false, err
	}
	destination := filepath.Join(c.tempDir, stage, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
		return false, err
	}
	if err := os.WriteFile(destination, data, 0o600); err != nil {
		return false, err
	}
	return true, nil
}

func (c *Capture) diffPath(rel string, beforeExists, afterExists bool) (string, bool, error) {
	before := "/dev/null"
	after := "/dev/null"
	if beforeExists {
		before = filepath.ToSlash(filepath.Join("before", filepath.FromSlash(rel)))
	}
	if afterExists {
		after = filepath.ToSlash(filepath.Join("after", filepath.FromSlash(rel)))
	}
	cmd := exec.Command("git", "-c", "core.quotePath=false", "diff", "--no-index", "--binary", "--no-ext-diff", "--", before, after)
	cmd.Dir = c.tempDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return "", false, nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return "", false, fmt.Errorf("render post-apply diff for %s: %s", rel, message)
	}
	return rewriteDiffPaths(stdout.String(), rel), true, nil
}

func rewriteDiffPaths(diff, rel string) string {
	for _, old := range []string{"a/before/" + rel, "b/before/" + rel, "a/after/" + rel, "b/after/" + rel} {
		prefix := old[:2]
		diff = strings.ReplaceAll(diff, old, prefix+rel)
	}
	return diff
}

func countChangedLines(diff string) (additions, deletions int) {
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "+++") || strings.HasPrefix(line, "---"):
			continue
		case strings.HasPrefix(line, "+"):
			additions++
		case strings.HasPrefix(line, "-"):
			deletions++
		}
	}
	return additions, deletions
}
