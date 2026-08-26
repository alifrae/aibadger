# Install AI Badger

This document covers released installs and source builds.

> [!IMPORTANT]
> The P0 hardening features documented in [Project Policy](project-policy.md) and the [PCS tutorial](pcs-tutorial.md) currently live on the `feat/p0-pcs-hardening` branch of the `alifrae/aibadger` fork. Upstream Homebrew/curl/PowerShell release installs do **not** contain those branch-only changes yet. Build the fork branch from source when testing snapshot pinning, DLP policy, patch-only writes, post-apply review, or explicit verification.

## Released upstream installs

Use these methods when you want the current published PVRLabs release.

### Homebrew

Install from the shared PVRLabs tap:

```bash
brew install pvrlabs/tap/badger
```

The tap pulls release tarballs from GitHub Releases.

### Curl installer (Linux and macOS)

Install the latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/PVRLabs/aibadger/main/install.sh | sh
```

The installer downloads the matching GitHub Release tarball for your platform,
verifies its SHA-256 checksum, and installs `badger` into `~/.local/bin` by
default. When that directory is not already on `PATH`, it tries to make
`badger` available immediately with a symlink and updates supported shell
configuration (`bash`, `zsh`, or `fish`) for future terminals. If needed,
restart the terminal or add the directory yourself:

```sh
export PATH="$HOME/.local/bin:$PATH"
```

Install a specific release:

```bash
curl -fsSL https://raw.githubusercontent.com/PVRLabs/aibadger/main/install.sh | BADGER_VERSION=vX.Y.Z sh
```

Install into a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/PVRLabs/aibadger/main/install.sh | BADGER_INSTALL_DIR="$HOME/bin" sh
```

### Windows released install

PowerShell one-liner (default install to `%LOCALAPPDATA%\Programs\Badger`):

```powershell
irm https://raw.githubusercontent.com/PVRLabs/aibadger/main/install.ps1 | iex
```

The installer adds this directory to your User `PATH` and updates the current
PowerShell session when possible. Restart other terminals that were already
open before installing.

Custom directory or version:

```powershell
$env:BADGER_INSTALL_DIR = "$HOME\bin"; irm https://raw.githubusercontent.com/PVRLabs/aibadger/main/install.ps1 | iex
$env:BADGER_VERSION = "vX.Y.Z"; irm https://raw.githubusercontent.com/PVRLabs/aibadger/main/install.ps1 | iex
```

Manual: download `badger_<version>_windows_amd64.zip` from the upstream latest release and extract `badger.exe` to your `PATH`.

A published upstream source install is also possible with:

```powershell
go install github.com/PVRLabs/aibadger/cmd/badger@latest
```

That command installs upstream, not the unreleased hardening fork branch.

---

## Build the P0 hardening branch from source

### Requirements

- Git;
- Go **1.26.5 or newer** (the current `go.mod` version for this branch);
- a supported terminal;
- Git on `PATH` at runtime when using patch-only writes or post-apply review.

Clone the fork:

```bash
git clone https://github.com/alifrae/aibadger.git
cd aibadger
git switch feat/p0-pcs-hardening
```

Confirm the branch:

```bash
git branch --show-current
```

Expected:

```text
feat/p0-pcs-hardening
```

### Linux/macOS development build

```bash
mkdir -p bin
go build -o ./bin/badger ./cmd/badger
./bin/badger --version
```

Use the binary from another project without copying it into that project:

```bash
export BADGER_DEV=/absolute/path/to/aibadger/bin/badger
cd /path/to/your-project
"$BADGER_DEV" code
```

Badger uses the current working directory as the interactive project root, so run the built binary **from the repository you want Badger to inspect**, not from the Badger source directory.

### Windows PowerShell development build

```powershell
git clone https://github.com/alifrae/aibadger.git
Set-Location aibadger
git switch feat/p0-pcs-hardening
New-Item -ItemType Directory -Force bin | Out-Null
go build -o .\bin\badger.exe .\cmd\badger
.\bin\badger.exe --version
```

Use it against another checkout:

```powershell
$env:BADGER_DEV = (Resolve-Path .\bin\badger.exe).Path
Set-Location C:\path\to\your-project
& $env:BADGER_DEV code
```

### Release-style source build

For a smaller binary:

Linux/macOS:

```bash
go build -tags aibadger_release -ldflags="-s -w" -o ./bin/badger ./cmd/badger
```

PowerShell:

```powershell
go build -tags aibadger_release -ldflags="-s -w" -o .\bin\badger.exe .\cmd\badger
```

For development and verification, the normal non-stripped build is easier to diagnose.

### Run repository checks before trusting a source build

From the Badger source repository:

```bash
go test ./...
go vet ./...
go build ./...
```

Do this before using a locally modified Badger build for sensitive repository work.

---

## Project configuration

Badger can read an optional `.badger.toml` from the target project root. No configuration file means legacy behavior.

A hardened starter profile is:

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

`verify.command` requires `write.post_apply_review = true`; verification is invoked only when the user explicitly presses `V` on the post-apply review screen.

See [Project Policy](project-policy.md) for every setting and [PCS End-to-End Tutorial](pcs-tutorial.md) for a complete configured workflow.

---

## VS Code

Install the official [AI Badger for VS Code](https://marketplace.visualstudio.com/items?itemName=pvrlabs.ai-badger)
extension from the Visual Studio Marketplace. Direct file copy and Git review
work without the CLI; Ask and Deep Review need a local `badger` on `PATH`.

The extension tracks the upstream product. Do not assume it exposes branch-only hardening behavior unless it is explicitly updated to do so.

Source and issues: [PVRLabs/aibadger-vscode](https://github.com/PVRLabs/aibadger-vscode).

## Agent Skills

Badger provides the official `handoff` and `badger-review` Agent Skills. Install
both through the [skills.sh](https://skills.sh/PVRLabs/aibadger) ecosystem:

```bash
npx skills add PVRLabs/aibadger
```

If the Badger binary is already installed, you can instead install its bundled
copies without network access:

```bash
badger skills install
```

When testing the hardening fork build, invoke that exact binary if you want its bundled skill version:

```bash
"$BADGER_DEV" skills install
```

Both methods install the Skill definitions. They do not install the `badger`
CLI itself. To complete the workflow, the CLI must be installed separately and
available on `PATH` or invoked by explicit path.

## Verify which Badger you are running

A machine can have both an upstream release and a fork build. Check explicitly.

Linux/macOS:

```bash
which badger
badger --version
"$BADGER_DEV" --version
```

PowerShell:

```powershell
Get-Command badger
badger --version
& $env:BADGER_DEV --version
```

Published installs should report the current release version. Source builds from
a development branch may report a development version until a release is prepared.

## Release notes

For release publishing and artifact details, see [releasing.md](releasing.md).
