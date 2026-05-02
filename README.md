# jmv

`jmv` is a minimal OpenJDK manager backed by the TUNA Adoptium mirror.

It installs JDK or JRE builds into a local `JMV_HOME`, switches the default runtime, and exposes activated
Java commands through shims.

## Installation

### One-line shell installer

After publishing release archives, install the latest binary with:

```bash
curl -fsSL https://raw.githubusercontent.com/fishandsheep/jmv/main/install.sh | sh
```

The installer downloads the matching `jmv_<os>_<arch>.tar.gz` release, installs `jmv` to
`$HOME/.local/bin`, creates `$HOME/.jmv`, and automatically adds the required environment
variables to your shell profile (Bash, Zsh, or Fish).

Customize the install location or version:

```bash
JMV_VERSION=v0.1.0 JMV_INSTALL_DIR="$HOME/bin" sh install.sh fishandsheep/okm
```

Skip shell profile changes:

```bash
JMV_NO_MODIFY_PROFILE=1 sh install.sh fishandsheep/okm
```

### Manual binary install

Download the release archive for your platform, then install the binary:

```bash
tar -xzf jmv_linux_amd64.tar.gz
mkdir -p "$HOME/.local/bin"
mv jmv "$HOME/.local/bin/jmv"
chmod +x "$HOME/.local/bin/jmv"
```

### Install from source

Install from source with Go 1.22 or newer:

```bash
git clone git@github.com:fishandsheep/okm.git
cd okm
go install ./cmd/jmv
```

Make sure Go's binary directory is on your `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

### Shell environment

`jmv` uses `$JMV_HOME` for installed runtimes and `$JMV_HOME/shims` for activated Java commands.

Bash or Zsh:

```bash
export JMV_HOME="$HOME/.jmv"
export JMV_MIRROR="https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
export PATH="$HOME/.local/bin:$JMV_HOME/shims:$PATH"
rm -rf "$JMV_HOME/sessions"
```

Fish:

```fish
set -gx JMV_HOME "$HOME/.jmv"
set -gx JMV_MIRROR "https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
fish_add_path "$HOME/.local/bin"
fish_add_path "$JMV_HOME/shims"
rm -rf "$JMV_HOME/sessions"
```

Verify the installation:

```bash
jmv list
```

## Commands

```bash
  list      or ls             [-r|--runtime [jdk]]
  install   or i              [-r|--runtime [jdk]] <major>
  uninstall or rm             [-r|--runtime [jdk]] <major>
  use       or u              [-r|--runtime [jdk]] <major>
  default   or d              [-r|--runtime [jdk]] <major>
  current   or c
  version   or v
  help      or h
```

`jdk` is the default runtime. Use `--runtime jre` for JRE operations.

### `use` vs `default`

- `jmv default <major>` — sets the persistent default. All new terminal sessions will use this version.
- `jmv use <major>` — switches the active version for the current session only. Opening a new terminal will still use the previous default.


## Development

```bash
go test ./...
go build ./cmd/jmv
go run ./cmd/jmv list
sh -n install.sh
```

Format Go files before committing:

```bash
gofmt -w cmd internal
```

## Continuous Integration

GitHub Actions runs on every push to `main` and on pull requests. The workflow checks Go formatting,
validates `install.sh` syntax, runs `go test ./...`, and builds the CLI with `go build ./cmd/jmv`.
