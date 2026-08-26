package tui

import (
	"fmt"
	"strings"

	"github.com/PVRLabs/aibadger/internal/postapply"
	"github.com/PVRLabs/aibadger/internal/protocol"
	"github.com/PVRLabs/aibadger/internal/verification"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) postApplyActive() bool {
	return len(m.postApply.Files) > 0 || strings.TrimSpace(m.postApply.Diff) != ""
}

func (m Model) viewPostApplyReview() string {
	var lines []string
	lines = append(lines, renderBold("Post-apply review"))
	lines = append(lines, "")
	lines = append(lines, fmt.Sprintf("Actual landed delta: %d file(s), +%d / -%d", len(m.postApply.Files), m.postApply.Additions, m.postApply.Deletions))
	if len(m.postApply.Files) > 0 {
		lines = append(lines, "Files: "+strings.Join(m.postApply.Files, ", "))
	}
	lines = append(lines, "")
	if m.verificationRan {
		if m.verification.Passed {
			lines = append(lines, "Verification: PASSED — "+formatCommand(m.verification.Command))
		} else if m.verification.TimedOut {
			lines = append(lines, "Verification: TIMED OUT — "+formatCommand(m.verification.Command))
		} else {
			lines = append(lines, fmt.Sprintf("Verification: FAILED (exit %d) — %s", m.verification.ExitCode, formatCommand(m.verification.Command)))
		}
		if output := strings.TrimSpace(m.verification.Output); output != "" {
			preview, hidden := textPreview(output, 12)
			lines = append(lines, "", "Verification output:", preview)
			if hidden > 0 {
				lines = append(lines, fmt.Sprintf("... [%d more lines hidden] ...", hidden))
			}
		}
	} else if m.eng != nil && len(m.eng.Policy.Verify.Command) > 0 {
		lines = append(lines, "Verification: NOT RUN — configured command: "+formatCommand(m.eng.Policy.Verify.Command))
	} else {
		lines = append(lines, "Verification: NOT CONFIGURED")
	}

	diff := strings.TrimSpace(m.postApply.Diff)
	if diff == "" {
		diff = "(no filesystem delta detected)"
	}
	preview, hidden := textPreview(diff, m.textResponsePreviewLineLimit())
	lines = append(lines, "", renderBold("Exact landed diff:"), "", preview)
	if hidden > 0 {
		lines = append(lines, "", helpStyle.Render(fmt.Sprintf("... [%d more lines hidden] ...", hidden)))
	}
	lines = append(lines, "", renderBold("R")+" independent AI review   "+renderBold("V")+" run verification   "+renderBold("Enter")+" finish")
	return m.renderBox(strings.Join(lines, "\n"))
}

func (m Model) preparePostApplyIndependentReview() (tea.Model, tea.Cmd) {
	if !m.postApplyActive() {
		return m, nil
	}
	attachment := newGoalGitDiffAttachmentWithStats(
		"Badger post-apply delta",
		m.postApply.Diff,
		len(m.postApply.Files),
		m.postApply.Additions,
		m.postApply.Deletions,
	)
	m.cfg.Focus = protocol.FocusReview
	m.state = stateHome
	m.goal = ""
	m.err = nil
	m.completion.suppressedKey = ""
	m.setGoalInputValue("Review only the exact Badger-applied delta attached below. Check whether the landed changes match the intended task and identify concrete correctness, regression, or safety issues. Do not review unrelated pre-existing worktree changes.")
	m.setGoalAttachments([]goalAttachment{attachment})
	m.postApply = postapply.Result{}
	m.verification = verification.Result{}
	m.verificationRan = false
	m.resizeGoalEditor()
	m.focusGoalEditor()
	m.paste.Blur()
	m.status = successMessage("Prepared an independent review from the exact landed delta. Press Enter to build the review prompt.")
	return m, textarea.Blink
}

func (m Model) startPostApplyVerification() (tea.Model, tea.Cmd) {
	if !m.postApplyActive() || m.eng == nil {
		return m, nil
	}
	command := m.eng.Policy.Verify.Command
	if len(command) == 0 {
		m.status = warningMessage("No verify.command is configured in .badger.toml.")
		return m, nil
	}
	m.status = neutralMessage(fmt.Sprintf("Running explicit verification: %s", formatCommand(command)))
	m.err = nil
	return m, verificationCmd(m.root, command)
}

func formatCommand(command []string) string {
	quoted := make([]string, 0, len(command))
	for _, arg := range command {
		if strings.ContainsAny(arg, " \t\n\"") {
			quoted = append(quoted, fmt.Sprintf("%q", arg))
		} else {
			quoted = append(quoted, arg)
		}
	}
	return strings.Join(quoted, " ")
}
