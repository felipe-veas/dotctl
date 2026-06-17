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

func TestBackupsRestoreTargetRestoresOnlySelectedFile(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	configHome := filepath.Join(base, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	snapshot := "20260101-010101.000013"
	selected := filepath.Join(home, "notes.txt")
	other := filepath.Join(home, "todo.txt")
	selectedBackup := filepath.Join(configHome, "dotctl", "backups", snapshot, "targets", strings.TrimPrefix(selected, string(filepath.Separator)))
	otherBackup := filepath.Join(configHome, "dotctl", "backups", snapshot, "targets", strings.TrimPrefix(other, string(filepath.Separator)))
	for _, path := range []string{selectedBackup, otherBackup} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir backup path: %v", err)
		}
	}
	if err := os.WriteFile(selectedBackup, []byte("restored\n"), 0o644); err != nil {
		t.Fatalf("write selected backup: %v", err)
	}
	if err := os.WriteFile(otherBackup, []byte("other-backup\n"), 0o644); err != nil {
		t.Fatalf("write other backup: %v", err)
	}
	if err := os.WriteFile(selected, []byte("current\n"), 0o644); err != nil {
		t.Fatalf("write selected target: %v", err)
	}
	if err := os.WriteFile(other, []byte("keep\n"), 0o644); err != nil {
		t.Fatalf("write other target: %v", err)
	}

	if _, err := executeCLI(t, "backups", "restore", snapshot, "--target", selected, "--force"); err != nil {
		t.Fatalf("restore with target failed: %v", err)
	}

	data, err := os.ReadFile(selected)
	if err != nil {
		t.Fatalf("read selected target: %v", err)
	}
	if string(data) != "restored\n" {
		t.Fatalf("selected target = %q", string(data))
	}
	data, err = os.ReadFile(other)
	if err != nil {
		t.Fatalf("read other target: %v", err)
	}
	if string(data) != "keep\n" {
		t.Fatalf("other target changed: %q", string(data))
	}
}

func TestBackupsRestoreTargetAcceptsTildeInput(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	configHome := filepath.Join(base, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	snapshot := "20260101-010101.000014"
	target := filepath.Join(home, ".zshrc")
	backupPath := filepath.Join(configHome, "dotctl", "backups", snapshot, "targets", strings.TrimPrefix(target, string(filepath.Separator)))
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		t.Fatalf("mkdir backup path: %v", err)
	}
	if err := os.WriteFile(backupPath, []byte("restored-from-tilde\n"), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}
	if err := os.WriteFile(target, []byte("current\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	if _, err := executeCLI(t, "backups", "restore", snapshot, "--target", "~/.zshrc", "--force"); err != nil {
		t.Fatalf("restore with tilde target failed: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read restored target: %v", err)
	}
	if string(data) != "restored-from-tilde\n" {
		t.Fatalf("restored target = %q", string(data))
	}
}

func TestBackupsRestoreTargetDryRunDoesNotMutate(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	configHome := filepath.Join(base, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	snapshot := "20260101-010101.000015"
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

	if _, err := executeCLI(t, "backups", "restore", snapshot, "--target", target, "--dry-run"); err != nil {
		t.Fatalf("restore dry-run with target failed: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "current\n" {
		t.Fatalf("target mutated: %q", string(data))
	}
}

func TestBackupsRestoreTargetJSONIncludesTargets(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	configHome := filepath.Join(base, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	snapshot := "20260101-010101.000018"
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

	raw, err := executeCLI(t, "backups", "restore", snapshot, "--target", target, "--dry-run", "--json")
	if err != nil {
		t.Fatalf("restore dry-run with json failed: %v", err)
	}

	var payload struct {
		Snapshot string   `json:"snapshot"`
		DryRun   bool     `json:"dry_run"`
		Targets  []string `json:"targets"`
		Entries  []struct {
			Status string `json:"status"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("parse restore json: %v\nraw: %s", err, raw)
	}
	if payload.Snapshot != snapshot {
		t.Fatalf("snapshot = %q, want %q", payload.Snapshot, snapshot)
	}
	if !payload.DryRun {
		t.Fatal("expected dry_run to be true")
	}
	if len(payload.Targets) != 1 || payload.Targets[0] != target {
		t.Fatalf("targets = %#v", payload.Targets)
	}
	if len(payload.Entries) != 1 || payload.Entries[0].Status != "planned" {
		t.Fatalf("entries = %#v", payload.Entries)
	}
}

func TestBackupsRestoreTargetRejectsMissingTarget(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	configHome := filepath.Join(base, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	snapshot := "20260101-010101.000016"
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

	_, err := executeCLI(t, "backups", "restore", snapshot, "--target", "~/missing.txt", "--force")
	if err == nil {
		t.Fatal("expected missing target restore to fail")
	}
	if !strings.Contains(err.Error(), "does not contain target") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "dotctl backups list") {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "current\n" {
		t.Fatalf("target mutated: %q", string(data))
	}
}

func TestBackupsRestoreTargetRejectsOutsideHome(t *testing.T) {
	base := t.TempDir()
	home := filepath.Join(base, "home")
	configHome := filepath.Join(base, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	snapshot := "20260101-010101.000017"
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

	_, err := executeCLI(t, "backups", "restore", snapshot, "--target", "/etc/passwd", "--dry-run")
	if err == nil {
		t.Fatal("expected outside-home target restore to fail")
	}
	if !strings.Contains(err.Error(), "must stay under home directory") {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "current\n" {
		t.Fatalf("target mutated: %q", string(data))
	}
}
