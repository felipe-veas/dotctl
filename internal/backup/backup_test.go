package backup

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestCreateBackupFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "testfile.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	// Override backup dir via XDG
	backupDir := filepath.Join(dir, "backups")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))

	backupPath, err := Create(src)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_ = backupDir

	if backupPath == "" {
		t.Fatal("backupPath is empty")
	}

	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("backup content = %q, want %q", string(data), "hello")
	}
}

func TestCreateBackupDir(t *testing.T) {
	dir := t.TempDir()
	srcDir := filepath.Join(dir, "mydir")
	if err := os.MkdirAll(filepath.Join(srcDir, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir source subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "a.txt"), []byte("aaa"), 0o644); err != nil {
		t.Fatalf("write source a.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "sub", "b.txt"), []byte("bbb"), 0o644); err != nil {
		t.Fatalf("write source sub/b.txt: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))

	backupPath, err := Create(srcDir)
	if err != nil {
		t.Fatalf("Create dir: %v", err)
	}

	// Verify files exist in backup
	data, err := os.ReadFile(filepath.Join(backupPath, "a.txt"))
	if err != nil {
		t.Fatalf("reading backed up a.txt: %v", err)
	}
	if string(data) != "aaa" {
		t.Errorf("a.txt = %q, want %q", string(data), "aaa")
	}

	data, err = os.ReadFile(filepath.Join(backupPath, "sub", "b.txt"))
	if err != nil {
		t.Fatalf("reading backed up sub/b.txt: %v", err)
	}
	if string(data) != "bbb" {
		t.Errorf("sub/b.txt = %q, want %q", string(data), "bbb")
	}
}

func TestCreateBackupSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "realfile")
	if err := os.WriteFile(target, []byte("content"), 0o644); err != nil {
		t.Fatalf("write symlink target: %v", err)
	}

	link := filepath.Join(dir, "mylink")
	if err := os.Symlink(target, link); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))

	backupPath, err := Create(link)
	if err != nil {
		t.Fatalf("Create symlink: %v", err)
	}

	linkDest, err := os.Readlink(backupPath)
	if err != nil {
		t.Fatalf("readlink backup: %v", err)
	}
	if linkDest != target {
		t.Errorf("backup link dest = %q, want %q", linkDest, target)
	}
}

func TestCreateNonexistent(t *testing.T) {
	_, err := Create("/nonexistent/file")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestRotateKeepsLatestSnapshots(t *testing.T) {
	dir := t.TempDir()
	configHome := filepath.Join(dir, "config")
	backupsDir := filepath.Join(configHome, "dotctl", "backups")
	t.Setenv("XDG_CONFIG_HOME", configHome)

	snapshots := []string{
		"20260101-010101.000001",
		"20260101-010101.000002",
		"20260101-010101.000003",
	}
	for _, snap := range snapshots {
		path := filepath.Join(backupsDir, snap)
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatalf("mkdir snapshot %s: %v", snap, err)
		}
		if err := os.WriteFile(filepath.Join(path, "a.txt"), []byte("x"), 0o644); err != nil {
			t.Fatalf("write snapshot file: %v", err)
		}
	}

	result, err := Rotate(2)
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}
	if result.Kept != 2 {
		t.Fatalf("Kept = %d, want 2", result.Kept)
	}
	if result.Removed != 1 {
		t.Fatalf("Removed = %d, want 1", result.Removed)
	}

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		t.Fatalf("read backups dir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) != 2 {
		t.Fatalf("remaining snapshots = %d, want 2 (%v)", len(names), names)
	}
	if names[0] != "20260101-010101.000002" || names[1] != "20260101-010101.000003" {
		t.Fatalf("remaining snapshots = %v, want [..000002 ..000003]", names)
	}
}
