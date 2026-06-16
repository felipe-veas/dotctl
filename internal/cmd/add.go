package cmd

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/felipe-veas/dotctl/internal/backup"
	"github.com/felipe-veas/dotctl/internal/linker"
	"github.com/felipe-veas/dotctl/internal/manifest"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

type addPlan struct {
	Input          string
	AbsTarget      string
	RelTarget      string
	TargetExpr     string
	Source         string
	RepoSourcePath string
	ManifestPath   string
	Sensitive      bool
}

func newAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <path>",
		Short: "Onboard one local dotfile into the repo",
		Args:  cobra.ExactArgs(1),
		RunE:  runAdd,
	}
}

func runAdd(cmd *cobra.Command, args []string) error {
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

	plan, err := buildAddPlan(args[0], cwd, homeDir, cfg.Repo.Path)
	if err != nil {
		return err
	}

	warnings, err := validateAddPlan(plan, flagForce)
	if err != nil {
		return err
	}

	manifestPath := plan.ManifestPath
	currentManifest, manifestExists, manifestBytes, err := loadAddManifest(manifestPath)
	if err != nil {
		return err
	}

	entry := manifest.FileEntry{
		Source: plan.Source,
		Target: plan.TargetExpr,
		Mode:   "symlink",
	}

	updatedManifest, manifestChanged, alreadyConfigured, err := planAddManifest(currentManifest, entry, flagForce)
	if err != nil {
		return err
	}

	repoSourceExists, err := pathExists(plan.RepoSourcePath)
	if err != nil {
		return err
	}

	if repoSourceExists && !alreadyConfigured && !flagForce {
		return fmt.Errorf("repo source already exists: %s (use --force to replace it)", plan.RepoSourcePath)
	}

	shouldCopy := !repoSourceExists || (flagForce && !alreadyConfigured)
	createdRepoSource := false
	if !flagDryRun && shouldCopy {
		if repoSourceExists {
			if err := os.RemoveAll(plan.RepoSourcePath); err != nil {
				return fmt.Errorf("removing existing repo source %s: %w", plan.RepoSourcePath, err)
			}
		}

		if err := copyPathRecursive(plan.AbsTarget, plan.RepoSourcePath); err != nil {
			return fmt.Errorf("copying %s to repo source %s: %w", plan.AbsTarget, plan.RepoSourcePath, err)
		}
		createdRepoSource = !repoSourceExists
	}

	if flagDryRun {
		return emitAddResult(out, addResult{
			Status:            dryRunStatus(alreadyConfigured, manifestChanged, shouldCopy),
			DryRun:            true,
			Target:            plan.AbsTarget,
			Source:            plan.Source,
			RepoSourcePath:    plan.RepoSourcePath,
			ManifestPath:      manifestPath,
			Warning:           firstWarning(warnings),
			ManifestChanged:   manifestChanged,
			AlreadyConfigured: alreadyConfigured,
			PlannedCopy:       shouldCopy,
			PlannedLink:       true,
			TargetExpr:        plan.TargetExpr,
		})
	}

	endSession := backup.BeginSession()
	defer endSession()

	actions := []manifest.Action{{
		Source: plan.Source,
		Target: plan.AbsTarget,
		Mode:   "symlink",
		Backup: true,
	}}
	results := linker.Apply(actions, cfg.Repo.Path, false)
	if hasLinkerError(results) {
		rollbackResults := linker.Rollback(results)
		_ = rollbackResults
		if createdRepoSource {
			_ = os.RemoveAll(plan.RepoSourcePath)
		}
		return firstLinkerError(results)
	}

	backupPath := ""
	for _, result := range results {
		if result.BackupPath != "" {
			backupPath = result.BackupPath
			break
		}
	}

	if manifestChanged {
		if err := writeAddManifest(manifestPath, updatedManifest); err != nil {
			rollbackResults := linker.Rollback(results)
			_ = rollbackResults
			if createdRepoSource {
				_ = os.RemoveAll(plan.RepoSourcePath)
			}
			if restoreErr := restoreAddManifest(manifestPath, manifestExists, manifestBytes); restoreErr != nil {
				return fmt.Errorf("writing manifest: %w (and restoring manifest failed: %v)", err, restoreErr)
			}
			return fmt.Errorf("writing manifest: %w", err)
		}
	}

	if err := addManagedSources(cfg.Repo.Path, []string{plan.Source}); err != nil {
		rollbackResults := linker.Rollback(results)
		_ = rollbackResults
		if createdRepoSource {
			_ = os.RemoveAll(plan.RepoSourcePath)
		}
		if manifestChanged {
			if restoreErr := restoreAddManifest(manifestPath, manifestExists, manifestBytes); restoreErr != nil {
				return fmt.Errorf("updating managed sources: %w (and restoring manifest failed: %v)", err, restoreErr)
			}
		}
		return fmt.Errorf("updating managed sources: %w", err)
	}

	return emitAddResult(out, addResult{
		Status:            successStatus(alreadyConfigured, manifestChanged, shouldCopy, backupPath),
		DryRun:            false,
		Target:            plan.AbsTarget,
		Source:            plan.Source,
		RepoSourcePath:    plan.RepoSourcePath,
		ManifestPath:      manifestPath,
		Warning:           firstWarning(warnings),
		BackupPath:        backupPath,
		ManifestChanged:   manifestChanged,
		AlreadyConfigured: alreadyConfigured,
		PlannedCopy:       shouldCopy,
		PlannedLink:       true,
		TargetExpr:        plan.TargetExpr,
	})
}

type addResult struct {
	Status            string `json:"status"`
	DryRun            bool   `json:"dry_run"`
	Target            string `json:"target"`
	Source            string `json:"source"`
	RepoSourcePath    string `json:"repo_source_path"`
	ManifestPath      string `json:"manifest_path"`
	TargetExpr        string `json:"-"`
	BackupPath        string `json:"backup_path,omitempty"`
	Warning           string `json:"warning,omitempty"`
	ManifestChanged   bool   `json:"-"`
	AlreadyConfigured bool   `json:"-"`
	PlannedCopy       bool   `json:"-"`
	PlannedLink       bool   `json:"-"`
}

func emitAddResult(out *output.Printer, result addResult) error {
	if out.IsJSON() {
		return out.JSON(result)
	}

	if result.Warning != "" {
		out.Warn("%s", result.Warning)
	}

	if result.DryRun {
		out.Header("Dry run")
		if result.PlannedCopy {
			out.Info("Would copy %s to %s", result.Target, result.RepoSourcePath)
		}
		if result.ManifestChanged {
			out.Info("Would update %s with %s -> %s", result.ManifestPath, result.Source, result.TargetExpr)
		} else {
			out.Info("Manifest already contains %s -> %s", result.Source, result.TargetExpr)
		}
		out.Info("Would replace %s with a symlink to %s", result.Target, result.RepoSourcePath)
		return nil
	}

	if result.AlreadyConfigured {
		out.Success("%s is already configured", result.Target)
	} else {
		out.Success("Added %s to %s", result.Target, result.RepoSourcePath)
	}
	out.Info("Manifest: %s", result.ManifestPath)
	if result.BackupPath != "" {
		out.Info("Backup: %s", result.BackupPath)
	}
	out.Info("Next: run dotctl diff and dotctl push")
	return nil
}

func buildAddPlan(rawPath, cwd, homeDir, repoPath string) (addPlan, error) {
	if strings.TrimSpace(rawPath) == "" {
		return addPlan{}, fmt.Errorf("path is required")
	}

	absTarget, err := normalizeAddInputPath(rawPath, cwd, homeDir)
	if err != nil {
		return addPlan{}, err
	}

	if exists, err := pathExists(absTarget); err != nil {
		return addPlan{}, err
	} else if !exists {
		return addPlan{}, fmt.Errorf("path does not exist: %s", absTarget)
	}

	homeAbs, err := filepath.Abs(filepath.Clean(homeDir))
	if err != nil {
		return addPlan{}, fmt.Errorf("resolving home directory: %w", err)
	}
	if containsPath(repoPath, absTarget) || containsPath(absTarget, repoPath) {
		return addPlan{}, fmt.Errorf("path %s overlaps the repo path %s", absTarget, repoPath)
	}

	relTarget, err := filepath.Rel(homeAbs, absTarget)
	if err != nil {
		return addPlan{}, fmt.Errorf("checking home containment: %w", err)
	}
	if relTarget == "." {
		return addPlan{}, fmt.Errorf("path %s is the home directory itself and cannot be added", absTarget)
	}
	if relTarget == ".." || strings.HasPrefix(relTarget, ".."+string(filepath.Separator)) || filepath.IsAbs(relTarget) {
		return addPlan{}, fmt.Errorf("path %s must stay under home directory %s", absTarget, homeAbs)
	}

	relSlash := filepath.ToSlash(relTarget)
	source := suggestedSourcePath(relSlash)
	repoSourcePath := filepath.Join(repoPath, filepath.FromSlash(source))
	if !containsPath(repoPath, repoSourcePath) {
		return addPlan{}, fmt.Errorf("resolved source path %s escapes repo path %s", repoSourcePath, repoPath)
	}

	return addPlan{
		Input:          rawPath,
		AbsTarget:      absTarget,
		RelTarget:      relSlash,
		TargetExpr:     path.Join("~", relSlash),
		Source:         source,
		RepoSourcePath: repoSourcePath,
		ManifestPath:   filepath.Join(repoPath, "manifest.yaml"),
		Sensitive:      isSensitiveManifestTargetRelPath(relSlash),
	}, nil
}

func normalizeAddInputPath(rawPath, cwd, homeDir string) (string, error) {
	trimmed := strings.TrimSpace(rawPath)
	if trimmed == "" {
		return "", fmt.Errorf("path is required")
	}

	expanded := trimmed
	if expanded == "~" {
		expanded = homeDir
	} else if strings.HasPrefix(expanded, "~/") {
		expanded = filepath.Join(homeDir, strings.TrimPrefix(expanded, "~/"))
	}

	abs := expanded
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(cwd, abs)
	}

	resolved, err := filepath.Abs(abs)
	if err != nil {
		return "", fmt.Errorf("resolving path %q: %w", rawPath, err)
	}

	return filepath.Clean(resolved), nil
}

func validateAddPlan(plan addPlan, force bool) ([]string, error) {
	if !plan.Sensitive {
		return nil, nil
	}

	warning := "path appears sensitive; use --force only if you intentionally want dotctl to manage it"
	if !force {
		return nil, fmt.Errorf("%s", warning)
	}

	return []string{warning}, nil
}

func loadAddManifest(manifestPath string) (*manifest.Manifest, bool, []byte, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &manifest.Manifest{Version: 1}, false, nil, nil
		}
		return nil, false, nil, fmt.Errorf("reading manifest: %w", err)
	}

	m, err := manifest.Parse(data)
	if err != nil {
		return nil, false, nil, err
	}

	if m.Version == 0 {
		m.Version = 1
	}

	return m, true, data, nil
}

func planAddManifest(m *manifest.Manifest, entry manifest.FileEntry, force bool) (*manifest.Manifest, bool, bool, error) {
	if m == nil {
		m = &manifest.Manifest{Version: 1}
	}

	if m.Version == 0 {
		m.Version = 1
	}

	updated := &manifest.Manifest{
		Version: m.Version,
		Vars:    copyStringMap(m.Vars),
		Ignore:  append([]string(nil), m.Ignore...),
	}

	files := make([]manifest.FileEntry, 0, len(m.Files)+1)
	exactMatch := false
	changed := false

	for _, existing := range m.Files {
		sameTarget := existing.Target == entry.Target
		sameSource := existing.Source == entry.Source
		sameMode := existing.LinkMode() == entry.LinkMode()

		if sameTarget && sameSource && sameMode {
			exactMatch = true
			files = append(files, existing)
			continue
		}

		if sameTarget {
			if !force {
				return nil, false, false, fmt.Errorf("manifest already has target %s mapped to %s", entry.Target, existing.Source)
			}
			changed = true
			continue
		}

		if sameSource {
			if !force {
				return nil, false, false, fmt.Errorf("manifest already tracks source %s at %s", entry.Source, existing.Target)
			}
			changed = true
			continue
		}

		files = append(files, existing)
	}

	if !exactMatch {
		files = append(files, entry)
		changed = true
	}

	updated.Files = files
	return updated, changed, exactMatch, nil
}

func writeAddManifest(manifestPath string, m *manifest.Manifest) error {
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
		return fmt.Errorf("creating manifest directory: %w", err)
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}

	if _, err := manifest.Parse(data); err != nil {
		return fmt.Errorf("validating manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0o644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	return nil
}

func restoreAddManifest(manifestPath string, existed bool, original []byte) error {
	if !existed {
		if err := os.Remove(manifestPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("removing manifest after failure: %w", err)
		}
		return nil
	}

	if err := os.WriteFile(manifestPath, original, 0o644); err != nil {
		return fmt.Errorf("restoring manifest: %w", err)
	}

	return nil
}

func containsPath(root, candidate string) bool {
	rootAbs, err := filepath.Abs(filepath.Clean(root))
	if err != nil {
		return false
	}
	candidateAbs, err := filepath.Abs(filepath.Clean(candidate))
	if err != nil {
		return false
	}

	rel, err := filepath.Rel(rootAbs, candidateAbs)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return false
	}
	return true
}

func dryRunStatus(alreadyConfigured, manifestChanged, shouldCopy bool) string {
	if alreadyConfigured && !manifestChanged && !shouldCopy {
		return "already_configured"
	}
	return "dry_run"
}

func successStatus(alreadyConfigured, manifestChanged, shouldCopy bool, backupPath string) string {
	if alreadyConfigured && !manifestChanged && !shouldCopy && backupPath == "" {
		return "already_configured"
	}
	return "ok"
}

func firstWarning(warnings []string) string {
	if len(warnings) == 0 {
		return ""
	}
	return warnings[0]
}

func hasLinkerError(results []linker.Result) bool {
	for _, result := range results {
		if result.Status == "error" {
			return true
		}
	}
	return false
}

func firstLinkerError(results []linker.Result) error {
	for _, result := range results {
		if result.Status == "error" && result.Error != nil {
			return fmt.Errorf("applying symlink for %s: %w", result.Action.Target, result.Error)
		}
	}
	return errors.New("linker returned an error state")
}

func copyStringMap(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
