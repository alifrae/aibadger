package engine

import (
	"path/filepath"
	"testing"

	"github.com/PVRLabs/aibadger/internal/model"
	"github.com/PVRLabs/aibadger/internal/writer"
)

func TestNamedExternalRootUsesTagAliasAndIncludeFilter(t *testing.T) {
	root := t.TempDir()
	external := t.TempDir()
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
	if roots[0].IsOmitted("src/core.rs", filepath.Join(external, "src/core.rs")) {
		t.Fatal("included external path was omitted")
	}
	if !roots[0].IsOmitted("secret/core.rs", filepath.Join(external, "secret/core.rs")) {
		t.Fatal("external include filter did not omit disallowed path")
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
