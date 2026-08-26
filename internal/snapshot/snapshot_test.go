package snapshot

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseEnvelope(t *testing.T) {
	id, selectors, err := ParseEnvelope("SNAPSHOT:abc123\nFILE:main.go\nNEAR:main.go#func main")
	if err != nil {
		t.Fatal(err)
	}
	if id != "abc123" {
		t.Fatalf("unexpected id %q", id)
	}
	if strings.Contains(selectors, "SNAPSHOT:") || !strings.Contains(selectors, "FILE:main.go") {
		t.Fatalf("unexpected selectors %q", selectors)
	}
}

func TestCaptureChangesWhenFilesystemChanges(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.txt")
	if err := os.WriteFile(path, []byte("one"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := Capture(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("a different size"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := Capture(root)
	if err != nil {
		t.Fatal(err)
	}
	if before.ID == after.ID {
		t.Fatal("snapshot id did not change")
	}
}

func TestValidateExpected(t *testing.T) {
	current := State{ID: "current"}
	if err := ValidateExpected("current", current, true); err != nil {
		t.Fatal(err)
	}
	if err := ValidateExpected("", current, true); err == nil {
		t.Fatal("expected missing snapshot error")
	}
	if err := ValidateExpected("old", current, true); err == nil {
		t.Fatal("expected stale snapshot error")
	}
}
