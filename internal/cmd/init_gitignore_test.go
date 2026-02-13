package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultInitGitignorePatternsBaseline(t *testing.T) {
	required := []string{
		".DS_Store",
		".env",
		".env.*",
		"*.pem",
		"*.key",
		"*.p12",
		"*.pfx",
		"*.token",
		"*credentials*",
		"*secret*",
		"configs/tmux/plugins/",
	}

	seen := make(map[string]bool, len(defaultInitGitignorePatterns))
	for _, pattern := range defaultInitGitignorePatterns {
		seen[pattern] = true
	}

	for _, pattern := range required {
		if !seen[pattern] {
			t.Fatalf("defaultInitGitignorePatterns missing required pattern %q", pattern)
		}
	}
}

func TestEnsureDefaultGitignorePatternsCreatesFile(t *testing.T) {
	repo := t.TempDir()

	update, err := ensureDefaultGitignorePatterns(repo, []string{"configs/tmux/plugins/"})
	if err != nil {
		t.Fatalf("ensureDefaultGitignorePatterns: %v", err)
	}
	if len(update.Added) != 1 || update.Added[0] != "configs/tmux/plugins/" {
		t.Fatalf("added patterns = %v", update.Added)
	}

	data, err := os.ReadFile(filepath.Join(repo, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "configs/tmux/plugins/") {
		t.Fatalf(".gitignore missing pattern, content:\n%s", content)
	}
}

func TestEnsureDefaultGitignorePatternsNoDuplicates(t *testing.T) {
	repo := t.TempDir()
	gitignorePath := filepath.Join(repo, ".gitignore")
	initial := "# existing\nconfigs/tmux/plugins/\n"
	if err := os.WriteFile(gitignorePath, []byte(initial), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	update, err := ensureDefaultGitignorePatterns(repo, []string{"configs/tmux/plugins/"})
	if err != nil {
		t.Fatalf("ensureDefaultGitignorePatterns: %v", err)
	}
	if len(update.Added) != 0 {
		t.Fatalf("expected no added patterns, got %v", update.Added)
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if string(data) != initial {
		t.Fatalf(".gitignore changed unexpectedly:\n%s", string(data))
	}
}
