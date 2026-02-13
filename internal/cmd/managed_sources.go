package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/felipe-veas/dotctl/internal/manifest"
)

const managedSourcesStateFile = ".dotctl/managed-sources.txt"

type sourcePruneResult struct {
	Source string
	Status string
	Error  string
}

type sourceBackfillResult struct {
	Source string
	Target string
	Status string
	Error  string
}

func writeManagedSources(repoPath string, sources []string) error {
	statePath := filepath.Join(repoPath, managedSourcesStateFile)
	cleaned := normalizeManagedSourceList(sources)

	if len(cleaned) == 0 {
		if err := os.Remove(statePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("removing managed sources state: %w", err)
		}
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(statePath), 0o755); err != nil {
		return fmt.Errorf("creating managed sources directory: %w", err)
	}

	content := strings.Join(cleaned, "\n") + "\n"
	if err := os.WriteFile(statePath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("writing managed sources state: %w", err)
	}

	return nil
}

func addManagedSources(repoPath string, sources []string) error {
	current, err := readManagedSources(repoPath)
	if err != nil {
		return err
	}

	combined := append(current, sources...)
	return writeManagedSources(repoPath, combined)
}

func readManagedSources(repoPath string) ([]string, error) {
	statePath := filepath.Join(repoPath, managedSourcesStateFile)
	data, err := os.ReadFile(statePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("reading managed sources state: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	return normalizeManagedSourceList(lines), nil
}

func pruneManagedSources(repoPath string, manifestFiles []manifest.FileEntry, dryRun bool) ([]sourcePruneResult, error) {
	managedSources, err := readManagedSources(repoPath)
	if err != nil {
		return nil, err
	}
	if len(managedSources) == 0 {
		return []sourcePruneResult{}, nil
	}

	active := make(map[string]bool, len(manifestFiles))
	activeSources := make([]string, 0, len(manifestFiles))
	for _, file := range manifestFiles {
		source, ok := normalizeManagedSource(file.Source)
		if !ok {
			continue
		}
		active[source] = true
		activeSources = append(activeSources, source)
	}

	results := make([]sourcePruneResult, 0)
	nextManaged := make([]string, 0, len(managedSources))
	errorCount := 0

	for _, source := range managedSources {
		if isManagedSourceStillActive(source, active, activeSources) {
			nextManaged = append(nextManaged, source)
			continue
		}

		repoSourcePath := filepath.Join(repoPath, filepath.FromSlash(source))
		exists, existsErr := pathExists(repoSourcePath)
		if existsErr != nil {
			errorCount++
			nextManaged = append(nextManaged, source)
			results = append(results, sourcePruneResult{
				Source: source,
				Status: "error",
				Error:  existsErr.Error(),
			})
			continue
		}

		if !exists {
			results = append(results, sourcePruneResult{
				Source: source,
				Status: "missing",
			})
			continue
		}

		if dryRun {
			nextManaged = append(nextManaged, source)
			results = append(results, sourcePruneResult{
				Source: source,
				Status: "would_remove",
			})
			continue
		}

		if err := os.RemoveAll(repoSourcePath); err != nil {
			errorCount++
			nextManaged = append(nextManaged, source)
			results = append(results, sourcePruneResult{
				Source: source,
				Status: "error",
				Error:  fmt.Sprintf("removing %s: %v", repoSourcePath, err),
			})
			continue
		}

		results = append(results, sourcePruneResult{
			Source: source,
			Status: "removed",
		})
	}

	if !dryRun {
		if err := writeManagedSources(repoPath, nextManaged); err != nil {
			return results, err
		}
	}

	if errorCount > 0 {
		return results, fmt.Errorf("pruning stale managed sources failed with %d error(s)", errorCount)
	}

	return results, nil
}

func isManagedSourceStillActive(source string, active map[string]bool, activeSources []string) bool {
	if active[source] {
		return true
	}

	for _, current := range activeSources {
		// Keep the source when either:
		// - source is a parent of an active entry (e.g. configs/tmux vs configs/tmux/tmux.conf)
		// - source is a child of an active entry (e.g. configs/tmux/tmux.conf vs configs/tmux)
		if hasManagedSourcePrefix(current, source) || hasManagedSourcePrefix(source, current) {
			return true
		}
	}

	return false
}

func hasManagedSourcePrefix(path, prefix string) bool {
	if path == prefix {
		return true
	}
	return strings.HasPrefix(path, prefix+"/")
}

func backfillMissingSourcesFromTargets(repoPath string, actions []manifest.Action, dryRun bool) ([]sourceBackfillResult, error) {
	results := make([]sourceBackfillResult, 0)
	copiedSources := make([]string, 0)
	errorCount := 0

	for _, action := range actions {
		source, ok := normalizeManagedSource(action.Source)
		if !ok {
			continue
		}

		repoSourcePath := filepath.Join(repoPath, filepath.FromSlash(source))
		exists, existsErr := pathExists(repoSourcePath)
		if existsErr != nil {
			errorCount++
			results = append(results, sourceBackfillResult{
				Source: source,
				Target: action.Target,
				Status: "error",
				Error:  existsErr.Error(),
			})
			continue
		}
		if exists {
			continue
		}

		targetPath, targetErr := resolveBackfillTargetPath(action.Target)
		if targetErr != nil {
			errorCount++
			results = append(results, sourceBackfillResult{
				Source: source,
				Target: action.Target,
				Status: "error",
				Error:  targetErr.Error(),
			})
			continue
		}

		if dryRun {
			results = append(results, sourceBackfillResult{
				Source: source,
				Target: action.Target,
				Status: "would_copy_from_target",
			})
			continue
		}

		if err := copyPathRecursive(targetPath, repoSourcePath); err != nil {
			errorCount++
			results = append(results, sourceBackfillResult{
				Source: source,
				Target: action.Target,
				Status: "error",
				Error:  fmt.Sprintf("copying target %s into repo source %s: %v", targetPath, repoSourcePath, err),
			})
			continue
		}

		copiedSources = append(copiedSources, source)
		results = append(results, sourceBackfillResult{
			Source: source,
			Target: action.Target,
			Status: "copied_from_target",
		})
	}

	if !dryRun && len(copiedSources) > 0 {
		if err := addManagedSources(repoPath, copiedSources); err != nil {
			return results, err
		}
	}

	if errorCount > 0 {
		return results, fmt.Errorf("backfilling missing manifest sources failed with %d error(s)", errorCount)
	}

	return results, nil
}

func resolveBackfillTargetPath(target string) (string, error) {
	info, err := os.Lstat(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("target %s does not exist locally", target)
		}
		return "", fmt.Errorf("reading target %s: %w", target, err)
	}

	if info.Mode()&os.ModeSymlink == 0 {
		return target, nil
	}

	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", fmt.Errorf("resolving symlink target %s: %w", target, err)
	}
	return resolved, nil
}

func normalizeManagedSourceList(raw []string) []string {
	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))

	for _, item := range raw {
		normalized, ok := normalizeManagedSource(item)
		if !ok || seen[normalized] {
			continue
		}
		seen[normalized] = true
		out = append(out, normalized)
	}

	sort.Strings(out)
	return out
}

func normalizeManagedSource(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", false
	}

	normalized := filepath.ToSlash(filepath.Clean(trimmed))
	if normalized == "." || strings.HasPrefix(normalized, "/") || strings.HasPrefix(normalized, "../") {
		return "", false
	}

	// Managed sources are created by manifest suggest under configs/.
	if !strings.HasPrefix(normalized, "configs/") {
		return "", false
	}

	return normalized, true
}
