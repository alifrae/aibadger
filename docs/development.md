# Development Notes

The public facade is:

```go
github.com/PVRLabs/aibadger/pkg/badger
```

Most implementation packages remain under `internal/` so the CLI and facade can evolve without exposing scanner, extractor, protocol, writer, security, or TUI internals as public API.

## Build the current checkout

This repository currently declares Go 1.26.5 in `go.mod`.

```bash
go build ./...
```

Build a runnable development binary:

```bash
mkdir -p bin
go build -o ./bin/badger ./cmd/badger
./bin/badger --version
```

On Windows PowerShell:

```powershell
New-Item -ItemType Directory -Force bin | Out-Null
go build -o .\bin\badger.exe .\cmd\badger
.\bin\badger.exe --version
```

## Run the source-built binary against another repository

Interactive Badger uses the current working directory as the project root. Build Badger in its own source checkout, then invoke that binary **from the repository you want to inspect**.

```bash
BADGER_DEV=/absolute/path/to/aibadger/bin/badger
cd /path/to/target-project
"$BADGER_DEV" code
```

Do not run the binary from the Badger source directory when your intention is to inspect PCS or another external checkout.

## Verify changes

Before treating a source build as trustworthy:

```bash
go test ./...
go vet ./...
go build ./...
```

Use focused package tests while developing and the broader checks above before publishing or merging a substantial cross-cutting change.

## P0 hardening branch

The `feat/p0-pcs-hardening` branch adds opt-in project policy, snapshot pinning, egress protection, guarded unified patches, exact post-apply review, and explicit verification.

User-facing references:

- [Installation & Source Builds](install.md)
- [Project Policy](project-policy.md)
- [PCS End-to-End Tutorial](pcs-tutorial.md)

## Release publishing

For release publishing and artifact details, see [releasing.md](releasing.md).
