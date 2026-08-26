package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PVRLabs/aibadger/internal/projectpolicy"
	"github.com/PVRLabs/aibadger/internal/protocol"
	"github.com/PVRLabs/aibadger/internal/writer"
)

func TestSnapshotPinnedSelectorsDetectDrift(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".badger.toml"), []byte("[session]\nrequire_snapshot = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "main.go")
	if err := os.WriteFile(path, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	eng, err := New(root, 0)
	if err != nil {
		t.Fatal(err)
	}
	prompt, _ := eng.GenerateMapDetailed("inspect main")
	if !strings.Contains(prompt, "[BADGER SNAPSHOT]") || eng.Snapshot.ID == "" {
		t.Fatalf("snapshot prompt missing: %q", prompt)
	}
	selectors := "SNAPSHOT:" + eng.Snapshot.ID + "\nFILE:main.go"
	parsed := eng.ParseCommandsDetailed(selectors)
	if len(parsed.Failures) != 0 || len(parsed.Commands) != 1 {
		t.Fatalf("unexpected parse result: %+v", parsed)
	}
	if err := os.WriteFile(path, []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	parsed = eng.ParseCommandsDetailed(selectors)
	if len(parsed.Failures) == 0 {
		t.Fatal("expected repository drift failure")
	}
}

func TestEgressPolicyBlocksSecretAndWarns(t *testing.T) {
	root := t.TempDir()
	policy := mustPolicyAtRoot(t, root, `[security]
deny = ["recordings/**"]
warn = ["calibration/**"]
block_secrets = true
`)
	eng := &Engine{Root: root, Policy: policy}
	items := []protocol.ExtractionResult{
		{Path: "recordings/a.dat", Content: "safe"},
		{Path: "calibration/a.yaml", Content: "safe"},
		{Path: "src/config.py", Content: "api_key = \"sk-AAAAAAAAAAAAAAAAAAAAAAAAAAAA\""},
		{Path: "src/main.py", Content: "print('ok')"},
	}
	kept, notices, err := eng.filterEgress(items)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept) != 2 {
		t.Fatalf("unexpected kept items: %+v", kept)
	}
	if len(notices) != 3 {
		t.Fatalf("unexpected notices: %+v", notices)
	}
}

func TestPatchOnlyRejectsWholeFileWrite(t *testing.T) {
	root := t.TempDir()
	policy := mustPolicyAtRoot(t, root, "[write]\npatch_only = true\n")
	eng := &Engine{Root: root, Policy: policy}
	if err := eng.validateUpdatePolicy(writer.FileUpdate{Path: "main.go", Kind: writer.UpdateKindWrite}); err == nil {
		t.Fatal("expected whole-file write rejection")
	}
}

func mustPolicyAtRoot(t *testing.T, root, content string) projectpolicy.Policy {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, ".badger.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	policy, err := projectpolicy.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return policy
}
