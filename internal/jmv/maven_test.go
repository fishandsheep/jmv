package jmv

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMavenInstallSettingsAndShimCoexist(t *testing.T) {
	disableProfileMutation(t)
	mavenArchive := pickMavenArchive(t)
	javaArchive := pickJDKArchive(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/apache/maven/maven-3/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<a href="3.9.10/">3.9.10/</a><a href="3.9.11/">3.9.11/</a>`))
	})
	mux.HandleFunc("/apache/maven/maven-3/3.9.11/binaries/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`ok`))
	})
	mux.HandleFunc("/apache/maven/maven-3/3.9.11/binaries/"+mavenAsset("3.9.11"), func(w http.ResponseWriter, r *http.Request) {
		w.Write(mavenArchive)
	})
	adoptiumMock(t, mux, "17", "17.0.19_10", javaArchive)
	server := httptest.NewServer(mux)
	defer server.Close()

	home := t.TempDir()
	cfg := Config{Home: home, Mirror: server.URL + "/Adoptium", MavenMirror: server.URL + "/apache/maven"}
	var out bytes.Buffer
	if err := install(context.Background(), cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	if err := mavenInstall(context.Background(), cfg, "latest", &out); err != nil {
		t.Fatal(err)
	}

	settings, err := os.ReadFile(mavenSettingsPath(home))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(settings), "https://maven.aliyun.com/repository/public") || !strings.Contains(string(settings), "<mirrorOf>*</mirrorOf>") {
		t.Fatalf("settings.xml missing Aliyun mirror:\n%s", string(settings))
	}
	if _, err := os.Stat(filepath.Join(home, "shims", "java")); err != nil {
		t.Fatalf("expected java shim: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "shims", "mvn")); err != nil {
		t.Fatalf("expected mvn shim: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "shims", "mvnDebug")); err != nil {
		t.Fatalf("expected mvnDebug shim: %v", err)
	}
	cur, err := readMavenCurrent(home)
	if err != nil {
		t.Fatal(err)
	}
	if cur.Major != "3.9.11" {
		t.Fatalf("expected latest Maven 3.9.11, got %#v", cur)
	}
}

func TestExtractMavenZip(t *testing.T) {
	archive := filepath.Join(t.TempDir(), "apache-maven.zip")
	if err := os.WriteFile(archive, tinyMavenZip(t), 0o644); err != nil {
		t.Fatal(err)
	}
	dest := t.TempDir()
	if err := extractArchive(archive, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dest, "bin", "mvn.cmd")); err != nil {
		t.Fatalf("expected extracted mvn.cmd: %v", err)
	}
}

func tinyMavenArchive(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	files := []string{
		"apache-maven-3.9.11/bin/mvn",
		"apache-maven-3.9.11/bin/mvnDebug",
	}
	for _, name := range files {
		body := []byte("#!/usr/bin/env sh\n")
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

func tinyMavenZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	h := &zip.FileHeader{Name: "apache-maven-3.9.11/bin/mvn.cmd", Method: zip.Deflate}
	h.SetMode(0o755)
	w, err := zw.CreateHeader(h)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("@echo off\r\n")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestMavenClientUsesZipOnWindows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only")
	}
	release := NewMavenClient("https://example.test/apache/maven").release("3.9.11")
	if !strings.HasSuffix(release.FileName, ".zip") {
		t.Fatalf("expected zip on Windows, got %s", release.FileName)
	}
}
