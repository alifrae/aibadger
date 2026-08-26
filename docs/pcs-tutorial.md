# PCS End-to-End Tutorial

This tutorial shows a complete hardened Badger workflow against a local Point Cloud Studio (PCS) checkout.

The concrete engineering task is:

> Expose the existing detection-probability (DP) statistics through the PCS agent-facing semantic API so Pia can consume them programmatically, without changing the DP algorithm or GUI behavior.

This is deliberately a real PCS-style task rather than a toy edit. It crosses an API boundary, must preserve existing behavior, should reuse current PCS patterns, and benefits from an independent second-model review.

The exact PCS file names are **not hard-coded in this document**. Badger's job is to map the current checkout, and the AI should request the files that exist in that specific PCS revision. Copying stale file paths from a tutorial would defeat snapshot pinning and context discovery.

> [!IMPORTANT]
> The P0 hardening features in this tutorial currently live on the `feat/p0-pcs-hardening` branch of the `alifrae/aibadger` fork. A normal upstream Homebrew/curl release does not contain these branch-only features yet.

## What the completed workflow looks like

```text
PCS checkout
    |
    |  badger code
    v
Prompt 1: current topology + snapshot ID
    |
    |  explicit copy
    v
ChatGPT / another independent AI
    |
    |  SNAPSHOT + FILE/PREFIX/NEAR selectors
    v
Badger extracts only requested context
    |
    |  DLP / deny / warn checks
    v
Prompt 2: bounded source context
    |
    |  explicit copy
    v
AI returns guarded unified patch
    |
    v
Badger validates snapshot + paths + policy + git apply --check
    |
    |  explicit write approval
    v
Patch lands in PCS
    |
    v
Post-apply screen: exact Badger-owned landed diff
    |                         |
    | V                       | R
    v                         v
project verification      independent AI review
    |
    v
manual accept -> normal PCS Git workflow
```

Badger does not commit, push, merge, or automatically trust the AI result.

---

## 1. Prerequisites

You need:

- a local PCS Git checkout;
- Git on `PATH`;
- Go **1.26.5 or newer** to build this Badger branch;
- Python/PCS development dependencies if you want the example verification command to run;
- a browser AI chat such as ChatGPT;
- a terminal in which you can copy/paste the Badger prompts.

Check the tools:

```bash
git --version
go version
python --version
```

On Windows PowerShell, `py --version` may be the appropriate Python command instead.

---

## 2. Build the hardened Badger branch from source

Clone the fork outside the PCS repository:

```bash
git clone https://github.com/alifrae/aibadger.git
cd aibadger
git switch feat/p0-pcs-hardening
```

Build it:

### Linux/macOS

```bash
mkdir -p bin
go build -o ./bin/badger ./cmd/badger
./bin/badger --version
```

### Windows PowerShell

```powershell
New-Item -ItemType Directory -Force bin | Out-Null
go build -o .\bin\badger.exe .\cmd\badger
.\bin\badger.exe --version
```

Keep the resulting binary separate from an upstream installed `badger` while this branch is under development. That prevents accidentally testing the wrong build.

For convenience, define the binary path for the current shell.

Linux/macOS example:

```bash
export BADGER_DEV=/absolute/path/to/aibadger/bin/badger
```

PowerShell example:

```powershell
$env:BADGER_DEV = "C:\path\to\aibadger\bin\badger.exe"
```

---

## 3. Configure PCS policy

Create `.badger.toml` in the **PCS repository root**.

A strong starting profile for PCS is:

```toml
[context]
always_include = ["AGENTS.md", "docs/architecture/", "docs/api/"]

[docs]
canonical_roots = ["docs/architecture/", "docs/development/", "docs/api/"]

[security]
deny = ["recordings/**", "customer_data/**", "**/*.pcap", "**/*.dat", "**/*.ifscan", "**/*.mcap"]
warn = ["calibration/**", "requirements/confidential/**"]
block_secrets = true

[session]
require_snapshot = true

[write]
patch_only = true
post_apply_review = true

[verify]
command = ["python", "-m", "pytest", "-q", "tests/api"]
```

Adapt only the paths and verification command to the actual PCS checkout.

### Why these settings are useful for PCS

`context.always_include` does **not** dump all those files into ChatGPT. It tells Prompt 1 that they are important routing context so the model can request the specific policy/architecture document it needs.

`docs.canonical_roots` advertises the canonical documentation locations instead of encouraging an AI to discover or create documentation in arbitrary directories.

`security.deny` prevents selected recordings and engineering data formats from entering extracted context or being write targets. Add any customer-specific or company-specific paths required by your environment.

`security.warn` does not block the file. It makes the sensitivity visible before handoff. Use it for areas such as calibration material that may sometimes be legitimate context but should never leave the machine casually.

`session.require_snapshot` binds the selector round-trip and final write to the repository state represented by Prompt 1. If Codex, OpenCode, another terminal, or you modify the repository in the middle of the handoff, Badger rejects stale continuation/write state instead of silently mixing revisions.

`write.patch_only` requires an AI to return a unified patch rather than full replacement files. This is the recommended PCS mode because it makes the requested change narrow and lets Git preflight the patch.

`write.post_apply_review` changes the success boundary. A successful filesystem write is not treated as task completion; Badger shows what actually landed.

`verify.command` is optional. If present, it is available only from the post-apply review screen and requires `write.post_apply_review = true`. Badger does not infer a language-specific command and does not run it automatically.

> [!NOTE]
> `verify.command = ["python", "-m", "pytest", "-q", "tests/api"]` is an example appropriate for an API-focused task. If the current PCS test layout uses another targeted entrypoint, use that exact command instead. Avoid turning every small Badger apply into an unnecessarily broad full-repository validation.

---

## 4. Inspect the PCS worktree before starting

Go to PCS, not the Badger source repository:

```bash
cd /path/to/PCS
git status --short
git branch --show-current
```

Badger snapshot pinning supports a dirty worktree. It does **not** require a clean branch. However, you should know what was already modified before the session starts.

If another coding agent is actively modifying the same checkout, finish or pause that work first. Snapshot pinning will correctly reject drift, but repeatedly changing the tree during the handoff creates unnecessary retries.

---

## 5. Start the concrete PCS task

Run the hardened binary from the PCS root:

Linux/macOS:

```bash
"$BADGER_DEV" code
```

PowerShell:

```powershell
& $env:BADGER_DEV code
```

Enter this goal:

```text
Expose the existing detection-probability statistics through the PCS agent-facing semantic API so Pia can consume them programmatically.

Constraints:
- preserve the existing DP calculation and GUI behavior;
- reuse existing PCS semantic API/result-envelope patterns;
- make the smallest maintainable change;
- do not introduce a second DP implementation;
- add or update targeted API tests;
- do not modify unrelated code or documentation;
- do not access recordings, customer data, or confidential calibration data.
```

Press Enter.

Badger scans the current PCS checkout and creates **Prompt 1**.

Because snapshot pinning is enabled, Prompt 1 also carries a repository snapshot identity. The exact value is generated from the current repository state.

---

## 6. Send Prompt 1 to the independent AI

Review the outbound Prompt 1 screen and copy it only when you are satisfied with what it contains.

Paste Prompt 1 into ChatGPT or another AI chat.

The model should use the current topology to decide what it actually needs. For this task it will typically need some combination of:

- the existing DP/statistics implementation or service;
- the current agent-facing PCS API surface;
- the API result/envelope conventions;
- relevant API contract documentation;
- targeted tests around the affected API.

The model must return only Badger selectors when more context is required.

With snapshot pinning enabled, the selector response has this shape:

```text
SNAPSHOT:<the exact snapshot ID from Prompt 1>
FILE:<a real path shown by the current PCS topology>
PREFIX:<another real path>#<literal prefix>
NEAR:<another real path>#<literal anchor>
```

Do **not** paste the placeholder example above into Badger. Paste the real selectors returned by the model.

Badger currently supports `FILE`, `PREFIX`, and `NEAR`; do not invent `SEARCH`, `SYMBOL`, or other selectors that this P0 branch has not implemented yet.

---

## 7. Let Badger extract only the requested PCS context

Paste the model's selector response into Badger.

Badger checks the snapshot first. Two outcomes are important.

### Snapshot unchanged

Badger resolves the requested files/spans and applies the configured egress policy.

### Repository changed since Prompt 1

Badger rejects the continuation. This is expected behavior.

For example, if OpenCode changed an API file after Prompt 1 was created, do not try to force the old selector response through. Restart the Badger task from the new repository state so the model reasons against one coherent revision.

### DLP/egress outcomes

For extracted context:

- a `deny` match is blocked;
- a `warn` match is surfaced for explicit attention;
- high-confidence secret patterns are blocked when `block_secrets = true`.

For this DP API task, source code and API docs should normally be enough. If the AI requests a raw recording or a customer-data file, treat that as a context-selection error and give it safer engineering evidence instead of weakening the policy.

When extraction is acceptable, Badger creates **Prompt 2** containing the focused source context.

---

## 8. Send Prompt 2 and require a guarded patch

Copy Prompt 2 and paste it into the same AI conversation.

The model now has enough evidence to propose the implementation.

Because PCS policy has:

```toml
[write]
patch_only = true
```

Badger accepts a guarded unified patch rather than whole-file replacement output. The final AI response should contain a patch block of the form:

```text
--- Patch ---
--- a/<existing PCS path>
+++ b/<existing PCS path>
@@ ...
 ...
--- End Patch ---
```

The patch can touch multiple files if the task actually requires it, for example the API implementation plus a targeted test. It should not include unrelated cleanup.

Paste the AI's final response into Badger.

---

## 9. Review the proposed write and approve it

Badger parses the response and reaches the write confirmation boundary.

Before writing, the hardened path checks:

- the repository still matches the pinned snapshot;
- the response satisfies patch-only policy;
- patch paths are repository-relative and do not traverse outside the root;
- configured `security.deny` paths are not write targets;
- external context remains read-only;
- symlink/path safety constraints hold;
- `git apply --check` accepts the patch.

Only after those checks and the interactive confirmation does Badger apply the patch.

Press `Y` only after the proposed operation is consistent with the DP API task.

If the snapshot has drifted, Badger should reject the write. Regenerate context rather than bypassing the stale-context protection.

---

## 10. Inspect the **actual landed PCS diff**

With `post_apply_review = true`, Badger does not immediately clear the task and print `Ready for the next goal`.

It records the pre-apply contents of only the files it is about to touch, applies the approved change, reads those files again, and displays the exact before/after delta introduced by that Badger operation.

The final screen contains:

```text
Post-apply review

Actual landed delta: N file(s), +A / -D
Files: ...

Verification: ...

Exact landed diff:
...

R independent AI review   V run verification   Enter finish
```

This is the ground truth to inspect. It is stronger than trusting the AI response or the earlier write-plan screen.

It also avoids mixing unrelated dirty-worktree changes elsewhere in PCS into the Badger-owned delta.

---

## 11. Run the configured targeted verification

Press `V`.

Badger shows the configured argv and runs it directly, without a shell.

For the example policy:

```text
python -m pytest -q tests/api
```

The post-apply screen then reports one of:

- `PASSED`;
- `FAILED` with the exit code;
- `TIMED OUT`.

Captured output is bounded so an unexpectedly verbose test run cannot flood the TUI indefinitely.

A failed verification does not cause Badger to pretend the task succeeded. Inspect the output and fix the implementation through your normal coding workflow.

Badger does not sandbox the verification executable. Only configure commands you already trust in the PCS development environment.

---

## 12. Request an independent second-model review

From the post-apply screen, press `R`.

Badger prepares a new review goal containing:

- the **original DP API task**;
- the **exact delta Badger actually applied**;
- explicit instructions not to review unrelated pre-existing changes.

This is a useful point to use ChatGPT as an independent reviewer even if Codex/OpenCode or another model produced the implementation.

Submit the resulting Badger review prompt to the reviewer.

A useful review target is:

```text
Does the landed change expose the existing DP statistics through the intended agent-facing PCS API without duplicating computation, changing GUI behavior, weakening API semantics, or missing required tests?
```

If the reviewer requests more context, use Badger's normal review continuation selectors. If it returns findings, send those findings to the coding agent responsible for the implementation or start another bounded Badger code task.

Do not automatically apply reviewer suggestions merely because they came from a second model.

---

## 13. Finish the Badger task

When you have inspected the landed delta and are satisfied with the verification/review state, press Enter on the post-apply screen.

Badger clears the task state and returns home.

Badger still does **not** commit or push anything.

Check PCS directly:

```bash
git status --short
git diff
```

Run any additional PCS verification warranted by the risk of the change. For a narrow API addition this should normally be targeted tests plus the appropriate API/integration check rather than automatically running every expensive validation step in the repository.

Then use your normal Git workflow to commit the PCS change.

---

## 14. Using Badger after Codex/OpenCode instead of applying code through Badger

Badger does not have to be the writer.

For the common PCS workflow where Codex or OpenCode performs the implementation, use Badger primarily as the independent review bridge:

```text
Codex/OpenCode implements PCS task
        |
        v
coding agent runs targeted deterministic tests
        |
        v
.badger-handoff / badger-review skill
        |
        v
badger continue
        |
        v
Badger collects authoritative current Git state
        |
        v
ChatGPT independent review
        |
        v
coding agent fixes findings
        |
        v
final deterministic verification
```

The bundled `badger-review` skill can write a compact `.badger-handoff` containing implementation intent, decisions, verification performed, and review focus. It should not copy the entire source tree or agent conversation into the handoff.

Run from the PCS root after the handoff is written:

```bash
"$BADGER_DEV" continue
```

This is likely the default PCS use case. Direct Badger patch application is useful when you explicitly want the browser reviewer to propose a small guarded change.

---

## 15. What remains manual

The hardening deliberately retains human authority at the important trust boundaries:

1. you decide when to start the handoff;
2. you approve what is copied from Badger into the AI chat;
3. you paste the AI's selectors back into Badger;
4. you approve the write;
5. you decide whether to run the configured verifier;
6. you decide whether to request an independent review;
7. you decide whether the final PCS change is acceptable and should be committed.

Badger automates the error-prone plumbing around those decisions: topology, bounded extraction, snapshot identity, DLP checks, patch preflight, exact landed-delta capture, and review-context construction.

---

## 16. Important PCS-specific limitations

### Rust/Cargo topology is not first-class yet

This branch still lacks a Cargo-aware Rust module detector. Rust files are visible to the generic scanner, but Cargo workspace/crate semantics are not modeled like Python/Go package boundaries.

For a PCS task crossing the Rust/PyO3 boundary, make the goal explicit and consider tagging/requesting the relevant `Cargo.toml`, crate entrypoint, Python binding surface, and tests. A first-class Cargo detector is a separate planned improvement.

### Verification is one static argv command

P0 supports one configured verification command. It does not yet select different commands based on changed files or risk level.

For PCS, prefer a narrow project-owned verification entrypoint. If no suitable stable command exists, leave `verify.command` unset and run targeted checks through the normal coding-agent workflow.

### Verification is trusted code execution

Badger uses `exec` directly rather than a shell, but the executable still runs with your local user permissions. `verify.command` is not a sandbox.

### Review-mode egress is a separate path

The strongest P0 egress controls are integrated into the Map → Extract context path and guarded writes. `badger review` builds its initial authoritative Git review context through a separate review-context path. Until that path receives equivalent complete policy coverage, inspect the initial review payload before copying it when working with sensitive PCS changes.

For highly sensitive source, prefer the `badger code` Map → Extract workflow described in this tutorial.

---

## 17. Troubleshooting

### `SNAPSHOT` mismatch

Cause: PCS changed after Prompt 1.

Action: restart the handoff from the new PCS state. Do not copy an old snapshot token into a new session.

### `patch preflight failed`

Cause: the patch no longer applies to the pinned files, has malformed paths, or conflicts with the current checkout.

Action: return the current context to the AI and request a fresh minimal patch. Do not manually strip safety checks from the patch.

### a file is blocked by `security.deny`

Cause: the AI requested or tried to modify a path the PCS policy forbids.

Action: use a safer source of engineering evidence. Change the policy only when the project owner deliberately intends that data class to be eligible.

### `No verify.command is configured`

Cause: no `[verify]` command exists in `.badger.toml`.

Action: either configure a trusted targeted PCS command or perform verification through Codex/OpenCode/manual development tooling.

### `verify.command requires write.post_apply_review = true`

Cause: a verifier was configured without enabling the post-apply screen from which verification is invoked.

Action:

```toml
[write]
post_apply_review = true
```

### you ran upstream `badger` instead of the fork build

Check the path explicitly:

Linux/macOS:

```bash
which badger
"$BADGER_DEV" --version
```

PowerShell:

```powershell
Get-Command badger
& $env:BADGER_DEV --version
```

While the hardening branch is unreleased, invoke the development binary explicitly to avoid ambiguity.

---

## 18. Recommended PCS operating model

Use Badger at bounded reasoning/review boundaries rather than turning it into another autonomous coding agent:

```text
implementation     Codex / OpenCode
       |
       v
verification       deterministic PCS tests/checks
       |
       v
context bridge     Badger
       |
       v
independent review ChatGPT / chosen browser model
       |
       v
fixes              Codex / OpenCode
       |
       v
final verification + commit
```

This preserves the key architectural separation: coding agents execute, Badger controls and proves the context/write handoff, and the browser model provides independent reasoning/review.
