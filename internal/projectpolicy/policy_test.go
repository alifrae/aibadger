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
post_apply_review = true

[verify]
command = ["go", "test", "./..."]
`
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	policy, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if !policy.Security.BlockSecrets || !policy.Session.RequireSnapshot || !policy.Write.PatchOnly || !policy.Write.PostApplyReview {
		t.Fatalf("policy flags not loaded: %+v", policy)
	}
	if len(policy.Verify.Command) != 3 || policy.Verify.Command[0] != "go" || policy.Verify.Command[2] != "./..." {
		t.Fatalf("verification command not loaded: %+v", policy.Verify.Command)
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
	if policy.Security.BlockSecrets || policy.Session.RequireSnapshot || policy.Write.PatchOnly || policy.Write.PostApplyReview || len(policy.Verify.Command) != 0 {
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

func TestVerificationCommandPreservesLiteralArguments(t *testing.T) {
	root := t.TempDir()
	content := "[write]\npost_apply_review = true\n\n[verify]\ncommand = [\"go\", \"test\", \"./pkg with spaces\"]\n"
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	policy, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := policy.Verify.Command[2]; got != "./pkg with spaces" {
		t.Fatalf("verification argument was normalized as a path: %q", got)
	}
}

func TestVerificationCommandRequiresPostApplyReview(t *testing.T) {
	root := t.TempDir()
	content := "[verify]\ncommand = [\"go\", \"test\", \"./...\"]\n"
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil {
		t.Fatal("expected verify.command without post-apply review to be rejected")
	}
}

func TestNamedExternalSourcePolicy(t *testing.T) {
	root := t.TempDir()
	content := `[external.algorithm-core]
root = "../algo_core"
include = ["src/**", "include/**"]
`
	if err := os.WriteFile(filepath.Join(root, ConfigFileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	policy, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(policy.External) != 1 {
		t.Fatalf("got %d external sources: %+v", len(policy.External), policy.External)
	}
	source := policy.External[0]
	if source.Label != "algorithm-core" || source.Root != "../algo_core" {
		t.Fatalf("unexpected external source: %+v", source)
	}
	if len(source.Include) != 2 || source.Include[0] != "src/**" {
		t.Fatalf("unexpected include filters: %+v", source.Include)
	}
}
