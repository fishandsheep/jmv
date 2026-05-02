package okm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

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
		fmt.Fprintln(out, "Run `okm default` to make it the default runtime, or `okm use` for current shell hints.")
		return nil
	}

	fmt.Fprintf(out, "Installing %s %s\n", rt, major)
	fmt.Fprintf(out, "Download URL: %s\n", release.URL)

	archivePath := filepath.Join(downloadsDir(cfg.Home), release.FileName)
	if err := download(ctx, release.URL, archivePath, out); err != nil {
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
	fmt.Fprintln(out, "Run `okm default` to make this version the default runtime.")
	return nil
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
				if percent != lastPercent && (percent%5 == 0 || percent == 100) {
					fmt.Fprintf(out, "  Download progress: %d%%\n", percent)
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
		if err := refreshShims(cfg.Home); err != nil {
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
	if err := refreshShims(cfg.Home); err != nil {
		return err
	}
	fmt.Fprintf(out, "Default %s set to %s (%s)\n", rt, major, meta.Home)
	return nil
}

func activate(cfg Config, rt Runtime, major string, out io.Writer) error {
	return activateDefault(cfg, rt, major, out)
}

func activateUse(cfg Config, rt Runtime, major string, out io.Writer) error {
	meta, err := readMetadata(cfg.Home, rt, major)
	if err != nil {
		return errf("%s %s is not installed", rt, major)
	}
	fmt.Fprintf(out, "Using %s %s for current shell session\n", rt, major)
	fmt.Fprintf(out, "export JAVA_HOME=%s\n", shellQuote(meta.Home))
	fmt.Fprintln(out, "export PATH=\"$JAVA_HOME/bin:$PATH\"")
	return nil
}
