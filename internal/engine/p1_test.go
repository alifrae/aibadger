package engine

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/PVRLabs/aibadger/internal/model"
	"github.com/PVRLabs/aibadger/internal/taggedfile"
	"github.com/PVRLabs/aibadger/internal/writer"
)

func TestNamedExternalRootUsesTagAliasAndIncludeFilter(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	if err := os.MkdirAll(filepath.Join(external, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	eng := &Engine{Root: root, Topology: &model.ProjectTopology{ExternalContext: []model.ExternalContext{{
		Path:    "@algorithm-core",
		AbsPath: external,
		Label:   "algorithm-core",
		Include: []string{"src/**"},
	}}}}

	roots := eng.ExternalRoots()
	if len(roots) != 1 || roots[0].Path != "algorithm-core" {
		t.Fatalf("unexpected tagged external root: %+v", roots)
	}
	if roots[0].IsOmitted("src", filepath.Join(external, "src")) {
		t.Fatal("included external parent directory was omitted")
	}
	if roots[0].IsOmitted("src/core.rs", filepath.Join(external, "src/core.rs")) {
		t.Fatal("included external path was omitted")
	}
	if !roots[0].IsOmitted("secret/core.rs", filepath.Join(external, "secret/core.rs")) {
		t.Fatal("external include filter did not omit disallowed path")
	}
}

func TestNamedExternalTaggedFileUsesCanonicalDisplayPath(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
	externalFile := filepath.Join(external, "src", "core.rs")
	if err := os.MkdirAll(filepath.Dir(externalFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(externalFile, []byte("pub fn external_detect() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A local path with the same label must not steal an explicitly named
	// external @label reference.
	localFile := filepath.Join(root, "algorithm-core", "src", "core.rs")
	if err := os.MkdirAll(filepath.Dir(localFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localFile, []byte("pub fn local_detect() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	eng := &Engine{Root: root, Topology: &model.ProjectTopology{ExternalContext: []model.ExternalContext{{
		Path:    "@algorithm-core",
		AbsPath: external,
		Label:   "algorithm-core",
		Include: []string{"src/**"},
	}}}}

	files, warnings := eng.resolveTaggedFiles("Inspect @algorithm-core/src/core.rs")
	if len(warnings) != 0 {
		t.Fatalf("unexpected tagged-file warnings: %+v", warnings)
	}
	if len(files) != 1 || files[0].Path != "@algorithm-core/src/core.rs" || files[0].IsLocal {
		t.Fatalf("unexpected canonical tagged file: %+v", files)
	}
}

func TestTaggedDisplayPathFallsBackForLegacyExternalContext(t *testing.T) {
	legacy := t.TempDir()
	eng := &Engine{Topology: &model.ProjectTopology{ExternalContext: []model.ExternalContext{{
		Path:    "../legacy",
		AbsPath: legacy,
	}}}}
	resolved := taggedfile.ResolvedPath{
		Path:       "../legacy/spec.md",
		AbsPath:    filepath.Join(legacy, "spec.md"),
		Source:     taggedfile.SourceExternal,
		SourceRoot: legacy,
	}
	if got := eng.taggedDisplayPath(resolved); got != resolved.Path {
		t.Fatalf("legacy external display changed: %q", got)
	}
}

func TestNamedExternalDisplayPathCannotBeWritten(t *testing.T) {
	eng := &Engine{Topology: &model.ProjectTopology{ExternalContext: []model.ExternalContext{{
		Path:  "@algorithm-core",
		Label: "algorithm-core",
	}}}}
	update := writer.FileUpdate{Path: "@algorithm-core/src/core.rs", Kind: writer.UpdateKindWrite}
	if err := eng.validateUpdatePolicy(update); err == nil {
		t.Fatal("named external display path should be read-only")
	}
}

func TestLocalPathMatchingExternalLabelRemainsWritable(t *testing.T) {
	root := t.TempDir()
	eng := &Engine{Root: root, Topology: &model.ProjectTopology{ExternalContext: []model.ExternalContext{{
		Path:  "@algorithm-core",
		Label: "algorithm-core",
	}}}}
	update := writer.FileUpdate{Path: "algorithm-core/src/core.rs", Kind: writer.UpdateKindWrite}
	if err := eng.validateUpdatePolicy(update); err != nil {
		t.Fatalf("local path should not be blocked by external label: %v", err)
	}
}
