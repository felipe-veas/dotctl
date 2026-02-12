package platform

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestConfigDir(t *testing.T) {
	dir := ConfigDir()
	if dir == "" {
		t.Fatal("ConfigDir returned empty string")
	}
	if !strings.HasSuffix(dir, filepath.Join("dotctl")) {
		t.Errorf("ConfigDir should end with 'dotctl', got: %s", dir)
	}
}

func TestConfigDirXDG(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg-test")
	got := ConfigDir()
	want := "/tmp/xdg-test/dotctl"
	if got != want {
		t.Errorf("ConfigDir with XDG_CONFIG_HOME = %q, want %q", got, want)
	}
}

func TestStateDirXDG(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")
	got := StateDir()
	want := "/tmp/xdg-state/dotctl"
	if got != want {
		t.Errorf("StateDir with XDG_STATE_HOME = %q, want %q", got, want)
	}
}

func TestRepoDir(t *testing.T) {
	dir := RepoDir()
	if !strings.HasSuffix(dir, filepath.Join("dotctl", "repo")) {
		t.Errorf("RepoDir should end with 'dotctl/repo', got: %s", dir)
	}
}

func TestBackupDir(t *testing.T) {
	dir := BackupDir()
	if !strings.HasSuffix(dir, filepath.Join("dotctl", "backups")) {
		t.Errorf("BackupDir should end with 'dotctl/backups', got: %s", dir)
	}
}

func TestStateDirDefault(t *testing.T) {
	// Ensure XDG vars are unset for default behavior
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	if runtime.GOOS == "linux" {
		dir := StateDir()
		if !strings.Contains(dir, ".local/state/dotctl") {
			t.Errorf("StateDir default on Linux should contain '.local/state/dotctl', got: %s", dir)
		}
	} else {
		// On macOS without XDG, StateDir == ConfigDir
		if StateDir() != ConfigDir() {
			t.Errorf("StateDir on macOS should equal ConfigDir")
		}
	}
}
