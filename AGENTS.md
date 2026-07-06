# Repository Guidelines

## Project Structure & Module Organization

`jmv` is a Go 1.22 command-line tool for managing OpenJDK/JRE installs through a local `JMV_HOME`.

- `cmd/jmv/`: CLI entry point. Keep command parsing and user-facing wiring here.
- `internal/`: private packages for install, mirror, shim, version, and filesystem logic.
- `install.sh`: POSIX shell installer used by the one-line install path.
- `bucket/jmv.json`: Scoop manifest updated during releases.
- `.github/workflows/ci.yml`: formatting, shell syntax, tests, and build checks.
- `.github/workflows/release.yml`: tagged-release build and manifest update workflow.

Place Go tests beside the package they cover using `*_test.go`.

## Build, Test, and Development Commands

Run these from the repository root:

```bash
go test ./...          # run all Go tests
go build ./cmd/jmv     # build the CLI
go run ./cmd/jmv list  # run a local command during development
sh -n install.sh       # validate installer shell syntax
gofmt -w cmd internal  # format Go source before committing
```

CI also checks `test -z "$(gofmt -l .)"`, so keep all Go files formatted.

## Coding Style & Naming Conventions

Use standard Go formatting and idioms. Prefer small package-level functions with clear error returns over hidden global state. Keep exported names only for APIs that must cross package boundaries; otherwise use unexported `camelCase` identifiers. Use concise command names and aliases that match README examples, such as `install/i`, `default/d`, and `current/c`.

Shell scripts should stay POSIX-compatible unless the shebang changes. Quote variables in `install.sh`, and validate edits with `sh -n install.sh`.

## Testing Guidelines

Use Go's built-in `testing` package. Name tests `TestThingDoesBehavior`, and prefer table-driven tests for command parsing, version selection, path handling, and mirror URL construction. Avoid tests that write to real `JMV_HOME`; use temporary directories from `t.TempDir()`.

Before opening a PR, run `go test ./...`, `go build ./cmd/jmv`, and `sh -n install.sh`.

## Commit & Pull Request Guidelines

Recent history uses short, imperative subjects with prefixes such as `fix:`, `chore:`, and `docs:`. Follow that style, for example `fix: handle missing default runtime`.

PRs should include a clear summary, test results, and any installer or release impact. Link related issues when available. Include screenshots or terminal output only when CLI behavior or install output changes.

## Security & Configuration Tips

Do not commit downloaded JDK/JRE archives, local runtime directories, or secrets. Treat `JMV_HOME`, `JMV_MIRROR`, and shell profile edits as user-controlled configuration.
