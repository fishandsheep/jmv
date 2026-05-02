package okm

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallActivateCurrentHomeAndUninstall(t *testing.T) {
	archive := tinyJDKArchive(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/Adoptium/17/jdk/x64/linux/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<a href="OpenJDK17U-jdk_x64_linux_hotspot_17.0.19_10.tar.gz">jdk</a>`))
	})
	mux.HandleFunc("/Adoptium/17/jdk/x64/linux/OpenJDK17U-jdk_x64_linux_hotspot_17.0.19_10.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != userAgent || r.Header.Get("Accept") != "*/*" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Write(archive)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	home := t.TempDir()
	cfg := Config{Home: home, Mirror: server.URL + "/Adoptium"}

	t.Setenv("OKM_HOME", home)
	t.Setenv("OKM_MIRROR", cfg.Mirror)

	var out bytes.Buffer
	if err := install(context.Background(), cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Download URL: "+server.URL+"/Adoptium/17/jdk/x64/linux/OpenJDK17U-jdk_x64_linux_hotspot_17.0.19_10.tar.gz") {
		t.Fatalf("missing download URL in output:\n%s", out.String())
	}
	javaPath := filepath.Join(home, "installs", "jdk", "17", "bin", "java")
	if _, err := os.Stat(javaPath); err != nil {
		t.Fatalf("expected extracted java at %s: %v", javaPath, err)
	}

	out.Reset()
	if err := activate(cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	cur, err := readCurrent(home)
	if err != nil {
		t.Fatal(err)
	}
	if cur.Runtime != RuntimeJDK || cur.Major != "17" {
		t.Fatalf("unexpected current: %#v", cur)
	}
	if _, err := os.Stat(filepath.Join(home, "shims", "java")); err != nil {
		t.Fatalf("expected java shim: %v", err)
	}

	out.Reset()
	if err := showHome(cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), filepath.Join(home, "installs", "jdk", "17")) {
		t.Fatalf("unexpected home output: %s", out.String())
	}

	out.Reset()
	if err := uninstall(cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(javaPath); !os.IsNotExist(err) {
		t.Fatalf("expected install removed, stat err=%v", err)
	}
}

func TestInstallShowsInstalledAndUseIsTemporary(t *testing.T) {
	archive := tinyJDKArchive(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/Adoptium/17/jdk/x64/linux/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<a href="OpenJDK17U-jdk_x64_linux_hotspot_17.0.19_10.tar.gz">jdk</a>`))
	})
	mux.HandleFunc("/Adoptium/17/jdk/x64/linux/OpenJDK17U-jdk_x64_linux_hotspot_17.0.19_10.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	})
	mux.HandleFunc("/Adoptium/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<a href="17/">17/</a>`))
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	home := t.TempDir()
	cfg := Config{Home: home, Mirror: server.URL + "/Adoptium"}

	var out bytes.Buffer
	if err := install(context.Background(), cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "[1/3] Downloading archive") || !strings.Contains(out.String(), "[2/3] Extracting archive") || !strings.Contains(out.String(), "[3/3] Finalizing configuration") {
		t.Fatalf("missing progress output: %s", out.String())
	}
	if _, err := os.Stat(filepath.Join(home, "downloads", "OpenJDK17U-jdk_x64_linux_hotspot_17.0.19_10.tar.gz")); !os.IsNotExist(err) {
		t.Fatalf("archive should be cleaned up, err=%v", err)
	}

	out.Reset()
	if err := install(context.Background(), cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "already installed") {
		t.Fatalf("expected already installed message, got: %s", out.String())
	}

	out.Reset()
	if err := list(context.Background(), cfg, RuntimeJDK, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "(installed)") {
		t.Fatalf("expected installed marker in list output: %s", out.String())
	}

	out.Reset()
	if err := activateUse(cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, "current.json")); !os.IsNotExist(err) {
		t.Fatalf("okm use must not persist current.json, err=%v", err)
	}
	if !strings.Contains(out.String(), "export JAVA_HOME=") {
		t.Fatalf("expected shell export output: %s", out.String())
	}
}

func tinyJDKArchive(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	files := map[string]string{
		"jdk-17/bin/java":  "#!/usr/bin/env sh\n",
		"jdk-17/bin/javac": "#!/usr/bin/env sh\n",
	}
	for name, content := range files {
		body := []byte(content)
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
