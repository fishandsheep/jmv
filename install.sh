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

	resolved=""
	if command -v curl >/dev/null 2>&1; then
		resolved="$(
			curl -fsSL \
				-H "Accept: application/vnd.github+json" \
				-H "User-Agent: okm-install-script" \
				"https://api.github.com/repos/$repo/releases/latest" \
				| sed -n 's/^[[:space:]]*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
				| head -n1
		)" || true
		if [ -z "$resolved" ]; then
			resolved="$(
				curl -fsSI -o - \
					-H "User-Agent: okm-install-script" \
					"https://github.com/$repo/releases/latest" \
					| sed -n 's#^[Ll]ocation: .*/releases/tag/\([^/[:space:]]*\).*#\1#p' \
					| tail -n1
			)" || true
		fi
	elif command -v wget >/dev/null 2>&1; then
		resolved="$(
			wget -qO- \
				--header="Accept: application/vnd.github+json" \
				--header="User-Agent: okm-install-script" \
				"https://api.github.com/repos/$repo/releases/latest" \
				| sed -n 's/^[[:space:]]*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' \
				| head -n1
		)" || true
		if [ -z "$resolved" ]; then
			resolved="$(
				wget -S --spider -O /dev/null \
					--max-redirect=0 \
					--header="User-Agent: okm-install-script" \
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
# okm configuration
export OKM_HOME="'"$okm_home"'"
export OKM_MIRROR="'"$mirror"'"
export PATH="'"$install_dir"':$OKM_HOME/shims:$PATH"
rm -f "$OKM_HOME/session.json"'

	fish_block='
# okm configuration
set -gx OKM_HOME "'"$okm_home"'"
set -gx OKM_MIRROR "'"$mirror"'"
fish_add_path "'"$install_dir"'"
fish_add_path "'"$okm_home"'/shims"
rm -f "$OKM_HOME/session.json"'

	case "$shell_name" in
		bash)
			if [ -f "$HOME/.bashrc" ]; then
				if ! grep -q '# okm configuration' "$HOME/.bashrc" 2>/dev/null; then
					printf '%s\n' "$config_block" >> "$HOME/.bashrc"
					printf '\nokm environment added to ~/.bashrc\n'
					printf 'Restart your terminal or run: source ~/.bashrc\n'
				else
					printf '\nokm environment already configured in ~/.bashrc\n'
				fi
			else
				printf '\nNo ~/.bashrc found. Add the following to your shell profile:\n'
				printf '%s\n' "$config_block"
			fi
			;;
		zsh)
			if [ -f "$HOME/.zshrc" ]; then
				if ! grep -q '# okm configuration' "$HOME/.zshrc" 2>/dev/null; then
					printf '%s\n' "$config_block" >> "$HOME/.zshrc"
					printf '\nokm environment added to ~/.zshrc\n'
					printf 'Restart your terminal or run: source ~/.zshrc\n'
				else
					printf '\nokm environment already configured in ~/.zshrc\n'
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
				if ! grep -q '# okm configuration' "$fish_config" 2>/dev/null; then
					printf '%s\n' "$fish_block" >> "$fish_config"
					printf '\nokm environment added to ~/.config/fish/config.fish\n'
					printf 'Restart your terminal or restart fish\n'
				else
					printf '\nokm environment already configured in ~/.config/fish/config.fish\n'
				fi
			else
				printf '%s\n' "$fish_block" > "$fish_config"
				printf '\nokm environment added to ~/.config/fish/config.fish\n'
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
export OKM_HOME="$HOME/.okm"
export OKM_MIRROR="https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
export PATH="$HOME/.local/bin:$OKM_HOME/shims:$PATH"
rm -f "$OKM_HOME/session.json"

# Fish (append to ~/.config/fish/config.fish):
set -gx OKM_HOME "$HOME/.okm"
set -gx OKM_MIRROR "https://mirrors.tuna.tsinghua.edu.cn/Adoptium"
fish_add_path "$HOME/.local/bin"
fish_add_path "$OKM_HOME/shims"
rm -f "$OKM_HOME/session.json"
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
if [ "${OKM_NO_MODIFY_PROFILE:-0}" != "1" ]; then
	configure_shell
else
	print_shell_manual
fi
