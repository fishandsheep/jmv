package jmv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const globalSessionPID = 0

func writeMetadata(home string, release Release, javaHome string) error {
	if err := os.MkdirAll(filepath.Dir(metadataPath(home, release.Runtime, release.Major)), 0o755); err != nil {
		return err
	}
	meta := Metadata{
		Runtime:     release.Runtime,
		Major:       release.Major,
		FileName:    release.FileName,
		URL:         release.URL,
		Platform:    release.Platform.Arch + "/" + release.Platform.OS,
		Home:        javaHome,
		InstalledAt: time.Now().UTC(),
	}
	return writeJSON(metadataPath(home, release.Runtime, release.Major), meta)
}

func readMetadata(home string, rt Runtime, major string) (Metadata, error) {
	var meta Metadata
	if err := readJSON(metadataPath(home, rt, major), &meta); err != nil {
		return Metadata{}, err
	}
	return meta, nil
}

func writeCurrent(home string, cur Current) error {
	return writeJSON(currentPath(home), cur)
}

func readCurrent(home string) (Current, error) {
	var cur Current
	if err := readJSON(currentPath(home), &cur); err != nil {
		return Current{}, err
	}
	return cur, nil
}

func clearCurrent(home string) error {
	err := os.Remove(currentPath(home))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func writeSession(home string, pid int, cur Current) error {
	return writeJSON(sessionPathForPID(home, pid), cur)
}

func readSession(home string, pid int) (Current, error) {
	var cur Current
	if err := readJSON(sessionPathForPID(home, pid), &cur); err != nil {
		return Current{}, err
	}
	return cur, nil
}

func clearSession(home string, pid int) error {
	err := os.Remove(sessionPathForPID(home, pid))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func clearAllSessions(home string) error {
	return os.RemoveAll(sessionDir(home))
}

func resolveCurrent(home string, sessionPID int) (Current, error) {
	if sessionPID > 0 {
		if cur, err := readSession(home, sessionPID); err == nil {
			return cur, nil
		}
	}
	if runtime.GOOS == "windows" {
		if cur, err := readSession(home, globalSessionPID); err == nil {
			return cur, nil
		}
	}
	return readCurrent(home)
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func readJSON(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
