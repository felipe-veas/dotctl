package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBackupsListJSONEmpty(t *testing.T) {
	base := t.TempDir()
	t.Setenv("HOME", filepath.Join(base, "home"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(base, "config"))

	raw, err := executeCLI(t, "backups", "list", "--json")
	if err != nil {
		t.Fatalf("backups list failed: %v", err)
	}

	var payload struct {
		BackupDir string `json:"backup_dir"`
		Snapshots []struct {
			Name    string `json:"name"`
			Entries int    `json:"entries"`
		} `json:"snapshots"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("parse list json: %v\nraw: %s", err, raw)
	}
	if payload.BackupDir == "" {
		t.Fatal("expected backup_dir in list output")
	}
	if len(payload.Snapshots) != 0 {
		t.Fatalf("expected empty snapshot list, got %#v", payload.Snapshots)
	}
}

func TestBackupsRestoreRequiresForce(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	configHome := filepath.Join(base, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	snapshot := "20260101-010101.000010"
	target := filepath.Join(home, "notes.txt")
	backupPath := filepath.Join(configHome, "dotctl", "backups", snapshot, "targets", strings.TrimPrefix(target, string(filepath.Separator)))
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		t.Fatalf("mkdir backup path: %v", err)
	}
	if err := os.WriteFile(backupPath, []byte("backup\n"), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	if err := os.WriteFile(target, []byte("current\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	_, err := executeCLI(t, "backups", "restore", snapshot)
	if err == nil {
		t.Fatal("expected restore without --force to fail")
	}
	if !strings.Contains(err.Error(), "rerun with --force") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBackupsRestoreDryRunDoesNotMutate(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	configHome := filepath.Join(base, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	snapshot := "20260101-010101.000011"
	target := filepath.Join(home, "notes.txt")
	backupPath := filepath.Join(configHome, "dotctl", "backups", snapshot, "targets", strings.TrimPrefix(target, string(filepath.Separator)))
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		t.Fatalf("mkdir backup path: %v", err)
	}
	if err := os.WriteFile(backupPath, []byte("backup\n"), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}
	if err := os.WriteFile(target, []byte("current\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	_, err := executeCLI(t, "backups", "restore", snapshot, "--dry-run")
	if err != nil {
		t.Fatalf("restore dry-run failed: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "current\n" {
		t.Fatalf("target mutated: %q", string(data))
	}
}

func TestBackupsRestoreForceRestoresFile(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	configHome := filepath.Join(base, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	snapshot := "20260101-010101.000012"
	target := filepath.Join(home, "notes.txt")
	backupPath := filepath.Join(configHome, "dotctl", "backups", snapshot, "targets", strings.TrimPrefix(target, string(filepath.Separator)))
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		t.Fatalf("mkdir backup path: %v", err)
	}
	if err := os.WriteFile(backupPath, []byte("restored\n"), 0o600); err != nil {
		t.Fatalf("write backup: %v", err)
	}
	if err := os.WriteFile(target, []byte("current\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	if _, err := executeCLI(t, "backups", "restore", snapshot, "--force"); err != nil {
		t.Fatalf("restore failed: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read restored target: %v", err)
	}
	if string(data) != "restored\n" {
		t.Fatalf("restored target = %q", string(data))
	}
}
