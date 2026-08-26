![AI Badger](assets/hero.png)

# AI Badger — Local AI Coding Context Tool

**Local-first codebase context extraction for any AI chat (Claude, ChatGPT, Grok, DeepSeek, etc.).**

Get **precise, token-efficient context** on demand without uploading your entire repo or wasting tokens on irrelevant files.

[![GitHub stars](https://img.shields.io/github/stars/PVRLabs/aibadger.svg)](https://github.com/PVRLabs/aibadger/stargazers)
[![Release](https://img.shields.io/github/v/release/PVRLabs/aibadger)](https://github.com/PVRLabs/aibadger/releases/latest)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Homebrew](https://img.shields.io/badge/Homebrew-available-brightgreen)](https://github.com/PVRLabs/homebrew-tap)
[![skills.sh](https://skills.sh/b/PVRLabs/aibadger)](https://skills.sh/PVRLabs/aibadger)

**No cloud • No API keys • No telemetry • Fully local**

[▶ Try Interactive Demo](https://pvrlabs.xyz/aibadger/demo.html) • [Install](#install)

[![AI Badger Interactive Demo](assets/demo.gif)](https://pvrlabs.xyz/aibadger/demo.html)

**Map → Extract → Apply:** Smart local context bridge that prepares focused codebase snippets for any LLM chat.

> [!NOTE]
> This fork's P0 hardening work is on `feat/p0-pcs-hardening`. Snapshot pinning, project policy, DLP rules, patch-only writes, exact post-apply review, and explicit verification are currently branch features rather than published upstream release features. See [Install](docs/install.md) for the source-build path.

## How it works

**1. Map**  
Enter your goal. Badger builds a prompt.  
↳ You copy it → paste into your AI chat

**2. Extract**  
AI replies asking for specific files.  
↳ You copy that → paste back into Badger

**3. Apply**  
Badger fetches those files, builds a second prompt.  
↳ You copy it → paste into AI → review before writing

With the optional hardening policy, Badger can additionally pin the repository snapshot, block sensitive egress, require unified patches, and show the exact landed diff before declaring the task complete.

✓ Fully local — nothing leaves your machine until you copy it  
✓ You control every paste and every write

## Why AI Badger?

- **Universal compatibility** — Works with any AI chat interface or local model
- **Local-first codebase context tool** — Explicit handoff instead of an inbound repo service
- **Token efficient** — Send only the context the reviewer actually asks for
- **Precise & lightweight** — Built in Go, fast, minimal overhead
- **Specialized modes** — `review` and `design` for common workflows
- **Optional hardening** — project policy, snapshot consistency, egress rules, guarded patches, and post-apply review on the P0 branch

## Install

### Published upstream release

Homebrew:

```bash
brew install pvrlabs/tap/badger
```

Quick curl install:

```bash
curl -fsSL https://raw.githubusercontent.com/PVRLabs/aibadger/main/install.sh | sh
```

### Build this hardening branch from source

The P0 features in this fork are not yet in the published upstream binaries.

```bash
git clone https://github.com/alifrae/aibadger.git
cd aibadger
git switch feat/p0-pcs-hardening
mkdir -p bin
go build -o ./bin/badger ./cmd/badger
```

Then run that binary from the project you want to inspect:

```bash
cd /path/to/your-project
/path/to/aibadger/bin/badger code
```

See [docs/install.md](docs/install.md) for Windows, source builds, verification, and avoiding confusion between upstream and fork binaries.

Also available with an official [VS Code companion](https://marketplace.visualstudio.com/items?itemName=pvrlabs.ai-badger). The extension follows the upstream product; do not assume it exposes unreleased fork-only hardening features.

## Optional project hardening

Create `.badger.toml` in the target repository root:

```toml
[security]
deny = ["recordings/**", "customer_data/**", "**/*.pcap", "**/*.dat"]
block_secrets = true

[session]
require_snapshot = true

[write]
patch_only = true
post_apply_review = true

[verify]
command = ["go", "test", "./..."]
```

No `.badger.toml` means existing Badger behavior is preserved.

See [Project Policy](docs/project-policy.md) for every setting and dependency.

## Agent Skills

Badger includes the `handoff` and `badger-review` skills for continuing an AI
coding session or requesting an independent review. See the [Agent Skills
guide](skills/README.md) for installation, usage, and details.

## Quick Start

1. Run `badger` (or the explicit source-built fork binary) in your project root.
2. Type your goal.
3. Copy **Prompt 1** → paste into your AI chat.
4. When the AI asks for files, copy its selector response → paste back into Badger.
5. Copy **Prompt 2** → paste back to the AI.
6. Paste the AI's final response into Badger.
7. Review and approve the proposed write.
8. If `post_apply_review` is enabled, inspect the **actual landed diff**, optionally press `V` for configured verification or `R` for an independent review, then press Enter to finish.

### Specialized Modes

- `badger code` — explicitly start in Code focus
- `badger review` — Git changes and bounded supporting context for immediate findings
- `badger design` — explicitly start in Design focus with an empty editor
- `badger continue` — consume an explicit `.badger-handoff` from another agent/session

Full usage: [docs/usage.md](docs/usage.md)

## PCS example

The [PCS End-to-End Tutorial](docs/pcs-tutorial.md) walks through a concrete engineering task: exposing existing detection-probability statistics through PCS's agent-facing semantic API for Pia. It covers source installation, a hardened PCS policy, snapshot-aware selectors, DLP decisions, guarded patch application, exact landed-diff review, targeted verification, and independent ChatGPT review.

## Learn More

- [Usage Examples & Walkthrough](docs/usage.md)
- [PCS End-to-End Tutorial](docs/pcs-tutorial.md)
- [Project Policy & Configurability](docs/project-policy.md)
- [Installation & Source Builds](docs/install.md)
- [Browser Handoff Guide](docs/handoff.md)
- [API Reference](docs/api.md) — Non-interactive commands for editors and scripts
- [Agent Integrations](docs/agents.md) — Compact repository orientation for coding agents
- [Articles](docs/articles/)
- [Protocol Reference](docs/protocol.md)
- [Limitations & Supported Projects](docs/limitations.md)
- [Privacy & Safety](docs/privacy.md)
- [Contributing](docs/development.md)

---

**Star if this local AI coding context tool solves a real pain for you ⭐**

Built in San Diego by [PVR Labs](https://pvrlabs.xyz). 🌊

[Website](https://pvrlabs.xyz/aibadger) • [X @kupolov](https://x.com/kupolov)
