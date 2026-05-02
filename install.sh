#!/bin/sh
set -eu

repo="${1:-${OKM_REPO:-fishandsheep/okm}}"
version="${OKM_VERSION:-latest}"
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

print_logo() {
	cat <<'LOGO'
          _____                    _____                    _____          
         /\    \                  /\    \                  /\    \         
        /::\    \                /::\____\                /::\____\        
        \:::\    \              /::::|   |               /:::/    /        
         \:::\    \            /:::::|   |              /:::/    /         
          \:::\    \          /::::::|   |             /:::/    /          
           \:::\    \        /:::/|::|   |            /:::/____/           
           /::::\    \      /:::/ |::|   |            |::|    |            
  _____   /::::::\    \    /:::/  |::|___|______      |::|    |     _____  
 /\    \ /:::/\:::\    \  /:::/   |::::::::\    \     |::|    |    /\    \ 
/::\    /:::/  \:::\____\/:::/    |:::::::::\____\    |::|    |   /::\____\
\:::\  /:::/    \::/    /\::/    / ~~~~~/:::/    /    |::|    |  /:::/    /
 \:::\/:::/    / \/____/  \/____/      /:::/    /     |::|    | /:::/    / 
  \::::::/    /                       /:::/    /      |::|____|/:::/    /  
   \::::/    /                       /:::/    /       |:::::::::::/    /   
    \::/    /                       /:::/    /        \::::::::::/____/    
     \/____/                       /:::/    /          ~~~~~~~~~~          
                                  /:::/    /                               
                                 /:::/    /                                
                                 \::/    /                                 
                                  \/____/                                  
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

	api_url="https://api.github.com/repos/$repo/releases/latest"
	if command -v curl >/dev/null 2>&1; then
		resolved="$(curl -sSL -H 'Accept: application/vnd.github+json' -H 'User-Agent: okm-installer' "$api_url" | sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
	elif command -v wget >/dev/null 2>&1; then
		resolved="$(wget -qO- --header='Accept: application/vnd.github+json' --user-agent='okm-installer' "$api_url" | sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
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
		curl -A "okm-installer" -fL# "$url" -o "$out"
	elif command -v wget >/dev/null 2>&1; then
		wget --progress=bar:force:noscroll -O "$out" "$url"
	else
		die "missing required command: curl or wget"
	fi
}

print_shell_examples() {
	printf '\nAdd the following to your shell profile manually:\n\n'
	printf 'For bash (append to ~/.bashrc):\n'
	cat <<EOF_BASH
export OKM_HOME="$okm_home"
export OKM_MIRROR="$mirror"
export PATH="$install_dir:\$OKM_HOME/shims:\$PATH"
EOF_BASH
	printf '\nFor zsh (append to ~/.zshrc):\n'
	cat <<EOF_ZSH
export OKM_HOME="$okm_home"
export OKM_MIRROR="$mirror"
export PATH="$install_dir:\$OKM_HOME/shims:\$PATH"
EOF_ZSH
	printf '\nFor fish (create config manually):\n'
	printf 'mkdir -p ~/.config/fish && cat <<\"EOF\" >> ~/.config/fish/config.fish\n'
	cat <<EOF_FISH
set -gx OKM_HOME "$okm_home"
set -gx OKM_MIRROR "$mirror"
fish_add_path "$install_dir" "$okm_home/shims"
EOF_FISH
	printf 'EOF\n'
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
archive="$tmp_dir/okm.tar.gz"
url="https://github.com/$repo/releases/download/$resolved_version/okm_${os}_${arch}.tar.gz"

printf 'Downloading %s\n' "$url"
download "$url" "$archive"

mkdir -p "$install_dir" "$okm_home"
tar -xzf "$archive" -C "$tmp_dir"
[ -f "$tmp_dir/okm" ] || die "release archive must contain an okm binary"
cp "$tmp_dir/okm" "$install_dir/okm"
chmod +x "$install_dir/okm"

printf '\nokm installed to %s/okm\n' "$install_dir"
print_shell_examples
