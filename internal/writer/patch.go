package writer

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	patchStartMarker = "--- Patch ---"
	patchEndMarker   = "--- End Patch ---"
)

func parsePatchBlock(input string) (string, bool, error) {
	start := strings.Index(input, patchStartMarker)
	if start < 0 {
		return "", false, nil
	}
	remaining := input[start+len(patchStartMarker):]
	end := strings.Index(remaining, patchEndMarker)
	if end < 0 {
		return "", true, fmt.Errorf("patch block is missing %s", patchEndMarker)
	}
	patch := strings.TrimSpace(remaining[:end])
	if patch == "" {
		return "", true, fmt.Errorf("patch block is empty")
	}
	if _, err := PatchPaths(patch); err != nil {
		return "", true, err
	}
	return patch + "\n", true, nil
}

func PatchPaths(patch string) ([]string, error) {
	seen := map[string]bool{}
	var paths []string
	for _, line := range strings.Split(patch, "\n") {
		var path string
		switch {
		case strings.HasPrefix(line, "+++ "):
			path = strings.TrimSpace(strings.TrimPrefix(line, "+++ "))
		case strings.HasPrefix(line, "--- "):
			path = strings.TrimSpace(strings.TrimPrefix(line, "--- "))
		default:
			continue
		}
		if path == "/dev/null" || path == "" {
			continue
		}
		path = strings.TrimPrefix(path, "a/")
		path = strings.TrimPrefix(path, "b/")
		if tab := strings.IndexByte(path, '\t'); tab >= 0 {
			path = path[:tab]
		}
		if err := validatePlannedPath(path); err != nil {
			return nil, fmt.Errorf("patch path %q: %w", path, err)
		}
		path = filepath.ToSlash(filepath.Clean(path))
		if !seen[path] {
			seen[path] = true
			paths = append(paths, path)
		}
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("patch contains no file paths")
	}
	return paths, nil
}

func ApplyUnifiedDiff(projectRoot, patch string) error {
	if _, err := PatchPaths(patch); err != nil {
		return err
	}
	if err := runGitApply(projectRoot, patch, true); err != nil {
		return fmt.Errorf("patch preflight failed: %w", err)
	}
	if err := runGitApply(projectRoot, patch, false); err != nil {
		return fmt.Errorf("patch apply failed after successful preflight: %w", err)
	}
	return nil
}

func runGitApply(projectRoot, patch string, check bool) error {
	args := []string{"apply", "--whitespace=nowarn"}
	if check {
		args = append(args, "--check")
	}
	args = append(args, "-")
	cmd := exec.Command("git", args...)
	cmd.Dir = projectRoot
	cmd.Stdin = strings.NewReader(patch)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("%s", message)
	}
	return nil
}
