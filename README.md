# jmv

`jmv` is a minimal OpenJDK and Apache Maven manager backed by China-friendly mirrors.

It installs JDK or JRE builds into a local `JMV_HOME`, switches the default runtime, and exposes activated
Java and Maven commands through shims.

## Installation

### One-line shell installer

After publishing release archives, install the latest binary with:

```bash
curl -fsSL https://raw.githubusercontent.com/fishandsheep/jmv/main/install.sh | sh
```

The shell installer supports Linux and macOS on amd64 and arm64. It downloads the matching
`jmv_<os>_<arch>.tar.gz` release, installs `jmv` to
`$HOME/.local/bin`, creates `$HOME/.jmv`, and automatically adds the required environment
variables to your shell profile (Bash, Zsh, or Fish).

Customize the install location or version:

```bash
JMV_VERSION=v0.1.0 JMV_INSTALL_DIR="$HOME/bin" sh install.sh fishandsheep/jmv
```

Skip shell profile changes:

```bash
JMV_NO_MODIFY_PROFILE=1 sh install.sh fishandsheep/jmv
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
git clone git@github.com:fishandsheep/jmv.git
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
export JMV_MAVEN_MIRROR="https://mirrors.aliyun.com/apache/maven"
export PATH="$HOME/.local/bin:$JMV_HOME/shims:$PATH"
rm -rf "$JMV_HOME/sessions"
```

Fish:

```fish
set -gx JMV_HOME "$HOME/.jmv"
set -gx JMV_MIRROR "https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
set -gx JMV_MAVEN_MIRROR "https://mirrors.aliyun.com/apache/maven"
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
  list      or ls             [-r|--runtime jdk|jre]
  install   or i              [-r|--runtime jdk|jre] <major>
  uninstall or rm             [-r|--runtime jdk|jre] <major>
  use       or u              [-r|--runtime jdk|jre] <major>
  default   or d              [-r|--runtime jdk|jre] <major>
  current   or c              [--home]
  env                         <print|java-home> [--shell bash|zsh|fish]
  maven                      <list|install|uninstall|use|default|current|config>
  version   or v
  help      or h
```

`jdk` is the default runtime. Use `--runtime jre` for JRE operations.
`--runtime` accepts only `jdk` or `jre`.

### `use` vs `default`

- `jmv default <major>` — sets the persistent default. All new terminal sessions will use this version.
- `jmv use <major>` — switches the active version for the current session only. Opening a new terminal will still use the previous default.

When installing another Java version after a default already exists, non-interactive input keeps the
existing default. Set `JMV_SET_DEFAULT=1` or answer `y` to switch during install.

After `jmv install`, `jmv default`, `jmv maven install`, or `jmv maven default`, `jmv` updates your
detected shell profile so `$JMV_HOME/shims` is on `PATH`. When a default JDK is active and `JAVA_HOME`
is not already set in the current environment, `jmv` also writes `JAVA_HOME` to that default JDK home.
Reload the profile or open a new terminal before running `java`, `javac`, or `mvn` directly:

```bash
source ~/.bashrc   # bash
source ~/.zshrc    # zsh
```

```fish
source ~/.config/fish/config.fish
```

If your shell cannot be detected or profile update is disabled with `JMV_NO_MODIFY_PROFILE=1`, `jmv`
prints the bash, zsh, and fish configuration blocks to add manually.

### Diagnosing the current environment

```bash
jmv current                 # show the active runtime, its home, metadata, and JAVA_HOME status
jmv current --home          # print just the home path of the active runtime (for scripts)
jmv env print               # print the bash/zsh/fish profile block(s) jmv would write
jmv env print --shell bash  # print the bash-only block
jmv env print bash          # same as above
jmv env java-home           # print just the default JDK home, or nothing if there is no JDK default
```

`jmv current` reports whether `JAVA_HOME` matches the active JDK. When it does not, reload the
profile (`source ~/.bashrc`) or use `jmv env print` to grab the correct block for the current
default.

### Maven

```bash
jmv maven list
jmv maven install latest
jmv maven default 3.9.11
jmv maven current
jmv maven config
```

Maven installs use the Aliyun Apache Maven mirror by default:
`https://mirrors.aliyun.com/apache/maven`. `jmv maven config` writes
`$JMV_HOME/config/maven/settings.xml` with the Aliyun public repository mirror:
`https://maven.aliyun.com/repository/public`.

Java and Maven shims share `$JMV_HOME/shims`, so `java`, `javac`, `mvn`, and `mvnDebug` can coexist.

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
