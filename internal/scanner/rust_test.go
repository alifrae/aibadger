package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRustDetectorDiscoversWorkspaceCrates(t *testing.T) {
	root := t.TempDir()
	mustWriteRustTest(t, filepath.Join(root, "Cargo.toml"), "[workspace]\nmembers = [\"crates/core\", \"crates/bridge\"]\n")
	mustWriteRustTest(t, filepath.Join(root, "crates/core/Cargo.toml"), "[package]\nname = \"pcs-core\"\nversion = \"0.1.0\"\n")
	mustWriteRustTest(t, filepath.Join(root, "crates/core/src/lib.rs"), "pub fn decode() {}\n")
	mustWriteRustTest(t, filepath.Join(root, "crates/core/tests/decode_test.rs"), "#[test]\nfn decodes() {}\n")
	mustWriteRustTest(t, filepath.Join(root, "crates/bridge/Cargo.toml"), "[package]\nname = \"pcs-pybridge\"\nversion = \"0.1.0\"\n")
	mustWriteRustTest(t, filepath.Join(root, "crates/bridge/src/lib.rs"), "pub fn bind() {}\n")
	mustWriteRustTest(t, filepath.Join(root, "crates/bridge/build.rs"), "fn main() {}\n")

	modules, err := NewRustDetector().Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(modules) != 2 {
		t.Fatalf("got %d Rust modules, want 2: %+v", len(modules), modules)
	}
	if modules[0].Name != "pcs-pybridge" && modules[1].Name != "pcs-pybridge" {
		t.Fatalf("missing bridge crate: %+v", modules)
	}
	if modules[0].Name != "pcs-core" && modules[1].Name != "pcs-core" {
		t.Fatalf("missing core crate: %+v", modules)
	}
	for _, module := range modules {
		if module.Language != "Rust" {
			t.Fatalf("module language=%q", module.Language)
		}
		if module.FileCount == 0 {
			t.Fatalf("empty crate module: %+v", module)
		}
	}

	topology, err := NewScanner(root).Scan()
	if err != nil {
		t.Fatal(err)
	}
	if topology.PrimaryLanguage != "Rust" {
		t.Fatalf("primary language=%q, want Rust", topology.PrimaryLanguage)
	}
	if !containsRustTestString(topology.Stack, "Cargo") {
		t.Fatalf("Cargo missing from stack: %+v", topology.Stack)
	}
}

func TestRustDetectorSkipsVirtualWorkspaceManifest(t *testing.T) {
	root := t.TempDir()
	mustWriteRustTest(t, filepath.Join(root, "Cargo.toml"), "[workspace]\nmembers = []\n")
	modules, err := NewRustDetector().Detect(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(modules) != 0 {
		t.Fatalf("virtual workspace should not become a crate: %+v", modules)
	}
}

func containsRustTestString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func mustWriteRustTest(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
