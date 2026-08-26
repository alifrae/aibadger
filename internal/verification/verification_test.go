package verification

import (
	"os/exec"
	"strings"
	"testing"
)

func TestRunPassesWithoutShell(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go unavailable")
	}
	result := Run(t.TempDir(), []string{"go", "version"})
	if !result.Passed || result.ExitCode != 0 {
		t.Fatalf("verification should pass: %+v", result)
	}
	if !strings.Contains(result.Output, "go version") {
		t.Fatalf("unexpected verification output: %q", result.Output)
	}
}

func TestRunReportsFailure(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go unavailable")
	}
	result := Run(t.TempDir(), []string{"go", "definitely-not-a-command"})
	if result.Passed || result.ExitCode == 0 {
		t.Fatalf("verification should fail: %+v", result)
	}
}

func TestRunWithoutCommandDoesNothing(t *testing.T) {
	result := Run(t.TempDir(), nil)
	if result.Passed || result.ExitCode != -1 || !strings.Contains(result.Output, "No verification command") {
		t.Fatalf("unexpected empty-command result: %+v", result)
	}
}

func TestBoundedBufferTruncates(t *testing.T) {
	buffer := &boundedBuffer{limit: 4}
	if _, err := buffer.Write([]byte("abcdef")); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(buffer.String(), "abcd") || !strings.Contains(buffer.String(), "truncated") {
		t.Fatalf("unexpected bounded buffer: %q", buffer.String())
	}
}
