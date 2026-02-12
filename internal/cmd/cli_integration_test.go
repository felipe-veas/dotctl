package cmd

import (
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipe-veas/dotctl/internal/config"
	"github.com/felipe-veas/dotctl/internal/gitops"
)

type cliTestEnv struct {
	remotePath string
	clonePath  string
	configPath string
	homePath   string
	binPath    string
}

func TestCLIInitIntegration(t *testing.T) {
	requireGit(t)
	env := setupCLIIntegration(t, false)

	_, err := executeCLI(t,
		"init",
		"--repo", env.remotePath,
		"--profile", "devserver",
		"--path", env.clonePath,
		"--config", env.configPath,
	)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}

	if !gitops.IsRepo(env.clonePath) {
		t.Fatalf("expected cloned git repo at %s", env.clonePath)
	}

	cfg, err := config.Load(env.configPath)
	if err != nil {
		t.Fatalf("loading config: %v", err)
	}
	if cfg.Profile != "devserver" {
		t.Fatalf("profile = %q, want devserver", cfg.Profile)
	}
	if cfg.Repo.Path != env.clonePath {
		t.Fatalf("repo.path = %q, want %q", cfg.Repo.Path, env.clonePath)
	}
}

func TestCLIPullIntegration(t *testing.T) {
	requireGit(t)
	env := setupCLIIntegration(t, false)
	initForIntegration(t, env)

	writer := filepath.Join(t.TempDir(), "writer")
	gitCmd(t, "", "clone", env.remotePath, writer)
	updated := "# zshrc updated from remote\n"
	if err := os.WriteFile(filepath.Join(writer, "configs", "zsh", ".zshrc"), []byte(updated), 0o644); err != nil {
		t.Fatalf("write remote update: %v", err)
	}
	gitCmd(t, writer, "add", "configs/zsh/.zshrc")
	gitCmd(t, writer, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "remote update")
	gitCmd(t, writer, "push", "origin", "HEAD")

	_, err := executeCLI(t, "pull", "--config", env.configPath)
	if err != nil {
		t.Fatalf("pull failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(env.clonePath, "configs", "zsh", ".zshrc"))
	if err != nil {
		t.Fatalf("read pulled file: %v", err)
	}
	if string(data) != updated {
		t.Fatalf("pulled content mismatch: got %q, want %q", string(data), updated)
	}
}

func TestCLIPushIntegration(t *testing.T) {
	requireGit(t)
	env := setupCLIIntegration(t, false)
	initForIntegration(t, env)

	newContent := "# zshrc local push\n"
	if err := os.WriteFile(filepath.Join(env.clonePath, "configs", "zsh", ".zshrc"), []byte(newContent), 0o644); err != nil {
		t.Fatalf("write local change: %v", err)
	}

	_, err := executeCLI(t, "push", "--config", env.configPath, "--message", "integration push")
	if err != nil {
		t.Fatalf("push failed: %v", err)
	}

	verifier := filepath.Join(t.TempDir(), "verifier")
	gitCmd(t, "", "clone", env.remotePath, verifier)

	data, err := os.ReadFile(filepath.Join(verifier, "configs", "zsh", ".zshrc"))
	if err != nil {
		t.Fatalf("read pushed file: %v", err)
	}
	if string(data) != newContent {
		t.Fatalf("pushed content mismatch: got %q, want %q", string(data), newContent)
	}

	subject := gitCmd(t, verifier, "log", "-1", "--pretty=%s")
	if subject != "integration push" {
		t.Fatalf("commit subject = %q, want integration push", subject)
	}
}

func TestCLISyncIntegration(t *testing.T) {
	requireGit(t)
	env := setupCLIIntegration(t, false)
	initForIntegration(t, env)

	writer := filepath.Join(t.TempDir(), "writer")
	gitCmd(t, "", "clone", env.remotePath, writer)
	updated := "# sync updated from remote\n"
	if err := os.WriteFile(filepath.Join(writer, "configs", "zsh", ".zshrc"), []byte(updated), 0o644); err != nil {
		t.Fatalf("write remote sync update: %v", err)
	}
	gitCmd(t, writer, "add", "configs/zsh/.zshrc")
	gitCmd(t, writer, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "sync remote update")
	gitCmd(t, writer, "push", "origin", "HEAD")

	_, err := executeCLI(t, "sync", "--config", env.configPath)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(env.clonePath, "configs", "zsh", ".zshrc"))
	if err != nil {
		t.Fatalf("read local repo file after sync: %v", err)
	}
	if string(data) != updated {
		t.Fatalf("sync did not pull latest change, got %q", string(data))
	}

	target := filepath.Join(env.homePath, ".zshrc")
	link, err := os.Readlink(target)
	if err != nil {
		t.Fatalf("expected symlink at %s: %v", target, err)
	}
	expectedLink := filepath.Join(env.clonePath, "configs", "zsh", ".zshrc")
	if link != expectedLink {
		t.Fatalf("symlink target = %q, want %q", link, expectedLink)
	}

	cfg, err := config.Load(env.configPath)
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if cfg.LastSync == nil {
		t.Fatal("expected last_sync to be set after sync")
	}
}

func TestCLIBootstrapIntegration(t *testing.T) {
	requireGit(t)
	env := setupCLIIntegration(t, false)
	initForIntegration(t, env)

	manifestBody := "version: 1\nfiles:\n  - source: configs/zsh/.zshrc\n    target: ~/.zshrc\nhooks:\n  bootstrap:\n    - command: printf bootstrap-ok > bootstrap.marker\n"
	if err := os.WriteFile(filepath.Join(env.clonePath, "manifest.yaml"), []byte(manifestBody), 0o644); err != nil {
		t.Fatalf("write manifest with bootstrap hook: %v", err)
	}

	raw, err := executeCLI(t, "bootstrap", "--config", env.configPath, "--json")
	if err != nil {
		t.Fatalf("bootstrap failed: %v", err)
	}

	var response struct {
		Hooks []struct {
			Status string `json:"status"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		t.Fatalf("parse bootstrap json: %v\nraw: %s", err, raw)
	}
	if len(response.Hooks) != 1 {
		t.Fatalf("hooks = %d, want 1", len(response.Hooks))
	}
	if response.Hooks[0].Status != "ok" {
		t.Fatalf("hook status = %q, want ok", response.Hooks[0].Status)
	}

	markerPath := filepath.Join(env.clonePath, "bootstrap.marker")
	data, err := os.ReadFile(markerPath)
	if err != nil {
		t.Fatalf("read bootstrap marker: %v", err)
	}
	if strings.TrimSpace(string(data)) != "bootstrap-ok" {
		t.Fatalf("marker content = %q, want bootstrap-ok", strings.TrimSpace(string(data)))
	}
}

func TestCLIBootstrapDryRunIntegration(t *testing.T) {
	requireGit(t)
	env := setupCLIIntegration(t, false)
	initForIntegration(t, env)

	manifestBody := "version: 1\nfiles:\n  - source: configs/zsh/.zshrc\n    target: ~/.zshrc\nhooks:\n  bootstrap:\n    - command: printf should-not-run > dry-run.marker\n"
	if err := os.WriteFile(filepath.Join(env.clonePath, "manifest.yaml"), []byte(manifestBody), 0o644); err != nil {
		t.Fatalf("write manifest with bootstrap hook: %v", err)
	}

	raw, err := executeCLI(t, "bootstrap", "--config", env.configPath, "--dry-run", "--json")
	if err != nil {
		t.Fatalf("bootstrap dry-run failed: %v", err)
	}

	var response struct {
		Hooks []struct {
			Status string `json:"status"`
		} `json:"hooks"`
	}
	if err := json.Unmarshal([]byte(raw), &response); err != nil {
		t.Fatalf("parse bootstrap dry-run json: %v\nraw: %s", err, raw)
	}
	if len(response.Hooks) != 1 {
		t.Fatalf("hooks = %d, want 1", len(response.Hooks))
	}
	if response.Hooks[0].Status != "would_run" {
		t.Fatalf("hook status = %q, want would_run", response.Hooks[0].Status)
	}

	if _, err := os.Stat(filepath.Join(env.clonePath, "dry-run.marker")); err == nil {
		t.Fatal("dry-run marker was created, expected no file")
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat dry-run marker: %v", err)
	}
}

func TestCLISyncRunsHooksIntegration(t *testing.T) {
	requireGit(t)
	env := setupCLIIntegration(t, false)
	initForIntegration(t, env)

	manifestBody := "version: 1\nfiles:\n  - source: configs/zsh/.zshrc\n    target: ~/.zshrc\nhooks:\n  pre_sync:\n    - command: printf pre > pre-sync.marker\n  post_sync:\n    - command: printf post > post-sync.marker\n"
	if err := os.WriteFile(filepath.Join(env.clonePath, "manifest.yaml"), []byte(manifestBody), 0o644); err != nil {
		t.Fatalf("write manifest with sync hooks: %v", err)
	}
	gitCmd(t, env.clonePath, "add", "manifest.yaml")
	gitCmd(t, env.clonePath, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "configure hooks for sync integration")
	gitCmd(t, env.clonePath, "push", "origin", "HEAD")

	if _, err := executeCLI(t, "sync", "--config", env.configPath); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	preData, err := os.ReadFile(filepath.Join(env.clonePath, "pre-sync.marker"))
	if err != nil {
		t.Fatalf("read pre-sync marker: %v", err)
	}
	if strings.TrimSpace(string(preData)) != "pre" {
		t.Fatalf("pre-sync marker content = %q, want pre", strings.TrimSpace(string(preData)))
	}

	postData, err := os.ReadFile(filepath.Join(env.clonePath, "post-sync.marker"))
	if err != nil {
		t.Fatalf("read post-sync marker: %v", err)
	}
	if strings.TrimSpace(string(postData)) != "post" {
		t.Fatalf("post-sync marker content = %q, want post", strings.TrimSpace(string(postData)))
	}
}

func TestCLISyncDecryptCopyIntegration(t *testing.T) {
	requireGit(t)
	env := setupCLIIntegration(t, false)
	initForIntegration(t, env)

	sopsScript := filepath.Join(env.binPath, "sops")
	sopsBody := "#!/bin/sh\nprintf 'api_key: decrypted-value\\n'"
	if err := os.WriteFile(sopsScript, []byte(sopsBody), 0o755); err != nil {
		t.Fatalf("write fake sops script: %v", err)
	}

	writer := filepath.Join(t.TempDir(), "writer")
	gitCmd(t, "", "clone", env.remotePath, writer)

	encPath := filepath.Join(writer, "configs", "secrets", "api.enc.yaml")
	if err := os.MkdirAll(filepath.Dir(encPath), 0o755); err != nil {
		t.Fatalf("mkdir encrypted source path: %v", err)
	}
	if err := os.WriteFile(encPath, []byte("ENC[AES256_GCM,data:...]\n"), 0o600); err != nil {
		t.Fatalf("write encrypted source: %v", err)
	}

	manifestBody := "version: 1\nfiles:\n  - source: configs/secrets/api.enc.yaml\n    target: ~/.config/decrypted/api.yaml\n    mode: copy\n    decrypt: true\n"
	if err := os.WriteFile(filepath.Join(writer, "manifest.yaml"), []byte(manifestBody), 0o644); err != nil {
		t.Fatalf("write decrypt manifest: %v", err)
	}

	gitCmd(t, writer, "add", "manifest.yaml", "configs/secrets/api.enc.yaml")
	gitCmd(t, writer, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "add decrypt copy entry")
	gitCmd(t, writer, "push", "origin", "HEAD")

	if _, err := executeCLI(t, "sync", "--config", env.configPath); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	target := filepath.Join(env.homePath, ".config", "decrypted", "api.yaml")
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read decrypted target: %v", err)
	}
	if strings.TrimSpace(string(data)) != "api_key: decrypted-value" {
		t.Fatalf("decrypted target = %q, want api_key: decrypted-value", strings.TrimSpace(string(data)))
	}
}

func TestCLISyncRollbackOnPostHookFailure(t *testing.T) {
	requireGit(t)
	env := setupCLIIntegration(t, false)
	initForIntegration(t, env)

	manifestBody := "version: 1\nfiles:\n  - source: configs/zsh/.zshrc\n    target: ~/.zshrc\nhooks:\n  post_sync:\n    - command: exit 9\n"
	if err := os.WriteFile(filepath.Join(env.clonePath, "manifest.yaml"), []byte(manifestBody), 0o644); err != nil {
		t.Fatalf("write manifest with failing post_sync hook: %v", err)
	}
	gitCmd(t, env.clonePath, "add", "manifest.yaml")
	gitCmd(t, env.clonePath, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "configure failing post hook")
	gitCmd(t, env.clonePath, "push", "origin", "HEAD")

	_, err := executeCLI(t, "sync", "--config", env.configPath)
	if err == nil {
		t.Fatal("expected sync failure due to failing post_sync hook")
	}
	if !strings.Contains(err.Error(), "post_sync hook failed") {
		t.Fatalf("unexpected sync error: %v", err)
	}

	target := filepath.Join(env.homePath, ".zshrc")
	if _, statErr := os.Lstat(target); !os.IsNotExist(statErr) {
		t.Fatalf("expected rollback to remove target %s, got err=%v", target, statErr)
	}
}

func TestCLIDoctorIntegration(t *testing.T) {
	requireGit(t)
	env := setupCLIIntegration(t, true)
	initForIntegration(t, env)

	_, err := executeCLI(t, "sync", "--config", env.configPath)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	raw, err := executeCLI(t, "doctor", "--config", env.configPath, "--json")
	if err != nil {
		t.Fatalf("doctor failed: %v", err)
	}

	var report struct {
		Healthy  bool     `json:"healthy"`
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(raw), &report); err != nil {
		t.Fatalf("parse doctor json: %v\nraw: %s", err, raw)
	}
	if !report.Healthy {
		t.Fatalf("expected healthy report, got: %s", raw)
	}
	if len(report.Warnings) == 0 {
		t.Fatalf("expected security warning in doctor output: %s", raw)
	}

	statusRaw, err := executeCLI(t, "status", "--config", env.configPath, "--json")
	if err != nil {
		t.Fatalf("status failed: %v", err)
	}
	var status struct {
		Warnings []string `json:"warnings"`
	}
	if err := json.Unmarshal([]byte(statusRaw), &status); err != nil {
		t.Fatalf("parse status json: %v\nraw: %s", err, statusRaw)
	}
	if len(status.Warnings) == 0 {
		t.Fatalf("expected warning in status output: %s", statusRaw)
	}
}

func initForIntegration(t *testing.T, env cliTestEnv) {
	t.Helper()
	_, err := executeCLI(t,
		"init",
		"--repo", env.remotePath,
		"--profile", "devserver",
		"--path", env.clonePath,
		"--config", env.configPath,
	)
	if err != nil {
		t.Fatalf("init failed: %v", err)
	}
}

func setupCLIIntegration(t *testing.T, includeSensitive bool) cliTestEnv {
	t.Helper()

	base := t.TempDir()
	homePath := filepath.Join(base, "home")
	xdgPath := filepath.Join(base, "xdg")
	binPath := filepath.Join(base, "bin")
	remotePath := filepath.Join(base, "remote.git")
	seedPath := filepath.Join(base, "seed")
	clonePath := filepath.Join(base, "repo")
	configPath := filepath.Join(base, "config.yaml")

	if err := os.MkdirAll(homePath, 0o755); err != nil {
		t.Fatalf("mkdir home: %v", err)
	}
	if err := os.MkdirAll(xdgPath, 0o755); err != nil {
		t.Fatalf("mkdir xdg: %v", err)
	}
	if err := os.MkdirAll(binPath, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}

	ghScript := filepath.Join(binPath, "gh")
	ghBody := "#!/bin/sh\nif [ \"$1\" = \"auth\" ] && [ \"$2\" = \"status\" ]; then\n  echo \"✓ Logged in to github.com account integration-user (keychain)\"\n  exit 0\nfi\necho \"unsupported gh invocation: $*\" >&2\nexit 1\n"
	if err := os.WriteFile(ghScript, []byte(ghBody), 0o755); err != nil {
		t.Fatalf("write fake gh script: %v", err)
	}

	gitCmd(t, "", "init", "--bare", remotePath)
	gitCmd(t, "", "clone", remotePath, seedPath)

	if err := os.MkdirAll(filepath.Join(seedPath, "configs", "zsh"), 0o755); err != nil {
		t.Fatalf("mkdir configs/zsh: %v", err)
	}
	manifest := "version: 1\nfiles:\n  - source: configs/zsh/.zshrc\n    target: ~/.zshrc\n"
	if err := os.WriteFile(filepath.Join(seedPath, "manifest.yaml"), []byte(manifest), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(filepath.Join(seedPath, "configs", "zsh", ".zshrc"), []byte("# zshrc\n"), 0o644); err != nil {
		t.Fatalf("write zshrc: %v", err)
	}
	if includeSensitive {
		if err := os.WriteFile(filepath.Join(seedPath, ".env"), []byte("TOKEN=super-secret\n"), 0o644); err != nil {
			t.Fatalf("write .env: %v", err)
		}
	}

	gitCmd(t, seedPath, "add", ".")
	gitCmd(t, seedPath, "-c", "user.name=test", "-c", "user.email=test@example.com", "commit", "-m", "seed")
	gitCmd(t, seedPath, "push", "origin", "HEAD")

	t.Setenv("HOME", homePath)
	t.Setenv("XDG_CONFIG_HOME", xdgPath)
	t.Setenv("PATH", binPath+string(os.PathListSeparator)+os.Getenv("PATH"))

	return cliTestEnv{
		remotePath: remotePath,
		clonePath:  clonePath,
		configPath: configPath,
		homePath:   homePath,
		binPath:    binPath,
	}
}

func executeCLI(t *testing.T, args ...string) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	os.Stdout = w

	root := NewRootCmd()
	root.SetArgs(args)
	execErr := root.Execute()

	_ = w.Close()
	os.Stdout = oldStdout

	data, readErr := io.ReadAll(r)
	_ = r.Close()
	if readErr != nil {
		t.Fatalf("read stdout: %v", readErr)
	}

	return strings.TrimSpace(string(data)), execErr
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
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
