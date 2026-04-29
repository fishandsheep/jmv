package okm

import (
	"os"
	"path/filepath"
	"runtime"
)

type Config struct {
	Home   string
	Mirror string
}

func LoadConfig() (Config, error) {
	home := os.Getenv("OKM_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return Config{}, err
		}
		home = filepath.Join(userHome, ".okm")
	}

	mirror := os.Getenv("OKM_MIRROR")
	if mirror == "" {
		mirror = DefaultMirror
	}

	return Config{Home: home, Mirror: trimSlash(mirror)}, nil
}

func DetectPlatform() (Platform, error) {
	var p Platform

	switch runtime.GOARCH {
	case "amd64":
		p.Arch = "x64"
	case "arm64":
		p.Arch = "aarch64"
	default:
		return Platform{}, errf("unsupported architecture: %s", runtime.GOARCH)
	}

	switch runtime.GOOS {
	case "linux":
		p.OS = "linux"
		p.Ext = ".tar.gz"
	case "darwin":
		p.OS = "mac"
		p.Ext = ".tar.gz"
	case "windows":
		p.OS = "windows"
		p.Ext = ".zip"
	default:
		return Platform{}, errf("unsupported operating system: %s", runtime.GOOS)
	}

	return p, nil
}

func trimSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}
