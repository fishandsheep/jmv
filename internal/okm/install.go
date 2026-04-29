package okm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
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

	fmt.Fprintf(out, "Installing %s %s\n", rt, major)
	fmt.Fprintf(out, "Download URL: %s\n", release.URL)

	archivePath := filepath.Join(downloadsDir(cfg.Home), release.FileName)
	if err := download(ctx, release.URL, archivePath); err != nil {
		return err
	}

	dest := installDir(cfg.Home, rt, major)
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

	if err := writeMetadata(cfg.Home, release, dest); err != nil {
		return err
	}
	fmt.Fprintf(out, "Installed %s %s at %s\n", rt, major, dest)
	fmt.Fprintln(out, "Run `okm default` to activate this version.")
	return nil
}

func download(ctx context.Context, url, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
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
	_, copyErr := io.Copy(f, resp.Body)
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

func activate(cfg Config, rt Runtime, major string, out io.Writer) error {
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
	fmt.Fprintf(out, "Using %s %s at %s\n", rt, major, meta.Home)
	fmt.Fprintf(out, "Add %s to PATH to use okm-managed Java commands.\n", shimsDir(cfg.Home))
	return nil
}
