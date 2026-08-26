package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestP1APIHelpAdvertisesDiscoverySelectors(t *testing.T) {
	for _, operation := range []string{"extract", "review-continuation"} {
		var output bytes.Buffer
		printAPIHelp(&output, operation)
		for _, selector := range []string{"SYMBOL", "REFERENCES", "TESTS", "SEARCH"} {
			if !strings.Contains(output.String(), selector) {
				t.Fatalf("%s help missing %s selector:\n%s", operation, selector, output.String())
			}
		}
	}
}

func TestP1APIExtractSupportsBoundedSearchSelector(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte("package sample\n\nfunc FrameProvider() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inputs := t.TempDir()
	goalPath := filepath.Join(inputs, "goal.txt")
	selectorPath := filepath.Join(inputs, "selectors.txt")
	if err := os.WriteFile(goalPath, []byte("Explain FrameProvider.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(selectorPath, []byte("SEARCH:FrameProvider\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	if err := runAPI([]string{
		"extract",
		"--root", root,
		"--input", selectorPath,
		"--goal-file", goalPath,
	}, &stdout, &stderr); err != nil {
		t.Fatalf("runAPI extract failed: %v; stderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "main.go") || !strings.Contains(stdout.String(), "FrameProvider") {
		t.Fatalf("bounded search context missing from API output:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected API diagnostics: %s", stderr.String())
	}
}
