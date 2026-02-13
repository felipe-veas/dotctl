package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectPushScopeDifferentRepoDirtyCWD(t *testing.T) {
	base := t.TempDir()
	configuredRepo := filepath.Join(base, "configured")
	cwdRepo := filepath.Join(base, "cwd")
	initGitRepo(t, configuredRepo)
	initGitRepo(t, cwdRepo)

	setCWD(t, cwdRepo)

	if err := os.WriteFile(filepath.Join(cwdRepo, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("write cwd change: %v", err)
	}
	gitCmdPushTest(t, cwdRepo, "add", "README.md")

	scope := detectPushScope(configuredRepo)
	if !scope.DifferentFromConfigured {
		t.Fatal("expected DifferentFromConfigured=true")
	}
	if !scope.CurrentDirIsRepo {
		t.Fatal("expected CurrentDirIsRepo=true")
	}
	if !scope.CurrentDirDirty {
		t.Fatal("expected CurrentDirDirty=true")
	}
	if !samePath(scope.CurrentDir, cwdRepo) {
		t.Fatalf("CurrentDir = %q, want same path as %q", scope.CurrentDir, cwdRepo)
	}
}

func TestDetectPushScopeSameRepo(t *testing.T) {
	repo := filepath.Join(t.TempDir(), "repo")
	initGitRepo(t, repo)

	setCWD(t, repo)

	if err := os.WriteFile(filepath.Join(repo, "README.md"), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("write repo change: %v", err)
	}
	gitCmdPushTest(t, repo, "add", "README.md")

	scope := detectPushScope(repo)
	if scope.DifferentFromConfigured {
		t.Fatal("expected DifferentFromConfigured=false")
	}
	if scope.CurrentDirDirty {
		t.Fatal("expected CurrentDirDirty=false when cwd == configured repo")
	}
}

func TestDetectPushScopeDifferentNonRepoCWD(t *testing.T) {
	base := t.TempDir()
	configuredRepo := filepath.Join(base, "configured")
	nonRepoDir := filepath.Join(base, "workdir")
	initGitRepo(t, configuredRepo)
	if err := os.MkdirAll(nonRepoDir, 0o755); err != nil {
		t.Fatalf("mkdir non-repo dir: %v", err)
	}

	setCWD(t, nonRepoDir)

	scope := detectPushScope(configuredRepo)
	if !scope.DifferentFromConfigured {
		t.Fatal("expected DifferentFromConfigured=true")
	}
	if scope.CurrentDirIsRepo {
		t.Fatal("expected CurrentDirIsRepo=false")
	}
	if scope.CurrentDirDirty {
		t.Fatal("expected CurrentDirDirty=false")
	}
}

func setCWD(t *testing.T, dir string) {
	t.Helper()

	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir %s: %v", dir, err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(prev)
	})
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()

	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir repo: %v", err)
	}

	gitCmdPushTest(t, dir, "init")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	gitCmdPushTest(t, dir, "add", "README.md")
	gitCmdPushTest(t, dir, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "seed")
}

func gitCmdPushTest(t *testing.T, dir string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(
		os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, string(out))
	}
	return strings.TrimSpace(string(out))
}
