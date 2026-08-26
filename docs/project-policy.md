# Project Policy

Badger can load an optional `.badger.toml` from the project root. When the file is absent, existing Badger behavior is preserved.

The policy is intentionally small and local. It controls context routing hints, egress protection, repository snapshot pinning, and guarded write mode without connecting Badger to an AI provider.

## Example

```toml
[context]
always_include = ["AGENTS.md", "docs/architecture/"]

[docs]
canonical_roots = ["docs/architecture/", "docs/development/", "docs/api/"]

[security]
deny = [
  "recordings/**",
  "customer_data/**",
  "**/*.pcap",
  "**/*.dat"
]
warn = ["calibration/**"]
block_secrets = true

[session]
require_snapshot = true

[write]
patch_only = true
```

Arrays use double-quoted strings. Unknown settings are rejected instead of silently ignored.

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
