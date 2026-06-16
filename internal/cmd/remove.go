package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/felipe-veas/dotctl/internal/manifest"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/spf13/cobra"
)

type removePlan struct {
	AbsTarget    string
	ManifestPath string
}

type removeResult struct {
	Status             string   `json:"status"`
	DryRun             bool     `json:"dry_run"`
	Target             string   `json:"target"`
	ManifestPath       string   `json:"manifest_path"`
	RemovedCount       int      `json:"removed_count"`
	RemovedSources     []string `json:"removed_sources"`
	LocalTargetDeleted bool     `json:"local_target_deleted"`
	RepoSourceDeleted  bool     `json:"repo_source_deleted"`
}

func newRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <path>",
		Short: "Untrack one dotfile from the manifest",
		Args:  cobra.ExactArgs(1),
		RunE:  runRemove,
	}
}

func runRemove(cmd *cobra.Command, args []string) error {
	out := output.New(flagJSON)

	cfg, _, err := resolveConfig()
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("detecting home directory: %w", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("detecting current directory: %w", err)
	}

	plan, err := buildRemovePlan(args[0], cwd, homeDir, cfg.Repo.Path)
	if err != nil {
		return err
	}

	currentManifest, manifestExists, manifestBytes, err := loadAddManifest(plan.ManifestPath)
	if err != nil {
		return err
	}
	if !manifestExists {
		return fmt.Errorf("manifest not found: %s", plan.ManifestPath)
	}

	updatedManifest, removedEntries, err := planRemoveManifest(currentManifest, plan.AbsTarget, homeDir)
	if err != nil {
		return err
	}
	removedSources := uniqueStrings(manifestEntrySources(removedEntries))

	if flagDryRun {
		return emitRemoveResult(out, removeResult{
			Status:             "dry_run",
			DryRun:             true,
			Target:             plan.AbsTarget,
			ManifestPath:       plan.ManifestPath,
			RemovedCount:       len(removedEntries),
			RemovedSources:     removedSources,
			LocalTargetDeleted: false,
			RepoSourceDeleted:  false,
		})
	}

	if err := writeAddManifest(plan.ManifestPath, updatedManifest); err != nil {
		if restoreErr := restoreAddManifest(plan.ManifestPath, manifestExists, manifestBytes); restoreErr != nil {
			return fmt.Errorf("writing manifest: %w (and restoring manifest failed: %v)", err, restoreErr)
		}
		return fmt.Errorf("writing manifest: %w", err)
	}

	if err := removeManagedSources(cfg.Repo.Path, removedSources); err != nil {
		if restoreErr := restoreAddManifest(plan.ManifestPath, manifestExists, manifestBytes); restoreErr != nil {
			return fmt.Errorf("updating managed sources: %w (and restoring manifest failed: %v)", err, restoreErr)
		}
		return fmt.Errorf("updating managed sources: %w", err)
	}

	return emitRemoveResult(out, removeResult{
		Status:             "ok",
		DryRun:             false,
		Target:             plan.AbsTarget,
		ManifestPath:       plan.ManifestPath,
		RemovedCount:       len(removedEntries),
		RemovedSources:     removedSources,
		LocalTargetDeleted: false,
		RepoSourceDeleted:  false,
	})
}

func buildRemovePlan(rawPath, cwd, homeDir, repoPath string) (removePlan, error) {
	if strings.TrimSpace(rawPath) == "" {
		return removePlan{}, fmt.Errorf("path is required")
	}

	absTarget, err := normalizeAddInputPath(rawPath, cwd, homeDir)
	if err != nil {
		return removePlan{}, err
	}

	homeAbs, err := filepath.Abs(filepath.Clean(homeDir))
	if err != nil {
		return removePlan{}, fmt.Errorf("resolving home directory: %w", err)
	}

	if containsPath(repoPath, absTarget) || containsPath(absTarget, repoPath) {
		return removePlan{}, fmt.Errorf("path %s overlaps the repo path %s", absTarget, repoPath)
	}

	relTarget, err := filepath.Rel(homeAbs, absTarget)
	if err != nil {
		return removePlan{}, fmt.Errorf("checking home containment: %w", err)
	}
	if relTarget == "." {
		return removePlan{}, fmt.Errorf("path %s is the home directory itself and cannot be removed", absTarget)
	}
	if relTarget == ".." || strings.HasPrefix(relTarget, ".."+string(filepath.Separator)) || filepath.IsAbs(relTarget) {
		return removePlan{}, fmt.Errorf("path %s must stay under home directory %s", absTarget, homeAbs)
	}

	return removePlan{
		AbsTarget:    absTarget,
		ManifestPath: filepath.Join(repoPath, "manifest.yaml"),
	}, nil
}

func planRemoveManifest(m *manifest.Manifest, target, homeDir string) (*manifest.Manifest, []manifest.FileEntry, error) {
	if m == nil {
		return nil, nil, fmt.Errorf("manifest is required")
	}

	updated := &manifest.Manifest{
		Version: m.Version,
		Vars:    copyStringMap(m.Vars),
		Ignore:  append([]string(nil), m.Ignore...),
	}
	if updated.Version == 0 {
		updated.Version = 1
	}

	ctx := manifest.RuntimeContext()
	ctx.Home = homeDir
	vars := manifest.MergeVars(m.Vars, ctx.Vars())

	files := make([]manifest.FileEntry, 0, len(m.Files))
	removed := make([]manifest.FileEntry, 0)

	for _, file := range m.Files {
		resolvedTarget, err := manifest.ResolveTarget(file.Target, vars)
		if err != nil {
			return nil, nil, err
		}

		if filepath.Clean(resolvedTarget) == target {
			removed = append(removed, file)
			continue
		}

		files = append(files, file)
	}

	if len(removed) == 0 {
		return nil, nil, fmt.Errorf("manifest has no entry targeting %s", target)
	}

	updated.Files = files
	return updated, removed, nil
}

func manifestEntrySources(entries []manifest.FileEntry) []string {
	sources := make([]string, 0, len(entries))
	for _, entry := range entries {
		sources = append(sources, entry.Source)
	}
	return sources
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func emitRemoveResult(out *output.Printer, result removeResult) error {
	if out.IsJSON() {
		return out.JSON(result)
	}

	if result.DryRun {
		out.Header("Dry run")
		if len(result.RemovedSources) == 0 {
			out.Info("No manifest entries would be removed")
		} else {
			word := "entries"
			if result.RemovedCount == 1 {
				word = "entry"
			}
			out.Info("Would remove %d manifest %s from %s", result.RemovedCount, word, result.ManifestPath)
			for _, source := range result.RemovedSources {
				out.Info("- %s", source)
			}
		}
		out.Info("Local target and repo source will not be deleted")
		return nil
	}

	word := "entries"
	if result.RemovedCount == 1 {
		word = "entry"
	}
	out.Success("Removed %d manifest %s from %s", result.RemovedCount, word, result.ManifestPath)
	out.Info("Local target and repo source were left untouched")
	if len(result.RemovedSources) > 0 {
		out.Info("Removed sources: %s", strings.Join(result.RemovedSources, ", "))
	}
	return nil
}
