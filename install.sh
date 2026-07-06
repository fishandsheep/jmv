#!/bin/sh
set -eu

repo="${1:-${JMV_REPO:-fishandsheep/jmv}}"
version="${JMV_VERSION:-latest}"
install_dir="${JMV_INSTALL_DIR:-$HOME/.local/bin}"
jmv_home="${JMV_HOME:-$HOME/.jmv}"
mirror="${JMV_MIRROR:-https://mirrors.tuna.tsinghua.edu.cn/Adoptium}"
maven_mirror="${JMV_MAVEN_MIRROR:-https://mirrors.aliyun.com/apache/maven}"

die() {
	printf 'jmv install: %s\n' "$*" >&2
	exit 1
}

need() {
	command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
}

print_logo() {
	cat <<'LOGO'
    ___  _____ ______   ___      ___
   |\  \|\   _ \  _   \|\  \    /  /|
   \ \  \ \  \\\__\ \  \ \  \  /  / /
 __ \ \  \ \  \\|__| \  \ \  \/  / /
|\  \\_\  \ \  \    \ \  \ \    / /
\ \________\ \__\    \ \__\ \__/ /
 \|________|\|__|     \|__|\|__|/    (JDK/JRE MANAGE VERSION)
LOGO
}

detect_os() {
	case "$(uname -s)" in
		Linux) printf linux ;;
		Darwin) printf darwin ;;
		*) die "unsupported OS: $(uname -s)" ;;
	esac
}

detect_arch() {
	case "$(uname -m)" in
		x86_64 | amd64) printf amd64 ;;
		arm64 | aarch64) printf arm64 ;;
		*) die "unsupported architecture: $(uname -m)" ;;
	esac
}

resolve_version() {
	if [ "$version" != "latest" ]; then
		printf '%s' "$version"
		return 0
	fi

	resolved=""
	if command -v curl >/dev/null 2>&1; then
		resolved="$(
			curl -fsSL \
				-H "Accept: application/vnd.github+json" \
				-H "User-Agent: jmv-install-script" \
				"https://api.github.com/repos/$repo/releases/latest" \
				| sed -n 's/^[[:space:]]*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
				| head -n1
		)" || true
		if [ -z "$resolved" ]; then
			resolved="$(
				curl -fsSI -o - \
					-H "User-Agent: jmv-install-script" \
					"https://github.com/$repo/releases/latest" \
					| sed -n 's#^[Ll]ocation: .*/releases/tag/\([^/[:space:]]*\).*#\1#p' \
					| tail -n1
			)" || true
		fi
	elif command -v wget >/dev/null 2>&1; then
		resolved="$(
			wget -qO- \
				--header="Accept: application/vnd.github+json" \
				--header="User-Agent: jmv-install-script" \
				"https://api.github.com/repos/$repo/releases/latest" \
				| sed -n 's/^[[:space:]]*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
				| head -n1
		)" || true
		if [ -z "$resolved" ]; then
			resolved="$(
				wget -S --spider -O /dev/null \
					--max-redirect=0 \
					--header="User-Agent: jmv-install-script" \
					"https://github.com/$repo/releases/latest" 2>&1 \
					| sed -n 's#^  Location: .*/releases/tag/\([^/[:space:]]*\).*#\1#p' \
					| tail -n1
			)" || true
		fi
	else
		die "missing required command: curl or wget"
	fi

	[ -n "$resolved" ] || die "failed to resolve latest release version"
	printf '%s' "$resolved"
}

download() {
	url="$1"
	out="$2"
	if command -v curl >/dev/null 2>&1; then
		curl -fL# "$url" -o "$out"
	elif command -v wget >/dev/null 2>&1; then
		wget --progress=bar:force:noscroll -O "$out" "$url"
	else
		die "missing required command: curl or wget"
	fi
}

configure_shell() {
	shell_name="$(basename "${SHELL:-}")"

	config_block='
# jmv configuration
export JMV_HOME="'"$jmv_home"'"
export JMV_MIRROR="'"$mirror"'"
export JMV_MAVEN_MIRROR="'"$maven_mirror"'"
export PATH="'"$install_dir"':$JMV_HOME/shims:$PATH"
rm -rf "$JMV_HOME/sessions"'

	fish_block='
# jmv configuration
set -gx JMV_HOME "'"$jmv_home"'"
set -gx JMV_MIRROR "'"$mirror"'"
set -gx JMV_MAVEN_MIRROR "'"$maven_mirror"'"
fish_add_path "'"$install_dir"'"
fish_add_path "'"$jmv_home"'/shims"
rm -rf "$JMV_HOME/sessions"'

	case "$shell_name" in
		bash)
			if [ -f "$HOME/.bashrc" ]; then
				if ! grep -q '# jmv configuration' "$HOME/.bashrc" 2>/dev/null; then
					printf '%s\n' "$config_block" >> "$HOME/.bashrc"
					printf '\njmv environment added to ~/.bashrc\n'
					printf 'Restart your terminal or run: source ~/.bashrc\n'
				else
					printf '\njmv environment already configured in ~/.bashrc\n'
				fi
			else
				printf '\nNo ~/.bashrc found. Add the following to your shell profile:\n'
				printf '%s\n' "$config_block"
			fi
			;;
		zsh)
			if [ -f "$HOME/.zshrc" ]; then
				if ! grep -q '# jmv configuration' "$HOME/.zshrc" 2>/dev/null; then
					printf '%s\n' "$config_block" >> "$HOME/.zshrc"
					printf '\njmv environment added to ~/.zshrc\n'
					printf 'Restart your terminal or run: source ~/.zshrc\n'
				else
					printf '\njmv environment already configured in ~/.zshrc\n'
				fi
			else
				printf '\nNo ~/.zshrc found. Add the following to your shell profile:\n'
				printf '%s\n' "$config_block"
			fi
			;;
		fish)
			fish_config="$HOME/.config/fish/config.fish"
			mkdir -p "$(dirname "$fish_config")"
			if [ -f "$fish_config" ]; then
				if ! grep -q '# jmv configuration' "$fish_config" 2>/dev/null; then
					printf '%s\n' "$fish_block" >> "$fish_config"
					printf '\njmv environment added to ~/.config/fish/config.fish\n'
					printf 'Restart your terminal or restart fish\n'
				else
					printf '\njmv environment already configured in ~/.config/fish/config.fish\n'
				fi
			else
				printf '%s\n' "$fish_block" > "$fish_config"
				printf '\njmv environment added to ~/.config/fish/config.fish\n'
				printf 'Restart your terminal or restart fish\n'
			fi
			;;
		*)
			printf '\nCould not detect shell (%s). Add the following to your shell profile:\n\n' "$shell_name"
			printf '%s\n' "$config_block"
			printf '\nFor fish:\n'
			printf '%s\n' "$fish_block"
			;;
	esac
}

print_shell_manual() {
	printf '\nAdd the following to your shell profile manually:\n\n'
	cat <<'EOF'
# Bash / Zsh (append to ~/.bashrc or ~/.zshrc):
export JMV_HOME="$HOME/.jmv"
export JMV_MIRROR="https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
export JMV_MAVEN_MIRROR="https://mirrors.aliyun.com/apache/maven"
export PATH="$HOME/.local/bin:$JMV_HOME/shims:$PATH"
rm -rf "$JMV_HOME/sessions"

# Fish (append to ~/.config/fish/config.fish):
set -gx JMV_HOME "$HOME/.jmv"
set -gx JMV_MIRROR "https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
set -gx JMV_MAVEN_MIRROR "https://mirrors.aliyun.com/apache/maven"
fish_add_path "$HOME/.local/bin"
fish_add_path "$JMV_HOME/shims"
rm -rf "$JMV_HOME/sessions"
EOF
}

need uname
need tar
need mkdir
need chmod

print_logo

os="$(detect_os)"
arch="$(detect_arch)"
resolved_version="$(resolve_version)"
tmp_dir="$(mktemp -d)"
archive="$tmp_dir/jmv.tar.gz"
url="https://github.com/$repo/releases/download/$resolved_version/jmv_${os}_${arch}.tar.gz"

printf 'Downloading %s\n' "$url"
download "$url" "$archive"

mkdir -p "$install_dir" "$jmv_home"
tar -xzf "$archive" -C "$tmp_dir"
[ -f "$tmp_dir/jmv" ] || die "release archive must contain an jmv binary"
cp "$tmp_dir/jmv" "$install_dir/jmv"
chmod +x "$install_dir/jmv"

printf '\njmv installed to %s/jmv\n' "$install_dir"
if [ "${JMV_NO_MODIFY_PROFILE:-0}" != "1" ]; then
	configure_shell
else
	print_shell_manual
fi
