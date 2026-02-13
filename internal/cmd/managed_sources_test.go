package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felipe-veas/dotctl/internal/manifest"
)

func TestWriteReadManagedSources(t *testing.T) {
	repo := t.TempDir()

	if err := writeManagedSources(repo, []string{
		"configs/zsh/.zshrc",
		"configs/zsh/.zshrc",
		"../unsafe",
		"configs/git/.gitconfig",
	}); err != nil {
		t.Fatalf("writeManagedSources: %v", err)
	}

	got, err := readManagedSources(repo)
	if err != nil {
		t.Fatalf("readManagedSources: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("managed source count = %d, want 2", len(got))
	}
	if got[0] != "configs/git/.gitconfig" || got[1] != "configs/zsh/.zshrc" {
		t.Fatalf("managed sources = %v", got)
	}
}

func TestPruneManagedSourcesRemovesStalePaths(t *testing.T) {
	repo := t.TempDir()

	zshPath := filepath.Join(repo, "configs", "zsh", ".zshrc")
	gitPath := filepath.Join(repo, "configs", "git", ".gitconfig")
	if err := os.MkdirAll(filepath.Dir(zshPath), 0o755); err != nil {
		t.Fatalf("mkdir zsh dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(gitPath), 0o755); err != nil {
		t.Fatalf("mkdir git dir: %v", err)
	}
	if err := os.WriteFile(zshPath, []byte("zsh"), 0o644); err != nil {
		t.Fatalf("write zsh: %v", err)
	}
	if err := os.WriteFile(gitPath, []byte("git"), 0o644); err != nil {
		t.Fatalf("write git: %v", err)
	}

	if err := writeManagedSources(repo, []string{"configs/zsh/.zshrc", "configs/git/.gitconfig"}); err != nil {
		t.Fatalf("writeManagedSources: %v", err)
	}

	results, err := pruneManagedSources(repo, []manifest.FileEntry{
		{Source: "configs/zsh/.zshrc", Target: "~/.zshrc"},
	}, false)
	if err != nil {
		t.Fatalf("pruneManagedSources: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("prune results = %d, want 1", len(results))
	}
	if results[0].Source != "configs/git/.gitconfig" || results[0].Status != "removed" {
		t.Fatalf("unexpected prune result: %+v", results[0])
	}

	if _, err := os.Stat(gitPath); !os.IsNotExist(err) {
		t.Fatalf("expected git source removed, stat err = %v", err)
	}
	if _, err := os.Stat(zshPath); err != nil {
		t.Fatalf("expected zsh source kept: %v", err)
	}

	managed, err := readManagedSources(repo)
	if err != nil {
		t.Fatalf("readManagedSources: %v", err)
	}
	if len(managed) != 1 || managed[0] != "configs/zsh/.zshrc" {
		t.Fatalf("managed sources after prune = %v", managed)
	}
}

func TestPruneManagedSourcesDryRunDoesNotDelete(t *testing.T) {
	repo := t.TempDir()
	gitPath := filepath.Join(repo, "configs", "git", ".gitconfig")
	if err := os.MkdirAll(filepath.Dir(gitPath), 0o755); err != nil {
		t.Fatalf("mkdir git dir: %v", err)
	}
	if err := os.WriteFile(gitPath, []byte("git"), 0o644); err != nil {
		t.Fatalf("write git: %v", err)
	}
	if err := writeManagedSources(repo, []string{"configs/git/.gitconfig"}); err != nil {
		t.Fatalf("writeManagedSources: %v", err)
	}

	results, err := pruneManagedSources(repo, []manifest.FileEntry{}, true)
	if err != nil {
		t.Fatalf("pruneManagedSources: %v", err)
	}
	if len(results) != 1 || results[0].Status != "would_remove" {
		t.Fatalf("unexpected dry-run prune results: %v", results)
	}

	if _, err := os.Stat(gitPath); err != nil {
		t.Fatalf("expected git source to remain during dry-run: %v", err)
	}
}

func TestPruneManagedSourcesKeepsParentWhenChildActive(t *testing.T) {
	repo := t.TempDir()
	tmuxPath := filepath.Join(repo, "configs", "tmux", "tmux.conf")
	if err := os.MkdirAll(filepath.Dir(tmuxPath), 0o755); err != nil {
		t.Fatalf("mkdir tmux dir: %v", err)
	}
	if err := os.WriteFile(tmuxPath, []byte("tmux"), 0o644); err != nil {
		t.Fatalf("write tmux config: %v", err)
	}
	if err := writeManagedSources(repo, []string{"configs/tmux"}); err != nil {
		t.Fatalf("writeManagedSources: %v", err)
	}

	results, err := pruneManagedSources(repo, []manifest.FileEntry{
		{Source: "configs/tmux/tmux.conf", Target: "~/.config/tmux/tmux.conf"},
	}, false)
	if err != nil {
		t.Fatalf("pruneManagedSources: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no prune results, got %v", results)
	}

	if _, err := os.Stat(tmuxPath); err != nil {
		t.Fatalf("expected tmux source kept: %v", err)
	}
}

func TestPruneManagedSourcesKeepsChildWhenParentActive(t *testing.T) {
	repo := t.TempDir()
	tmuxPath := filepath.Join(repo, "configs", "tmux", "tmux.conf")
	if err := os.MkdirAll(filepath.Dir(tmuxPath), 0o755); err != nil {
		t.Fatalf("mkdir tmux dir: %v", err)
	}
	if err := os.WriteFile(tmuxPath, []byte("tmux"), 0o644); err != nil {
		t.Fatalf("write tmux config: %v", err)
	}
	if err := writeManagedSources(repo, []string{"configs/tmux/tmux.conf"}); err != nil {
		t.Fatalf("writeManagedSources: %v", err)
	}

	results, err := pruneManagedSources(repo, []manifest.FileEntry{
		{Source: "configs/tmux", Target: "~/.config/tmux"},
	}, false)
	if err != nil {
		t.Fatalf("pruneManagedSources: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no prune results, got %v", results)
	}

	if _, err := os.Stat(tmuxPath); err != nil {
		t.Fatalf("expected tmux source kept: %v", err)
	}
}

func TestBackfillMissingSourcesFromTargetsCopiesFile(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()

	targetFile := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(targetFile, []byte("export Z=1\n"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}

	results, err := backfillMissingSourcesFromTargets(repo, []manifest.Action{
		{
			Source: "configs/zsh/.zshrc",
			Target: targetFile,
			Mode:   "symlink",
		},
	}, false)
	if err != nil {
		t.Fatalf("backfillMissingSourcesFromTargets: %v", err)
	}
	if len(results) != 1 || results[0].Status != "copied_from_target" {
		t.Fatalf("unexpected backfill results: %+v", results)
	}

	repoSource := filepath.Join(repo, "configs", "zsh", ".zshrc")
	data, err := os.ReadFile(repoSource)
	if err != nil {
		t.Fatalf("read backfilled source: %v", err)
	}
	if string(data) != "export Z=1\n" {
		t.Fatalf("backfilled content = %q", string(data))
	}

	managed, err := readManagedSources(repo)
	if err != nil {
		t.Fatalf("readManagedSources: %v", err)
	}
	if len(managed) != 1 || managed[0] != "configs/zsh/.zshrc" {
		t.Fatalf("managed sources after backfill = %v", managed)
	}
}

func TestBackfillMissingSourcesFromTargetsDryRun(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()

	targetFile := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(targetFile, []byte("export Z=1\n"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}

	results, err := backfillMissingSourcesFromTargets(repo, []manifest.Action{
		{
			Source: "configs/zsh/.zshrc",
			Target: targetFile,
			Mode:   "symlink",
		},
	}, true)
	if err != nil {
		t.Fatalf("backfillMissingSourcesFromTargets: %v", err)
	}
	if len(results) != 1 || results[0].Status != "would_copy_from_target" {
		t.Fatalf("unexpected backfill results: %+v", results)
	}

	repoSource := filepath.Join(repo, "configs", "zsh", ".zshrc")
	if _, err := os.Stat(repoSource); !os.IsNotExist(err) {
		t.Fatalf("expected no source file in dry-run, stat err = %v", err)
	}
}
