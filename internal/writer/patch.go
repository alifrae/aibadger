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

func removePatchBlock(input string) string {
	start := strings.Index(input, patchStartMarker)
	if start < 0 {
		return input
	}
	remaining := input[start+len(patchStartMarker):]
	end := strings.Index(remaining, patchEndMarker)
	if end < 0 {
		return input[:start]
	}
	end += start + len(patchStartMarker) + len(patchEndMarker)
	return input[:start] + input[end:]
}

// PatchPaths returns every repository-relative path named by a standard
// unified-diff file header. It deliberately requires paired --- a/... and
// +++ b/... headers (with /dev/null allowed for create/delete) so source lines
// inside hunks that happen to begin with --- or +++ are not treated as paths.
func PatchPaths(patch string) ([]string, error) {
	lines := strings.Split(patch, "\n")
	seen := map[string]bool{}
	var paths []string
	for i := 0; i+1 < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "--- ") || !strings.HasPrefix(lines[i+1], "+++ ") {
			continue
		}
		oldPath, oldOK, err := parseUnifiedHeaderPath(strings.TrimSpace(strings.TrimPrefix(lines[i], "--- ")), "a/")
		if err != nil {
			return nil, err
		}
		newPath, newOK, err := parseUnifiedHeaderPath(strings.TrimSpace(strings.TrimPrefix(lines[i+1], "+++ ")), "b/")
		if err != nil {
			return nil, err
		}
		if !oldOK || !newOK {
			continue
		}
		for _, path := range []string{oldPath, newPath} {
			if path == "" || seen[path] {
				continue
			}
			seen[path] = true
			paths = append(paths, path)
		}
		i++
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("patch contains no valid unified-diff file headers")
	}
	return paths, nil
}

func parseUnifiedHeaderPath(raw, expectedPrefix string) (string, bool, error) {
	if tab := strings.IndexByte(raw, '\t'); tab >= 0 {
		raw = raw[:tab]
	}
	if raw == "/dev/null" {
		return "", true, nil
	}
	if !strings.HasPrefix(raw, expectedPrefix) {
		return "", false, nil
	}
	path := strings.TrimPrefix(raw, expectedPrefix)
	if err := validatePlannedPath(path); err != nil {
		return "", false, fmt.Errorf("patch path %q: %w", path, err)
	}
	return filepath.ToSlash(filepath.Clean(path)), true, nil
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
