package jmv

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func refreshShims(home string, sessionPID int) error {
	dir := shimsDir(home)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := clearShimFiles(dir); err != nil {
		return err
	}

	okmExe, err := os.Executable()
	if err != nil {
		return err
	}
	if cur, err := resolveCurrent(home, sessionPID); err == nil {
		if err := writeShimsForBin(dir, filepath.Join(cur.Home, "bin"), okmExe, home); err != nil {
			return err
		}
	}
	if cur, err := readMavenCurrent(home); err == nil {
		if err := writeShimsForBin(dir, filepath.Join(cur.Home, "bin"), okmExe, home); err != nil {
			return err
		}
	}
	return nil
}

func writeShimsForBin(dir, binDir, okmExe, home string) error {
	entries, err := os.ReadDir(binDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !isExecutable(info) {
			continue
		}
		name := entry.Name()
		if runtime.GOOS == "windows" {
			name = strings.TrimSuffix(name, ".exe")
			script := "@echo off\r\nset \"JMV_HOME=" + home + "\"\r\n\"" + okmExe + "\" shim " + name + " %*\r\n"
			if err := os.WriteFile(filepath.Join(dir, name+".cmd"), []byte(script), 0o755); err != nil {
				return err
			}
		} else {
			script := "#!/usr/bin/env sh\nJMV_HOME=" + shellQuote(home) + " exec " + shellQuote(okmExe) + " shim " + name + " \"$@\"\n"
			if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
				return err
			}
		}
	}
	return nil
}

func clearShimFiles(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
			return err
		}
	}
	return nil
}

func isExecutable(info fs.FileInfo) bool {
	if runtime.GOOS == "windows" {
		return strings.HasSuffix(strings.ToLower(info.Name()), ".exe")
	}
	return info.Mode()&0o111 != 0
}

func runShim(home string, exe string, args []string) error {
	for _, cur := range activeHomes(home, os.Getppid()) {
		target := filepath.Join(cur.Home, "bin", exe)
		if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(target), ".exe") {
			target += ".exe"
		}
		if _, err := os.Stat(target); err != nil {
			continue
		}
		cmd := exec.Command(target, args...)
		if cur.Runtime == RuntimeMaven {
			cmd = exec.Command(target, append([]string{"-s", mavenSettingsPath(home)}, args...)...)
		}
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}
	return errf("no active runtime provides %s", exe)
}

func activeHomes(home string, sessionPID int) []Current {
	var out []Current
	if cur, err := resolveCurrent(home, sessionPID); err == nil {
		out = append(out, cur)
	}
	if cur, err := readMavenCurrent(home); err == nil {
		out = append(out, cur)
	}
	return out
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
