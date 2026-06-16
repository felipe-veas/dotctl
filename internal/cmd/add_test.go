package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipe-veas/dotctl/internal/config"
	"github.com/felipe-veas/dotctl/internal/manifest"
	"github.com/spf13/cobra"
)

func TestBuildAddPlan(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	cwd := home
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export Z=1\n"), 0o644); err != nil {
		t.Fatalf("write .zshrc: %v", err)
	}

	plan, err := buildAddPlan(".zshrc", cwd, home, repo)
	if err != nil {
		t.Fatalf("buildAddPlan: %v", err)
	}

	if plan.Source != "configs/zsh/.zshrc" {
		t.Fatalf("source = %q, want configs/zsh/.zshrc", plan.Source)
	}
	if plan.TargetExpr != "~/.zshrc" {
		t.Fatalf("target expr = %q, want ~/.zshrc", plan.TargetExpr)
	}
	wantRepoSource := filepath.Join(repo, "configs", "zsh", ".zshrc")
	if plan.RepoSourcePath != wantRepoSource {
		t.Fatalf("repo source = %q, want %q", plan.RepoSourcePath, wantRepoSource)
	}
	if plan.Sensitive {
		t.Fatal("expected .zshrc to be non-sensitive")
	}
}

func TestBuildAddPlanRejectsOutsideHome(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	outside := filepath.Join(t.TempDir(), "outside.txt")
	if err := os.WriteFile(outside, []byte("x\n"), 0o644); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	if _, err := buildAddPlan(outside, home, home, repo); err == nil {
		t.Fatal("expected error for path outside home")
	}
}

func TestBuildAddPlanRejectsHomeRoot(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	if _, err := buildAddPlan(home, home, home, repo); err == nil {
		t.Fatal("expected error for home directory root")
	}
	if _, err := buildAddPlan("~", home, home, repo); err == nil {
		t.Fatal("expected error for tilde home directory root")
	}
}

func TestBuildAddPlanRejectsRepoOverlap(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(home, ".config", "dotctl", "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	configDir := filepath.Join(home, ".config")
	if _, err := buildAddPlan(configDir, home, home, repo); err == nil {
		t.Fatal("expected error for path containing repo")
	}
	if _, err := buildAddPlan(repo, home, home, repo); err == nil {
		t.Fatal("expected error for repo path itself")
	}
}

func TestValidateAddPlanRejectsSensitiveUnlessForced(t *testing.T) {
	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(filepath.Join(home, ".ssh"), 0o755); err != nil {
		t.Fatalf("mkdir .ssh: %v", err)
	}
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	keyPath := filepath.Join(home, ".ssh", "id_rsa")
	if err := os.WriteFile(keyPath, []byte("key\n"), 0o600); err != nil {
		t.Fatalf("write id_rsa: %v", err)
	}
	plan, err := buildAddPlan(keyPath, home, home, repo)
	if err != nil {
		t.Fatalf("buildAddPlan: %v", err)
	}
	if !plan.Sensitive {
		t.Fatal("expected id_rsa path to be sensitive")
	}
	if _, err := validateAddPlan(plan, false); err == nil {
		t.Fatal("expected sensitive path rejection without force")
	}
	warnings, err := validateAddPlan(plan, true)
	if err != nil {
		t.Fatalf("validateAddPlan(force): %v", err)
	}
	if len(warnings) != 1 || warnings[0] == "" {
		t.Fatalf("unexpected warnings: %#v", warnings)
	}
}

func TestRunAddOnboardsLocalPath(t *testing.T) {
	defer restoreAddFlags()()
	flagJSON = true
	flagDryRun = false
	flagForce = false

	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export Z=1\n"), 0o644); err != nil {
		t.Fatalf("write .zshrc: %v", err)
	}

	t.Setenv("HOME", home)
	if err := config.Save(configPath, &config.Config{Repo: config.RepoConfig{URL: "git@example.com:dotfiles.git", Path: repo}}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	flagConfig = configPath

	if err := runAdd(&cobra.Command{}, []string{filepath.Join(home, ".zshrc")}); err != nil {
		t.Fatalf("runAdd: %v", err)
	}

	repoSource := filepath.Join(repo, "configs", "zsh", ".zshrc")
	data, err := os.ReadFile(repoSource)
	if err != nil {
		t.Fatalf("read repo source: %v", err)
	}
	if string(data) != "export Z=1\n" {
		t.Fatalf("repo source content = %q", string(data))
	}

	target := filepath.Join(home, ".zshrc")
	linkTarget, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("readlink target: %v", err)
	}
	if linkTarget != repoSource {
		t.Fatalf("symlink target = %q, want %q", linkTarget, repoSource)
	}

	manifestPath := filepath.Join(repo, "manifest.yaml")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	m, err := manifest.Parse(manifestData)
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if len(m.Files) != 1 {
		t.Fatalf("manifest file count = %d, want 1", len(m.Files))
	}
	if m.Files[0].Source != "configs/zsh/.zshrc" || m.Files[0].Target != "~/.zshrc" || m.Files[0].LinkMode() != "symlink" {
		t.Fatalf("unexpected manifest entry: %+v", m.Files[0])
	}

	managed, err := os.ReadFile(filepath.Join(repo, ".dotctl", "managed-sources.txt"))
	if err != nil {
		t.Fatalf("read managed sources: %v", err)
	}
	if strings.TrimSpace(string(managed)) != "configs/zsh/.zshrc" {
		t.Fatalf("managed sources = %q", string(managed))
	}
}

func TestRunAddDryRunDoesNotMutate(t *testing.T) {
	defer restoreAddFlags()()
	flagJSON = true
	flagDryRun = true
	flagForce = false

	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export Z=1\n"), 0o644); err != nil {
		t.Fatalf("write .zshrc: %v", err)
	}
	t.Setenv("HOME", home)
	if err := config.Save(configPath, &config.Config{Repo: config.RepoConfig{URL: "git@example.com:dotfiles.git", Path: repo}}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	flagConfig = configPath

	if err := runAdd(&cobra.Command{}, []string{filepath.Join(home, ".zshrc")}); err != nil {
		t.Fatalf("runAdd dry-run: %v", err)
	}

	if _, err := os.Stat(filepath.Join(repo, "configs", "zsh", ".zshrc")); !os.IsNotExist(err) {
		t.Fatalf("expected no repo source created, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "manifest.yaml")); !os.IsNotExist(err) {
		t.Fatalf("expected no manifest created, got err=%v", err)
	}
	info, err := os.Lstat(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatalf("lstat target: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Fatal("expected local target to remain a regular file")
	}
}

func TestRunAddRejectsExistingRepoSourceWithoutForce(t *testing.T) {
	defer restoreAddFlags()()
	flagJSON = true
	flagDryRun = false
	flagForce = false

	home := t.TempDir()
	repo := filepath.Join(t.TempDir(), "repo")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.MkdirAll(filepath.Join(repo, "configs", "zsh"), 0o755); err != nil {
		t.Fatalf("mkdir repo source dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repo, "configs", "zsh", ".zshrc"), []byte("existing\n"), 0o644); err != nil {
		t.Fatalf("write existing source: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("export Z=1\n"), 0o644); err != nil {
		t.Fatalf("write .zshrc: %v", err)
	}
	t.Setenv("HOME", home)
	if err := config.Save(configPath, &config.Config{Repo: config.RepoConfig{URL: "git@example.com:dotfiles.git", Path: repo}}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	flagConfig = configPath

	err := runAdd(&cobra.Command{}, []string{filepath.Join(home, ".zshrc")})
	if err == nil {
		t.Fatal("expected error for existing repo source without force")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Fatalf("error = %v, want --force hint", err)
	}
}

func restoreAddFlags() func() {
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
