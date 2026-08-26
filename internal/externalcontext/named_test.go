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
	if err := os.MkdirAll(project, 0o755); err != nil { t.Fatal(err) }
	mustWriteNamedTest(t, filepath.Join(external, "src/core.rs"), "pub fn detect() {}\n")
	mustWriteNamedTest(t, filepath.Join(external, "secret/internal.rs"), "pub fn hidden() {}\n")
	policy := `[external.algorithm-core]
root = "../algo"
include = ["src/**"]
`
	mustWriteNamedTest(t, filepath.Join(project, ".badger.toml"), policy)

	contexts, err := LoadNamed(project)
	if err != nil { t.Fatal(err) }
	if len(contexts) != 1 { t.Fatalf("got %d contexts: %+v", len(contexts), contexts) }
	ctx := contexts[0]
	if ctx.Label != "algorithm-core" || ctx.Path != "@algorithm-core" { t.Fatalf("unexpected named context: %+v", ctx) }
	if len(ctx.Include) != 1 || ctx.Include[0] != "src/**" { t.Fatalf("include not preserved: %+v", ctx.Include) }

	allowed := ResolveFileFiltered(project, contexts, "@algorithm-core/src/core.rs")
	if !allowed.Found() { t.Fatalf("included file did not resolve: %+v", allowed.Matches) }
	blocked := ResolveFileFiltered(project, contexts, "@algorithm-core/secret/internal.rs")
	if len(blocked.Matches) != 0 { t.Fatalf("excluded file resolved: %+v", blocked.Matches) }
	if !IsDisplayPath(contexts, "@algorithm-core/src/core.rs") { t.Fatal("named display path not recognized") }
}

func mustWriteNamedTest(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil { t.Fatal(err) }
}
