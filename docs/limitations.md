# Limitations

Badger is a context bridge, not an AI provider or autonomous coding agent.

## Current constraints

- The selector protocol remains intentionally bounded. `FILE:`, `PREFIX:`, and `NEAR:` are direct selectors; P1 adds `SYMBOL:`, `REFERENCES:`, `TESTS:`, and `SEARCH:` without turning Badger into a code-indexing service.
- `SYMBOL:` currently resolves a bounded span in a known file using Badger's existing local span extraction. It is not AST- or LSP-aware.
- `REFERENCES:` and `SEARCH:` are bounded literal text searches, not compiler/type-aware reference queries. They return at most 12 matched-file spans.
- `TESTS:` is a bounded literal search over likely test-file paths and returns at most 8 matched-file spans.
- Discovery selectors inspect at most 5,000 eligible project files per request, skip symlinks and files larger than 1 MiB, and intentionally skip common dependency/build/noise directories.
- A failed discovery selector does not discard successful selectors in the same request; it is reported through Badger's existing partial-success diagnostics.
- Non-interactive review automation uses the stable `badger api review-context` operation.
- Binary and generated files are intentionally excluded or minimized to keep prompts compact.
- The P0 verifier supports one configured argv command; it does not yet select commands by changed path, language, or risk level.
- Verification commands are trusted local execution, not sandboxed execution.

## Language topology

P1 adds a first-class Cargo-aware Rust detector. Cargo manifests with a `[package]` section become Rust modules and common crate layout is surfaced for `src/`, `tests/`, `benches/`, `examples/`, and `build.rs`. Virtual workspace-only manifests are not incorrectly presented as crates. Rust projects also report `Cargo` in the detected stack.

This is **structural Cargo topology**, not a full Cargo metadata implementation:

- Badger does not invoke `cargo metadata`;
- dependency graphs, feature resolution, target triples, proc-macro expansion, and cfg evaluation are not modeled;
- Cargo manifests are discovered through Badger's normal initial marker scan, currently bounded to four directory levels below the project root;
- crate discovery follows local manifests found by that scan rather than evaluating every workspace-members glob exactly as Cargo would.

For PyO3 or other mixed Python/Rust systems this is enough to expose crate boundaries and likely bridge files, but it does not prove ABI/API compatibility across the language boundary.

## Named external context

P1 named external sources provide labels, include filters, read-only enforcement, and the external Git `HEAD` when available.

The provenance field identifies the checkout that Badger read; it is not a trust or correctness assertion. A non-Git directory simply has no Git revision.

`include` is a path allow-scope for that external source. It does not replace normal sensitive-path filtering or the project's `security.deny` policy.

Legacy `.badger-context` remains supported but does not gain a synthetic Git revision or include policy unless migrated to a named `.badger.toml` source.

## P0/P1 branch scope

`feat/p0-pcs-hardening` contains the project-policy, snapshot, DLP, guarded-patch, post-apply-review, and verification layer.

`feat/p1-context-quality` is stacked on P0 and adds Cargo/Rust topology, bounded discovery selectors, and named/provenanced external context. It should not be treated as an independently releasable branch until its P0 base is integrated.

The strongest egress-policy integration is still in the Map → Extract flow and guarded writes. The initial `badger review` Git payload is produced through a separate review-context path and should still be manually inspected before copy when sensitive material may be present.

## Snapshot scope

Snapshot pinning prevents Badger from mixing repository revisions during a handoff. It does not freeze external systems, generated services, databases, or files outside the captured project state.

Named external Git provenance is reported separately; P0 project snapshot pinning does not currently freeze or reject drift in those external roots during the main project handoff.

## Post-apply review scope

Post-apply review shows the exact before/after delta for files Badger targeted. It intentionally does not claim that the rest of the worktree is clean or unchanged.

## What Badger does not do

Badger does not:

- log into or automate a hosted AI provider;
- run an autonomous coding loop;
- commit, push, merge, or rebase Git history;
- automatically trust a second-model review;
- provide complete enterprise DLP;
- provide AST/LSP/compiler-semantic code search;
- sandbox project verification commands.

See [Project Policy](project-policy.md), [Protocol](protocol.md), [Privacy and Safety](privacy.md), and the [PCS End-to-End Tutorial](pcs-tutorial.md) for the operational boundaries.
