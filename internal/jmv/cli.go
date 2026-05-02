package jmv

import (
	"context"
	"fmt"
	"io"
	"os"
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
		if len(args) != 0 {
			return usage("jmv current")
		}
		return showCurrent(cfg, out)
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
			if args[i+1] != "jdk" && args[i+1] != "jre" {
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

func showCurrent(cfg Config, out io.Writer) error {
	pid := os.Getppid()
	cur, err := resolveCurrent(cfg.Home, pid)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(out, "No active Java version.")
			return nil
		}
		return err
	}
	if _, sessErr := readSession(cfg.Home, pid); sessErr == nil {
		fmt.Fprintf(out, "%s %s (session)\n", cur.Runtime, cur.Major)
	} else {
		fmt.Fprintf(out, "%s %s (default)\n", cur.Runtime, cur.Major)
	}
	fmt.Fprintf(out, "Home: %s\n", cur.Home)
	meta, err := readMetadata(cfg.Home, cur.Runtime, cur.Major)
	if err == nil {
		fmt.Fprintf(out, "Download URL: %s\n", meta.URL)
	}
	return nil
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
  list      or ls             [-r|--runtime [jdk]]
  install   or i              [-r|--runtime [jdk]] <major>
  uninstall or rm             [-r|--runtime [jdk]] <major>
  use       or u              [-r|--runtime [jdk]] <major>
  default   or d              [-r|--runtime [jdk]] <major>
  current   or c
  version   or v
  help      or h

Options:
  --runtime, -r [jdk|jre]     Defaults to jdk.

Environment:
  JMV_HOME                    Defaults to ~/.jmv.
  JMV_MIRROR                  Defaults to TUNA Adoptium mirror.

Examples:
  jmv list
  jmv install 17
  jmv install --runtime jre 17
  jmv default 17
  jmv use 17`)
}

func usage(s string) error {
	return errf("usage: %s", s)
}
