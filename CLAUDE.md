# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`jmv` is a minimal OpenJDK version manager written in Go. It installs JDK/JRE builds from the TUNA Adoptium mirror into a local directory (`JMV_HOME`, default `~/.jmv`), manages version switching, and exposes activated Java commands through shims.

## Build & Development Commands

```bash
go build ./cmd/jmv                    # Build the CLI binary
go test ./...                         # Run all tests
go test ./internal/jmv/ -run TestX    # Run a single test
go run ./cmd/jmv list                 # Run CLI locally
gofmt -w cmd internal                 # Format Go code
sh -n install.sh                      # Validate installer script syntax
```

CI (GitHub Actions) checks formatting, validates `install.sh`, runs tests, and builds the binary on every push to `main` and on PRs. Release workflow cross-compiles for Linux/macOS/Windows and publishes to GitHub Releases + Scoop bucket.

## Architecture

Single Go module (`jmv`). Entry point at `cmd/jmv/main.go` calls `jmv.Run()`, which dispatches via a `switch` on the command name (with aliases like `ls`/`i`/`rm`/`u`/`d`). All business logic lives in `internal/jmv/`.

**Command flow**: `cli.go` routes → loads `Config` (platform detection, env vars) → calls domain functions in `install.go`, `state.go`, `shim.go`, or `mirror.go`.

**`--runtime` flag**: Every version-aware command (`list`, `install`, `uninstall`, `use`, `default`) accepts `--runtime jdk|jre`. Parsed by `parseRuntime()` which strips the flag and returns the remaining args.

**Mirror scraping** (`mirror.go`): `MirrorClient` crawls the TUNA Adoptium HTML index. `Majors()` extracts version directories from the root, `Resolve()` navigates `/<major>/<runtime>/<arch>/<os>/` and picks the latest archive by filename sort. No API key needed.

**Shim mechanism** (`shim.go`): `refreshShims()` generates wrapper scripts in `$JMV_HOME/shims/` (shell scripts on Unix, `.cmd` on Windows) that invoke `jmv shim <executable>`, which resolves the active version via parent PID session tracking.

**State management** (`state.go`): Two layers — `session.json` (keyed by parent PID, for `jmv use`) and a persistent default file. `resolveCurrent()` checks session first, then falls back to default.

## Key Design Concepts

- **Shim-based activation**: Instead of modifying PATH to point to a specific JDK, shims in `$JMV_HOME/shims` forward invocations to the active version. Users add `$JMV_HOME/shims` to PATH once.
- **Session vs default**: `jmv use <ver>` sets a session-only version (stored in `session.json`, cleared on new shell). `jmv default <ver>` sets the persistent default.
- **Mirror scraping**: The tool parses HTML from the TUNA Adoptium mirror rather than using the Adoptium API, keeping it simple with no API key requirement.
- **Cross-platform**: Supports Linux, macOS, and Windows with platform-specific archive formats and shim generation.

## Go Version

Requires Go 1.22+.
