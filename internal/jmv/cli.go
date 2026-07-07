package jmv

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

func Run(ctx context.Context, args []string, out, errOut io.Writer) error {
	if len(args) == 0 {
		printHelp(out)
		return nil
	}

	cfg, err := LoadConfig()
	if err != nil {
		return err
	}

	cmd := args[0]
	args = args[1:]

	switch cmd {
	case "help", "h", "--help", "-h":
		printHelp(out)
		return nil
	case "version", "--version", "-v":
		printLogo(out)
		fmt.Fprintf(out, "\njmv %s\n", Version)
		return nil
	case "list", "ls":
		rt, rest, err := parseRuntime(args)
		if err != nil {
			return err
		}
		if len(rest) != 0 {
			return usage("jmv list [--runtime jdk|jre]")
		}
		return list(ctx, cfg, rt, out)
	case "install", "i":
		rt, rest, err := parseRuntime(args)
		if err != nil {
			return err
		}
		if len(rest) != 1 {
			return usage("jmv install [--runtime jdk|jre] <major>")
		}
		return install(ctx, cfg, rt, rest[0], out)
	case "uninstall", "rm":
		rt, rest, err := parseRuntime(args)
		if err != nil {
			return err
		}
		if len(rest) != 1 {
			return usage("jmv uninstall [--runtime jdk|jre] <major>")
		}
		return uninstall(cfg, rt, rest[0], out)
	case "default", "d":
		rt, rest, err := parseRuntime(args)
		if err != nil {
			return err
		}
		if len(rest) != 1 {
			return usage("jmv " + cmd + " [--runtime jdk|jre] <major>")
		}
		return activateDefault(cfg, rt, rest[0], out)
	case "use", "u":
		rt, rest, err := parseRuntime(args)
		if err != nil {
			return err
		}
		if len(rest) != 1 {
			return usage("jmv " + cmd + " [--runtime jdk|jre] <major>")
		}
		return activateUse(cfg, rt, rest[0], out)
	case "current", "c":
		homeOnly := false
		for _, a := range args {
			if a == "--home" {
				homeOnly = true
				continue
			}
			return usage("jmv current [--home]")
		}
		return showCurrent(cfg, out, homeOnly)
	case "env":
		return envCommand(cfg, args, out)
	case "maven", "mvn":
		return runMaven(ctx, cfg, args, out)
	case "shim":
		if len(args) < 1 {
			return usage("jmv shim <executable> [args...]")
		}
		return runShim(cfg.Home, args[0], args[1:])
	default:
		fmt.Fprintf(errOut, "Unknown command: %s\n\n", cmd)
		printHelp(errOut)
		return errf("unknown command")
	}
}

func parseRuntime(args []string) (Runtime, []string, error) {
	rt := RuntimeJDK
	var rest []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--runtime", "-r":
			if i+1 >= len(args) || args[i+1] == "" || args[i+1][0] == '-' {
				rt = RuntimeJDK
				continue
			}
			parsed, err := normalizeRuntime(args[i+1])
			if err != nil {
				return "", nil, err
			}
			rt = parsed
			i++
		default:
			rest = append(rest, args[i])
		}
	}
	return rt, rest, nil
}

func normalizeRuntime(s string) (Runtime, error) {
	switch s {
	case "jdk":
		return RuntimeJDK, nil
	case "jre":
		return RuntimeJRE, nil
	default:
		return "", errf("runtime must be jdk or jre")
	}
}

func list(ctx context.Context, cfg Config, rt Runtime, out io.Writer) error {
	platform, err := DetectPlatform()
	if err != nil {
		return err
	}
	releases, err := NewMirrorClient(cfg.Mirror).List(ctx, rt, platform)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "Available %s versions from %s\n", rt, cfg.Mirror)
	fmt.Fprintf(out, "Platform: %s/%s\n", platform.Arch, platform.OS)
	for _, release := range releases {
		installed := ""
		if _, err := readMetadata(cfg.Home, rt, release.Major); err == nil {
			installed = "\t(installed)"
		}
		fmt.Fprintf(out, "%s\t%s%s\n", release.Major, release.FileName, installed)
	}
	return nil
}

func showCurrent(cfg Config, out io.Writer, homeOnly bool) error {
	pid := os.Getppid()
	cur, err := resolveCurrent(cfg.Home, pid)
	if err != nil {
		if os.IsNotExist(err) {
			if homeOnly {
				return errf("no active runtime")
			}
			fmt.Fprintln(out, "No active Java version.")
			return nil
		}
		return err
	}
	if homeOnly {
		fmt.Fprintln(out, cur.Home)
		return nil
	}
	if currentFromSession(cfg.Home, pid) {
		fmt.Fprintf(out, "%s %s (session)\n", cur.Runtime, cur.Major)
	} else {
		fmt.Fprintf(out, "%s %s (default)\n", cur.Runtime, cur.Major)
	}
	fmt.Fprintf(out, "Home: %s\n", cur.Home)
	meta, err := readMetadata(cfg.Home, cur.Runtime, cur.Major)
	if err == nil {
		fmt.Fprintf(out, "Download URL: %s\n", meta.URL)
	}
	if cur.Runtime == RuntimeJDK {
		reportJavaHomeStatus(cur.Home, out)
	}
	return nil
}

func reportJavaHomeStatus(jdkHome string, out io.Writer) {
	actual := os.Getenv("JAVA_HOME")
	switch {
	case actual == "":
		fmt.Fprintln(out, "Hint: JAVA_HOME is not set in this shell. Run `source ~/.bashrc` or `jmv env print` to enable it for tools that need it.")
	case samePath(actual, jdkHome):
		// already aligned, stay quiet
	default:
		fmt.Fprintf(out, "Warning: JAVA_HOME=%s does not match this runtime's home (%s). Reload your shell profile or run `jmv env print`.\n", actual, jdkHome)
	}
}

// samePath returns true when a and b refer to the same directory on disk,
// resolving symlinks so platform differences (notably macOS /var vs
// /private/var) do not produce false mismatches.
//
// Fallback constraint: when a (or b) does not exist or has a broken symlink,
// EvalSymlinks fails and we treat the unresolved path as its literal value
// for the comparison. This means two non-existent paths that happen to be
// lexically different will correctly return false, and two non-existent paths
// that share their lexical form will be considered equal — callers that want
// "must exist" semantics should pre-validate both paths.
func samePath(a, b string) bool {
	if filepath.Clean(a) == filepath.Clean(b) {
		return true
	}
	aReal, errA := filepath.EvalSymlinks(a)
	if errA != nil {
		aReal = a
	}
	bReal, errB := filepath.EvalSymlinks(b)
	if errB != nil {
		bReal = b
	}
	return filepath.Clean(aReal) == filepath.Clean(bReal)
}

func envCommand(cfg Config, args []string, out io.Writer) error {
	if len(args) == 0 {
		printManualShellConfig(cfg, out)
		return nil
	}
	switch args[0] {
	case "print":
		return envPrint(cfg, args[1:], out)
	case "java-home":
		return envJavaHome(cfg, out)
	default:
		return usage("jmv env <print|java-home> [...]")
	}
}

func envJavaHome(cfg Config, out io.Writer) error {
	cur, err := readCurrent(cfg.Home)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if cur.Runtime != RuntimeJDK {
		return nil
	}
	fmt.Fprintln(out, cur.Home)
	return nil
}

func envPrint(cfg Config, args []string, out io.Writer) error {
	rest := args
	javaHome := profileJavaHome(cfg)
	var shells []string
	for i := 0; i < len(rest); i++ {
		switch rest[i] {
		case "--shell", "-s":
			if i+1 >= len(rest) {
				return usage("jmv env print [--shell bash|zsh|fish]")
			}
			name, err := normalizeShellName(rest[i+1])
			if err != nil {
				return usage("jmv env print [--shell bash|zsh|fish]")
			}
			shells = append(shells, name)
			i++
		default:
			name, err := normalizeShellName(rest[i])
			if err != nil {
				return usage("jmv env print [--shell bash|zsh|fish]")
			}
			shells = append(shells, name)
		}
	}
	if len(shells) == 0 {
		printManualShellConfig(cfg, out)
		return nil
	}
	for idx, name := range shells {
		if idx > 0 {
			fmt.Fprintln(out)
		}
		fmt.Fprintf(out, "# %s\n", name)
		fmt.Fprint(out, shellConfigBlock(cfg, name, javaHome))
	}
	return nil
}

func normalizeShellName(s string) (string, error) {
	switch s {
	case "bash":
		return "bash", nil
	case "zsh":
		return "zsh", nil
	case "fish":
		return "fish", nil
	default:
		return "", errf("shell must be bash, zsh, or fish")
	}
}

func currentFromSession(home string, pid int) bool {
	if pid > 0 {
		if _, err := readSession(home, pid); err == nil {
			return true
		}
	}
	if runtime.GOOS == "windows" {
		if _, err := readSession(home, globalSessionPID); err == nil {
			return true
		}
	}
	return false
}

func runMaven(ctx context.Context, cfg Config, args []string, out io.Writer) error {
	if len(args) == 0 {
		return usage("jmv maven <list|install|uninstall|use|default|current|config>")
	}
	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "list", "ls":
		if len(rest) != 0 {
			return usage("jmv maven list")
		}
		return mavenList(ctx, cfg, out)
	case "install", "i":
		if len(rest) != 1 {
			return usage("jmv maven install <version|latest>")
		}
		return mavenInstall(ctx, cfg, rest[0], out)
	case "uninstall", "rm":
		if len(rest) != 1 {
			return usage("jmv maven uninstall <version>")
		}
		return mavenUninstall(cfg, rest[0], out)
	case "use", "u":
		if len(rest) != 1 {
			return usage("jmv maven use <version>")
		}
		return mavenUse(cfg, rest[0], out)
	case "default", "d":
		if len(rest) != 1 {
			return usage("jmv maven default <version>")
		}
		return mavenDefault(cfg, rest[0], out)
	case "current", "c":
		if len(rest) != 0 {
			return usage("jmv maven current")
		}
		return mavenCurrent(cfg, out)
	case "config":
		if len(rest) != 0 {
			return usage("jmv maven config")
		}
		return mavenConfig(cfg, out)
	default:
		return usage("jmv maven <list|install|uninstall|use|default|current|config>")
	}
}

func printLogo(out io.Writer) {
	fmt.Fprintln(out, `    ___  _____ ______   ___      ___
   |\  \|\   _ \  _   \|\  \    /  /|
   \ \  \ \  \\\__\ \  \ \  \  /  / /
 __ \ \  \ \  \\|__| \  \ \  \/  / /
|\  \\_\  \ \  \    \ \  \ \    / /
\ \________\ \__\    \ \__\ \__/ /
 \|________|\|__|     \|__|\|__|/    (JDK/JRE MANAGE VERSION)`)
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, `Usage: jmv <command> [options]

Commands:
  list      or ls             [-r|--runtime jdk|jre]
  install   or i              [-r|--runtime jdk|jre] <major>
  uninstall or rm             [-r|--runtime jdk|jre] <major>
  use       or u              [-r|--runtime jdk|jre] <major>
  default   or d              [-r|--runtime jdk|jre] <major>
  current   or c              [--home]
  env                         [print] [--shell bash|zsh|fish]
  maven                        <list|install|uninstall|use|default|current|config>
  version   or v
  help      or h

Options:
  --runtime, -r jdk|jre       Defaults to jdk.

Environment:
  JMV_HOME                    Defaults to ~/.jmv.
  JMV_MIRROR                  Defaults to TUNA Adoptium mirror.
  JMV_MAVEN_MIRROR            Defaults to Aliyun Apache Maven mirror.

Examples:
  jmv list
  jmv install 17
  jmv install --runtime jre 17
  jmv default 17
  jmv use 17
  jmv env
  jmv env print --shell bash
  jmv env java-home
  jmv current --home
  jmv maven install latest
  jmv maven default 3.9.11`)
}

func usage(s string) error {
	return errf("usage: %s", s)
}
