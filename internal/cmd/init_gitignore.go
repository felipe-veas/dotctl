package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var defaultInitGitignorePatterns = []string{
	// OS/editor noise.
	".DS_Store",
	"Thumbs.db",

	// Common sensitive files.
	".env",
	".env.*",
	"*.pem",
	"*.key",
	"*.p12",
	"*.pfx",
	"*.token",
	"*credentials*",
	"*secret*",

	// Keep common encrypted config folders trackable even with broad patterns above.
	"!configs/secrets/",
	"!configs/secrets/**",
	"!configs/credentials/",
	"!configs/credentials/**",

	// Common dotfiles runtime/plugin content that should not be versioned.
	"configs/tmux/plugins/",
}

type gitignoreUpdate struct {
	Path  string
	Added []string
}

func ensureDefaultGitignorePatterns(repoPath string, patterns []string) (gitignoreUpdate, error) {
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	update := gitignoreUpdate{
		Path:  gitignorePath,
		Added: []string{},
	}

	data, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return update, fmt.Errorf("reading %s: %w", gitignorePath, err)
	}

	existing := make(map[string]bool)
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		existing[trimmed] = true
	}

	for _, raw := range patterns {
		pattern := strings.TrimSpace(raw)
		if pattern == "" || existing[pattern] {
			continue
		}
		update.Added = append(update.Added, pattern)
	}

	if len(update.Added) == 0 {
		return update, nil
	}

	f, err := os.OpenFile(gitignorePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return update, fmt.Errorf("opening %s: %w", gitignorePath, err)
	}
	defer func() { _ = f.Close() }()

	if len(data) > 0 && data[len(data)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return update, fmt.Errorf("writing newline to %s: %w", gitignorePath, err)
		}
	}
	if len(data) > 0 {
		if _, err := f.WriteString("\n"); err != nil {
			return update, fmt.Errorf("writing separator to %s: %w", gitignorePath, err)
		}
	}

	if _, err := f.WriteString("# dotctl defaults\n"); err != nil {
		return update, fmt.Errorf("writing header to %s: %w", gitignorePath, err)
	}
	for _, pattern := range update.Added {
		if _, err := f.WriteString(pattern + "\n"); err != nil {
			return update, fmt.Errorf("writing pattern %q to %s: %w", pattern, gitignorePath, err)
		}
	}

	return update, nil
}
