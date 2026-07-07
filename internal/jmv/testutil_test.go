package jmv

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"testing"
)

func disableProfileMutation(t *testing.T) {
	t.Helper()
	t.Setenv("JMV_NO_MODIFY_PROFILE", "1")
}

// adoptiumIndexPath returns the per-platform Adoptium directory mock pattern,
// e.g. /Adoptium/17/jdk/x64/windows/ on Windows lizard mimicry.
func adoptiumIndexPath(major string) string {
	p, _ := DetectPlatform()
	return fmt.Sprintf("/Adoptium/%s/jdk/%s/%s/", major, p.Arch, p.OS)
}

// adoptiumAsset returns the per-platform asset filename the install code will
// match against the index listing, e.g.
//
//	OpenJDK17U-jdk_x64_linux_hotspot_17.0.19_10.tar.gz on Linux/macOS
//	OpenJDK17U-jdk_x64_windows_hotspot_17.0.19_10.zip  on Windows
func adoptiumAsset(major, fullVersion string) string {
	p, _ := DetectPlatform()
	return fmt.Sprintf("OpenJDK%sU-jdk_%s_%s_hotspot_%s%s",
		major, p.Arch, p.OS, fullVersion, p.Ext)
}

// adoptiumMock registers both the index listing and the asset download handler
// with platform-aware paths/filenames, so tests run identically on every
// runner regardless of DetectPlatform() arch/OS/ext.
//
// Note: the .sha256 URL is intentionally NOT registered. mirror.go's
// verifyArchiveChecksum gracefully returns nil on 404, which means tests
// exercise the install download + extract path but skip checksum verification.
// SHA256 verification is instead validated in real-mirror live CI runs.
func adoptiumMock(t *testing.T, mux *http.ServeMux, major, fullVersion string, archive []byte) {
	t.Helper()
	index := adoptiumIndexPath(major)
	asset := adoptiumAsset(major, fullVersion)
	mux.HandleFunc(index, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<a href="%s">jdk</a>`, asset)
	})
	mux.HandleFunc(index+asset, func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	})
}

// adoptiumMockStrict is adoptiumMock plus the User-Agent / Accept header
// verification that mirrors mirror.go::setRequestHeaders.
func adoptiumMockStrict(t *testing.T, mux *http.ServeMux, major, fullVersion string, archive []byte) {
	t.Helper()
	index := adoptiumIndexPath(major)
	asset := adoptiumAsset(major, fullVersion)
	mux.HandleFunc(index, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `<a href="%s">jdk</a>`, asset)
	})
	mux.HandleFunc(index+asset, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != userAgent || r.Header.Get("Accept") != "*/*" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Write(archive)
	})
}

// mavenAsset returns the per-platform Maven archive filename apache-maven
// downloads, mirroring MavenClient.release().
func mavenAsset(version string) string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("apache-maven-%s-bin%s", version, ext)
}

func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func shimPath(home, name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "shims", name+".cmd")
	}
	return filepath.Join(home, "shims", name)
}

// tinyJDKZip mirrors tinyJDKArchive but produces a zip archive. Used on
// Windows runners where DetectPlatform() reports Ext=".zip".
func tinyJDKZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"jdk-17/bin/java.exe":  "@echo off\r\n",
		"jdk-17/bin/javac.exe": "@echo off\r\n",
	}
	for name, content := range files {
		h := &zip.FileHeader{Name: name, Method: zip.Deflate}
		h.SetMode(0o755)
		w, err := zw.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// pickJDKArchive returns the JDK archive bytes the current platform expects:
// tar.gz on Linux/macOS, zip on Windows.
func pickJDKArchive(t *testing.T) []byte {
	t.Helper()
	p, _ := DetectPlatform()
	if p.Ext == ".zip" {
		return tinyJDKZip(t)
	}
	return tinyJDKArchive(t)
}

// pickMavenArchive mirrors pickJDKArchive for Maven.
func pickMavenArchive(t *testing.T) []byte {
	t.Helper()
	if runtime.GOOS == "windows" {
		return tinyMavenZip(t)
	}
	return tinyMavenArchive(t)
}
