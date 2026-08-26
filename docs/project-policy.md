# Project Policy

Badger can load an optional `.badger.toml` from the project root. When the file is absent, existing Badger behavior is preserved.

The policy is intentionally small and local. It controls context routing hints, egress protection, repository snapshot pinning, guarded writes, post-apply review, and explicit verification without connecting Badger to an AI provider.

## Example

```toml
[context]
always_include = ["AGENTS.md", "docs/architecture/"]

[docs]
canonical_roots = ["docs/architecture/", "docs/development/", "docs/api/"]

[security]
deny = ["recordings/**", "customer_data/**", "**/*.pcap", "**/*.dat"]
warn = ["calibration/**"]
block_secrets = true

[session]
require_snapshot = true

[write]
patch_only = true
post_apply_review = true

[verify]
command = ["go", "test", "./..."]
```

The current policy parser intentionally supports only this small TOML subset: section headers, booleans, and single-line arrays of double-quoted strings. Unknown settings are rejected instead of silently ignored.

## Context and documentation hints

`context.always_include` and `docs.canonical_roots` are routing hints in Prompt 1. Badger does not automatically dump those files into the prompt. The model should request only the specific files it needs.

## Egress controls

`security.deny` blocks matching extracted context and matching write targets. `security.warn` keeps matching extracted context but surfaces a safety warning before interactive prompt delivery; API callers receive the warning on stderr.

When `security.block_secrets = true`, Badger blocks extracted content containing high-confidence credential patterns such as private-key blocks and common provider API-token formats. This is a safety layer, not a complete enterprise DLP system.

Existing hard-coded sensitive-path exclusions remain in force independently of `.badger.toml`.

## Snapshot pinning

With `session.require_snapshot = true`, Prompt 1 includes a repository snapshot ID. A selector response must echo it as its first non-empty line:

```text
SNAPSHOT:<id>
FILE:internal/client/client.go
NEAR:internal/client/config.go#Timeout
```

Badger rejects the continuation if the current repository no longer matches the snapshot. The same snapshot guard is checked once immediately before an approved write batch, preventing writes against stale context.

Git repositories are fingerprinted from HEAD, status, staged and unstaged diffs, plus bounded untracked-file content. Non-Git projects use a filesystem metadata fingerprint.

## Patch-only writes

With `write.patch_only = true`, whole-file write and delete blocks are rejected. The AI response must contain a guarded unified diff:

```text
--- Patch ---
--- a/internal/client/client.go
+++ b/internal/client/client.go
@@ -10,1 +10,1 @@
-old
+new
--- End Patch ---
```

Badger validates every patch path, rejects traversal and configured deny paths, checks that external context remains read-only, then runs `git apply --check` before `git apply`. The normal interactive write confirmation still applies.

Patch-only mode therefore requires `git` to be installed and is intended primarily for Git-backed development workspaces.

## Post-apply review

With `write.post_apply_review = true`, Badger does not treat a successful file write as task completion. Before applying the approved update, it saves temporary copies of only the files that Badger is about to touch. After the apply, it compares those copies with the resulting files and shows the exact delta introduced by that Badger operation.

This is deliberately narrower than a normal worktree-wide `git diff`: unrelated changes that were already present elsewhere in the repository are not mixed into the landed-change review.

The post-apply screen reports the changed files and line counts, shows the actual landed diff, and distinguishes applying a change from reviewing or verifying it. From that screen:

- `R` prepares an independent Badger review using the exact landed delta as the review attachment.
- `V` runs the configured verification command, if one exists.
- `Enter` accepts the landed delta and returns to the next goal.

Post-apply review is opt-in so repositories without `.badger.toml` retain the existing Badger write flow. It requires `git` because the exact before/after delta is rendered using Git's diff machinery.

## Explicit verification

`verify.command` is an argv array, not a shell command string. For example:

```toml
[verify]
command = ["go", "test", "./..."]
```

Badger never runs this command merely because the project contains `.badger.toml`. The user must explicitly press `V` on the post-apply review screen. Badger invokes the executable directly without shell expansion, applies a bounded execution timeout, and limits captured output.

The configured command should be a trusted, deterministic, preferably non-mutating project check. Badger does not sandbox verification commands and does not infer language-specific checks such as `cargo check` or Python linting automatically; projects choose the appropriate verification entrypoint themselves.
