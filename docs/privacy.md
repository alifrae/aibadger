# Privacy And Safety

Badger is local-first: scanning, selection, and policy checks happen on the machine where Badger runs. Content leaves the machine only when the user explicitly transfers it to an external AI chat.

## Baseline guarantees

- All scanning and extraction runs locally.
- No telemetry is collected by Badger.
- No cloud sync is used by Badger.
- No source code is copied until you approve the handoff.
- No file writes happen until you review the write boundary and confirm.
- Read-only external context cannot be used as a patch target.

"Local-first" does **not** mean selected source never leaves the machine. If you paste a Badger prompt into ChatGPT or another hosted AI service, that selected prompt content has left the machine under your control.

## Built-in path exclusions

Badger automatically excludes obvious secret-bearing and sensitive paths from scanning and extraction, including:

- **Credentials & Secrets**: `.env` and most `.env.*` files, `.npmrc`, `.pypirc`, `.netrc`. Common environment template files such as `.env.example`, `.env.template`, and `.env.sample` may be extracted.
- **Keys & Certificates**: `*.pem`, `*.key`, `*.p12`, `*.pfx`, `id_rsa`, `id_dsa`, and other common private-key formats.
- **Cloud Configs**: `.aws/credentials`, `.aws/config`, `.gcp/credentials.json`, `.azure/` directories.
- **System & Internal**: `.git`, `.kubeconfig`, and binary artifacts.

These exclusions remain active independently of project policy.

## Optional P0 project egress policy

The `feat/p0-pcs-hardening` branch can load `.badger.toml` from the project root.

Example:

```toml
[security]
deny = ["recordings/**", "customer_data/**", "**/*.pcap", "**/*.dat"]
warn = ["calibration/**"]
block_secrets = true
```

- `deny` prevents matching extracted context from being handed off and prevents matching write targets.
- `warn` allows matching extracted context but makes the sensitivity visible before handoff.
- `block_secrets` scans extracted text for high-confidence credential patterns in addition to path-level exclusions.

These controls reduce accidental disclosure; they are not a substitute for an enterprise DLP product or your employer's data-handling policy.

## Important review-mode limitation

The Map → Extract path and guarded writes have the strongest P0 policy integration.

`badger review` constructs its initial authoritative Git review payload through a separate review-context path. Until that path receives equivalent complete policy coverage, inspect the initial review payload before copying it when the worktree may contain sensitive material.

For highly sensitive source, prefer `badger code` with the snapshot-aware Map → Extract workflow.

## Snapshot consistency

With:

```toml
[session]
require_snapshot = true
```

Badger pins the Map → Extract/write round trip to one repository state. This is a correctness and evidence-consistency control, not a confidentiality control.

## Guarded writes

With:

```toml
[write]
patch_only = true
post_apply_review = true
```

Badger requires a unified patch, validates paths, runs `git apply --check`, requires interactive approval, then shows the exact before/after delta it actually introduced.

## Verification commands

`verify.command` is trusted local code execution. Badger invokes the configured argv directly without a shell, but the process still runs with the user's local permissions.

Do not configure verification commands from an untrusted repository without reviewing them first. Badger does not sandbox the verifier.

## Consent model

The intended authority boundaries are explicit:

1. the user chooses the task;
2. the user approves outbound prompt transfer;
3. the user approves writes;
4. the user explicitly invokes optional verification;
5. the user accepts or rejects the final landed result.

## External context

Read-only external directories can be listed in `.badger-context`.
They are summarized separately from the main project and cannot be used as patch targets.

See [Project Policy](project-policy.md) for configuration details and [PCS End-to-End Tutorial](pcs-tutorial.md) for a concrete sensitive engineering workflow.
