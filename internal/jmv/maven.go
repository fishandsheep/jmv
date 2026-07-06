package jmv

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

type MavenClient struct {
	BaseURL string
	HTTP    *http.Client
}

func NewMavenClient(baseURL string) MavenClient {
	return MavenClient{BaseURL: trimSlash(baseURL), HTTP: http.DefaultClient}
}

func (m MavenClient) List(ctx context.Context) ([]Release, error) {
	body, err := m.getText(ctx, m.BaseURL+"/maven-3/")
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	for _, href := range hrefs(body) {
		version := strings.TrimSuffix(strings.TrimPrefix(href, "./"), "/")
		if validMavenVersion(version) {
			seen[version] = true
		}
	}

	versions := make([]string, 0, len(seen))
	for version := range seen {
		versions = append(versions, version)
	}
	sortVersions(versions)

	releases := make([]Release, 0, len(versions))
	for _, version := range versions {
		releases = append(releases, m.release(version))
	}
	return releases, nil
}

func (m MavenClient) Resolve(ctx context.Context, version string) (Release, error) {
	if version == "latest" {
		releases, err := m.List(ctx)
		if err != nil {
			return Release{}, err
		}
		if len(releases) == 0 {
			return Release{}, errf("no Maven releases found")
		}
		return releases[len(releases)-1], nil
	}
	release := m.release(version)
	if _, err := m.getText(ctx, m.BaseURL+"/maven-3/"+version+"/binaries/"); err != nil {
		return Release{}, err
	}
	return release, nil
}

func (m MavenClient) release(version string) Release {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	name := "apache-maven-" + version + "-bin" + ext
	return Release{
		Runtime:  RuntimeMaven,
		Major:    version,
		FileName: name,
		URL:      m.BaseURL + "/maven-3/" + version + "/binaries/" + name,
		Platform: Platform{Arch: "all", OS: runtime.GOOS, Ext: ext},
	}
}

func (m MavenClient) getText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	setRequestHeaders(req)
	resp, err := m.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", errf("GET %s returned %s", url, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func validMavenVersion(s string) bool {
	if s == "" || strings.Contains(s, "/") {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && r != '.' {
			return false
		}
	}
	return strings.Contains(s, ".")
}

func sortVersions(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		a := strings.Split(versions[i], ".")
		b := strings.Split(versions[j], ".")
		for k := 0; k < len(a) || k < len(b); k++ {
			var av, bv string
			if k < len(a) {
				av = a[k]
			}
			if k < len(b) {
				bv = b[k]
			}
			if len(av) != len(bv) {
				return len(av) < len(bv)
			}
			if av != bv {
				return av < bv
			}
		}
		return false
	})
}

func mavenList(ctx context.Context, cfg Config, out io.Writer) error {
	releases, err := NewMavenClient(cfg.MavenMirror).List(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Available Maven versions from %s\n", cfg.MavenMirror)
	for _, release := range releases {
		installed := ""
		if _, err := readMetadata(cfg.Home, RuntimeMaven, release.Major); err == nil {
			installed = "\t(installed)"
		}
		fmt.Fprintf(out, "%s\t%s%s\n", release.Major, release.FileName, installed)
	}
	return nil
}

func mavenInstall(ctx context.Context, cfg Config, version string, out io.Writer) error {
	if err := ensureLayout(cfg.Home); err != nil {
		return err
	}
	release, err := NewMavenClient(cfg.MavenMirror).Resolve(ctx, version)
	if err != nil {
		return err
	}
	dest := installDir(cfg.Home, RuntimeMaven, release.Major)
	if _, err := os.Stat(dest); err == nil {
		fmt.Fprintf(out, "Maven %s already installed at %s\n", release.Major, dest)
		configureShellEnvironment(cfg, out)
		return nil
	}

	fmt.Fprintf(out, "Installing Maven %s\n", release.Major)
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
	if err := writeMavenSettings(cfg.Home); err != nil {
		return err
	}
	if err := os.Remove(archivePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	fmt.Fprintf(out, "Installed Maven %s at %s\n", release.Major, dest)
	if _, err := readMavenCurrent(cfg.Home); os.IsNotExist(err) {
		return mavenDefault(cfg, release.Major, out)
	}
	configureShellEnvironment(cfg, out)
	return nil
}

func mavenUninstall(cfg Config, version string, out io.Writer) error {
	if err := os.RemoveAll(installDir(cfg.Home, RuntimeMaven, version)); err != nil {
		return err
	}
	err := os.Remove(metadataPath(cfg.Home, RuntimeMaven, version))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	cur, err := readMavenCurrent(cfg.Home)
	if err == nil && cur.Major == version {
		if err := clearMavenCurrent(cfg.Home); err != nil {
			return err
		}
		if err := refreshShims(cfg.Home, os.Getppid()); err != nil {
			return err
		}
	}
	fmt.Fprintf(out, "Uninstalled Maven %s\n", version)
	return nil
}

func mavenDefault(cfg Config, version string, out io.Writer) error {
	meta, err := readMetadata(cfg.Home, RuntimeMaven, version)
	if err != nil {
		return errf("Maven %s is not installed", version)
	}
	cur := Current{Runtime: RuntimeMaven, Major: version, Home: meta.Home}
	if err := writeMavenCurrent(cfg.Home, cur); err != nil {
		return err
	}
	if err := writeMavenSettings(cfg.Home); err != nil {
		return err
	}
	if err := refreshShims(cfg.Home, os.Getppid()); err != nil {
		return err
	}
	fmt.Fprintf(out, "Default Maven set to %s (%s)\n", version, meta.Home)
	configureShellEnvironment(cfg, out)
	return nil
}

func mavenUse(cfg Config, version string, out io.Writer) error {
	return mavenDefault(cfg, version, out)
}

func mavenCurrent(cfg Config, out io.Writer) error {
	cur, err := readMavenCurrent(cfg.Home)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(out, "No active Maven version.")
			return nil
		}
		return err
	}
	fmt.Fprintf(out, "maven %s (default)\n", cur.Major)
	fmt.Fprintf(out, "Home: %s\n", cur.Home)
	fmt.Fprintf(out, "Settings: %s\n", mavenSettingsPath(cfg.Home))
	return nil
}

func mavenConfig(cfg Config, out io.Writer) error {
	if err := writeMavenSettings(cfg.Home); err != nil {
		return err
	}
	fmt.Fprintf(out, "Maven settings written to %s\n", mavenSettingsPath(cfg.Home))
	return nil
}

func writeMavenSettings(home string) error {
	if err := os.MkdirAll(mavenConfigDir(home), 0o755); err != nil {
		return err
	}
	const settings = `<settings xmlns="http://maven.apache.org/SETTINGS/1.2.0"
          xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
          xsi:schemaLocation="http://maven.apache.org/SETTINGS/1.2.0 https://maven.apache.org/xsd/settings-1.2.0.xsd">
  <mirrors>
    <mirror>
      <id>aliyun-public</id>
      <mirrorOf>*</mirrorOf>
      <name>Aliyun Maven Public</name>
      <url>https://maven.aliyun.com/repository/public</url>
    </mirror>
  </mirrors>
</settings>
`
	return os.WriteFile(mavenSettingsPath(home), []byte(settings), 0o644)
}
