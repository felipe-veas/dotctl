package gitops

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIsSSHURL(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{in: "git@github.com:felipe-veas/dotfiles.git", want: true},
		{in: "ssh://git@github.com/felipe-veas/dotfiles.git", want: true},
		{in: "https://github.com/felipe-veas/dotfiles.git", want: false},
		{in: "github.com/felipe-veas/dotfiles", want: false},
	}

	for _, tc := range cases {
		if got := IsSSHURL(tc.in); got != tc.want {
			t.Errorf("IsSSHURL(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestNormalizeCloneURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "github.com/felipe-veas/dotfiles", want: "https://github.com/felipe-veas/dotfiles.git"},
		{in: "felipe-veas/dotfiles", want: "https://github.com/felipe-veas/dotfiles.git"},
		{in: "https://github.com/felipe-veas/dotfiles", want: "https://github.com/felipe-veas/dotfiles.git"},
		{in: "git@github.com:felipe-veas/dotfiles", want: "git@github.com:felipe-veas/dotfiles.git"},
		{in: "git@github.com:felipe-veas/dotfiles.git", want: "git@github.com:felipe-veas/dotfiles.git"},
	}

	for _, tc := range cases {
		if got := NormalizeCloneURL(tc.in); got != tc.want {
			t.Errorf("NormalizeCloneURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestBrowserURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "git@github.com:felipe-veas/dotfiles.git", want: "https://github.com/felipe-veas/dotfiles"},
		{in: "ssh://git@github.com/felipe-veas/dotfiles.git", want: "https://github.com/felipe-veas/dotfiles"},
		{in: "github.com/felipe-veas/dotfiles", want: "https://github.com/felipe-veas/dotfiles"},
		{in: "https://github.com/felipe-veas/dotfiles.git", want: "https://github.com/felipe-veas/dotfiles"},
	}

	for _, tc := range cases {
		if got := BrowserURL(tc.in); got != tc.want {
			t.Errorf("BrowserURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestClone(t *testing.T) {
	requireGit(t)

	remote := setupRemoteRepo(t)
	clonePath := filepath.Join(t.TempDir(), "clone")

	if err := Clone(remote, clonePath); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if !IsRepo(clonePath) {
		t.Fatalf("expected %s to be a git repo", clonePath)
	}
}

func TestPullRebase(t *testing.T) {
	requireGit(t)

	remote := setupRemoteRepo(t)
	client := filepath.Join(t.TempDir(), "client")
	writer := filepath.Join(t.TempDir(), "writer")

	gitCmd(t, "", "clone", remote, client)
	gitCmd(t, "", "clone", remote, writer)

	if err := os.WriteFile(filepath.Join(writer, "README.md"), []byte("updated\n"), 0o644); err != nil {
		t.Fatalf("write updated file: %v", err)
	}
	gitCmd(t, writer, "add", "README.md")
	gitCmd(t, writer, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "update")
	gitCmd(t, writer, "push", "origin", "HEAD")

	if _, err := PullRebase(client); err != nil {
		t.Fatalf("PullRebase: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(client, "README.md"))
	if err != nil {
		t.Fatalf("read pulled file: %v", err)
	}
	if strings.TrimSpace(string(data)) != "updated" {
		t.Fatalf("unexpected pulled content: %q", string(data))
	}
}

func TestPullRebaseDirty(t *testing.T) {
	requireGit(t)

	remote := setupRemoteRepo(t)
	client := filepath.Join(t.TempDir(), "client")
	gitCmd(t, "", "clone", remote, client)

	if err := os.WriteFile(filepath.Join(client, "README.md"), []byte("local change\n"), 0o644); err != nil {
		t.Fatalf("write local change: %v", err)
	}

	_, err := PullRebase(client)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrRepoDirty) {
		t.Fatalf("expected ErrRepoDirty, got: %v", err)
	}
}

func TestPush(t *testing.T) {
	requireGit(t)

	remote := setupRemoteRepo(t)
	client := filepath.Join(t.TempDir(), "client")
	verifier := filepath.Join(t.TempDir(), "verifier")

	gitCmd(t, "", "clone", remote, client)

	if err := os.WriteFile(filepath.Join(client, "new.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write new file: %v", err)
	}

	now := time.Date(2026, 2, 12, 10, 30, 0, 0, time.UTC)
	res, err := Push(client, "", "devserver", now)
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if !res.Committed || !res.Pushed {
		t.Fatalf("unexpected push result: %+v", res)
	}
	if !strings.Contains(res.Message, "devserver") {
		t.Fatalf("message = %q, expected profile", res.Message)
	}

	gitCmd(t, "", "clone", remote, verifier)
	if _, err := os.Stat(filepath.Join(verifier, "new.txt")); err != nil {
		t.Fatalf("expected new.txt in remote clone: %v", err)
	}
}

func TestPushNothingToPush(t *testing.T) {
	requireGit(t)

	remote := setupRemoteRepo(t)
	client := filepath.Join(t.TempDir(), "client")
	gitCmd(t, "", "clone", remote, client)

	res, err := Push(client, "", "devserver", time.Now())
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if !res.NothingToPush {
		t.Fatalf("expected NothingToPush=true, got %+v", res)
	}
}

func TestTrackedFiles(t *testing.T) {
	requireGit(t)

	remote := setupRemoteRepo(t)
	client := filepath.Join(t.TempDir(), "client")
	gitCmd(t, "", "clone", remote, client)

	files, err := TrackedFiles(client)
	if err != nil {
		t.Fatalf("TrackedFiles: %v", err)
	}

	if len(files) == 0 {
		t.Fatal("expected tracked files")
	}

	foundReadme := false
	for _, f := range files {
		if f == "README.md" {
			foundReadme = true
			break
		}
	}
	if !foundReadme {
		t.Fatalf("expected README.md in tracked files, got: %v", files)
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
}

func setupRemoteRepo(t *testing.T) string {
	t.Helper()

	base := t.TempDir()
	remote := filepath.Join(base, "remote.git")
	seed := filepath.Join(base, "seed")

	gitCmd(t, "", "init", "--bare", remote)
	gitCmd(t, "", "clone", remote, seed)

	if err := os.WriteFile(filepath.Join(seed, "README.md"), []byte("seed\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}
	gitCmd(t, seed, "add", "README.md")
	gitCmd(t, seed, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "seed")
	gitCmd(t, seed, "push", "origin", "HEAD")

	return remote
}

func gitCmd(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
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
