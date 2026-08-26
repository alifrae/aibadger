package writer

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParsePatchBlock(t *testing.T) {
	input := `Notes before.
--- Patch ---
--- a/main.txt
+++ b/main.txt
@@ -1 +1 @@
-old
+new
--- End Patch ---
Notes after.`
	result := ParseAIResponseDetailed(input)
	if len(result.Errors) != 0 {
		t.Fatalf("unexpected errors: %v", result.Errors)
	}
	if len(result.Updates) != 1 || result.Updates[0].Kind != UpdateKindPatch {
		t.Fatalf("unexpected updates: %+v", result.Updates)
	}
	if result.Text != "Notes before.\nNotes after." {
		t.Fatalf("unexpected notes %q", result.Text)
	}
}

func TestApplyUnifiedDiff(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.txt"), []byte("old\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	patch := `--- a/main.txt
+++ b/main.txt
@@ -1 +1 @@
-old
+new
`
	if err := ApplyUnifiedDiff(root, patch); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(root, "main.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new\n" {
		t.Fatalf("unexpected file content %q", got)
	}
}

func TestPatchRejectsTraversal(t *testing.T) {
	patch := `--- a/../outside.txt
+++ b/../outside.txt
@@ -1 +1 @@
-old
+new
`
	if _, err := PatchPaths(patch); err == nil {
		t.Fatal("expected traversal rejection")
	}
}
