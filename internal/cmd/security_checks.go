package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/felipe-veas/dotctl/internal/gitops"
)

var sensitiveSuffixes = []string{
	".pem",
	".key",
	".p12",
	".pfx",
	".token",
	".secret",
	".secrets",
}

func trackedSensitiveFiles(repoPath string) ([]string, error) {
	files, err := gitops.TrackedFiles(repoPath)
	if err != nil {
		return nil, err
	}

	findings := make([]string, 0)
	for _, file := range files {
		if isSensitiveTrackedPath(file) {
			findings = append(findings, file)
		}
	}

	sort.Strings(findings)
	return findings, nil
}

func isSensitiveTrackedPath(p string) bool {
	normalized := filepath.ToSlash(strings.TrimSpace(p))
	normalized = strings.TrimPrefix(normalized, "./")
	lower := strings.ToLower(normalized)
	base := strings.ToLower(path.Base(lower))

	// Encrypted files are safe — skip them.
	if strings.Contains(base, ".enc.") || strings.HasSuffix(base, ".enc") {
		return false
	}

	if strings.HasPrefix(lower, ".ssh/") || strings.Contains(lower, "/.ssh/") {
		return true
	}

	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}

	if base == "id_rsa" || strings.HasPrefix(base, "id_rsa.") {
		return true
	}
	if base == "id_ed25519" || strings.HasPrefix(base, "id_ed25519.") {
		return true
	}

	for _, suffix := range sensitiveSuffixes {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}

	return false
}

func sensitiveTrackedFilesWarning(files []string) string {
	if len(files) == 0 {
		return ""
	}

	const maxPreview = 5
	preview := files
	if len(files) > maxPreview {
		preview = files[:maxPreview]
	}

	message := fmt.Sprintf("potentially sensitive tracked files: %s", strings.Join(preview, ", "))
	if len(files) > maxPreview {
		message += fmt.Sprintf(" (+%d more)", len(files)-maxPreview)
	}

	return message
}

func missingGitignorePatterns(repoPath string, patterns []string) ([]string, error) {
	required := uniqueNonEmptyPatterns(patterns)
	if len(required) == 0 {
		return nil, nil
	}

	gitignorePath := filepath.Join(repoPath, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return required, nil
		}
		return nil, fmt.Errorf("reading %s: %w", gitignorePath, err)
	}

	existing := make(map[string]bool)
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		existing[trimmed] = true
	}

	missing := make([]string, 0)
	for _, pattern := range required {
		if !existing[pattern] {
			missing = append(missing, pattern)
		}
	}
	return missing, nil
}

func uniqueNonEmptyPatterns(patterns []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(patterns))

	for _, raw := range patterns {
		p := strings.TrimSpace(raw)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}

	sort.Strings(out)
	return out
}
