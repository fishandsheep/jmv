package jmv

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCurrentHomePrintsHomePath(t *testing.T) {
	disableProfileMutation(t)
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	t.Setenv("JMV_HOME", jmvHome)

	javaHome := installDir(jmvHome, RuntimeJDK, "17")
	if err := writeCurrent(jmvHome, Current{Runtime: RuntimeJDK, Major: "17", Home: javaHome}); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := Run(context.Background(), []string{"current", "--home"}, &out, &out); err != nil {
		t.Fatalf("current --home: %v", err)
	}
	if strings.TrimSpace(out.String()) != javaHome {
		t.Fatalf("expected %s, got %s", javaHome, strings.TrimSpace(out.String()))
	}
}

func TestCurrentHomeFailsWithoutDefaultRuntime(t *testing.T) {
	disableProfileMutation(t)
	homeDir := t.TempDir()
	t.Setenv("JMV_HOME", filepath.Join(homeDir, ".jmv"))

	var out bytes.Buffer
	err := Run(context.Background(), []string{"current", "--home"}, &out, &out)
	if err == nil {
		t.Fatalf("expected error when no default runtime, got output: %q", out.String())
	}
}

func TestCurrentReportsHintWhenJavaHomeUnset(t *testing.T) {
	disableProfileMutation(t)
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	t.Setenv("JMV_HOME", jmvHome)
	t.Setenv("JAVA_HOME", "")

	javaHome := installDir(jmvHome, RuntimeJDK, "17")
	release := Release{Runtime: RuntimeJDK, Major: "17", FileName: "OpenJDK17.tar.gz", URL: "https://example.test/jdk.tgz"}
	if err := os.MkdirAll(filepath.Join(javaHome, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeMetadata(jmvHome, release, javaHome); err != nil {
		t.Fatal(err)
	}
	if err := writeCurrent(jmvHome, Current{Runtime: RuntimeJDK, Major: "17", Home: javaHome}); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := Run(context.Background(), []string{"current"}, &out, &out); err != nil {
		t.Fatalf("current: %v", err)
	}
	if !strings.Contains(out.String(), "JAVA_HOME is not set in this shell") {
		t.Fatalf("expected JAVA_HOME hint, got:\n%s", out.String())
	}
}

func TestCurrentReportsWarningWhenJavaHomeMismatches(t *testing.T) {
	disableProfileMutation(t)
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	t.Setenv("JMV_HOME", jmvHome)
	t.Setenv("JAVA_HOME", "/some/other/jdk")

	javaHome := installDir(jmvHome, RuntimeJDK, "17")
	release := Release{Runtime: RuntimeJDK, Major: "17", FileName: "OpenJDK17.tar.gz", URL: "https://example.test/jdk.tgz"}
	if err := os.MkdirAll(filepath.Join(javaHome, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeMetadata(jmvHome, release, javaHome); err != nil {
		t.Fatal(err)
	}
	if err := writeCurrent(jmvHome, Current{Runtime: RuntimeJDK, Major: "17", Home: javaHome}); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := Run(context.Background(), []string{"current"}, &out, &out); err != nil {
		t.Fatalf("current: %v", err)
	}
	if !strings.Contains(out.String(), "Warning: JAVA_HOME=/some/other/jdk does not match this runtime's home") {
		t.Fatalf("expected mismatch warning, got:\n%s", out.String())
	}
}

func TestCurrentStaysQuietWhenJavaHomeMatches(t *testing.T) {
	disableProfileMutation(t)
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	javaHome := installDir(jmvHome, RuntimeJDK, "17")
	t.Setenv("JMV_HOME", jmvHome)
	// Use the resolved-cleanable form whose Clean() matches javaHome.
	t.Setenv("JAVA_HOME", javaHome)

	release := Release{Runtime: RuntimeJDK, Major: "17", FileName: "OpenJDK17.tar.gz", URL: "https://example.test/jdk.tgz"}
	if err := os.MkdirAll(filepath.Join(javaHome, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeMetadata(jmvHome, release, javaHome); err != nil {
		t.Fatal(err)
	}
	if err := writeCurrent(jmvHome, Current{Runtime: RuntimeJDK, Major: "17", Home: javaHome}); err != nil {
		t.Fatal(err)
	}

	var out bytes.Buffer
	if err := Run(context.Background(), []string{"current"}, &out, &out); err != nil {
		t.Fatalf("current: %v", err)
	}
	if strings.Contains(out.String(), "Hint: JAVA_HOME") || strings.Contains(out.String(), "Warning: JAVA_HOME") {
		t.Fatalf("unexpected JAVA_HOME status line when matched:\n%s", out.String())
	}
}

func TestEnvPrintOutputsAllShells(t *testing.T) {
	disableProfileMutation(t)
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	javaHome := installDir(jmvHome, RuntimeJDK, "17")
	t.Setenv("JMV_HOME", jmvHome)
	t.Setenv("JAVA_HOME", "")
	if err := writeCurrent(jmvHome, Current{Runtime: RuntimeJDK, Major: "17", Home: javaHome}); err != nil {
		t.Fatal(err)
	}

	cfg := Config{Home: jmvHome, Mirror: DefaultMirror, MavenMirror: DefaultMavenMirror}
	var out bytes.Buffer
	if err := envPrint(cfg, nil, &out); err != nil {
		t.Fatalf("env print: %v", err)
	}
	got := out.String()
	for _, want := range []string{
		"# Bash (~/.bashrc)",
		"# Zsh (~/.zshrc)",
		"# Fish (~/.config/fish/config.fish)",
		`export JAVA_HOME="` + javaHome + `"`,
		`set -gx JAVA_HOME "` + javaHome + `"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("env print missing %q:\n%s", want, got)
		}
	}
}

func TestEnvPrintFiltersSingleShell(t *testing.T) {
	disableProfileMutation(t)
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	javaHome := installDir(jmvHome, RuntimeJDK, "17")
	t.Setenv("JMV_HOME", jmvHome)
	t.Setenv("JAVA_HOME", "")
	if err := writeCurrent(jmvHome, Current{Runtime: RuntimeJDK, Major: "17", Home: javaHome}); err != nil {
		t.Fatal(err)
	}

	cfg := Config{Home: jmvHome, Mirror: DefaultMirror, MavenMirror: DefaultMavenMirror}
	var out bytes.Buffer
	if err := envPrint(cfg, []string{"--shell", "bash"}, &out); err != nil {
		t.Fatalf("env print --shell bash: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "# bash") {
		t.Fatalf("expected bash header in output:\n%s", got)
	}
	if strings.Contains(got, "# zsh") || strings.Contains(got, "# fish") {
		t.Fatalf("expected only bash block, got:\n%s", got)
	}
	if !strings.Contains(got, `export JAVA_HOME="`+javaHome+`"`) {
		t.Fatalf("missing JAVA_HOME export:\n%s", got)
	}
}

func TestEnvPrintAcceptsPositionalShellNames(t *testing.T) {
	disableProfileMutation(t)
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	javaHome := installDir(jmvHome, RuntimeJDK, "17")
	t.Setenv("JMV_HOME", jmvHome)
	t.Setenv("JAVA_HOME", "")
	if err := writeCurrent(jmvHome, Current{Runtime: RuntimeJDK, Major: "17", Home: javaHome}); err != nil {
		t.Fatal(err)
	}

	cfg := Config{Home: jmvHome, Mirror: DefaultMirror, MavenMirror: DefaultMavenMirror}

	for _, name := range []string{"bash", "zsh", "fish"} {
		var out bytes.Buffer
		if err := envPrint(cfg, []string{name}, &out); err != nil {
			t.Fatalf("env print %s: %v", name, err)
		}
		got := out.String()
		if !strings.Contains(got, "# "+name) {
			t.Fatalf("expected #%s header, got:\n%s", name, got)
		}
		if !strings.Contains(got, javaHome) {
			t.Fatalf("output missing JDK home %s:\n%s", javaHome, got)
		}
	}
}

func TestEnvPrintRejectsUnknownShellUniformly(t *testing.T) {
	disableProfileMutation(t)
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	t.Setenv("JMV_HOME", jmvHome)
	t.Setenv("JAVA_HOME", "")

	cfg := Config{Home: jmvHome, Mirror: DefaultMirror, MavenMirror: DefaultMavenMirror}
	for _, args := range [][]string{{"--shell", "tcsh"}, {"tcsh"}} {
		var out bytes.Buffer
		err := envPrint(cfg, args, &out)
		if err == nil {
			t.Fatalf("expected usage error for args %v, got nil", args)
		}
		if !strings.HasPrefix(err.Error(), "usage:") {
			t.Fatalf("expected usage-prefixed error for args %v, got: %v", args, err)
		}
	}
}

func TestEnvJavaHomePrintsDefaultJDKPath(t *testing.T) {
	disableProfileMutation(t)
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	javaHome := installDir(jmvHome, RuntimeJDK, "17")
	t.Setenv("JMV_HOME", jmvHome)
	if err := writeCurrent(jmvHome, Current{Runtime: RuntimeJDK, Major: "17", Home: javaHome}); err != nil {
		t.Fatal(err)
	}

	cfg := Config{Home: jmvHome, Mirror: DefaultMirror, MavenMirror: DefaultMavenMirror}
	var out bytes.Buffer
	if err := envJavaHome(cfg, &out); err != nil {
		t.Fatalf("env java-home: %v", err)
	}
	if strings.TrimSpace(out.String()) != javaHome {
		t.Fatalf("expected %s, got %s", javaHome, strings.TrimSpace(out.String()))
	}
}

func TestEnvJavaHomeEmptyWhenNoDefaultJDKRuntime(t *testing.T) {
	disableProfileMutation(t)
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	t.Setenv("JMV_HOME", jmvHome)

	cfg := Config{Home: jmvHome, Mirror: DefaultMirror, MavenMirror: DefaultMavenMirror}
	var out bytes.Buffer
	if err := envJavaHome(cfg, &out); err != nil {
		t.Fatalf("env java-home: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Fatalf("expected empty output, got %q", out.String())
	}
}

func TestEnvJavaHomeEmptyForNonJDKDefault(t *testing.T) {
	disableProfileMutation(t)
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	mavenHome := installDir(jmvHome, RuntimeMaven, "3.9.11")
	t.Setenv("JMV_HOME", jmvHome)
	if err := writeCurrent(jmvHome, Current{Runtime: RuntimeMaven, Major: "3.9.11", Home: mavenHome}); err != nil {
		t.Fatal(err)
	}

	cfg := Config{Home: jmvHome, Mirror: DefaultMirror, MavenMirror: DefaultMavenMirror}
	var out bytes.Buffer
	if err := envJavaHome(cfg, &out); err != nil {
		t.Fatalf("env java-home: %v", err)
	}
	if strings.TrimSpace(out.String()) != "" {
		t.Fatalf("expected empty output for maven default, got %q", out.String())
	}
}
