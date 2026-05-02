# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`okm` is a minimal OpenJDK manager backed by the TUNA Adoptium mirror. It installs JDK/JRE builds into a local directory, switches the default runtime, and exposes Java commands through shims. Written in Go with zero external dependencies (stdlib only).

## Build, Test, and Lint Commands

```bash
go build ./cmd/okm          # Build the CLI binary
go test ./...               # Run all tests
go run ./cmd/okm list       # Run CLI locally
gofmt -w cmd internal       # Format Go files (required before committing)
sh -n install.sh            # Validate installer script syntax
test -z "$(gofmt -l .)"     # CI formatting check
```

## Architecture

- **`cmd/okm/main.go`** — Entry point, delegates to `okm.Run()`
- **`internal/okm/`** — All core logic in a single package:
  - `cli.go` — Command routing and main orchestrator
  - `config.go` — Configuration from env vars (`OKM_HOME`, `OKM_MIRROR`)
  - `mirror.go` — Adoptium mirror HTML scraper for available versions
  - `install.go` — Download and install flow
  - `archive.go` — Archive extraction (tar.gz, zip)
  - `shim.go` — Symlink/command-file generation for Java binaries
  - `paths.go` — Platform-specific path resolution
  - `state.go` — Local state (installed versions, current default)
  - `types.go` — Shared type definitions
  - `errors.go` — Error formatting utilities

Tests are colocated with implementation as `_test.go` files.

## Conventions

- Go 1.22+, idiomatic Go, small package APIs
- Table-driven tests; keep tests deterministic (temp dirs, stubs, no live network)
- Conventional commit prefixes: `fix:`, `docs:`, `test:`, `chore:`
- Default runtime is JDK; use `--runtime jre` for JRE operations
- Mirror URL defaults to TUNA Adoptium; configurable via `OKM_MIRROR`
