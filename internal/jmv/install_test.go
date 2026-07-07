package jmv

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

func TestInstallActivateCurrentAndUninstall(t *testing.T) {
	disableProfileMutation(t)
	originalPrompt := installPromptIn
	installPromptIn = strings.NewReader("\n")
	t.Cleanup(func() { installPromptIn = originalPrompt })

	archive := pickJDKArchive(t)
	mux := http.NewServeMux()
	adoptiumMockStrict(t, mux, "17", "17.0.19_10", archive)
	server := httptest.NewServer(mux)
	defer server.Close()

	home := t.TempDir()
	cfg := Config{Home: home, Mirror: server.URL + "/Adoptium"}

	t.Setenv("JMV_HOME", home)
	t.Setenv("JMV_MIRROR", cfg.Mirror)

	var out bytes.Buffer
	if err := install(context.Background(), cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	wantURL := "Download URL: " + server.URL + adoptiumIndexPath("17") + adoptiumAsset("17", "17.0.19_10")
	if !strings.Contains(out.String(), wantURL) {
		t.Fatalf("missing download URL in output:\n%s", out.String())
	}
	javaPath := filepath.Join(home, "installs", "jdk", "17", "bin", executableName("java"))
	if _, err := os.Stat(javaPath); err != nil {
		t.Fatalf("expected extracted java at %s: %v", javaPath, err)
	}

	curAuto, err := readCurrent(home)
	if err != nil {
		t.Fatal(err)
	}
	if curAuto.Major != "17" {
		t.Fatalf("expected first install to auto-set default to 17, got %#v", curAuto)
	}

	out.Reset()
	if err := activateDefault(cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	cur, err := readCurrent(home)
	if err != nil {
		t.Fatal(err)
	}
	if cur.Runtime != RuntimeJDK || cur.Major != "17" {
		t.Fatalf("unexpected current: %#v", cur)
	}
	if _, err := os.Stat(shimPath(home, "java")); err != nil {
		t.Fatalf("expected java shim: %v", err)
	}

	out.Reset()
	if err := uninstall(cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(javaPath); !os.IsNotExist(err) {
		t.Fatalf("expected install removed, stat err=%v", err)
	}
}

func TestInstallShowsInstalledAndUseSetsSessionOverride(t *testing.T) {
	disableProfileMutation(t)
	originalPrompt := installPromptIn
	installPromptIn = strings.NewReader("\n")
	t.Cleanup(func() { installPromptIn = originalPrompt })
	archive := pickJDKArchive(t)
	mux := http.NewServeMux()
	adoptiumMock(t, mux, "17", "17.0.19_10", archive)
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
	wantArchive := filepath.Join("downloads", adoptiumAsset("17", "17.0.19_10"))
	if _, err := os.Stat(filepath.Join(home, wantArchive)); !os.IsNotExist(err) {
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
	pid := os.Getppid()
	if _, err := os.Stat(sessionPathForPID(home, pid)); err != nil {
		t.Fatalf("jmv use should create session file, err=%v", err)
	}
	cur, err := readCurrent(home)
	if err != nil {
		t.Fatalf("expected current.json to remain after use: %v", err)
	}
	if cur.Major != "17" {
		t.Fatalf("expected default to remain 17 after use, got %#v", cur)
	}
	if !strings.Contains(out.String(), "session") {
		t.Fatalf("expected session indicator in use output: %s", out.String())
	}
}

func TestInstallPromptCanKeepExistingDefault(t *testing.T) {
	disableProfileMutation(t)
	archive := pickJDKArchive(t)
	mux := http.NewServeMux()
	adoptiumMock(t, mux, "17", "17.0.19_10", archive)
	adoptiumMock(t, mux, "8", "8.0.432_6", archive)
	server := httptest.NewServer(mux)
	defer server.Close()

	home := t.TempDir()
	cfg := Config{Home: home, Mirror: server.URL + "/Adoptium"}

	originalPrompt := installPromptIn
	installPromptIn = strings.NewReader("n\n")
	t.Cleanup(func() { installPromptIn = originalPrompt })

	var out bytes.Buffer
	if err := install(context.Background(), cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := install(context.Background(), cfg, RuntimeJDK, "8", &out); err != nil {
		t.Fatal(err)
	}

	cur, err := readCurrent(home)
	if err != nil {
		t.Fatal(err)
	}
	if cur.Major != "17" {
		t.Fatalf("expected to keep existing default 17, got %#v", cur)
	}
}

func TestInstallPromptEOFKeepsExistingDefault(t *testing.T) {
	disableProfileMutation(t)
	archive := pickJDKArchive(t)
	mux := http.NewServeMux()
	adoptiumMock(t, mux, "17", "17.0.19_10", archive)
	adoptiumMock(t, mux, "21", "21.0.7_6", archive)
	server := httptest.NewServer(mux)
	defer server.Close()

	home := t.TempDir()
	cfg := Config{Home: home, Mirror: server.URL + "/Adoptium"}
	originalPrompt := installPromptIn
	installPromptIn = strings.NewReader("")
	t.Cleanup(func() { installPromptIn = originalPrompt })

	var out bytes.Buffer
	if err := install(context.Background(), cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	if err := install(context.Background(), cfg, RuntimeJDK, "21", &out); err != nil {
		t.Fatal(err)
	}
	cur, err := readCurrent(home)
	if err != nil {
		t.Fatal(err)
	}
	if cur.Major != "17" {
		t.Fatalf("expected EOF prompt to keep existing default 17, got %#v", cur)
	}
}

func TestCopyWithProgressSingleLine(t *testing.T) {
	src := strings.NewReader(strings.Repeat("a", 1024))
	var downloaded bytes.Buffer
	var out bytes.Buffer
	if err := copyWithProgress(&downloaded, src, 1024, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Count(out.String(), "\n") != 1 {
		t.Fatalf("expected a single progress line, got output: %q", out.String())
	}
	if !strings.Contains(out.String(), "100%") {
		t.Fatalf("expected 100%% output, got: %q", out.String())
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
