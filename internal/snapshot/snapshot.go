package snapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

const envelopePrefix = "SNAPSHOT:"

type State struct {
	ID      string
	GitHead string
	Git     bool
}

func Capture(root string) (State, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return State{}, fmt.Errorf("snapshot root: %w", err)
	}
	if state, ok, err := captureGit(absRoot); ok || err != nil {
		return state, err
	}
	return captureFilesystem(absRoot)
}

func ParseEnvelope(input string) (snapshotID, selectors string, err error) {
	var kept []string
	for _, line := range strings.Split(input, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, envelopePrefix) {
			id := strings.TrimSpace(strings.TrimPrefix(trimmed, envelopePrefix))
			if id == "" {
				return "", "", fmt.Errorf("snapshot line is missing an id")
			}
			if snapshotID != "" {
				return "", "", fmt.Errorf("multiple snapshot lines are not allowed")
			}
			snapshotID = id
			continue
		}
		kept = append(kept, line)
	}
	return snapshotID, strings.Join(kept, "\n"), nil
}

func PromptSuffix(state State) string {
	if state.ID == "" {
		return ""
	}
	return fmt.Sprintf("\n[BADGER SNAPSHOT]\nID: %s\nIf you return FILE:, PREFIX:, or NEAR: selectors, the first non-empty line MUST be SNAPSHOT:%s. This pins follow-up context to the exact repository state used for this prompt.\n", state.ID, state.ID)
}

func ValidateExpected(expected string, current State, required bool) error {
	if strings.TrimSpace(expected) == "" {
		if required {
			return fmt.Errorf("snapshot id required; regenerate Prompt 1 from the current repository state")
		}
		return nil
	}
	if current.ID == "" {
		return fmt.Errorf("cannot validate snapshot id")
	}
	if expected != current.ID {
		return fmt.Errorf("repository changed since Prompt 1 (expected snapshot %s, current %s); regenerate context before continuing", short(expected), short(current.ID))
	}
	return nil
}

func short(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

func captureGit(root string) (State, bool, error) {
	top, err := gitOutput(root, "rev-parse", "--show-toplevel")
	if err != nil {
		return State{}, false, nil
	}
	repoRoot := strings.TrimSpace(string(top))
	if repoRoot == "" {
		return State{}, false, nil
	}

	h := sha256.New()
	writeString(h, "badger-snapshot-v1\n")
	head, _ := gitOutput(root, "rev-parse", "HEAD")
	headText := strings.TrimSpace(string(head))
	writeString(h, "HEAD\x00"+headText+"\x00")

	status, err := gitOutput(root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		return State{}, true, fmt.Errorf("snapshot git status: %w", err)
	}
	writeBytes(h, status)

	if err := hashGitStream(h, root, "diff", "--no-ext-diff", "--binary"); err != nil {
		return State{}, true, fmt.Errorf("snapshot git diff: %w", err)
	}
	if err := hashGitStream(h, root, "diff", "--cached", "--no-ext-diff", "--binary"); err != nil {
		return State{}, true, fmt.Errorf("snapshot staged diff: %w", err)
	}

	for _, path := range untrackedPaths(status) {
		hashFileBounded(h, filepath.Join(repoRoot, filepath.FromSlash(path)), path)
	}
	return State{ID: hex.EncodeToString(h.Sum(nil)), GitHead: headText, Git: true}, true, nil
}

func gitOutput(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	return cmd.Output()
}

func hashGitStream(h hash.Hash, root string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Stdout = h
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func untrackedPaths(status []byte) []string {
	var paths []string
	for _, record := range bytes.Split(status, []byte{0}) {
		if len(record) > 3 && bytes.HasPrefix(record, []byte("?? ")) {
			paths = append(paths, filepath.ToSlash(string(record[3:])))
		}
	}
	sort.Strings(paths)
	return paths
}

func hashFileBounded(h hash.Hash, path, display string) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return
	}
	writeString(h, "FILE\x00"+display+"\x00")
	writeString(h, fmt.Sprintf("%d\x00%d\x00", info.Size(), info.ModTime().UnixNano()))
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	const fullLimit = int64(1024 * 1024)
	if info.Size() <= fullLimit {
		_, _ = io.Copy(h, file)
		return
	}
	buf := make([]byte, 64*1024)
	if n, _ := file.Read(buf); n > 0 {
		writeBytes(h, buf[:n])
	}
	if _, err := file.Seek(-int64(len(buf)), io.SeekEnd); err == nil {
		if n, _ := file.Read(buf); n > 0 {
			writeBytes(h, buf[:n])
		}
	}
}

func captureFilesystem(root string) (State, error) {
	h := sha256.New()
	writeString(h, "badger-snapshot-v1-fs\n")
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "target", "build", "dist", ".venv", "venv", "__pycache__":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		info, err := entry.Info()
		if err != nil || !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err == nil {
			paths = append(paths, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		return State{}, fmt.Errorf("snapshot filesystem: %w", err)
	}
	sort.Strings(paths)
	for _, rel := range paths {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		writeString(h, fmt.Sprintf("%s\x00%d\x00%d\x00", rel, info.Size(), info.ModTime().UnixNano()))
	}
	return State{ID: hex.EncodeToString(h.Sum(nil))}, nil
}

func writeString(h hash.Hash, value string) { _, _ = io.WriteString(h, value) }
func writeBytes(h hash.Hash, value []byte)   { _, _ = h.Write(value) }
