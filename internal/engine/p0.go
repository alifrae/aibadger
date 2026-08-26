package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/PVRLabs/aibadger/internal/egress"
	"github.com/PVRLabs/aibadger/internal/externalcontext"
	"github.com/PVRLabs/aibadger/internal/extractor"
	"github.com/PVRLabs/aibadger/internal/projectpolicy"
	"github.com/PVRLabs/aibadger/internal/protocol"
	"github.com/PVRLabs/aibadger/internal/snapshot"
	"github.com/PVRLabs/aibadger/internal/writer"
)

func loadP0State(root string) (projectpolicy.Policy, snapshot.State, error) {
	policy, err := projectpolicy.Load(root)
	if err != nil {
		return projectpolicy.Policy{}, snapshot.State{}, err
	}
	if !policy.Session.RequireSnapshot {
		return policy, snapshot.State{}, nil
	}
	state, err := snapshot.Capture(root)
	if err != nil {
		return projectpolicy.Policy{}, snapshot.State{}, err
	}
	return policy, state, nil
}

func (e *Engine) decorateMap(prompt string) (string, []string) {
	if e == nil {
		return prompt, nil
	}
	var warnings []string
	if e.policyErr != nil {
		warnings = append(warnings, e.policyErr.Error())
	}
	if suffix := e.policyPromptSuffix(); suffix != "" {
		prompt += suffix
	}
	if e.Policy.Session.RequireSnapshot {
		prompt += snapshot.PromptSuffix(e.Snapshot)
	}
	return prompt, warnings
}

func (e *Engine) policyPromptSuffix() string {
	if e == nil {
		return ""
	}
	var lines []string
	if len(e.Policy.Context.AlwaysInclude) > 0 {
		lines = append(lines, "Context hints: "+strings.Join(e.Policy.Context.AlwaysInclude, ", "))
	}
	if len(e.Policy.Docs.CanonicalRoots) > 0 {
		lines = append(lines, "Canonical docs: "+strings.Join(e.Policy.Docs.CanonicalRoots, ", "))
	}
	if e.Topology != nil {
		for _, ctx := range e.Topology.ExternalContext {
			if ctx.Label == "" {
				continue
			}
			line := "External source: " + ctx.Path + " [read-only]"
			if ctx.GitRevision != "" {
				line += " revision=" + shortRevision(ctx.GitRevision)
			}
			if len(ctx.Include) > 0 {
				line += " include=" + strings.Join(ctx.Include, ",")
			}
			lines = append(lines, line)
		}
	}
	if e.Policy.Write.PatchOnly {
		lines = append(lines, "Write mode: unified diff only. For code changes use --- Patch --- / --- End Patch --- around a standard unified diff with repository-relative a/ and b/ paths.")
	}
	if len(lines) == 0 {
		return ""
	}
	return "\n[PROJECT POLICY]\n" + strings.Join(lines, "\n") + "\nTreat context and documentation entries as routing hints; request only the specific files needed for the task.\n"
}

func (e *Engine) parseSnapshotInput(input string) (string, error) {
	id, selectors, err := snapshot.ParseEnvelope(input)
	if err != nil {
		return "", err
	}
	if e == nil {
		return selectors, nil
	}
	if e.snapshotErr != nil && e.Policy.Session.RequireSnapshot {
		return "", e.snapshotErr
	}
	if err := snapshot.ValidateExpected(id, e.Snapshot, e.Policy.Session.RequireSnapshot); err != nil {
		return "", err
	}
	if err := e.ValidateSnapshotCurrent(); err != nil {
		return "", err
	}
	return selectors, nil
}

func (e *Engine) filterEgress(extractions []protocol.ExtractionResult) ([]protocol.ExtractionResult, []string, error) {
	if e == nil {
		return extractions, nil, nil
	}
	if e.policyErr != nil {
		return nil, nil, e.policyErr
	}
	kept := make([]protocol.ExtractionResult, 0, len(extractions))
	var notices []string
	blocked := 0
	for _, item := range extractions {
		decision := egress.Inspect(item.Path, item.Content, e.Policy)
		if decision.Blocked {
			blocked++
			notices = append(notices, decision.Warning)
			continue
		}
		if decision.Warning != "" {
			notices = append(notices, decision.Warning)
		}
		kept = append(kept, item)
	}
	if len(kept) == 0 && blocked > 0 {
		return nil, notices, extractor.ErrNoSafePrompt2Files
	}
	return kept, notices, nil
}

// ValidateSnapshotCurrent rejects continuation or writes when snapshot pinning
// is enabled and the repository changed after Prompt 1 was generated.
func (e *Engine) ValidateSnapshotCurrent() error {
	if e == nil || !e.Policy.Session.RequireSnapshot || e.Snapshot.ID == "" {
		return nil
	}
	current, err := snapshot.Capture(e.Root)
	if err != nil {
		return fmt.Errorf("validating repository snapshot: %w", err)
	}
	if current.ID != e.Snapshot.ID {
		return fmt.Errorf("repository changed after Prompt 1; expected snapshot %s, current %s; regenerate context before continuing", shortSnapshot(e.Snapshot.ID), shortSnapshot(current.ID))
	}
	return nil
}

// ValidateWriteBase rejects stale write batches before the first file changes.
func (e *Engine) ValidateWriteBase() error {
	if err := e.ValidateSnapshotCurrent(); err != nil {
		return fmt.Errorf("refusing stale write: %w", err)
	}
	return nil
}

func shortSnapshot(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

func shortRevision(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:12]
}

func (e *Engine) validateUpdatePolicy(update writer.FileUpdate) error {
	if e == nil {
		return nil
	}
	if e.Policy.Write.PatchOnly && update.Kind != writer.UpdateKindPatch {
		return fmt.Errorf(".badger.toml write.patch_only requires unified-diff patches")
	}
	paths := []string{update.Path}
	if update.Kind == writer.UpdateKindPatch {
		var err error
		paths, err = writer.PatchPaths(update.Content)
		if err != nil {
			return err
		}
	}
	for _, path := range paths {
		if e.Policy.Denies(path) {
			return fmt.Errorf("write blocked by .badger.toml security.deny: %s", path)
		}
		if e.Topology != nil && externalcontext.IsDisplayPath(e.Topology.ExternalContext, path) {
			return externalContextWriteError(path)
		}
		if e.isExternalContextPath(path) {
			return externalContextWriteError(path)
		}
		if err := validateExistingTarget(e.Root, path); err != nil {
			return err
		}
	}
	return nil
}

func validateExistingTarget(root, rel string) error {
	full := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Lstat(full)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("inspect write target %s: %w", rel, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, err := filepath.EvalSymlinks(full)
		if err != nil {
			return fmt.Errorf("resolve write target %s: %w", rel, err)
		}
		rootAbs, _ := filepath.Abs(root)
		resolvedAbs, _ := filepath.Abs(resolved)
		relToRoot, err := filepath.Rel(rootAbs, resolvedAbs)
		if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
			return fmt.Errorf("write target escapes project root through symlink: %s", rel)
		}
	}
	return nil
}
