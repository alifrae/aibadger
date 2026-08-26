package tui

import (
	"strings"
	"testing"

	"github.com/PVRLabs/aibadger/internal/engine"
	"github.com/PVRLabs/aibadger/internal/postapply"
	"github.com/PVRLabs/aibadger/internal/projectpolicy"
	"github.com/PVRLabs/aibadger/internal/protocol"
	tea "github.com/charmbracelet/bubbletea"
)

func postApplyTestModel() Model {
	m := NewModel("/tmp/project", DefaultConfig())
	m.state = stateTextResponse
	m.postApply = postapply.Result{
		Files:     []string{"src/main.go"},
		Diff:      "--- a/src/main.go\n+++ b/src/main.go\n@@ -1 +1 @@\n-old\n+new\n",
		Additions: 1,
		Deletions: 1,
	}
	return m
}

func TestPostApplyReviewShowsGroundTruthAndActions(t *testing.T) {
	m := postApplyTestModel()
	view := m.viewTextResponse()
	for _, want := range []string{"Post-apply review", "src/main.go", "+1 / -1", "Exact landed diff", "independent AI review", "run verification"} {
		if !strings.Contains(view, want) {
			t.Fatalf("post-apply view missing %q:\n%s", want, view)
		}
	}
}

func TestPostApplyReviewShortcutBuildsExactDeltaHandoff(t *testing.T) {
	m := postApplyTestModel()
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	got := next.(Model)
	if got.state != stateHome || protocol.NormalizeFocus(got.cfg.Focus) != protocol.FocusReview {
		t.Fatalf("unexpected review handoff state=%v focus=%v", got.state, got.cfg.Focus)
	}
	if len(got.goalAttachments) != 1 || !strings.Contains(got.goalAttachments[0].Text, "src/main.go") {
		t.Fatalf("exact landed delta was not attached: %+v", got.goalAttachments)
	}
	if got.postApplyActive() {
		t.Fatal("post-apply state should be cleared after it is copied into the review attachment")
	}
}

func TestPostApplyVerifyWithoutCommandDoesNotExecute(t *testing.T) {
	m := postApplyTestModel()
	m.eng = &engine.Engine{Policy: projectpolicy.Policy{}}
	next, cmd := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	got := next.(Model)
	if cmd != nil {
		t.Fatal("verification command should not run without explicit project configuration")
	}
	if !strings.Contains(got.status.text, "No verify.command") {
		t.Fatalf("unexpected verification warning: %q", got.status.text)
	}
}

func TestPostApplyEnterFinishesAndClearsDelta(t *testing.T) {
	m := postApplyTestModel()
	next, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	got := next.(Model)
	if got.state != stateHome || got.postApplyActive() {
		t.Fatalf("post-apply finish did not reset state: state=%v postApply=%+v", got.state, got.postApply)
	}
}
