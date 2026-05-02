package okm

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUseVsDefaultDifferentBehavior(t *testing.T) {
	archive := tinyJDKArchive(t)
	mux := http.NewServeMux()
	mux.HandleFunc("/Adoptium/17/jdk/x64/linux/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<a href="OpenJDK17U-jdk_x64_linux_hotspot_17.0.19_10.tar.gz">jdk</a>`))
	})
	mux.HandleFunc("/Adoptium/17/jdk/x64/linux/OpenJDK17U-jdk_x64_linux_hotspot_17.0.19_10.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	})
	mux.HandleFunc("/Adoptium/8/jdk/x64/linux/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<a href="OpenJDK8U-jdk_x64_linux_hotspot_8.0.432_6.tar.gz">jdk</a>`))
	})
	mux.HandleFunc("/Adoptium/8/jdk/x64/linux/OpenJDK8U-jdk_x64_linux_hotspot_8.0.432_6.tar.gz", func(w http.ResponseWriter, r *http.Request) {
		w.Write(archive)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	home := t.TempDir()
	cfg := Config{Home: home, Mirror: server.URL + "/Adoptium"}
	t.Setenv("OKM_HOME", home)

	// Install jdk 17 and jdk 8
	var out bytes.Buffer
	if err := install(context.Background(), cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatalf("install 17: %v", err)
	}
	if err := install(context.Background(), cfg, RuntimeJDK, "8", &out); err != nil {
		t.Fatalf("install 8: %v", err)
	}

	// Step 1: Set default to jdk 17
	out.Reset()
	if err := activateDefault(cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}

	// Verify: current.json exists, session.json does NOT
	if _, err := os.Stat(filepath.Join(home, "current.json")); err != nil {
		t.Fatal("current.json should exist after default")
	}
	if _, err := os.Stat(filepath.Join(home, "session.json")); !os.IsNotExist(err) {
		t.Fatal("session.json should NOT exist after default")
	}

	// Verify: resolveCurrent returns jdk 17
	cur, err := resolveCurrent(home)
	if err != nil {
		t.Fatal(err)
	}
	if cur.Major != "17" {
		t.Fatalf("expected jdk 17, got %s %s", cur.Runtime, cur.Major)
	}

	// Step 2: okm use 8 (session override)
	out.Reset()
	if err := activateUse(cfg, RuntimeJDK, "8", &out); err != nil {
		t.Fatal(err)
	}

	// Verify: session.json exists, current.json still has jdk 17
	if _, err := os.Stat(filepath.Join(home, "session.json")); err != nil {
		t.Fatal("session.json should exist after use")
	}
	curJson, _ := os.ReadFile(filepath.Join(home, "current.json"))
	if !strings.Contains(string(curJson), `"17"`) {
		t.Fatalf("current.json should still have jdk 17, got: %s", string(curJson))
	}

	// Verify: resolveCurrent returns jdk 8 (session takes priority)
	cur, err = resolveCurrent(home)
	if err != nil {
		t.Fatal(err)
	}
	if cur.Major != "8" {
		t.Fatalf("expected jdk 8 from session, got %s %s", cur.Runtime, cur.Major)
	}

	// Step 3: Simulate new terminal - delete session.json
	os.Remove(filepath.Join(home, "session.json"))

	// Verify: resolveCurrent now returns jdk 17 (default)
	cur, err = resolveCurrent(home)
	if err != nil {
		t.Fatal(err)
	}
	if cur.Major != "17" {
		t.Fatalf("expected jdk 17 (default) after session cleanup, got %s %s", cur.Runtime, cur.Major)
	}

	t.Log("PASS: use vs default behavior verified")
}
