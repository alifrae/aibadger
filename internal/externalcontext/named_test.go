package externalcontext

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNamedExternalContextLoadsAndFilters(t *testing.T) {
	base := t.TempDir()
	project := filepath.Join(base, "project")
	external := filepath.Join(base, "algo")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteNamedTest(t, filepath.Join(external, "src/core.rs"), "pub fn detect() {}\n")
	mustWriteNamedTest(t, filepath.Join(external, "secret/internal.rs"), "pub fn hidden() {}\n")
	policy := `[external.algorithm-core]
root = "../algo"
include = ["src/**"]
`
	mustWriteNamedTest(t, filepath.Join(project, ".badger.toml"), policy)

	contexts, err := LoadNamed(project)
	if err != nil {
		t.Fatal(err)
	}
	if len(contexts) != 1 {
		t.Fatalf("got %d contexts: %+v", len(contexts), contexts)
	}
	ctx := contexts[0]
	if ctx.Label != "algorithm-core" || ctx.Path != "@algorithm-core" {
		t.Fatalf("unexpected named context: %+v", ctx)
	}
	if len(ctx.Include) != 1 || ctx.Include[0] != "src/**" {
		t.Fatalf("include not preserved: %+v", ctx.Include)
	}

	allowed := ResolveFileFiltered(project, contexts, "@algorithm-core/src/core.rs")
	if !allowed.Found() {
		t.Fatalf("included file did not resolve: %+v", allowed.Matches)
	}
	blocked := ResolveFileFiltered(project, contexts, "@algorithm-core/secret/internal.rs")
	if len(blocked.Matches) != 0 {
		t.Fatalf("excluded file resolved: %+v", blocked.Matches)
	}
	if !IsDisplayPath(contexts, "@algorithm-core/src/core.rs") {
		t.Fatal("canonical named display path not recognized")
	}
	if IsDisplayPath(contexts, "algorithm-core/src/core.rs") {
		t.Fatal("local-looking label path should not be treated as the external display namespace")
	}
	if !IsAllowedPath(ctx, "src", true) {
		t.Fatal("include filter should allow a parent directory needed for completion")
	}
}

func TestExplicitNamedPathSelectsOnlyRequestedRoot(t *testing.T) {
	base := t.TempDir()
	project := filepath.Join(base, "project")
	first := filepath.Join(base, "first")
	second := filepath.Join(base, "second")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteNamedTest(t, filepath.Join(first, "src/core.rs"), "first\n")
	mustWriteNamedTest(t, filepath.Join(second, "src/core.rs"), "second\n")

	contexts := []model.ExternalContext{
		{Path: "@first", AbsPath: first, Label: "first"},
		{Path: "@second", AbsPath: second, Label: "second"},
	}
	resolution := ResolveFileFiltered(project, contexts, "@second/src/core.rs")
	match, ok := resolution.Match()
	if !ok {
		t.Fatalf("explicit named path should resolve exactly once: %+v", resolution.Matches)
	}
	if match.Context.Label != "second" || match.DisplayPath != "@second/src/core.rs" {
		t.Fatalf("wrong named source resolved: %+v", match)
	}
}

func mustWriteNamedTest(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
