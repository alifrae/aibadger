# Using Badger From Source

Use this guide when you want to run a locally built Badger checkout without installing or replacing your normal released `badger` binary.

This is the recommended way to test the unreleased `feat/p0-pcs-hardening` branch.

## 1. Build once in the Badger source repository

```bash
git clone https://github.com/alifrae/aibadger.git
cd aibadger
git switch feat/p0-pcs-hardening
mkdir -p bin
go build -o ./bin/badger ./cmd/badger
```

PowerShell:

```powershell
git clone https://github.com/alifrae/aibadger.git
Set-Location aibadger
git switch feat/p0-pcs-hardening
New-Item -ItemType Directory -Force bin | Out-Null
go build -o .\bin\badger.exe .\cmd\badger
```

You do not need to `go install` the fork globally.

## 2. Keep an explicit path to the development binary

Linux/macOS:

```bash
export BADGER_DEV=/absolute/path/to/aibadger/bin/badger
```

PowerShell:

```powershell
$env:BADGER_DEV = "C:\absolute\path\to\aibadger\bin\badger.exe"
```

This avoids confusing the branch build with a released `badger` already on `PATH`.

## 3. Interactive usage targets the current directory

For normal TUI usage, change into the project you want Badger to inspect first.

```bash
cd /path/to/PCS
"$BADGER_DEV" code
```

PowerShell:

```powershell
Set-Location C:\path\to\PCS
& $env:BADGER_DEV code
```

Do **not** run the binary from the Badger source checkout and expect it to inspect PCS. Interactive mode uses the current working directory as the project root.

Common modes:

```bash
"$BADGER_DEV"          # Design/default interactive focus
"$BADGER_DEV" code     # implementation-oriented Map -> Extract -> Apply
"$BADGER_DEV" review   # review current Git changes
"$BADGER_DEV" design   # design/exploration focus
"$BADGER_DEV" continue # consume .badger-handoff in the current project
```

## 4. Project policy is read from the target repository

If `/path/to/PCS/.badger.toml` exists, the source-built binary loads that policy while operating on PCS.

It does not load `.badger.toml` from the Badger source repository merely because the executable was built there.

Example target layout:

```text
/home/me/src/aibadger/
    bin/badger              <- executable

/home/me/work/PCS/
    .badger.toml            <- policy Badger will use
    AGENTS.md
    docs/
    ...
```

See [Project Policy](project-policy.md) for configuration details.

## 5. Non-interactive API usage can name the root explicitly

The stable API commands accept `--root`, so they do not depend on the current working directory in the same way as the interactive TUI.

Examples:

```bash
"$BADGER_DEV" api topology --root /path/to/PCS
```

```bash
"$BADGER_DEV" api prompt \
  --root /path/to/PCS \
  --focus code \
  --input /tmp/goal.txt
```

Review context:

```bash
"$BADGER_DEV" api review-context --root /path/to/PCS
```

The API output remains AI-facing text, not a structured JSON topology API.

## 6. Update and rebuild the branch

From the Badger source checkout:

```bash
git switch feat/p0-pcs-hardening
git pull --ff-only
go test ./...
go vet ./...
go build -o ./bin/badger ./cmd/badger
```

If you have local edits, inspect them before pulling. Do not use destructive reset commands merely to update the build.

## 7. Run repository verification before sensitive use

Before using a newly modified source build against sensitive engineering source:

```bash
go test ./...
go vet ./...
go build ./...
```

A successful `go build -o ./bin/badger ./cmd/badger` only proves the CLI target compiled. The broader commands above are the intended final source verification.

## 8. Use multiple Badger builds safely

It is reasonable to keep both:

- `badger` on `PATH`: published upstream release;
- `$BADGER_DEV`: explicit fork/branch build.

Check them separately:

```bash
badger --version
"$BADGER_DEV" --version
```

When testing P0 behavior, always invoke `$BADGER_DEV` explicitly until those features are released.

## 9. Source build + PCS

For a complete workflow using a source-built Badger against PCS—including `.badger.toml`, snapshot pinning, DLP rules, patch-only writes, exact landed-diff review, verification, and second-model review—continue with the [PCS End-to-End Tutorial](pcs-tutorial.md).
