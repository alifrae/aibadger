# Project Policy

Badger can load an optional `.badger.toml` from the project root. When the file is absent, existing Badger behavior is preserved.

The policy is intentionally small and local. It controls context-routing hints, egress protection, repository snapshot pinning, guarded writes, post-apply review, explicit verification, and named read-only external context without connecting Badger to an AI provider.

> [!IMPORTANT]
> P0 hardening lives on `feat/p0-pcs-hardening`. P1 context-quality features—including named external sources—live on the stacked `feat/p1-context-quality` branch of the `alifrae/aibadger` fork. Build the branch you want to evaluate from source until the functionality is released upstream.

## Complete example

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

[external.algorithm-core]
root = "../algo_core"
include = ["src/**", "include/**"]
```

The current parser intentionally supports only a small TOML subset: section headers, booleans, double-quoted strings, and **single-line arrays of double-quoted strings**. Unknown settings are rejected instead of silently ignored.

## Settings reference

| Setting | Type | Default | Effect |
| --- | --- | --- | --- |
| `context.always_include` | string array | `[]` | Advertises important project files/directories as context-routing hints in Prompt 1; does not automatically dump their contents. |
| `docs.canonical_roots` | string array | `[]` | Advertises canonical documentation roots to discourage ad-hoc documentation placement. |
| `security.deny` | glob array | `[]` | Blocks matching extracted context and matching write targets. |
| `security.warn` | glob array | `[]` | Allows matching extracted context but surfaces a sensitivity warning. |
| `security.block_secrets` | boolean | `false` | Blocks high-confidence credential patterns in extracted context. |
| `session.require_snapshot` | boolean | `false` | Pins the Map → Extract/write round trip to one repository state. |
| `write.patch_only` | boolean | `false` | Rejects whole-file write/delete blocks and requires a guarded unified patch. |
| `write.post_apply_review` | boolean | `false` | Captures and displays the exact delta introduced by a Badger apply before task completion. |
| `verify.command` | argv string array | unset | Adds an explicit `V` verification action to the post-apply screen. Requires `write.post_apply_review = true`. |
| `external.<label>.root` | string | unset | Adds a named local directory as read-only external context. |
| `external.<label>.include` | glob array | `[]` | Restricts a named external source to the matching relative paths. |

## Context and documentation hints

`context.always_include` and `docs.canonical_roots` are **routing hints**, not unconditional context injection.

For example:

```toml
[context]
always_include = ["AGENTS.md", "docs/architecture/"]
```

Badger tells the model that these locations matter, but the model should still request only the specific file or span it needs through the normal selector protocol.

This avoids turning repository governance files into permanent prompt overhead.

## Named external sources

P1 adds named, read-only external context directly to `.badger.toml`:

```toml
[external.algorithm-core]
root = "../algo_core"
include = ["src/**", "include/**"]

[external.requirements]
root = "/absolute/path/to/requirements"
include = ["api/**", "system/**"]
```

The section suffix is the source label. Labels must be simple names without spaces, slashes, backslashes, or `@`.

Each named source has these properties:

- `root` is required and must identify a local directory. Relative paths are resolved from the project root; absolute local paths are also accepted.
- `include` is optional. When omitted, the existing external-context omission rules apply to the complete root. When present, only matching relative paths may be resolved, tagged, or suggested.
- the source remains read-only. Patch/write targets using the displayed external path are rejected.
- Badger records the external repository's current Git `HEAD` when the directory is a Git worktree and exposes a shortened revision in Prompt 1 policy metadata.
- named sources appear as `@label` in external-context/provenance output.

Example selector paths:

```text
FILE:@algorithm-core/src/detection/energy.rs
SYMBOL:@algorithm-core/src/detection/energy.rs#compute_energy
```

For a goal-editor file tag, use normal tagged-file syntax:

```text
Review the PCS boundary against @algorithm-core/src/detection/energy.rs
```

The tag parser consumes the first `@`; Badger maps the remaining `algorithm-core/...` path back to the named source internally.

Legacy `.badger-context` remains supported. Named sources are additive rather than a migration requirement. Prefer named sources when source identity, include scoping, and Git provenance matter.

Named external provenance is evidence about **which checkout Badger read**. It does not prove that the external code is correct or compatible with the main project.

## Path globs

`security.deny`, `security.warn`, and `external.<label>.include` use slash-normalized glob patterns.

Examples:

```toml
[security]
deny = ["recordings/**", "**/*.pcap", "**/*.dat"]
warn = ["calibration/**"]
```

`**/*.dat` matches both root-level and nested `.dat` files.

Absolute paths and parent traversal such as `../outside/**` are rejected for security patterns and include patterns. External `root` itself may intentionally point to a sibling directory such as `../algo_core`.

## Egress controls

`security.deny` blocks matching **extracted** context and matching write targets. `security.warn` keeps matching extracted context but surfaces a safety warning before interactive prompt delivery; API callers receive the warning on stderr.

When `security.block_secrets = true`, Badger blocks extracted content containing high-confidence credential patterns such as private-key blocks and common provider API-token formats. This is a safety layer, not a complete enterprise DLP system.

Existing hard-coded sensitive-path exclusions remain in force independently of `.badger.toml`.

> [!WARNING]
> `badger review` constructs its initial authoritative Git review payload through a separate review-context path. The strongest P0 DLP integration currently applies to the Map → Extract flow and guarded writes. Inspect an initial review payload before copying it when the worktree may contain sensitive material. For highly sensitive source, prefer `badger code` with Map → Extract until review-context receives equivalent complete policy coverage.

## Snapshot pinning

With:

```toml
[session]
require_snapshot = true
```

Prompt 1 includes a repository snapshot ID. A selector response must echo it as its first non-empty line:

```text
SNAPSHOT:<id>
FILE:internal/client/client.go
NEAR:internal/client/config.go#Timeout
```

Badger rejects the continuation if the current repository no longer matches the snapshot. The same snapshot guard is checked again immediately before an approved write batch, preventing writes against stale context.

Git repositories are fingerprinted from HEAD, status, staged and unstaged diffs, plus bounded untracked-file content. Non-Git projects use a filesystem metadata fingerprint.

Snapshot pinning is intended to prevent mixed-revision reasoning, not to replace Git history.

## Patch-only writes

With:

```toml
[write]
patch_only = true
```

whole-file write and delete blocks are rejected. The AI response must contain a guarded unified diff:

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

Patch-only mode requires Git to be installed and is intended primarily for Git-backed development workspaces.

Because unified patches are applied through Git, legacy smart whitespace normalization is not applied to this patch path.

## Post-apply review

With:

```toml
[write]
post_apply_review = true
```

Badger does not treat a successful filesystem write as task completion.

Before applying the approved update, it saves temporary copies of only the files that Badger is about to touch. After the apply, it compares those copies with the resulting files and shows the exact delta introduced by that Badger operation.

This is deliberately narrower than a worktree-wide `git diff`: unrelated changes that were already present elsewhere in the repository are not mixed into the landed-change review.

The post-apply screen reports changed files and line counts, shows the actual landed diff, and distinguishes applying a change from reviewing or verifying it.

From that screen:

- `R` prepares an independent Badger review using the exact landed delta **plus the original task**;
- `V` runs the configured verification command, if one exists;
- `Enter` accepts the landed delta and returns to the next goal.

Post-apply review is opt-in so repositories without `.badger.toml` retain the existing Badger write flow. It requires Git because the exact before/after delta is rendered using Git's diff machinery.

## Explicit verification

`verify.command` is an argv array, not a shell command string.

Example:

```toml
[write]
post_apply_review = true

[verify]
command = ["go", "test", "./..."]
```

A verifier without post-apply review is invalid configuration because there would be no UI boundary from which the user could invoke it.

Badger never runs this command merely because the project contains `.badger.toml`. The user must explicitly press `V` on the post-apply review screen.

Badger invokes the executable directly without shell expansion, applies a bounded timeout, and limits captured output.

The configured command should be a trusted, deterministic, preferably non-mutating project check. Badger does not sandbox verification commands and does not infer language-specific checks such as `cargo check`, Python linting, or targeted test selection automatically.

For large repositories, prefer a stable project-owned verification entrypoint or a narrow targeted command rather than forcing every small Badger change through an expensive full suite.

## Example configurations

### Context-only / legacy-safe

```toml
[context]
always_include = ["AGENTS.md"]
```

No write or snapshot behavior changes.

### Read-only sensitive-source analysis

```toml
[security]
deny = ["recordings/**", "customer_data/**", "**/*.pcap", "**/*.dat"]
warn = ["calibration/**"]
block_secrets = true

[session]
require_snapshot = true
```

### Hardened code application

```toml
[security]
deny = ["recordings/**", "customer_data/**"]
block_secrets = true

[session]
require_snapshot = true

[write]
patch_only = true
post_apply_review = true
```

### Hardened application with verification

```toml
[session]
require_snapshot = true

[write]
patch_only = true
post_apply_review = true

[verify]
command = ["python", "-m", "pytest", "-q", "tests/api"]
```

### PCS with a named algorithm source

```toml
[security]
deny = ["recordings/**", "**/*.pcap", "**/*.dat"]
block_secrets = true

[session]
require_snapshot = true

[write]
patch_only = true
post_apply_review = true

[external.algorithm-core]
root = "../algo_core"
include = ["dsp/**", "include/**"]
```

## Configuration errors

Badger fails closed on unsupported or internally inconsistent policy settings.

Examples that are rejected:

```toml
[security]
deny = ["../outside/**"]
```

```toml
[verify]
command = ["go", "test", "./..."]
```

The second example is invalid because `write.post_apply_review` is not enabled.

A named external source without `root`, a label containing spaces/slashes, or an unsafe `include` pattern is also rejected.

## PCS profile

For a complete engineering example using PCS, Pia's agent-facing API, snapshot pinning, egress policy, guarded patching, post-apply verification, and independent review, see [PCS End-to-End Tutorial](pcs-tutorial.md).
