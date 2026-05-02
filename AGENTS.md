# Repository Guidelines

## Project Structure & Module Organization

This repository contains `jmv`, a minimal OpenJDK manager backed by the TUNA Adoptium mirror. The Go module
is defined in `go.mod` and targets Go 1.22. The CLI entry point is `cmd/jmv/main.go`. Core implementation
code lives in `internal/jmv/`, including configuration, path handling, mirror access, archive handling,
installation, shims, and CLI command wiring. Tests are colocated with package code in `internal/jmv/` using
Go `_test.go` files, for example `install_test.go` and `mirror_test.go`.

## Build, Test, and Development Commands

- `go test ./...` runs the full Go test suite.
- `go run ./cmd/jmv list` runs the CLI locally and lists available JDK versions.
- `go run ./cmd/jmv install 17` exercises a local install flow for JDK 17.
- `go run ./cmd/jmv list --runtime jre` lists JRE versions instead of the default JDK runtime.
- `go build ./cmd/jmv` compiles the CLI binary for local verification.
- `gofmt -w <files>` formats edited Go files before committing.

Configuration can be overridden with environment variables:

```bash
export JMV_HOME="$HOME/.jmv"
export JMV_MIRROR="https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
export PATH="$HOME/.jmv/shims:$PATH"
```

## Coding Style & Naming Conventions

Use idiomatic Go and keep package APIs small. Format all Go changes with `gofmt`. Prefer clear, focused
functions and table-driven tests when adding cases. Keep unexported helpers lowercase unless they are part
of the CLI-facing behavior that must be shared across files. Name test files with the `_test.go` suffix and
test functions as `Test<Behavior>`.

## Testing Guidelines

Add or update tests for changes to install behavior, mirror parsing, path selection, archive handling, or
configuration defaults. Keep tests deterministic: use temporary directories, local fixtures, or stubbed
responses rather than relying on live mirror state when practical. Run `go test ./...` before submitting
changes.

## Commit & Pull Request Guidelines

Use concise, focused commits. Conventional prefixes such as `fix:`, `docs:`, `test:`, and `chore:` are
preferred when they fit the change. Pull requests should explain the user-visible behavior, note any config
or filesystem impact, and include the test command run. Keep PRs small enough to review in one pass.

## Security & Configuration Tips

Do not commit downloaded JDK/JRE archives, local `JMV_HOME` contents, credentials, or generated binaries.
Be explicit when changing default mirror URLs, archive extraction behavior, or shim generation because those
paths affect user machines directly.
