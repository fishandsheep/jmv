package jmv

import (
	"fmt"
	"os"
	"path/filepath"
)

func ensureLayout(home string) error {
	for _, dir := range []string{
		filepath.Join(home, "installs"),
		filepath.Join(home, "metadata"),
		filepath.Join(home, "downloads"),
		filepath.Join(home, "shims"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func installDir(home string, rt Runtime, major string) string {
	return filepath.Join(home, "installs", string(rt), major)
}

func metadataPath(home string, rt Runtime, major string) string {
	return filepath.Join(home, "metadata", string(rt), major+".json")
}

func currentPath(home string) string {
	return filepath.Join(home, "current.json")
}

func downloadsDir(home string) string {
	return filepath.Join(home, "downloads")
}

func shimsDir(home string) string {
	return filepath.Join(home, "shims")
}

func sessionDir(home string) string {
	return filepath.Join(home, "sessions")
}

func sessionPathForPID(home string, pid int) string {
	return filepath.Join(home, "sessions", fmt.Sprintf("%d.json", pid))
}
