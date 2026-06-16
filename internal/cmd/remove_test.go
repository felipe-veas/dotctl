package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipe-veas/dotctl/internal/config"
	"github.com/felipe-veas/dotctl/internal/manifest"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func TestBuildRemovePlanRejectsOutsideHome(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("x\n"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	if _, err := buildRemovePlan(outside, home, home, repo); err == nil {
		t.Fatal("expected error for path outside home")
	}
}

func TestBuildRemovePlanRejectsHomeRoot(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	if _, err := buildRemovePlan(home, home, home, repo); err == nil {
		t.Fatal("expected error for home directory root")
	}
	if _, err := buildRemovePlan("~", home, home, repo); err == nil {
		t.Fatal("expected error for tilde home directory root")
	}
}

func TestBuildRemovePlanAllowsMissingTarget(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	target := filepath.Join(home, ".missing-dotfile")
	plan, err := buildRemovePlan(target, home, home, repo)
	if err != nil {
		t.Fatalf("buildRemovePlan: %v", err)
	}
	if plan.AbsTarget != filepath.Clean(target) {
		t.Fatalf("AbsTarget = %q, want %q", plan.AbsTarget, filepath.Clean(target))
	}
}

func TestBuildRemovePlanRejectsRepoOverlap(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, ".config", "dotctl", "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	configDir := filepath.Join(home, ".config")
	if _, err := buildRemovePlan(configDir, home, home, repo); err == nil {
		t.Fatal("expected error for path containing repo")
	}
	if _, err := buildRemovePlan(repo, home, home, repo); err == nil {
		t.Fatal("expected error for repo path itself")
	}
}

func TestRunRemoveUpdatesManifestAndManagedSources(t *testing.T) {
	defer restoreRemoveFlags()()
	flagJSON = true
	flagDryRun = false
	flagForce = false

	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.MkdirAll(filepath.Join(repo, "configs", "zsh"), 0o755); err != nil {
		t.Fatalf("mkdir repo source dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(repo, "configs", "git"), 0o755); err != nil {
		t.Fatalf("mkdir repo source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "configs", "zsh", ".zshrc"), []byte("zsh\n"), 0o644); err != nil {
		t.Fatalf("write repo zsh source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "configs", "git", ".gitconfig"), []byte("git\n"), 0o644); err != nil {
		t.Fatalf("write repo git source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("local zsh\n"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}
	linkPath := filepath.Join(home, ".gitconfig")
	if err := os.Symlink(filepath.Join(repo, "configs", "git", ".gitconfig"), linkPath); err != nil {
		t.Fatalf("create symlink target: %v", err)
	}

	manifestPath := filepath.Join(repo, "manifest.yaml")
	manifestData, err := yaml.Marshal(&manifest.Manifest{
		Version: 1,
		Files: []manifest.FileEntry{
			{Source: "configs/zsh/.zshrc", Target: "~/.zshrc"},
			{Source: "configs/git/.gitconfig", Target: "~/.gitconfig"},
		},
	})
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := writeManagedSources(repo, []string{"configs/zsh/.zshrc", "configs/git/.gitconfig"}); err != nil {
		t.Fatalf("write managed sources: %v", err)
	}

	t.Setenv("HOME", home)
	if err := config.Save(configPath, &config.Config{Repo: config.RepoConfig{URL: "git@example.com:dotfiles.git", Path: repo}}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	flagConfig = configPath

	if err := runRemove(&cobra.Command{}, []string{filepath.Join(home, ".zshrc")}); err != nil {
		t.Fatalf("runRemove: %v", err)
	}

	updatedManifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	m, err := manifest.Parse(updatedManifest)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if len(m.Files) != 1 {
		t.Fatalf("manifest file count = %d, want 1", len(m.Files))
	}
	if m.Files[0].Source != "configs/git/.gitconfig" {
		t.Fatalf("unexpected remaining manifest entry: %+v", m.Files[0])
	}

	managed, err := readManagedSources(repo)
	if err != nil {
		t.Fatalf("read managed sources: %v", err)
	}
	if len(managed) != 1 || managed[0] != "configs/git/.gitconfig" {
		t.Fatalf("managed sources = %v", managed)
	}
	removedTargetData, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatalf("read removed local target: %v", err)
	}
	if string(removedTargetData) != "local zsh\n" {
		t.Fatalf("removed local target mutated: %q", string(removedTargetData))
	}
	removedRepoSourceData, err := os.ReadFile(filepath.Join(repo, "configs", "zsh", ".zshrc"))
	if err != nil {
		t.Fatalf("read removed repo source: %v", err)
	}
	if string(removedRepoSourceData) != "zsh\n" {
		t.Fatalf("removed repo source mutated: %q", string(removedRepoSourceData))
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatalf("lstat target symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("expected local target symlink to remain untouched")
	}
	linkTarget, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("readlink target: %v", err)
	}
	if linkTarget != filepath.Join(repo, "configs", "git", ".gitconfig") {
		t.Fatalf("symlink target = %q", linkTarget)
	}
}

func TestRunRemoveDryRunDoesNotMutate(t *testing.T) {
	defer restoreRemoveFlags()()
	flagJSON = true
	flagDryRun = true
	flagForce = false

	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.MkdirAll(filepath.Join(repo, "configs", "zsh"), 0o755); err != nil {
		t.Fatalf("mkdir repo source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "configs", "zsh", ".zshrc"), []byte("zsh\n"), 0o644); err != nil {
		t.Fatalf("write repo zsh source: %v", err)
	}
	manifestPath := filepath.Join(repo, "manifest.yaml")
	manifestData, err := yaml.Marshal(&manifest.Manifest{
		Version: 1,
		Files:   []manifest.FileEntry{{Source: "configs/zsh/.zshrc", Target: "~/.zshrc"}},
	})
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := writeManagedSources(repo, []string{"configs/zsh/.zshrc"}); err != nil {
		t.Fatalf("write managed sources: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("local zsh\n"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}

	t.Setenv("HOME", home)
	if err := config.Save(configPath, &config.Config{Repo: config.RepoConfig{URL: "git@example.com:dotfiles.git", Path: repo}}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	flagConfig = configPath

	if err := runRemove(&cobra.Command{}, []string{filepath.Join(home, ".zshrc")}); err != nil {
		t.Fatalf("runRemove dry-run: %v", err)
	}

	dataAfter, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest after dry-run: %v", err)
	}
	if string(dataAfter) != string(manifestData) {
		t.Fatalf("manifest mutated during dry-run")
	}
	managed, err := readManagedSources(repo)
	if err != nil {
		t.Fatalf("read managed sources: %v", err)
	}
	if len(managed) != 1 || managed[0] != "configs/zsh/.zshrc" {
		t.Fatalf("managed sources mutated during dry-run: %v", managed)
	}
}

func TestRunRemoveMissingManifest(t *testing.T) {
	defer restoreRemoveFlags()()
	flagJSON = true
	flagDryRun = false
	flagForce = false

	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("local zsh\n"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}

	t.Setenv("HOME", home)
	if err := config.Save(configPath, &config.Config{Repo: config.RepoConfig{URL: "git@example.com:dotfiles.git", Path: repo}}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	flagConfig = configPath

	err := runRemove(&cobra.Command{}, []string{filepath.Join(home, ".zshrc")})
	if err == nil || !strings.Contains(err.Error(), "manifest not found") {
		t.Fatalf("error = %v, want manifest not found", err)
	}
}

func TestRunRemoveNoMatchingEntry(t *testing.T) {
	defer restoreRemoveFlags()()
	flagJSON = true
	flagDryRun = false
	flagForce = false

	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.MkdirAll(filepath.Join(repo, "configs", "git"), 0o755); err != nil {
		t.Fatalf("mkdir repo source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "configs", "git", ".gitconfig"), []byte("git\n"), 0o644); err != nil {
		t.Fatalf("write repo source: %v", err)
	}
	manifestPath := filepath.Join(repo, "manifest.yaml")
	manifestData, err := yaml.Marshal(&manifest.Manifest{
		Version: 1,
		Files:   []manifest.FileEntry{{Source: "configs/git/.gitconfig", Target: "~/.gitconfig"}},
	})
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, manifestData, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("local zsh\n"), 0o644); err != nil {
		t.Fatalf("write target file: %v", err)
	}

	t.Setenv("HOME", home)
	if err := config.Save(configPath, &config.Config{Repo: config.RepoConfig{URL: "git@example.com:dotfiles.git", Path: repo}}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	flagConfig = configPath

	err = runRemove(&cobra.Command{}, []string{filepath.Join(home, ".zshrc")})
	if err == nil || !strings.Contains(err.Error(), "manifest has no entry targeting") {
		t.Fatalf("error = %v, want no matching entry", err)
	}
}

func restoreRemoveFlags() func() {
	oldJSON := flagJSON
	oldDryRun := flagDryRun
	oldForce := flagForce
	oldConfig := flagConfig
	return func() {
		flagJSON = oldJSON
		flagDryRun = oldDryRun
		flagForce = oldForce
		flagConfig = oldConfig
	}
}
