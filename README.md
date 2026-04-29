# okm

`okm` is a minimal OpenJDK manager backed by the TUNA Adoptium mirror.

## Installation

### One-line shell installer

After publishing release archives, install the latest binary with:

```bash
curl -fsSL https://raw.githubusercontent.com/<owner>/<repo>/main/install.sh | sh -s -- <owner>/<repo>
```

The installer downloads the matching `okm_<os>_<arch>.tar.gz` release, installs `okm` to
`$HOME/.local/bin`, creates `$HOME/.okm`, and adds `okm` environment variables to Bash, Zsh, or Fish
configuration when possible.

Customize the install location or version:

```bash
OKM_VERSION=v0.1.0 OKM_INSTALL_DIR="$HOME/bin" sh install.sh <owner>/<repo>
```

Skip shell profile changes:

```bash
OKM_NO_MODIFY_PROFILE=1 sh install.sh <owner>/<repo>
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
git clone <repository-url>
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
```

Fish:

```fish
set -gx OKM_HOME "$HOME/.okm"
set -gx OKM_MIRROR "https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
fish_add_path "$HOME/.local/bin"
fish_add_path "$OKM_HOME/shims"
```

Verify the installation:

```bash
okm list
```

## Commands

```bash
okm list
okm list --runtime jre
okm install 17
okm install --runtime jre 17
okm default 17
okm current
okm home 17
okm uninstall 17
```

`jdk` is the default runtime. Use `--runtime jre` for JRE operations.

## Configuration

```bash
export OKM_HOME="$HOME/.okm"
export OKM_MIRROR="https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
```

After activating a version, add the shim directory to `PATH`:

```bash
export PATH="$HOME/.okm/shims:$PATH"
```

## Development

```bash
go test ./...
go run ./cmd/okm list
```
