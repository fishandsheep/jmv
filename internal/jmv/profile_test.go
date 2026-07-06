package jmv

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestMavenDefaultConfiguresBashProfileForMvnLookup(t *testing.T) {
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("SHELL", "/bin/bash")
	t.Setenv("JMV_NO_MODIFY_PROFILE", "0")
	t.Setenv("JAVA_HOME", "")

	mavenHome := installDir(jmvHome, RuntimeMaven, "3.9.11")
	if err := os.MkdirAll(filepath.Join(mavenHome, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mavenHome, "bin", "mvn"), []byte("#!/usr/bin/env sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	release := Release{Runtime: RuntimeMaven, Major: "3.9.11", FileName: "apache-maven-3.9.11-bin.tar.gz", URL: "https://example.test/maven.tgz"}
	if err := writeMetadata(jmvHome, release, mavenHome); err != nil {
		t.Fatal(err)
	}

	cfg := Config{Home: jmvHome, Mirror: DefaultMirror, MavenMirror: DefaultMavenMirror}
	var out bytes.Buffer
	if err := mavenDefault(cfg, "3.9.11", &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Reload your shell environment or open a new terminal") {
		t.Fatalf("missing reload guidance:\n%s", out.String())
	}

	bashrc := filepath.Join(homeDir, ".bashrc")
	data, err := os.ReadFile(bashrc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `export PATH="$JMV_HOME/shims:$PATH"`) {
		t.Fatalf("bash profile missing shims PATH:\n%s", string(data))
	}

	if runtime.GOOS == "windows" {
		return
	}
	cmd := exec.Command("bash", "-lc", "source ~/.bashrc && command -v mvn")
	cmd.Env = append(os.Environ(), "HOME="+homeDir)
	found, err := cmd.Output()
	if err != nil {
		t.Fatalf("mvn not found after sourcing bashrc: %v", err)
	}
	want := filepath.Join(jmvHome, "shims", "mvn")
	if strings.TrimSpace(string(found)) != want {
		t.Fatalf("expected %s, got %s", want, strings.TrimSpace(string(found)))
	}
}

func TestJDKDefaultConfiguresJavaHomeWhenMissing(t *testing.T) {
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("SHELL", "/bin/bash")
	t.Setenv("JMV_NO_MODIFY_PROFILE", "0")
	t.Setenv("JAVA_HOME", "")

	javaHome := installDir(jmvHome, RuntimeJDK, "17")
	if err := os.MkdirAll(filepath.Join(javaHome, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(javaHome, "bin", "java"), []byte("#!/usr/bin/env sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	release := Release{Runtime: RuntimeJDK, Major: "17", FileName: "OpenJDK17.tar.gz", URL: "https://example.test/jdk.tgz"}
	if err := writeMetadata(jmvHome, release, javaHome); err != nil {
		t.Fatal(err)
	}

	cfg := Config{Home: jmvHome, Mirror: DefaultMirror, MavenMirror: DefaultMavenMirror}
	var out bytes.Buffer
	if err := activateDefault(cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}

	bashrc := filepath.Join(homeDir, ".bashrc")
	data, err := os.ReadFile(bashrc)
	if err != nil {
		t.Fatal(err)
	}
	wantLine := `export JAVA_HOME="` + javaHome + `"`
	if !strings.Contains(string(data), wantLine) {
		t.Fatalf("bash profile missing JAVA_HOME %q:\n%s", wantLine, string(data))
	}

	if runtime.GOOS == "windows" {
		return
	}
	cmd := exec.Command("bash", "-lc", "source ~/.bashrc && printf '%s' \"$JAVA_HOME\"")
	cmd.Env = append(os.Environ(), "HOME="+homeDir)
	got, err := cmd.Output()
	if err != nil {
		t.Fatalf("JAVA_HOME not available after sourcing bashrc: %v", err)
	}
	if string(got) != javaHome {
		t.Fatalf("expected JAVA_HOME=%s, got %s", javaHome, string(got))
	}
}

func TestJDKDefaultDoesNotOverrideExistingJavaHome(t *testing.T) {
	homeDir := t.TempDir()
	jmvHome := filepath.Join(homeDir, ".jmv")
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("SHELL", "/bin/bash")
	t.Setenv("JMV_NO_MODIFY_PROFILE", "0")
	t.Setenv("JAVA_HOME", "/existing/java")

	javaHome := installDir(jmvHome, RuntimeJDK, "17")
	if err := os.MkdirAll(filepath.Join(javaHome, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	release := Release{Runtime: RuntimeJDK, Major: "17", FileName: "OpenJDK17.tar.gz", URL: "https://example.test/jdk.tgz"}
	if err := writeMetadata(jmvHome, release, javaHome); err != nil {
		t.Fatal(err)
	}

	cfg := Config{Home: jmvHome, Mirror: DefaultMirror, MavenMirror: DefaultMavenMirror}
	var out bytes.Buffer
	if err := activateDefault(cfg, RuntimeJDK, "17", &out); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(homeDir, ".bashrc"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "JAVA_HOME") {
		t.Fatalf("profile should not write JAVA_HOME when it already exists:\n%s", string(data))
	}
}

func TestConfigureShellEnvironmentWritesZshAndFishProfiles(t *testing.T) {
	for _, tc := range []struct {
		name         string
		shell        string
		profile      string
		wantPath     string
		wantJavaHome string
	}{
		{name: "zsh", shell: "/bin/zsh", profile: ".zshrc", wantPath: `export PATH="$JMV_HOME/shims:$PATH"`, wantJavaHome: `export JAVA_HOME="`},
		{name: "fish", shell: "/usr/bin/fish", profile: filepath.Join(".config", "fish", "config.fish"), wantPath: `fish_add_path "$JMV_HOME/shims"`, wantJavaHome: `set -gx JAVA_HOME "`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			homeDir := t.TempDir()
			jmvHome := filepath.Join(homeDir, ".jmv")
			javaHome := installDir(jmvHome, RuntimeJDK, "17")
			t.Setenv("HOME", homeDir)
			t.Setenv("USERPROFILE", homeDir)
			t.Setenv("SHELL", tc.shell)
			t.Setenv("JMV_NO_MODIFY_PROFILE", "0")
			t.Setenv("JAVA_HOME", "")
			if err := writeCurrent(jmvHome, Current{Runtime: RuntimeJDK, Major: "17", Home: javaHome}); err != nil {
				t.Fatal(err)
			}

			cfg := Config{Home: jmvHome, Mirror: DefaultMirror, MavenMirror: DefaultMavenMirror}
			var out bytes.Buffer
			configureShellEnvironment(cfg, &out)

			data, err := os.ReadFile(filepath.Join(homeDir, tc.profile))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), tc.wantPath) {
				t.Fatalf("profile missing PATH config:\n%s", string(data))
			}
			if !strings.Contains(string(data), tc.wantJavaHome+javaHome+`"`) {
				t.Fatalf("profile missing JAVA_HOME config:\n%s", string(data))
			}
			if !strings.Contains(out.String(), "open a new terminal") {
				t.Fatalf("missing new terminal guidance:\n%s", out.String())
			}
		})
	}
}

func TestConfigureShellEnvironmentPrintsManualConfigForUnknownShell(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)
	t.Setenv("SHELL", "/bin/unknownshell")
	t.Setenv("JMV_NO_MODIFY_PROFILE", "0")
	t.Setenv("JAVA_HOME", "")

	jmvHome := filepath.Join(homeDir, ".jmv")
	javaHome := installDir(jmvHome, RuntimeJDK, "17")
	if err := writeCurrent(jmvHome, Current{Runtime: RuntimeJDK, Major: "17", Home: javaHome}); err != nil {
		t.Fatal(err)
	}
	cfg := Config{Home: jmvHome, Mirror: DefaultMirror, MavenMirror: DefaultMavenMirror}
	var out bytes.Buffer
	configureShellEnvironment(cfg, &out)

	got := out.String()
	for _, want := range []string{
		"# Bash (~/.bashrc)",
		"# Zsh (~/.zshrc)",
		"# Fish (~/.config/fish/config.fish)",
		`export PATH="$JMV_HOME/shims:$PATH"`,
		`export JAVA_HOME="` + javaHome + `"`,
		`fish_add_path "$JMV_HOME/shims"`,
		`set -gx JAVA_HOME "` + javaHome + `"`,
		"open a new terminal",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("manual output missing %q:\n%s", want, got)
		}
	}
}
