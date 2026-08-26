# Limitations

Badger is a context bridge, not an AI provider or autonomous coding agent.

## Current constraints

- Extraction commands are intentionally simple: `FILE:`, `PREFIX:`, and `NEAR:`.
- `SEARCH:`, `SYMBOL:`, `REFERENCES:`, and `TESTS:` selectors are not implemented in this P0 branch.
- Non-interactive review automation uses the stable `badger api review-context` operation.
- Binary and generated files are intentionally excluded or minimized to keep prompts compact.
- The P0 verifier supports one configured argv command; it does not yet select commands by changed path, language, or risk level.
- Verification commands are trusted local execution, not sandboxed execution.

## Language topology

The supported language detectors improve ranking for common project shapes, but Badger is designed to work across arbitrary repositories.

Rust source files are recognized by the generic scanner, but there is currently **no Cargo-aware Rust module detector**. Cargo workspaces, crate boundaries, `lib.rs`, `main.rs`, and `mod.rs` are therefore not modeled with first-class Cargo semantics yet.

For mixed Python/Rust or PyO3 projects, explicitly request/tag relevant Cargo manifests and boundary files when needed.

## P0 hardening scope

The `feat/p0-pcs-hardening` branch adds project policy, snapshot pinning, DLP rules, guarded unified patches, and exact post-apply review. These are currently unreleased fork features.

The strongest egress-policy integration is in the Map → Extract flow and guarded writes. The initial `badger review` Git payload is produced through a separate review-context path and should still be manually inspected before copy when sensitive material may be present.

## Snapshot scope

Snapshot pinning prevents Badger from mixing repository revisions during a handoff. It does not freeze external systems, generated services, databases, or files outside the captured project state.

## Post-apply review scope

Post-apply review shows the exact before/after delta for files Badger targeted. It intentionally does not claim that the rest of the worktree is clean or unchanged.

## What Badger does not do

Badger does not:

- log into or automate a hosted AI provider;
- run an autonomous coding loop;
- commit, push, merge, or rebase Git history;
- automatically trust a second-model review;
- provide complete enterprise DLP;
- sandbox project verification commands.

See [Project Policy](project-policy.md), [Privacy and Safety](privacy.md), and the [PCS End-to-End Tutorial](pcs-tutorial.md) for the operational boundaries.
