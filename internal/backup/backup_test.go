package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
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

func TestCreateWritesMetadataForFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "testfile.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	configHome := filepath.Join(dir, "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)

	backupPath, err := Create(src)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	snapshot := snapshotFromBackupPath(t, configHome, backupPath)
	metadata := readSnapshotMetadata(t, snapshot)
	if metadata.Version != snapshotMetadataVersion {
		t.Fatalf("metadata version = %d, want %d", metadata.Version, snapshotMetadataVersion)
	}
	if metadata.CreatedAt.IsZero() {
		t.Fatal("metadata created_at is zero")
	}
	if len(metadata.Entries) != 1 {
		t.Fatalf("metadata entries = %d, want 1", len(metadata.Entries))
	}
	entry := metadata.Entries[0]
	if entry.Target != metadataTargetPath(src) {
		t.Fatalf("metadata target = %q, want %q", entry.Target, metadataTargetPath(src))
	}
	if entry.BackupPath != backupPath {
		t.Fatalf("metadata backup_path = %q, want %q", entry.BackupPath, backupPath)
	}
	if entry.Kind != "file" {
		t.Fatalf("metadata kind = %q, want file", entry.Kind)
	}
	if entry.Mode == "" {
		t.Fatal("metadata mode is empty")
	}
}

func TestCreateWritesMetadataForDirectoryAndRestoreUsesMetadata(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	configHome := filepath.Join(dir, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	targetDir := filepath.Join(home, ".config", "nvim")
	if err := os.MkdirAll(filepath.Join(targetDir, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "init.lua"), []byte("print('hello')\n"), 0o644); err != nil {
		t.Fatalf("write init.lua: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "sub", "plugin.lua"), []byte("return {}\n"), 0o644); err != nil {
		t.Fatalf("write plugin.lua: %v", err)
	}

	endSession := BeginSession()
	defer endSession()

	backupPath, err := Create(targetDir)
	if err != nil {
		t.Fatalf("Create dir: %v", err)
	}

	snapshot := snapshotFromBackupPath(t, configHome, backupPath)
	metadata := readSnapshotMetadata(t, snapshot)
	if len(metadata.Entries) != 1 {
		t.Fatalf("metadata entries = %d, want 1", len(metadata.Entries))
	}
	entry := metadata.Entries[0]
	if entry.Kind != "dir" {
		t.Fatalf("metadata kind = %q, want dir", entry.Kind)
	}

	entries, err := SnapshotEntries(snapshot)
	if err != nil {
		t.Fatalf("SnapshotEntries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("snapshot entries = %d, want 1", len(entries))
	}
	if entries[0].Kind != "dir" {
		t.Fatalf("snapshot entry kind = %q, want dir", entries[0].Kind)
	}
	if entries[0].BackupPath != backupPath {
		t.Fatalf("snapshot entry backup_path = %q, want %q", entries[0].BackupPath, backupPath)
	}

	if err := os.WriteFile(filepath.Join(targetDir, "old.txt"), []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "init.lua"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("mutate init.lua: %v", err)
	}

	results, err := RestoreSnapshot(snapshot, false)
	if err != nil {
		t.Fatalf("RestoreSnapshot: %v", err)
	}
	if len(results) != 1 || results[0].Status != "restored" {
		t.Fatalf("results = %#v", results)
	}

	data, err := os.ReadFile(filepath.Join(targetDir, "init.lua"))
	if err != nil {
		t.Fatalf("read restored init.lua: %v", err)
	}
	if string(data) != "print('hello')\n" {
		t.Fatalf("restored init.lua = %q", string(data))
	}
	if _, err := os.Stat(filepath.Join(targetDir, "old.txt")); !os.IsNotExist(err) {
		t.Fatalf("old.txt should be removed, err=%v", err)
	}
}

func TestRestoreSnapshotTargetsRestoresMetadataBackedDirectoryOnly(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	configHome := filepath.Join(dir, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	targetDir := filepath.Join(home, ".config", "nvim")
	if err := os.MkdirAll(filepath.Join(targetDir, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "init.lua"), []byte("print('hello')\n"), 0o644); err != nil {
		t.Fatalf("write init.lua: %v", err)
	}
	otherTarget := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(otherTarget, []byte("keep\n"), 0o644); err != nil {
		t.Fatalf("write other target: %v", err)
	}

	endSession := BeginSession()
	defer endSession()

	backupPath, err := Create(targetDir)
	if err != nil {
		t.Fatalf("Create dir: %v", err)
	}

	snapshot := snapshotFromBackupPath(t, configHome, backupPath)
	if err := os.WriteFile(filepath.Join(targetDir, "init.lua"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("mutate init.lua: %v", err)
	}
	if err := os.WriteFile(filepath.Join(targetDir, "stale.txt"), []byte("stale\n"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}
	if err := os.WriteFile(otherTarget, []byte("keep-updated\n"), 0o644); err != nil {
		t.Fatalf("mutate other target: %v", err)
	}

	results, err := RestoreSnapshotTargets(snapshot, []string{targetDir}, false)
	if err != nil {
		t.Fatalf("RestoreSnapshotTargets: %v", err)
	}
	if len(results) != 1 || results[0].Status != "restored" {
		t.Fatalf("results = %#v", results)
	}

	data, err := os.ReadFile(filepath.Join(targetDir, "init.lua"))
	if err != nil {
		t.Fatalf("read restored init.lua: %v", err)
	}
	if string(data) != "print('hello')\n" {
		t.Fatalf("restored init.lua = %q", string(data))
	}
	if _, err := os.Stat(filepath.Join(targetDir, "stale.txt")); !os.IsNotExist(err) {
		t.Fatalf("stale.txt should be removed, err=%v", err)
	}
	data, err = os.ReadFile(otherTarget)
	if err != nil {
		t.Fatalf("read other target: %v", err)
	}
	if string(data) != "keep-updated\n" {
		t.Fatalf("other target mutated: %q", string(data))
	}
}

func TestRestoreSnapshotTargetsAcceptsRelativePathWithinWorkingDir(t *testing.T) {
	dir := t.TempDir()
	home := filepath.Join(dir, "home")
	configHome := filepath.Join(dir, "config")
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	target := filepath.Join(home, "notes.txt")
	if err := os.WriteFile(target, []byte("current\n"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	backupPath := filepath.Join(configHome, "dotctl", "backups", "20260101-010101.000013", "targets", strings.TrimPrefix(target, string(filepath.Separator)))
	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
		t.Fatalf("mkdir backup path: %v", err)
	}
	if err := os.WriteFile(backupPath, []byte("backup\n"), 0o644); err != nil {
		t.Fatalf("write backup: %v", err)
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(home); err != nil {
		t.Fatalf("chdir home: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})

	results, err := RestoreSnapshotTargets("20260101-010101.000013", []string{"notes.txt"}, true)
	if err != nil {
		t.Fatalf("RestoreSnapshotTargets dry-run: %v", err)
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

func TestCreateAppendsMetadataForRepeatedTargets(t *testing.T) {
	dir := t.TempDir()
	configHome := filepath.Join(dir, "config")
	t.Setenv("XDG_CONFIG_HOME", configHome)

	src := filepath.Join(dir, "target.txt")
	if err := os.WriteFile(src, []byte("one"), 0o644); err != nil {
		t.Fatalf("write source #1: %v", err)
	}

	endSession := BeginSession()
	defer endSession()

	backupOne, err := Create(src)
	if err != nil {
		t.Fatalf("Create #1: %v", err)
	}

	if err := os.WriteFile(src, []byte("two"), 0o644); err != nil {
		t.Fatalf("write source #2: %v", err)
	}
	backupTwo, err := Create(src)
	if err != nil {
		t.Fatalf("Create #2: %v", err)
	}

	snapshot := snapshotFromBackupPath(t, configHome, backupOne)
	if snapshot != snapshotFromBackupPath(t, configHome, backupTwo) {
		t.Fatal("expected repeated targets to share a snapshot")
	}

	metadata := readSnapshotMetadata(t, snapshot)
	if len(metadata.Entries) != 2 {
		t.Fatalf("metadata entries = %d, want 2", len(metadata.Entries))
	}
	if metadata.Entries[0].BackupPath != backupOne {
		t.Fatalf("entry[0] backup_path = %q, want %q", metadata.Entries[0].BackupPath, backupOne)
	}
	if metadata.Entries[1].BackupPath != backupTwo {
		t.Fatalf("entry[1] backup_path = %q, want %q", metadata.Entries[1].BackupPath, backupTwo)
	}
	if backupOne == backupTwo {
		t.Fatal("expected unique backup paths")
	}
	if !strings.HasPrefix(backupTwo, backupOne+"~") {
		t.Fatalf("backupTwo = %q, expected suffix on %q", backupTwo, backupOne)
	}
}

func TestSnapshotEntriesFallsBackWithoutMetadata(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))

	snapshot := "20260101-010101.000050"
	root := filepath.Join(dir, "config", "dotctl", "backups", snapshot, "targets")
	if err := os.MkdirAll(filepath.Join(root, "home", "alice", ".config", "nvim"), 0o755); err != nil {
		t.Fatalf("mkdir legacy tree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "home", "alice", ".config", "nvim", "init.lua"), []byte("print('nvim')\n"), 0o644); err != nil {
		t.Fatalf("write init.lua: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "home", "alice", ".config", "nvim", "lua"), 0o755); err != nil {
		t.Fatalf("mkdir lua dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "home", "alice", ".config", "nvim", "lua", "plugins.lua"), []byte("return {}\n"), 0o644); err != nil {
		t.Fatalf("write plugins.lua: %v", err)
	}

	entries, err := SnapshotEntries(snapshot)
	if err != nil {
		t.Fatalf("SnapshotEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
	for _, entry := range entries {
		if entry.Kind != "file" {
			t.Fatalf("entry kind = %q, want file", entry.Kind)
		}
	}
}

func TestSnapshotEntriesRejectsMetadataOutsideSnapshotDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))

	snapshot := "20260101-010101.000051"
	snapshotDir := filepath.Join(dir, "config", "dotctl", "backups", snapshot)
	if err := os.MkdirAll(filepath.Join(snapshotDir, "targets"), 0o755); err != nil {
		t.Fatalf("mkdir snapshot dir: %v", err)
	}
	metadata := snapshotMetadata{
		Version:   snapshotMetadataVersion,
		CreatedAt: time.Now().UTC(),
		Entries: []snapshotMetadataEntry{{
			Target:     filepath.Join("/", "home", "alice", ".zshrc"),
			BackupPath: filepath.Join(dir, "outside", "evil.txt"),
			Kind:       "file",
			Mode:       "-rw-r--r--",
		}},
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}
	if err := os.WriteFile(metadataFilePath(snapshot), append(data, '\n'), 0o600); err != nil {
		t.Fatalf("write metadata: %v", err)
	}
	if _, err := SnapshotEntries(snapshot); err == nil {
		t.Fatal("expected metadata containment error")
	}
}

func readSnapshotMetadata(t *testing.T, snapshot string) snapshotMetadata {
	t.Helper()
	data, err := os.ReadFile(metadataFilePath(snapshot))
	if err != nil {
		t.Fatalf("read metadata: %v", err)
	}
	var metadata snapshotMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		t.Fatalf("decode metadata: %v", err)
	}
	return metadata
}

func snapshotFromBackupPath(t *testing.T, configHome, backupPath string) string {
	t.Helper()
	backupRoot := filepath.Join(configHome, "dotctl", "backups")
	rel, err := filepath.Rel(backupRoot, backupPath)
	if err != nil {
		t.Fatalf("rel backup path: %v", err)
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) == 0 || parts[0] == "" {
		t.Fatalf("unexpected backup path layout: %q", rel)
	}
	return parts[0]
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
