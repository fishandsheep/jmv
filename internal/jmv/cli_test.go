package jmv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseRuntimeDefaultsToJDK(t *testing.T) {
	rt, rest, err := parseRuntime([]string{"17"})
	if err != nil {
		t.Fatal(err)
	}
	if rt != RuntimeJDK {
		t.Fatalf("expected default runtime jdk, got %s", rt)
	}
	if len(rest) != 1 || rest[0] != "17" {
		t.Fatalf("unexpected rest args: %#v", rest)
	}
}

func TestParseRuntimeRejectsInvalidValue(t *testing.T) {
	rt, rest, err := parseRuntime([]string{"-r", "17"})
	if err == nil {
		t.Fatalf("expected invalid runtime error, got rt=%s rest=%#v", rt, rest)
	}
	if err.Error() != "runtime must be jdk or jre" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseRuntimeFlagWithValue(t *testing.T) {
	rt, rest, err := parseRuntime([]string{"--runtime", "jdk", "17"})
	if err != nil {
		t.Fatal(err)
	}
	if rt != RuntimeJDK {
		t.Fatalf("expected runtime jdk, got %s", rt)
	}
	if len(rest) != 1 || rest[0] != "17" {
		t.Fatalf("unexpected rest args: %#v", rest)
	}
}

func TestSamePath(t *testing.T) {
	tmp := t.TempDir()
	linked := t.TempDir()
	if err := os.Symlink(linked, filepath.Join(tmp, "alias")); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}

	cases := []struct {
		name string
		a, b string
		want bool
	}{
		{name: "identical", a: linked, b: linked, want: true},
		{name: "trailing slash matters not", a: linked + "/", b: linked, want: true},
		{name: "symlink resolves to target", a: filepath.Join(tmp, "alias"), b: linked, want: true},
		{name: "different paths", a: linked, b: filepath.Join(tmp, "elsewhere"), want: false},
		{name: "exists vs nonexistent lex different", a: linked, b: "/nonexistent/path/xyz", want: false},
		{name: "both nonexistent lexically equal still true", a: "/nonexistent/path/a", b: "/nonexistent/path/a", want: true},
		{name: "both nonexistent lexically different false", a: "/nonexistent/path/a", b: "/nonexistent/path/b", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := samePath(tc.a, tc.b); got != tc.want {
				t.Fatalf("samePath(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
