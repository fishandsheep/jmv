package okm

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func refreshShims(home string) error {
	dir := shimsDir(home)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := clearShimFiles(dir); err != nil {
		return err
	}

	cur, err := readCurrent(home)
	if err != nil {
		return nil
	}
	binDir := filepath.Join(cur.Home, "bin")
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return err
	}

	okmExe, err := os.Executable()
	if err != nil {
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
			script := "@echo off\r\nset \"OKM_HOME=" + home + "\"\r\n\"" + okmExe + "\" shim " + name + " %*\r\n"
			if err := os.WriteFile(filepath.Join(dir, name+".cmd"), []byte(script), 0o755); err != nil {
				return err
			}
		} else {
			script := "#!/usr/bin/env sh\nOKM_HOME=" + shellQuote(home) + " exec " + shellQuote(okmExe) + " shim " + name + " \"$@\"\n"
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
	cur, err := readCurrent(home)
	if err != nil {
		return errf("no active Java version; run `okm default <major>` first")
	}
	target := filepath.Join(cur.Home, "bin", exe)
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(target), ".exe") {
		target += ".exe"
	}
	if _, err := os.Stat(target); err != nil {
		return err
	}
	cmd := exec.Command(target, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
