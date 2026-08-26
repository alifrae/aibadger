package postapply

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PVRLabs/aibadger/internal/writer"
)

func TestCaptureExactDeltaIgnoresUnrelatedChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	root := t.TempDir()
	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("target.txt", "before\n")
	write("unrelated.txt", "already dirty\n")
	capture, err := Begin(root, []writer.FileUpdate{{Path: "target.txt", Kind: writer.UpdateKindWrite}})
	if err != nil {
		t.Fatal(err)
	}
	defer capture.Cleanup()
	write("target.txt", "after\n")
	result, err := capture.Finish()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Files) != 1 || result.Files[0] != "target.txt" {
		t.Fatalf("unexpected files: %+v", result.Files)
	}
	if strings.Contains(result.Diff, "unrelated.txt") {
		t.Fatalf("diff included unrelated worktree state: %s", result.Diff)
	}
	if !strings.Contains(result.Diff, "--- a/target.txt") || !strings.Contains(result.Diff, "+++ b/target.txt") {
		t.Fatalf("diff paths were not normalized: %s", result.Diff)
	}
	if result.Additions != 1 || result.Deletions != 1 {
		t.Fatalf("unexpected stats: +%d -%d\n%s", result.Additions, result.Deletions, result.Diff)
	}
}

func TestCaptureNewAndDeletedFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	root := t.TempDir()
	deleted := filepath.Join(root, "deleted.txt")
	if err := os.WriteFile(deleted, []byte("gone\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	updates := []writer.FileUpdate{{Path: "new.txt", Kind: writer.UpdateKindWrite}, {Path: "deleted.txt", Kind: writer.UpdateKindDelete}}
	capture, err := Begin(root, updates)
	if err != nil {
		t.Fatal(err)
	}
	defer capture.Cleanup()
	if err := os.WriteFile(filepath.Join(root, "new.txt"), []byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(deleted); err != nil {
		t.Fatal(err)
	}
	result, err := capture.Finish()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Diff, "--- /dev/null") || !strings.Contains(result.Diff, "+++ /dev/null") {
		t.Fatalf("expected new/delete markers: %s", result.Diff)
	}
}
