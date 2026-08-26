package extractor

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseSemanticSelectors(t *testing.T) {
	e := &Extractor{}
	got := e.ParseStrictCommandsDetailed("SYMBOL:src/api.py#FrameProvider\nREFERENCES:FrameProvider\nTESTS:FrameProvider\nSEARCH:open_live_source\n")
	if len(got.Failures) != 0 {
		t.Fatalf("unexpected parse failures: %+v", got.Failures)
	}
	if len(got.Commands) != 4 {
		t.Fatalf("got %d commands: %+v", len(got.Commands), got.Commands)
	}
	if got.Commands[0].Type != "SYMBOL" || got.Commands[0].Pattern != "FrameProvider" {
		t.Fatalf("unexpected SYMBOL: %+v", got.Commands[0])
	}
}

func TestReferencesAndTestsAreBoundedAndFiltered(t *testing.T) {
	root := t.TempDir()
	mustWriteSelectorTest(t, filepath.Join(root, "src/provider.go"), "type FrameProvider struct{}\n")
	mustWriteSelectorTest(t, filepath.Join(root, "tests/provider_test.go"), "func TestFrameProvider() { _ = FrameProvider{} }\n")
	mustWriteSelectorTest(t, filepath.Join(root, "docs/note.md"), "FrameProvider documentation\n")
	for i := 0; i < 20; i++ {
		mustWriteSelectorTest(t, filepath.Join(root, "src", "refs", string(rune('a'+i))+".go"), "var _ = FrameProvider{}\n")
	}

	e := NewExtractor(root, nil)
	refs, failures := e.expandCommands([]Command{{Type: "REFERENCES", Path: "FrameProvider"}})
	if len(failures) != 0 {
		t.Fatalf("unexpected reference failures: %+v", failures)
	}
	if len(refs) == 0 || len(refs) > maxSelectorMatches {
		t.Fatalf("reference count=%d", len(refs))
	}
	for _, cmd := range refs {
		if cmd.Type != "NEAR" || cmd.Pattern != "FrameProvider" {
			t.Fatalf("unexpected expanded reference: %+v", cmd)
		}
	}

	tests, failures := e.expandCommands([]Command{{Type: "TESTS", Path: "FrameProvider"}})
	if len(failures) != 0 {
		t.Fatalf("unexpected test failures: %+v", failures)
	}
	if len(tests) != 1 || tests[0].Path != "tests/provider_test.go" {
		t.Fatalf("unexpected test matches: %+v", tests)
	}
}

func TestSymbolExpandsToNear(t *testing.T) {
	e := &Extractor{}
	commands, failures := e.expandCommands([]Command{{Type: "SYMBOL", Path: "src/api.py", Pattern: "FrameProvider"}})
	if len(failures) != 0 {
		t.Fatalf("unexpected symbol failures: %+v", failures)
	}
	if len(commands) != 1 || commands[0].Type != "NEAR" || commands[0].Path != "src/api.py" {
		t.Fatalf("unexpected expansion: %+v", commands)
	}
}

func TestDiscoveryFailurePreservesSuccessfulExtraction(t *testing.T) {
	root := t.TempDir()
	mustWriteSelectorTest(t, filepath.Join(root, "src/main.go"), "package main\n\nfunc main() {}\n")
	e := NewExtractor(root, nil)

	results, err := e.Extract([]Command{
		{Type: "FILE", Path: "src/main.go"},
		{Type: "SEARCH", Path: "definitely_missing_literal"},
	})
	if len(results) != 1 || results[0].Path != "src/main.go" {
		t.Fatalf("usable extraction was lost: %+v", results)
	}
	var extractionErr *ExtractionError
	if !errors.As(err, &extractionErr) || !extractionErr.CanProceed || len(extractionErr.Failures) != 1 {
		t.Fatalf("expected partial-success extraction error, got %v", err)
	}
}

func TestSearchProjectSkipsSymlinkedFiles(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.go")
	mustWriteSelectorTest(t, outside, "var EscapedSecret = true\n")
	link := filepath.Join(root, "linked.go")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	e := NewExtractor(root, nil)
	matches, err := e.searchProject("EscapedSecret", false, maxSelectorMatches)
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("selector search followed symlink outside project: %+v", matches)
	}
}

func mustWriteSelectorTest(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
