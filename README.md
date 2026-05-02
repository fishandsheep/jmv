# okm

`okm` is a minimal OpenJDK manager backed by the TUNA Adoptium mirror.

It installs JDK or JRE builds into a local `OKM_HOME`, switches the default runtime, and exposes activated
Java commands through shims.

## Installation

### One-line shell installer

After publishing release archives, install the latest binary with:

```bash
curl -fsSL https://raw.githubusercontent.com/fishandsheep/okm/main/install.sh | sh 
```

The installer downloads the matching `okm_<os>_<arch>.tar.gz` release, installs `okm` to
`$HOME/.local/bin`, creates `$HOME/.okm`, and automatically adds the required environment
variables to your shell profile (Bash, Zsh, or Fish).

Customize the install location or version:

```bash
OKM_VERSION=v0.1.0 OKM_INSTALL_DIR="$HOME/bin" sh install.sh fishandsheep/okm
```

Skip shell profile changes:

```bash
OKM_NO_MODIFY_PROFILE=1 sh install.sh fishandsheep/okm
```

### Manual binary install

Download the release archive for your platform, then install the binary:

```bash
tar -xzf okm_linux_amd64.tar.gz
mkdir -p "$HOME/.local/bin"
mv okm "$HOME/.local/bin/okm"
chmod +x "$HOME/.local/bin/okm"
```

### Install from source

Install from source with Go 1.22 or newer:

```bash
git clone git@github.com:fishandsheep/okm.git
cd okm
go install ./cmd/okm
```

Make sure Go's binary directory is on your `PATH`:

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
```

### Shell environment

`okm` uses `$OKM_HOME` for installed runtimes and `$OKM_HOME/shims` for activated Java commands.

Bash or Zsh:

```bash
export OKM_HOME="$HOME/.okm"
export OKM_MIRROR="https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
export PATH="$HOME/.local/bin:$OKM_HOME/shims:$PATH"
rm -f "$OKM_HOME/session.json"
```

Fish:

```fish
set -gx OKM_HOME "$HOME/.okm"
set -gx OKM_MIRROR "https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
fish_add_path "$HOME/.local/bin"
fish_add_path "$OKM_HOME/shims"
rm -f "$OKM_HOME/session.json"
```

Verify the installation:

```bash
okm list
```

## Commands

```bash
  list      or ls             [-r|--runtime [jdk]]
  install   or i              [-r|--runtime [jdk]] <major>
  uninstall or rm             [-r|--runtime [jdk]] <major>
  use       or u              [-r|--runtime [jdk]] <major>
  default   or d              [-r|--runtime [jdk]] <major>
  current   or c
  home      or h              [-r|--runtime [jdk]] <major>
  version   or v
  help
```

`jdk` is the default runtime. Use `--runtime jre` for JRE operations.

### `use` vs `default`

- `okm default <major>` — sets the persistent default. All new terminal sessions will use this version.
- `okm use <major>` — switches the active version for the current session only. Opening a new terminal will still use the previous default.


## Development

```bash
go test ./...
go build ./cmd/okm
go run ./cmd/okm list
sh -n install.sh
```

Format Go files before committing:

```bash
gofmt -w cmd internal
```

## Continuous Integration

GitHub Actions runs on every push to `main` and on pull requests. The workflow checks Go formatting,
validates `install.sh` syntax, runs `go test ./...`, and builds the CLI with `go build ./cmd/okm`.
