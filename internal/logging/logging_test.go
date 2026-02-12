package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPathUsesStateDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", filepath.Join(t.TempDir(), "state"))
	got := Path()
	wantSuffix := filepath.Join("dotctl", "dotctl.log")
	if !strings.HasSuffix(got, wantSuffix) {
		t.Fatalf("Path() = %q, want suffix %q", got, wantSuffix)
	}
}

func TestInitCreatesLogFile(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state")
	t.Setenv("XDG_STATE_HOME", state)

	if err := Init(true); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	t.Cleanup(func() { _ = Close() })

	Info("hello", "token", "ghp_aaaaaaaa")

	if _, err := os.Stat(Path()); err != nil {
		t.Fatalf("expected log file at %s: %v", Path(), err)
	}
}

func TestRedact(t *testing.T) {
	const sample = "ghp_aaaaaaaa"

	got := redact("token " + sample + " should be hidden")
	if strings.Contains(got, sample) {
		t.Fatalf("expected token to be redacted, got: %s", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Fatalf("expected [REDACTED] marker, got: %s", got)
	}
}
