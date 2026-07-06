package jmv

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const profileMarker = "# jmv configuration"

func configureShellEnvironment(cfg Config, out io.Writer) {
	if os.Getenv("JMV_NO_MODIFY_PROFILE") == "1" {
		printManualShellConfig(cfg, out)
		return
	}

	profile, shellName, ok := detectShellProfile()
	if !ok {
		fmt.Fprintf(out, "\nCould not detect shell profile for %s.\n", shellName)
		printManualShellConfig(cfg, out)
		return
	}

	block := shellConfigBlock(cfg, shellName, profileJavaHome(cfg))
	if err := ensureProfileBlock(profile, block); err != nil {
		fmt.Fprintf(out, "\nCould not update %s: %v\n", profile, err)
		printManualShellConfig(cfg, out)
		return
	}

	fmt.Fprintf(out, "\njmv environment configured in %s\n", profile)
	fmt.Fprintln(out, "Reload your shell environment or open a new terminal before running shim commands.")
	switch shellName {
	case "fish":
		fmt.Fprintln(out, "For current fish session: source ~/.config/fish/config.fish")
	default:
		fmt.Fprintf(out, "For current %s session: source %s\n", shellName, profile)
	}
}

func detectShellProfile() (string, string, bool) {
	shellName := filepath.Base(os.Getenv("SHELL"))
	shellName = strings.TrimSuffix(shellName, ".exe")
	if shellName == "" || shellName == "." || shellName == string(filepath.Separator) {
		shellName = "unknown"
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", shellName, false
	}
	switch shellName {
	case "bash":
		return filepath.Join(home, ".bashrc"), shellName, true
	case "zsh":
		return filepath.Join(home, ".zshrc"), shellName, true
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish"), shellName, true
	default:
		return "", shellName, false
	}
}

func ensureProfileBlock(path, block string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	hasMarker := strings.Contains(string(data), profileMarker)
	hasShims := strings.Contains(string(data), "$JMV_HOME/shims")
	needsJavaHome := strings.Contains(block, "JAVA_HOME")
	hasJavaHome := strings.Contains(string(data), "JAVA_HOME")
	if hasMarker && hasShims && (!needsJavaHome || hasJavaHome) {
		return nil
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if len(data) > 0 && !strings.HasSuffix(string(data), "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	_, err = f.WriteString("\n" + block)
	return err
}

func shellConfigBlock(cfg Config, shellName string, javaHome string) string {
	if shellName == "fish" {
		javaHomeLine := ""
		if javaHome != "" {
			javaHomeLine = fmt.Sprintf("set -gx JAVA_HOME \"%s\"\n", javaHome)
		}
		return fmt.Sprintf(`# jmv configuration
set -gx JMV_HOME "%s"
set -gx JMV_MIRROR "%s"
set -gx JMV_MAVEN_MIRROR "%s"
%sfish_add_path "$JMV_HOME/shims"
rm -rf "$JMV_HOME/sessions"
`, cfg.Home, cfg.Mirror, cfg.MavenMirror, javaHomeLine)
	}
	javaHomeLine := ""
	if javaHome != "" {
		javaHomeLine = fmt.Sprintf("export JAVA_HOME=\"%s\"\n", javaHome)
	}
	return fmt.Sprintf(`# jmv configuration
export JMV_HOME="%s"
export JMV_MIRROR="%s"
export JMV_MAVEN_MIRROR="%s"
%sexport PATH="$JMV_HOME/shims:$PATH"
rm -rf "$JMV_HOME/sessions"
`, cfg.Home, cfg.Mirror, cfg.MavenMirror, javaHomeLine)
}

func printManualShellConfig(cfg Config, out io.Writer) {
	javaHome := profileJavaHome(cfg)
	fmt.Fprintln(out, "\nAdd the following to your shell profile, then reload it or open a new terminal:")
	fmt.Fprintln(out, "\n# Bash (~/.bashrc)")
	fmt.Fprint(out, shellConfigBlock(cfg, "bash", javaHome))
	fmt.Fprintln(out, "\n# Zsh (~/.zshrc)")
	fmt.Fprint(out, shellConfigBlock(cfg, "zsh", javaHome))
	fmt.Fprintln(out, "\n# Fish (~/.config/fish/config.fish)")
	fmt.Fprint(out, shellConfigBlock(cfg, "fish", javaHome))
}

func profileJavaHome(cfg Config) string {
	if os.Getenv("JAVA_HOME") != "" {
		return ""
	}
	cur, err := readCurrent(cfg.Home)
	if err != nil || cur.Runtime != RuntimeJDK {
		return ""
	}
	return cur.Home
}
