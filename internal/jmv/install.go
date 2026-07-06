package jmv

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var installPromptIn io.Reader = os.Stdin

func install(ctx context.Context, cfg Config, rt Runtime, major string, out io.Writer) error {
	if err := ensureLayout(cfg.Home); err != nil {
		return err
	}
	platform, err := DetectPlatform()
	if err != nil {
		return err
	}
	release, err := NewMirrorClient(cfg.Mirror).Resolve(ctx, rt, major, platform)
	if err != nil {
		return err
	}

	dest := installDir(cfg.Home, rt, major)
	if _, err := os.Stat(dest); err == nil {
		fmt.Fprintf(out, "%s %s already installed at %s\n", rt, major, dest)
		fmt.Fprintln(out, "Run `jmv default` to switch shims to this runtime.")
		configureShellEnvironment(cfg, out)
		return nil
	}

	fmt.Fprintf(out, "Installing %s %s\n", rt, major)
	fmt.Fprintf(out, "Download URL: %s\n", release.URL)

	archivePath := filepath.Join(downloadsDir(cfg.Home), release.FileName)
	if err := download(ctx, release.URL, archivePath, out); err != nil {
		return err
	}
	if err := verifyArchiveChecksum(ctx, release, archivePath, out); err != nil {
		_ = os.Remove(archivePath)
		return err
	}

	fmt.Fprintln(out, "[2/3] Extracting archive...")
	tmpDest := dest + ".tmp"
	_ = os.RemoveAll(tmpDest)
	if err := os.MkdirAll(tmpDest, 0o755); err != nil {
		return err
	}
	if err := extractArchive(archivePath, tmpDest); err != nil {
		_ = os.RemoveAll(tmpDest)
		return err
	}
	_ = os.RemoveAll(dest)
	if err := os.Rename(tmpDest, dest); err != nil {
		_ = os.RemoveAll(tmpDest)
		return err
	}

	fmt.Fprintln(out, "[3/3] Finalizing configuration...")
	if err := writeMetadata(cfg.Home, release, dest); err != nil {
		return err
	}
	if err := os.Remove(archivePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	fmt.Fprintf(out, "Installed %s %s at %s\n", rt, major, dest)
	return maybeConfigureDefaultAfterInstall(cfg, rt, major, out)
}

func maybeConfigureDefaultAfterInstall(cfg Config, rt Runtime, major string, out io.Writer) error {
	hasDefault := false
	if cur, err := readCurrent(cfg.Home); err == nil {
		hasDefault = cur.Runtime == rt
	}

	if !hasDefault {
		if err := activateDefault(cfg, rt, major, out); err != nil {
			return err
		}
		fmt.Fprintf(out, "Automatically set default %s to %s (first install).\n", rt, major)
		return nil
	}

	if shouldSetDefault(out) {
		if err := activateDefault(cfg, rt, major, out); err != nil {
			return err
		}
		fmt.Fprintf(out, "Default %s updated to %s.\n", rt, major)
		return nil
	}

	fmt.Fprintf(out, "Keeping existing default. Run `jmv default --runtime %s %s` to switch later.\n", rt, major)
	configureShellEnvironment(cfg, out)
	return nil
}

func shouldSetDefault(out io.Writer) bool {
	if os.Getenv("JMV_SET_DEFAULT") == "1" {
		return true
	}
	fmt.Fprint(out, "Set this version as default now? (y/N, default: n): ")
	reader := bufio.NewReader(installPromptIn)
	answer, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false
	}
	if err == io.EOF && answer == "" {
		return false
	}
	answer = strings.ToLower(strings.TrimSpace(answer))
	return answer == "y" || answer == "yes"
}

func verifyArchiveChecksum(ctx context.Context, release Release, archivePath string, out io.Writer) error {
	sum := strings.TrimSpace(release.SHA256)
	if sum == "" {
		var err error
		sum, err = downloadChecksum(ctx, release.URL+".sha256")
		if err != nil {
			return nil
		}
	}
	if sum == "" {
		return nil
	}
	actual, err := fileSHA256(archivePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, sum) {
		return errf("sha256 mismatch for %s", release.FileName)
	}
	fmt.Fprintln(out, "Verified sha256 checksum.")
	return nil
}

func downloadChecksum(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	setRequestHeaders(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", errf("checksum not found")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errf("GET %s returned %s", url, resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if err != nil {
		return "", err
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return "", nil
	}
	if _, err := hex.DecodeString(fields[0]); err != nil || len(fields[0]) != sha256.Size*2 {
		return "", nil
	}
	return fields[0], nil
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func download(ctx context.Context, url, path string, out io.Writer) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	setRequestHeaders(req)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errf("GET %s returned %s", url, resp.Status)
	}

	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	fmt.Fprintln(out, "[1/3] Downloading archive...")
	copyErr := copyWithProgress(f, resp.Body, resp.ContentLength, out)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(tmp)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return closeErr
	}
	return os.Rename(tmp, path)
}

func copyWithProgress(dst io.Writer, src io.Reader, total int64, out io.Writer) error {
	buf := make([]byte, 32*1024)
	var downloaded int64
	var lastPercent int64 = -1
	for {
		n, err := src.Read(buf)
		if n > 0 {
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
			downloaded += int64(n)
			if total > 0 {
				percent := downloaded * 100 / total
				if percent != lastPercent {
					hashes := int(percent / 5)
					if hashes > 20 {
						hashes = 20
					}
					fmt.Fprintf(out, "\r  Download progress: %-20s %3d%%", strings.Repeat("#", hashes), percent)
					if percent == 100 {
						fmt.Fprintln(out)
					}
					lastPercent = percent
				}
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}
	if total <= 0 {
		fmt.Fprintf(out, "  Downloaded %s\n", strings.TrimSpace(byteCount(downloaded)))
	}
	return nil
}

func byteCount(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for nn := n / unit; nn >= unit; nn /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func setRequestHeaders(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "*/*")
}

func uninstall(cfg Config, rt Runtime, major string, out io.Writer) error {
	if err := os.RemoveAll(installDir(cfg.Home, rt, major)); err != nil {
		return err
	}
	err := os.Remove(metadataPath(cfg.Home, rt, major))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	cur, err := readCurrent(cfg.Home)
	if err == nil && cur.Runtime == rt && cur.Major == major {
		if err := clearCurrent(cfg.Home); err != nil {
			return err
		}
		if err := refreshShims(cfg.Home, os.Getppid()); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "Uninstalled %s %s\n", rt, major)
	return nil
}

func activateDefault(cfg Config, rt Runtime, major string, out io.Writer) error {
	meta, err := readMetadata(cfg.Home, rt, major)
	if err != nil {
		return errf("%s %s is not installed", rt, major)
	}
	cur := Current{Runtime: rt, Major: major, Home: meta.Home}
	if err := writeCurrent(cfg.Home, cur); err != nil {
		return err
	}
	_ = clearSession(cfg.Home, os.Getppid())
	if runtime.GOOS == "windows" {
		_ = clearSession(cfg.Home, globalSessionPID)
	}
	if err := refreshShims(cfg.Home, os.Getppid()); err != nil {
		return err
	}
	fmt.Fprintf(out, "Default %s set to %s (%s)\n", rt, major, meta.Home)
	configureShellEnvironment(cfg, out)
	return nil
}

func activateUse(cfg Config, rt Runtime, major string, out io.Writer) error {
	meta, err := readMetadata(cfg.Home, rt, major)
	if err != nil {
		return errf("%s %s is not installed", rt, major)
	}
	cur := Current{Runtime: rt, Major: major, Home: meta.Home}
	pid := os.Getppid()
	if runtime.GOOS == "windows" {
		pid = globalSessionPID
	}
	if err := writeSession(cfg.Home, pid, cur); err != nil {
		return err
	}
	if err := refreshShims(cfg.Home, pid); err != nil {
		return err
	}
	var defaultNote string
	if def, err := readCurrent(cfg.Home); err == nil {
		defaultNote = fmt.Sprintf(" (default: %s %s)", def.Runtime, def.Major)
	}
	fmt.Fprintf(out, "Now using %s %s in this session%s\n", rt, major, defaultNote)
	return nil
}
