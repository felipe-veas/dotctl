package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenRejectsEmptyRepoURL(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	configHome := filepath.Join(base, "config")
	configPath := filepath.Join(base, "config.yaml")

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	if err := os.MkdirAll(configHome, 0o755); err != nil {
		t.Fatalf("mkdir config home: %v", err)
	}
	if err := os.WriteFile(configPath, []byte("repo:\n  url: \"\"\n  path: /tmp/repo\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	_, err := executeCLI(t, "open", "--config", configPath)
	if err == nil {
		t.Fatal("expected open to fail for empty repo URL")
	}
	if !strings.Contains(err.Error(), "GitHub SSH or HTTPS repo URL") {
		t.Fatalf("unexpected error: %v", err)
	}
}
