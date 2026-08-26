package projectpolicy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadProjectPolicy(t *testing.T) {
	root := t.TempDir()
	content := `[context]
always_include = ["AGENTS.md", "docs/architecture/"]

[docs]
canonical_roots = ["docs/architecture/", "docs/api/"]

[security]
deny = ["recordings/**", "customer/**", "**/*.dat"]
warn = ["calibration/**"]
block_secrets = true

[session]
require_snapshot = true

[write]
patch_only = true
`
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	policy, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !policy.Security.BlockSecrets || !policy.Session.RequireSnapshot || !policy.Write.PatchOnly {
		t.Fatalf("policy flags not loaded: %+v", policy)
	}
	if !policy.Denies("recordings/run.dat") || !policy.Denies("customer/a/b.txt") {
		t.Fatalf("deny globs not applied: %+v", policy.Security.Deny)
	}
	if !policy.Denies("capture.dat") || !policy.Denies("nested/capture.dat") {
		t.Fatalf("recursive glob must match root and nested files: %+v", policy.Security.Deny)
	}
	if policy.Denies("src/main.go") {
		t.Fatal("unexpected deny match")
	}
	if !policy.Warns("calibration/foo.yaml") {
		t.Fatal("warn glob not applied")
	}
}

func TestLoadMissingPolicyPreservesDefaults(t *testing.T) {
	policy, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if policy.Security.BlockSecrets || policy.Session.RequireSnapshot || policy.Write.PatchOnly {
		t.Fatalf("missing policy must preserve legacy behavior: %+v", policy)
	}
}

func TestRejectUnsafePattern(t *testing.T) {
	root := t.TempDir()
	content := "[security]\ndeny = [\"../outside/**\"]\n"
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("expected unsafe pattern error")
	}
}
