package gitops

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	runGitCommand = runGit

	// ErrNotGitRepo indicates the path is not a Git repository.
	ErrNotGitRepo = errors.New("not a git repository")

	// ErrRepoDirty indicates the repository has uncommitted local changes.
	ErrRepoDirty = errors.New("repository has uncommitted changes")
)

// InspectResult contains basic repository metadata.
type InspectResult struct {
	Branch     string
	LastCommit string
	Dirty      bool
}

// PushResult describes the outcome of a push operation.
type PushResult struct {
	Message       string `json:"message,omitempty"`
	Committed     bool   `json:"committed"`
	Pushed        bool   `json:"pushed"`
	NothingToPush bool   `json:"nothing_to_push"`
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")

	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		op := "command"
		if len(args) > 0 {
			op = args[0]
		}
		if output != "" {
			return "", fmt.Errorf("git %s failed: %s", op, output)
		}
		return "", fmt.Errorf("git %s failed: %w", op, err)
	}

	return output, nil
}

// IsSSHURL reports whether repoURL is an SSH Git URL.
func IsSSHURL(repoURL string) bool {
	repoURL = strings.TrimSpace(repoURL)
	return strings.HasPrefix(repoURL, "git@") || strings.HasPrefix(repoURL, "ssh://")
}

// NormalizeCloneURL converts user-provided repo URLs into clone-ready URLs.
func NormalizeCloneURL(repoURL string) string {
	repoURL = strings.TrimSpace(repoURL)
	repoURL = strings.TrimSuffix(repoURL, "/")
	if repoURL == "" {
		return ""
	}

	switch {
	case strings.HasPrefix(repoURL, "http://"), strings.HasPrefix(repoURL, "https://"), IsSSHURL(repoURL):
		if strings.HasSuffix(repoURL, ".git") {
			return repoURL
		}
		return repoURL + ".git"
	case strings.HasPrefix(repoURL, "github.com/"):
		repoURL = "https://" + repoURL
		if strings.HasSuffix(repoURL, ".git") {
			return repoURL
		}
		return repoURL + ".git"
	case strings.Count(repoURL, "/") == 1 && !strings.Contains(repoURL, ":"):
		repoURL = "https://github.com/" + repoURL
		if strings.HasSuffix(repoURL, ".git") {
			return repoURL
		}
		return repoURL + ".git"
	default:
		return repoURL
	}
}

// BrowserURL converts repoURL into a browser-friendly HTTPS URL.
func BrowserURL(repoURL string) string {
	repoURL = strings.TrimSpace(repoURL)
	repoURL = strings.TrimSuffix(repoURL, "/")
	repoURL = strings.TrimSuffix(repoURL, ".git")

	switch {
	case strings.HasPrefix(repoURL, "git@github.com:"):
		return "https://github.com/" + strings.TrimPrefix(repoURL, "git@github.com:")
	case strings.HasPrefix(repoURL, "ssh://git@github.com/"):
		return "https://github.com/" + strings.TrimPrefix(repoURL, "ssh://git@github.com/")
	case strings.HasPrefix(repoURL, "github.com/"):
		return "https://" + repoURL
	case strings.HasPrefix(repoURL, "http://github.com/"):
		return "https://" + strings.TrimPrefix(repoURL, "http://")
	case strings.Count(repoURL, "/") == 1 && !strings.Contains(repoURL, ":"):
		return "https://github.com/" + repoURL
	default:
		return repoURL
	}
}

// GitVersion returns the installed git version string.
func GitVersion() (string, error) {
	return runGitCommand("", "--version")
}

// IsRepo reports whether path looks like a Git repository.
func IsRepo(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && info.IsDir()
}

func ensureRepo(path string) error {
	if !IsRepo(path) {
		return fmt.Errorf("%w: %s", ErrNotGitRepo, path)
	}
	return nil
}

// Clone clones repoURL into path. If path already contains a git repo, it is treated as success.
func Clone(repoURL, path string) error {
	if IsRepo(path) {
		return nil
	}

	if info, err := os.Stat(path); err == nil {
		if !info.IsDir() {
			return fmt.Errorf("clone destination exists and is not a directory: %s", path)
		}
		entries, readErr := os.ReadDir(path)
		if readErr != nil {
			return fmt.Errorf("reading clone destination: %w", readErr)
		}
		if len(entries) > 0 {
			return fmt.Errorf("clone destination already exists and is not a git repo: %s", path)
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating clone parent dir: %w", err)
	}

	if _, err := runGitCommand("", "clone", NormalizeCloneURL(repoURL), path); err != nil {
		return fmt.Errorf("cloning repository: %w", err)
	}

	return nil
}

// PullRebase runs git pull --rebase for path.
func PullRebase(path string) (string, error) {
	if err := ensureRepo(path); err != nil {
		return "", err
	}

	dirty, err := IsDirty(path)
	if err != nil {
		return "", err
	}
	if dirty {
		return "", fmt.Errorf("%w: commit or stash your local changes before pulling", ErrRepoDirty)
	}

	out, err := runGitCommand(path, "pull", "--rebase")
	if err != nil {
		return "", fmt.Errorf("pulling latest changes: %w", err)
	}

	return out, nil
}

// IsDirty reports whether the repository has uncommitted changes.
func IsDirty(path string) (bool, error) {
	if err := ensureRepo(path); err != nil {
		return false, err
	}

	out, err := runGitCommand(path, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("checking repo status: %w", err)
	}
	return strings.TrimSpace(out) != "", nil
}

// TrackedFiles returns files currently tracked by git (git ls-files).
func TrackedFiles(path string) ([]string, error) {
	if err := ensureRepo(path); err != nil {
		return nil, err
	}

	out, err := runGitCommand(path, "ls-files")
	if err != nil {
		return nil, fmt.Errorf("listing tracked files: %w", err)
	}

	if strings.TrimSpace(out) == "" {
		return []string{}, nil
	}

	lines := strings.Split(out, "\n")
	files := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		files = append(files, line)
	}

	return files, nil
}

// Branch returns the current git branch name.
func Branch(path string) (string, error) {
	if err := ensureRepo(path); err != nil {
		return "", err
	}
	out, err := runGitCommand(path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("getting current branch: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// LastCommit returns the short hash of HEAD.
func LastCommit(path string) (string, error) {
	if err := ensureRepo(path); err != nil {
		return "", err
	}
	out, err := runGitCommand(path, "rev-parse", "--short", "HEAD")
	if err != nil {
		return "", fmt.Errorf("getting last commit: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// Inspect gathers branch, commit and dirty state in one call.
func Inspect(path string) (InspectResult, error) {
	if err := ensureRepo(path); err != nil {
		return InspectResult{}, err
	}

	branch, err := Branch(path)
	if err != nil {
		return InspectResult{}, err
	}
	commit, err := LastCommit(path)
	if err != nil {
		return InspectResult{}, err
	}
	dirty, err := IsDirty(path)
	if err != nil {
		return InspectResult{}, err
	}

	return InspectResult{Branch: branch, LastCommit: commit, Dirty: dirty}, nil
}

// DefaultCommitMessage builds the default message used by dotctl push.
func DefaultCommitMessage(profile string, now time.Time) string {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		profile = "unknown"
	}
	return fmt.Sprintf("dotctl push from %s @ %s", profile, now.Format("2006-01-02 15:04:05"))
}

// Push stages, commits and pushes local changes.
func Push(path, message, profile string, now time.Time) (PushResult, error) {
	result := PushResult{}

	if err := ensureRepo(path); err != nil {
		return result, err
	}

	if _, err := runGitCommand(path, "add", "-A"); err != nil {
		return result, fmt.Errorf("staging changes: %w", err)
	}

	dirty, err := IsDirty(path)
	if err != nil {
		return result, err
	}
	if !dirty {
		result.NothingToPush = true
		return result, nil
	}

	message = strings.TrimSpace(message)
	if message == "" {
		message = DefaultCommitMessage(profile, now)
	}

	if _, err := runGitCommand(path,
		"-c", "user.name=dotctl",
		"-c", "user.email=dotctl@localhost",
		"-c", "commit.gpgsign=false",
		"commit", "-m", message,
	); err != nil {
		return result, fmt.Errorf("creating commit: %w", err)
	}
	result.Committed = true
	result.Message = message

	if _, err := runGitCommand(path, "push"); err != nil {
		return result, fmt.Errorf("pushing to origin: %w", err)
	}
	result.Pushed = true

	return result, nil
}
