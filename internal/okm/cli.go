package okm

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	case "help", "--help", "-h":
		printHelp(out)
		return nil
	case "version", "--version", "-v":
		fmt.Fprintf(out, "okm %s\n", Version)
		return nil
	case "list", "ls":
		rt, rest, err := parseRuntime(args)
		if err != nil {
			return err
		}
		if len(rest) != 0 {
			return usage("okm list [--runtime jre]")
		}
		return list(ctx, cfg, rt, out)
	case "install", "i":
		rt, rest, err := parseRuntime(args)
		if err != nil {
			return err
		}
		if len(rest) != 1 {
			return usage("okm install [--runtime jre] <major>")
		}
		return install(ctx, cfg, rt, rest[0], out)
	case "uninstall", "rm":
		rt, rest, err := parseRuntime(args)
		if err != nil {
			return err
		}
		if len(rest) != 1 {
			return usage("okm uninstall [--runtime jre] <major>")
		}
		return uninstall(cfg, rt, rest[0], out)
	case "default", "d", "use", "u":
		rt, rest, err := parseRuntime(args)
		if err != nil {
			return err
		}
		if len(rest) != 1 {
			return usage("okm " + cmd + " [--runtime jre] <major>")
		}
		return activate(cfg, rt, rest[0], out)
	case "current", "c":
		if len(args) != 0 {
			return usage("okm current")
		}
		return showCurrent(cfg, out)
	case "home", "h":
		rt, rest, err := parseRuntime(args)
		if err != nil {
			return err
		}
		if len(rest) != 1 {
			return usage("okm home [--runtime jre] <major>")
		}
		return showHome(cfg, rt, rest[0], out)
	case "shim":
		if len(args) < 1 {
			return usage("okm shim <executable> [args...]")
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
			if i+1 >= len(args) {
				return "", nil, usage("--runtime requires jdk or jre")
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
		fmt.Fprintf(out, "%s\t%s\n", release.Major, release.FileName)
	}
	return nil
}

func showCurrent(cfg Config, out io.Writer) error {
	cur, err := readCurrent(cfg.Home)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(out, "No active Java version.")
			return nil
		}
		return err
	}
	fmt.Fprintf(out, "%s %s\n", cur.Runtime, cur.Major)
	fmt.Fprintf(out, "Home: %s\n", cur.Home)
	meta, err := readMetadata(cfg.Home, cur.Runtime, cur.Major)
	if err == nil {
		fmt.Fprintf(out, "Download URL: %s\n", meta.URL)
	}
	return nil
}

func showHome(cfg Config, rt Runtime, major string, out io.Writer) error {
	meta, err := readMetadata(cfg.Home, rt, major)
	if err != nil {
		return errf("%s %s is not installed", rt, major)
	}
	fmt.Fprintln(out, filepath.Clean(meta.Home))
	return nil
}

func printHelp(out io.Writer) {
	fmt.Fprintln(out, `Usage: okm <command> [options]

Commands:
  list      or ls             [--runtime jre]
  install   or i              [--runtime jre] <major>
  uninstall or rm             [--runtime jre] <major>
  use       or u              [--runtime jre] <major>
  default   or d              [--runtime jre] <major>
  current   or c
  home      or h              [--runtime jre] <major>
  version   or v
  help

Options:
  --runtime, -r jdk|jre       Defaults to jdk.

Environment:
  OKM_HOME                    Defaults to ~/.okm.
  OKM_MIRROR                  Defaults to TUNA Adoptium mirror.

Examples:
  okm list
  okm install 17
  okm install --runtime jre 17
  okm default 17`)
}

func usage(s string) error {
	return errf("usage: %s", s)
}
