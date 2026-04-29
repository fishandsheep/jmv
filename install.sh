#!/bin/sh
set -eu

repo="${1:-${OKM_REPO:-fishandsheep/okm}}"
version="${OKM_VERSION:-0.0.2-beta}"
install_dir="${OKM_INSTALL_DIR:-$HOME/.local/bin}"
okm_home="${OKM_HOME:-$HOME/.okm}"
mirror="${OKM_MIRROR:-https://mirrors.tuna.tsinghua.edu.cn/Adoptium}"

die() {
	printf 'okm install: %s\n' "$*" >&2
	exit 1
}

need() {
	command -v "$1" >/dev/null 2>&1 || die "missing required command: $1"
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

download() {
	url="$1"
	out="$2"
	if command -v curl >/dev/null 2>&1; then
		curl -fsSL "$url" -o "$out"
	elif command -v wget >/dev/null 2>&1; then
		wget -qO "$out" "$url"
	else
		die "missing required command: curl or wget"
	fi
}

append_if_missing() {
	file="$1"
	text="$2"
	marker="$3"
	mkdir -p "$(dirname "$file")"
	touch "$file"
	if ! grep -q "$marker" "$file"; then
		printf '\n%s\n' "$text" >>"$file"
		printf 'Updated %s\n' "$file"
	fi
}

configure_shells() {
	[ "${OKM_NO_MODIFY_PROFILE:-}" = "1" ] && return 0

	sh_block="# okm
export OKM_HOME=\"$okm_home\"
export OKM_MIRROR=\"$mirror\"
export PATH=\"$install_dir:\$OKM_HOME/shims:\$PATH\"
# okm end"

	fish_block="# okm
set -gx OKM_HOME \"$okm_home\"
set -gx OKM_MIRROR \"$mirror\"
fish_add_path \"$install_dir\"
fish_add_path \"\$OKM_HOME/shims\"
# okm end"

	[ -n "${BASH_VERSION:-}" ] && append_if_missing "$HOME/.bashrc" "$sh_block" "# okm"
	[ -n "${ZSH_VERSION:-}" ] && append_if_missing "$HOME/.zshrc" "$sh_block" "# okm"
	[ -d "$HOME/.config/fish" ] && append_if_missing "$HOME/.config/fish/conf.d/okm.fish" "$fish_block" "# okm"
}

need uname
need tar
need mkdir
need chmod

os="$(detect_os)"
arch="$(detect_arch)"
tmp_dir="$(mktemp -d)"
archive="$tmp_dir/okm.tar.gz"
url="https://github.com/$repo/releases/download/$version/okm_${os}_${arch}.tar.gz"

printf 'Downloading %s\n' "$url"
download "$url" "$archive"

mkdir -p "$install_dir" "$okm_home"
tar -xzf "$archive" -C "$tmp_dir"
[ -f "$tmp_dir/okm" ] || die "release archive must contain an okm binary"
cp "$tmp_dir/okm" "$install_dir/okm"
chmod +x "$install_dir/okm"

configure_shells

printf 'okm installed to %s/okm\n' "$install_dir"
printf 'Restart your shell or run: export PATH="%s:%s/shims:$PATH"\n' "$install_dir" "$okm_home"
