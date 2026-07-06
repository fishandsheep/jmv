package jmv

import "testing"

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
