package jmv

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractTarGzRejectsUnsafeSymlink(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{
		Name:     "jdk-17/bin/java",
		Mode:     0o755,
		Typeflag: tar.TypeSymlink,
		Linkname: "../../outside",
	}); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(t.TempDir(), "bad.tar.gz")
	if err := os.WriteFile(archive, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	err := extractArchive(archive, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "unsafe symlink") {
		t.Fatalf("expected unsafe symlink error, got %v", err)
	}
}
