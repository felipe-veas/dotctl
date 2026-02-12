package linker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipe-veas/dotctl/internal/manifest"
)

// setupRepo creates a fake repo with source files.
func setupRepo(t *testing.T) (repoRoot string, targetDir string) {
	t.Helper()
	dir := t.TempDir()
	repoRoot = filepath.Join(dir, "repo")
	targetDir = filepath.Join(dir, "home")

	// Set XDG so backups go to temp dir
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config"))

	if err := os.MkdirAll(filepath.Join(repoRoot, "configs", "zsh"), 0o755); err != nil {
		t.Fatalf("mkdir zsh dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repoRoot, "configs", "nvim", "lua"), 0o755); err != nil {
		t.Fatalf("mkdir nvim dir: %v", err)
	}
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}

	if err := os.WriteFile(filepath.Join(repoRoot, "configs", "zsh", ".zshrc"), []byte("# zshrc"), 0o644); err != nil {
		t.Fatalf("write zshrc: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "configs", "nvim", "init.lua"), []byte("-- nvim"), 0o644); err != nil {
		t.Fatalf("write init.lua: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "configs", "nvim", "lua", "plugins.lua"), []byte("-- plugins"), 0o644); err != nil {
		t.Fatalf("write plugins.lua: %v", err)
	}

	return repoRoot, targetDir
}

func TestApplySymlinkCreate(t *testing.T) {
	repoRoot, targetDir := setupRepo(t)

	actions := []manifest.Action{
		{Source: "configs/zsh/.zshrc", Target: filepath.Join(targetDir, ".zshrc"), Mode: "symlink", Backup: true},
	}

	results := Apply(actions, repoRoot, false)
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}

	r := results[0]
	if r.Status != "created" {
		t.Errorf("status = %q, want %q", r.Status, "created")
	}
	if r.Error != nil {
		t.Errorf("error = %v", r.Error)
	}

	// Verify symlink exists
	link, err := os.Readlink(filepath.Join(targetDir, ".zshrc"))
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	expected := filepath.Join(repoRoot, "configs", "zsh", ".zshrc")
	if link != expected {
		t.Errorf("link = %q, want %q", link, expected)
	}
}

func TestApplySymlinkIdempotent(t *testing.T) {
	repoRoot, targetDir := setupRepo(t)

	actions := []manifest.Action{
		{Source: "configs/zsh/.zshrc", Target: filepath.Join(targetDir, ".zshrc"), Mode: "symlink", Backup: true},
	}

	// First apply
	Apply(actions, repoRoot, false)

	// Second apply — should be idempotent
	results := Apply(actions, repoRoot, false)
	if results[0].Status != "already_linked" {
		t.Errorf("second apply status = %q, want %q", results[0].Status, "already_linked")
	}
}

func TestApplySymlinkWithBackup(t *testing.T) {
	repoRoot, targetDir := setupRepo(t)

	// Create an existing regular file at target
	targetPath := filepath.Join(targetDir, ".zshrc")
	if err := os.WriteFile(targetPath, []byte("old content"), 0o644); err != nil {
		t.Fatalf("write existing target: %v", err)
	}

	actions := []manifest.Action{
		{Source: "configs/zsh/.zshrc", Target: targetPath, Mode: "symlink", Backup: true},
	}

	results := Apply(actions, repoRoot, false)
	r := results[0]

	if r.Status != "backed_up" {
		t.Errorf("status = %q, want %q", r.Status, "backed_up")
	}
	if r.BackupPath == "" {
		t.Error("BackupPath should not be empty")
	}

	// Verify backup content
	data, err := os.ReadFile(r.BackupPath)
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if string(data) != "old content" {
		t.Errorf("backup = %q, want %q", string(data), "old content")
	}

	// Verify symlink was created
	link, err := os.Readlink(targetPath)
	if err != nil {
		t.Fatalf("readlink: %v", err)
	}
	if link != filepath.Join(repoRoot, "configs", "zsh", ".zshrc") {
		t.Errorf("link = %q", link)
	}
}

func TestApplyDryRun(t *testing.T) {
	repoRoot, targetDir := setupRepo(t)

	actions := []manifest.Action{
		{Source: "configs/zsh/.zshrc", Target: filepath.Join(targetDir, ".zshrc"), Mode: "symlink", Backup: true},
	}

	results := Apply(actions, repoRoot, true)
	if results[0].Status != "would_create" {
		t.Errorf("dry-run status = %q, want %q", results[0].Status, "would_create")
	}

	// Verify nothing was created
	if _, err := os.Lstat(filepath.Join(targetDir, ".zshrc")); err == nil {
		t.Error("file should not exist in dry-run mode")
	}
}

func TestApplyDryRunWithExisting(t *testing.T) {
	repoRoot, targetDir := setupRepo(t)

	targetPath := filepath.Join(targetDir, ".zshrc")
	if err := os.WriteFile(targetPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write existing target for dry-run: %v", err)
	}

	actions := []manifest.Action{
		{Source: "configs/zsh/.zshrc", Target: targetPath, Mode: "symlink", Backup: true},
	}

	results := Apply(actions, repoRoot, true)
	if results[0].Status != "would_backup_and_link" {
		t.Errorf("dry-run status = %q, want %q", results[0].Status, "would_backup_and_link")
	}

	// Verify original file is untouched
	data, _ := os.ReadFile(targetPath)
	if string(data) != "existing" {
		t.Error("original file should be untouched in dry-run")
	}
}

func TestApplyCopy(t *testing.T) {
	repoRoot, targetDir := setupRepo(t)

	actions := []manifest.Action{
		{Source: "configs/zsh/.zshrc", Target: filepath.Join(targetDir, ".zshrc"), Mode: "copy", Backup: true},
	}

	results := Apply(actions, repoRoot, false)
	if results[0].Status != "copied" {
		t.Errorf("status = %q, want %q", results[0].Status, "copied")
	}

	// Verify it's a regular file, not a symlink
	info, err := os.Lstat(filepath.Join(targetDir, ".zshrc"))
	if err != nil {
		t.Fatalf("lstat: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("copy mode should create a regular file, not a symlink")
	}

	data, _ := os.ReadFile(filepath.Join(targetDir, ".zshrc"))
	if string(data) != "# zshrc" {
		t.Errorf("copy content = %q, want %q", string(data), "# zshrc")
	}
}

func TestApplyCopyDir(t *testing.T) {
	repoRoot, targetDir := setupRepo(t)

	actions := []manifest.Action{
		{Source: "configs/nvim", Target: filepath.Join(targetDir, ".config", "nvim"), Mode: "copy", Backup: true},
	}

	results := Apply(actions, repoRoot, false)
	if results[0].Status != "copied" {
		t.Errorf("status = %q, want %q (error: %v)", results[0].Status, "copied", results[0].Error)
	}

	// Verify files were copied
	data, err := os.ReadFile(filepath.Join(targetDir, ".config", "nvim", "init.lua"))
	if err != nil {
		t.Fatalf("reading copied file: %v", err)
	}
	if string(data) != "-- nvim" {
		t.Errorf("copied content = %q, want %q", string(data), "-- nvim")
	}
}

func TestApplySourceMissing(t *testing.T) {
	repoRoot, targetDir := setupRepo(t)

	actions := []manifest.Action{
		{Source: "configs/nonexistent", Target: filepath.Join(targetDir, "x"), Mode: "symlink"},
	}

	results := Apply(actions, repoRoot, false)
	if results[0].Status != "error" {
		t.Errorf("status = %q, want %q", results[0].Status, "error")
	}
}

func TestApplyCreatesParentDirs(t *testing.T) {
	repoRoot, targetDir := setupRepo(t)

	actions := []manifest.Action{
		{Source: "configs/zsh/.zshrc", Target: filepath.Join(targetDir, "deep", "nested", ".zshrc"), Mode: "symlink"},
	}

	results := Apply(actions, repoRoot, false)
	if results[0].Status != "created" {
		t.Errorf("status = %q, want %q (error: %v)", results[0].Status, "created", results[0].Error)
	}
}

func TestSummarize(t *testing.T) {
	results := []Result{
		{Status: "created"},
		{Status: "created"},
		{Status: "already_linked"},
		{Status: "backed_up"},
		{Status: "error"},
	}

	s := Summarize(results)
	if s.Created != 2 {
		t.Errorf("Created = %d, want 2", s.Created)
	}
	if s.AlreadyOK != 1 {
		t.Errorf("AlreadyOK = %d, want 1", s.AlreadyOK)
	}
	if s.BackedUp != 1 {
		t.Errorf("BackedUp = %d, want 1", s.BackedUp)
	}
	if s.Errors != 1 {
		t.Errorf("Errors = %d, want 1", s.Errors)
	}
}

func TestApplySymlinkWrongTarget(t *testing.T) {
	repoRoot, targetDir := setupRepo(t)

	targetPath := filepath.Join(targetDir, ".zshrc")

	// Create a symlink pointing to the wrong place
	if err := os.Symlink("/some/other/path", targetPath); err != nil {
		t.Fatalf("create wrong symlink: %v", err)
	}

	actions := []manifest.Action{
		{Source: "configs/zsh/.zshrc", Target: targetPath, Mode: "symlink", Backup: true},
	}

	results := Apply(actions, repoRoot, false)
	r := results[0]

	if r.Status != "backed_up" {
		t.Errorf("status = %q, want %q", r.Status, "backed_up")
	}

	// Verify the new symlink points to the right place
	link, _ := os.Readlink(targetPath)
	expected := filepath.Join(repoRoot, "configs", "zsh", ".zshrc")
	if link != expected {
		t.Errorf("link = %q, want %q", link, expected)
	}
}

func TestRollbackCreatedRemovesTarget(t *testing.T) {
	repoRoot, targetDir := setupRepo(t)
	targetPath := filepath.Join(targetDir, ".zshrc")

	actions := []manifest.Action{
		{Source: "configs/zsh/.zshrc", Target: targetPath, Mode: "symlink", Backup: true},
	}
	results := Apply(actions, repoRoot, false)
	if results[0].Status != "created" {
		t.Fatalf("status = %q, want created", results[0].Status)
	}

	rollback := Rollback(results)
	if len(rollback) != 1 {
		t.Fatalf("rollback len = %d, want 1", len(rollback))
	}
	if rollback[0].Status != "removed" {
		t.Fatalf("rollback status = %q, want removed", rollback[0].Status)
	}
	if _, err := os.Lstat(targetPath); !os.IsNotExist(err) {
		t.Fatalf("target should be removed after rollback, err=%v", err)
	}
}

func TestRollbackBackedUpRestoresOriginalFile(t *testing.T) {
	repoRoot, targetDir := setupRepo(t)
	targetPath := filepath.Join(targetDir, ".zshrc")
	if err := os.WriteFile(targetPath, []byte("original"), 0o644); err != nil {
		t.Fatalf("write original target: %v", err)
	}

	actions := []manifest.Action{
		{Source: "configs/zsh/.zshrc", Target: targetPath, Mode: "symlink", Backup: true},
	}
	results := Apply(actions, repoRoot, false)
	if results[0].Status != "backed_up" {
		t.Fatalf("status = %q, want backed_up", results[0].Status)
	}

	rollback := Rollback(results)
	if len(rollback) != 1 {
		t.Fatalf("rollback len = %d, want 1", len(rollback))
	}
	if rollback[0].Status != "restored" {
		t.Fatalf("rollback status = %q, want restored", rollback[0].Status)
	}

	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read restored target: %v", err)
	}
	if string(data) != "original" {
		t.Fatalf("restored content = %q, want original", string(data))
	}
}

func TestApplySourceSymlinkLoop(t *testing.T) {
	repoRoot, targetDir := setupRepo(t)

	loopSource := filepath.Join(repoRoot, "configs", "loop")
	if err := os.Symlink(loopSource, loopSource); err != nil {
		t.Fatalf("create source symlink loop: %v", err)
	}

	actions := []manifest.Action{
		{Source: "configs/loop", Target: filepath.Join(targetDir, "loop-target"), Mode: "symlink", Backup: true},
	}
	results := Apply(actions, repoRoot, false)
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Status != "error" {
		t.Fatalf("status = %q, want error", results[0].Status)
	}
	if results[0].Error == nil || !strings.Contains(strings.ToLower(results[0].Error.Error()), "symlink loop") {
		t.Fatalf("expected symlink loop error, got: %v", results[0].Error)
	}
}

func TestApplyPermissionDeniedTarget(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission test is not reliable when running as root")
	}

	repoRoot, targetDir := setupRepo(t)
	blockedDir := filepath.Join(targetDir, "blocked")
	if err := os.MkdirAll(blockedDir, 0o755); err != nil {
		t.Fatalf("mkdir blocked dir: %v", err)
	}
	if err := os.Chmod(blockedDir, 0o555); err != nil {
		t.Fatalf("chmod blocked dir read-only: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(blockedDir, 0o755)
	})

	actions := []manifest.Action{
		{Source: "configs/zsh/.zshrc", Target: filepath.Join(blockedDir, ".zshrc"), Mode: "symlink", Backup: true},
	}
	results := Apply(actions, repoRoot, false)
	if len(results) != 1 {
		t.Fatalf("results = %d, want 1", len(results))
	}
	if results[0].Status != "error" {
		t.Fatalf("status = %q, want error", results[0].Status)
	}
	if results[0].Error == nil || !strings.Contains(strings.ToLower(results[0].Error.Error()), "permission denied") {
		t.Fatalf("expected permission denied error, got: %v", results[0].Error)
	}
}
