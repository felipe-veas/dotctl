package backup

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestCreateBackupFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "testfile.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	// Override backup dir via XDG
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))

	backupPath, err := Create(src)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

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

func TestCreateSessionGroupsBackupsIntoSingleSnapshot(t *testing.T) {
	dir := t.TempDir()
	configHome := filepath.Join(dir, "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)

	srcA := filepath.Join(dir, "targets", "a.txt")
	srcB := filepath.Join(dir, "targets", "sub", "b.txt")
	if err := os.MkdirAll(filepath.Dir(srcB), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.WriteFile(srcA, []byte("A"), 0o644); err != nil {
		t.Fatalf("write srcA: %v", err)
	}
	if err := os.WriteFile(srcB, []byte("B"), 0o644); err != nil {
		t.Fatalf("write srcB: %v", err)
	}

	endSession := BeginSession()
	defer endSession()

	pathA, err := Create(srcA)
	if err != nil {
		t.Fatalf("Create srcA: %v", err)
	}
	pathB, err := Create(srcB)
	if err != nil {
		t.Fatalf("Create srcB: %v", err)
	}

	backupRoot := filepath.Join(configHome, "dotctl", "backups")
	relA, err := filepath.Rel(backupRoot, pathA)
	if err != nil {
		t.Fatalf("rel pathA: %v", err)
	}
	relB, err := filepath.Rel(backupRoot, pathB)
	if err != nil {
		t.Fatalf("rel pathB: %v", err)
	}

	partsA := strings.Split(relA, string(filepath.Separator))
	partsB := strings.Split(relB, string(filepath.Separator))
	if len(partsA) < 3 || len(partsB) < 3 {
		t.Fatalf("unexpected backup path layout: %q %q", relA, relB)
	}
	if partsA[0] != partsB[0] {
		t.Fatalf("expected same snapshot, got %q and %q", partsA[0], partsB[0])
	}

	wantA := filepath.Join(backupRoot, partsA[0], "targets", targetRelativePath(srcA))
	wantB := filepath.Join(backupRoot, partsB[0], "targets", targetRelativePath(srcB))
	if pathA != wantA {
		t.Fatalf("pathA = %q, want %q", pathA, wantA)
	}
	if pathB != wantB {
		t.Fatalf("pathB = %q, want %q", pathB, wantB)
	}
}

func TestCreateSessionStoresRepeatedTargetAsUniquePaths(t *testing.T) {
	dir := t.TempDir()
	configHome := filepath.Join(dir, "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)

	src := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(src, []byte("one"), 0o644); err != nil {
		t.Fatalf("write src #1: %v", err)
	}

	endSession := BeginSession()
	defer endSession()

	backupOne, err := Create(src)
	if err != nil {
		t.Fatalf("Create #1: %v", err)
	}

	if err := os.WriteFile(src, []byte("two"), 0o644); err != nil {
		t.Fatalf("write src #2: %v", err)
	}
	backupTwo, err := Create(src)
	if err != nil {
		t.Fatalf("Create #2: %v", err)
	}

	if backupOne == backupTwo {
		t.Fatalf("expected unique paths for repeated target, got %q", backupOne)
	}
	if !strings.HasPrefix(backupTwo, backupOne+"~") {
		t.Fatalf("backupTwo = %q, expected suffix on %q", backupTwo, backupOne)
	}

	dataOne, err := os.ReadFile(backupOne)
	if err != nil {
		t.Fatalf("read backupOne: %v", err)
	}
	if string(dataOne) != "one" {
		t.Fatalf("backupOne content = %q, want %q", string(dataOne), "one")
	}

	dataTwo, err := os.ReadFile(backupTwo)
	if err != nil {
		t.Fatalf("read backupTwo: %v", err)
	}
	if string(dataTwo) != "two" {
		t.Fatalf("backupTwo content = %q, want %q", string(dataTwo), "two")
	}
}

func TestRestorePathRestoresDirectoryContents(t *testing.T) {
	dir := t.TempDir()
	backupDir := filepath.Join(dir, "backup-dir")
	targetDir := filepath.Join(dir, "target-dir")
	if err := os.MkdirAll(filepath.Join(backupDir, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir backup subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backupDir, "sub", "file.txt"), []byte("restored\n"), 0o644); err != nil {
		t.Fatalf("write backup file: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "old.txt"), []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write old target file: %v", err)
	}

	if err := RestorePath(backupDir, targetDir); err != nil {
		t.Fatalf("RestorePath: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old target file should be removed, err=%v", err)
	}
	data, err := os.ReadFile(filepath.Join(targetDir, "sub", "file.txt"))
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(data) != "restored\n" {
		t.Fatalf("restored file content = %q", string(data))
	}
}

func TestTargetFromBackupPathOnlyStripsSuffixFromLeaf(t *testing.T) {
	dir := t.TempDir()
	targetsRoot := filepath.Join(dir, "targets")
	backupPath := filepath.Join(targetsRoot, "Users", "me~1", ".zshrc~2")

	got := targetFromBackupPath(targetsRoot, backupPath)
	want := filepath.Join(string(filepath.Separator), "Users", "me~1", ".zshrc")
	if got != want {
		t.Fatalf("target = %q, want %q", got, want)
	}
}

func TestValidateSnapshotNameRejectsTraversal(t *testing.T) {
	for _, name := range []string{"", "..", ".", "../evil", "snap/child", "/abs"} {
		if err := validateSnapshotName(name); err == nil {
			t.Fatalf("expected validation error for %q", name)
		}
	}
}

func TestListReturnsNewestSnapshotsFirst(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))

	for _, snap := range []string{"20260101-010101.000001", "20260101-010101.000003"} {
		root := filepath.Join(dir, "config", "dotctl", "backups", snap, "targets")
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("mkdir snapshot %s: %v", snap, err)
		}
		filePath := filepath.Join(root, "home", "alice", ".zshrc")
		if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
			t.Fatalf("mkdir snapshot file dir: %v", err)
		}
		if err := os.WriteFile(filePath, []byte("zsh\n"), 0o644); err != nil {
			t.Fatalf("write snapshot file: %v", err)
		}
	}

	snapshots, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(snapshots) != 2 {
		t.Fatalf("snapshot count = %d, want 2", len(snapshots))
	}
	if snapshots[0].Name != "20260101-010101.000003" || snapshots[1].Name != "20260101-010101.000001" {
		t.Fatalf("snapshots = %#v", snapshots)
	}
}

func TestRestoreSnapshotDryRunDoesNotMutate(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	configHome := filepath.Join(dir, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	snapshot := "20260101-010101.000001"
	target := filepath.Join(home, "notes.txt")
	if err := os.WriteFile(target, []byte("current\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	backupPath := filepath.Join(configHome, "dotctl", "backups", snapshot, "targets", strings.TrimPrefix(target, string(filepath.Separator)))
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		t.Fatalf("mkdir backup path: %v", err)
	}
	if err := os.WriteFile(backupPath, []byte("backup\n"), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	results, err := RestoreSnapshot(snapshot, true)
	if err != nil {
		t.Fatalf("RestoreSnapshot dry-run: %v", err)
	}
	if len(results) != 1 || results[0].Status != "planned" {
		t.Fatalf("results = %#v", results)
	}

	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(data) != "current\n" {
		t.Fatalf("target mutated: %q", string(data))
	}
}

func TestRestoreSnapshotOverwritesFileAndPreservesSymlink(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	configHome := filepath.Join(dir, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}

	snapshot := "20260101-010101.000002"
	fileTarget := filepath.Join(home, "notes.txt")
	if err := os.WriteFile(fileTarget, []byte("current\n"), 0o644); err != nil {
		t.Fatalf("write file target: %v", err)
	}
	linkTarget := filepath.Join(home, ".profile-link")
	if err := os.WriteFile(filepath.Join(dir, "profile"), []byte("profile\n"), 0o644); err != nil {
		t.Fatalf("write profile file: %v", err)
	}
	if err := os.Symlink(filepath.Join(dir, "profile"), linkTarget); err != nil {
		t.Fatalf("create local symlink: %v", err)
	}

	backupRoot := filepath.Join(configHome, "dotctl", "backups", snapshot, "targets")
	fileBackup := filepath.Join(backupRoot, strings.TrimPrefix(fileTarget, string(filepath.Separator)))
	if err := os.MkdirAll(filepath.Dir(fileBackup), 0o755); err != nil {
		t.Fatalf("mkdir file backup: %v", err)
	}
	if err := os.WriteFile(fileBackup, []byte("restored\n"), 0o600); err != nil {
		t.Fatalf("write file backup: %v", err)
	}
	linkBackup := filepath.Join(backupRoot, strings.TrimPrefix(linkTarget, string(filepath.Separator)))
	if err := os.MkdirAll(filepath.Dir(linkBackup), 0o755); err != nil {
		t.Fatalf("mkdir link backup: %v", err)
	}
	if err := os.Symlink(filepath.Join(dir, "profile"), linkBackup); err != nil {
		t.Fatalf("create link backup: %v", err)
	}

	results, err := RestoreSnapshot(snapshot, false)
	if err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %#v", results)
	}

	data, err := os.ReadFile(fileTarget)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if string(data) != "restored\n" {
		t.Fatalf("restored file = %q", string(data))
	}
	if gotMode, err := os.Lstat(linkTarget); err != nil {
		t.Fatalf("lstat restored symlink: %v", err)
	} else if gotMode.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected restored symlink to remain a symlink")
	}
	if dest, err := os.Readlink(linkTarget); err != nil {
		t.Fatalf("readlink restored symlink: %v", err)
	} else if dest != filepath.Join(dir, "profile") {
		t.Fatalf("symlink destination = %q", dest)
	}
}

func TestRestoreSnapshotRejectsOutsideHome(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	configHome := filepath.Join(dir, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	snapshot := "20260101-010101.000003"
	backupPath := filepath.Join(configHome, "dotctl", "backups", snapshot, "targets", "etc", "passwd")
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		t.Fatalf("mkdir backup path: %v", err)
	}
	if err := os.WriteFile(backupPath, []byte("root:x:0:0:root:/root:/bin/sh\n"), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	results, err := RestoreSnapshot(snapshot, true)
	if err == nil {
		t.Fatalf("expected restore error, got results=%#v", results)
	}
	if len(results) != 1 || results[0].Status != "rejected" {
		t.Fatalf("results = %#v", results)
	}
}
