# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`jmv` is a minimal OpenJDK version manager written in Go. It installs JDK/JRE builds from the TUNA Adoptium mirror into a local directory (`JMV_HOME`, default `~/.jmv`), manages version switching, and exposes activated Java commands through shims.

## Build & Development Commands

```bash
go build ./cmd/jmv          # Build the CLI binary
go test ./...               # Run all tests
go run ./cmd/jmv list       # Run CLI locally
gofmt -w cmd internal       # Format Go code
sh -n install.sh            # Validate installer script syntax
```

CI (GitHub Actions) checks formatting, validates `install.sh`, runs tests, and builds the binary on every push to `main` and on PRs.

## Architecture

Single Go module with all logic in `internal/jmv/` and the entry point at `cmd/jmv/main.go`.

- **cli.go** — Command routing and argument parsing. Dispatches to subcommands (`list`, `install`, `use`, `default`, etc.)
- **config.go** — Configuration loading, platform detection (OS/arch), environment variables (`JMV_HOME`, `JMV_MIRROR`)
- **mirror.go** — Scrapes the Adoptium mirror HTML to discover available JDK/JRE versions and resolve download URLs
- **install.go** — Downloads archives, extracts them, and tracks installation metadata
- **shim.go** — Generates wrapper executables in `$JMV_HOME/shims` that route to the active JDK's binaries
- **state.go** — Manages `session.json` (current session version) and the persistent default version
- **types.go** — Shared type definitions (Runtime type: JDK/JRE, version structs)
- **archive.go** — Archive extraction utilities
- **paths.go** — Path resolution helpers
- **errors.go** — Custom error formatting

## Key Design Concepts

- **Shim-based activation**: Instead of modifying PATH to point to a specific JDK, shims in `$JMV_HOME/shims` forward invocations to the active version. Users add `$JMV_HOME/shims` to PATH once.
- **Session vs default**: `jmv use <ver>` sets a session-only version (stored in `session.json`, cleared on new shell). `jmv default <ver>` sets the persistent default.
- **Mirror scraping**: The tool parses HTML from the TUNA Adoptium mirror rather than using the Adoptium API, keeping it simple with no API key requirement.
- **Cross-platform**: Supports Linux, macOS, and Windows with platform-specific archive formats and shim generation.

## Go Version

Requires Go 1.22+.
