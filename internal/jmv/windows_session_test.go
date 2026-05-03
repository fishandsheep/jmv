package jmv

import (
	"runtime"
	"testing"
)

func TestResolveCurrentWindowsGlobalSessionFallback(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows-only behavior")
	}

	home := t.TempDir()
	def := Current{Runtime: RuntimeJDK, Major: "17", Home: "C:/jdk-17"}
	if err := writeCurrent(home, def); err != nil {
		t.Fatal(err)
	}

	override := Current{Runtime: RuntimeJDK, Major: "21", Home: "C:/jdk-21"}
	if err := writeSession(home, globalSessionPID, override); err != nil {
		t.Fatal(err)
	}

	cur, err := resolveCurrent(home, 123456)
	if err != nil {
		t.Fatal(err)
	}
	if cur.Major != "21" {
		t.Fatalf("expected global session override 21, got %s", cur.Major)
	}
}
