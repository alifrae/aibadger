package tui

import (
	"fmt"
	"strings"

	"github.com/PVRLabs/aibadger/internal/protocol"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) postApplyActive() bool {
	return len(m.postApply.Files) > 0 || strings.TrimSpace(m.postApply.Diff) != ""
}

func (m Model) preparePostApplyIndependentReview() (tea.Model, tea.Cmd) {
	if !m.postApplyActive() {
		return m, nil
	}
	m.cfg.Focus = protocol.FocusReview
	m.state = stateHome
	m.goal = ""
	m.err = nil
	m.completion.suppressedKey = ""
	m.setGoalInputValue("Review only the exact Badger-applied delta attached below. Check whether the landed changes match the intended task and identify concrete correctness, regression, or safety issues. Do not review unrelated pre-existing worktree changes.")
	m.setGoalAttachments([]goalAttachment{newGoalGitDiffAttachmentWithStats(
		"Badger post-apply delta",
		m.postApply.Diff,
		len(m.postApply.Files),
		m.postApply.Additions,
		m.postApply.Deletions,
	)})
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
