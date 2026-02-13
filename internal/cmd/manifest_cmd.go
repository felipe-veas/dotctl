package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/felipe-veas/dotctl/internal/manifest"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var defaultManifestScanCandidates = []string{
	".zshrc",
	".zprofile",
	".bashrc",
	".bash_profile",
	".profile",
	".gitconfig",
	".gitignore",
	".tmux.conf",
	".vimrc",
	".config/nvim",
	".config/wezterm",
	".config/kitty",
	".config/alacritty",
	".config/starship.toml",
	".config/fish",
	".config/gh",
	".config/bat",
	".config/tmux",
	".config/helix",
	".config/lazygit",
	".config/ghostty",
}

var homeSourceCategoryByBase = map[string]string{
	".zshrc":        "zsh",
	".zprofile":     "zsh",
	".bashrc":       "bash",
	".bash_profile": "bash",
	".gitconfig":    "git",
	".gitignore":    "git",
	".tmux.conf":    "tmux",
	".vimrc":        "vim",
}

type manifestScanCandidate struct {
	Relative string `json:"relative"`
	Target   string `json:"target"`
	absPath  string
}

type suggestedManifest struct {
	Version int                     `yaml:"version"`
	Files   []suggestedManifestFile `yaml:"files"`
}

type suggestedManifestFile struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
	Mode   string `yaml:"mode,omitempty"`
}

type sourceCopyResult struct {
	Source    string `json:"source"`
	LocalPath string `json:"local_path"`
	RepoPath  string `json:"repo_path"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
}

func newManifestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "manifest",
		Short: "Manifest helpers",
	}

	cmd.AddCommand(newManifestSuggestCmd())
	return cmd
}

func newManifestSuggestCmd() *cobra.Command {
	var outputPath string
	var noCopySources bool

	cmd := &cobra.Command{
		Use:   "suggest",
		Short: "Scan common config paths and generate a suggested manifest",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runManifestSuggest(outputPath, !noCopySources)
		},
	}

	cmd.Flags().StringVar(&outputPath, "output", "", "output path for suggested manifest (default: <repo>/manifest.suggested.yaml)")
	cmd.Flags().BoolVar(&noCopySources, "no-copy-sources", false, "do not copy detected local config sources into repo paths referenced by the suggested manifest")
	return cmd
}

func runManifestSuggest(outputPath string, copySources bool) error {
	out := output.New(flagJSON)

	cfg, _, err := resolveConfig()
	if err != nil {
		return err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("detecting home directory: %w", err)
	}
	configHome := xdgConfigHome(homeDir)

	if !flagForce {
		if out.IsJSON() {
			return fmt.Errorf("--json requires --force for manifest suggest (confirmation is interactive)")
		}
		confirmed, err := requestScanAuthorization(os.Stdin, os.Stdout, homeDir, configHome, cfg.Repo.Path, copySources)
		if err != nil {
			return err
		}
		if !confirmed {
			out.Info("Manifest scan canceled")
			return nil
		}
	}

	candidates, skippedSensitive, err := discoverManifestCandidates(homeDir, configHome, defaultManifestScanCandidates)
	if err != nil {
		return err
	}
	entries := buildSuggestedManifestFiles(candidates)

	if len(entries) == 0 {
		if out.IsJSON() {
			return out.JSON(map[string]any{
				"status":               "no_suggestions",
				"repo_path":            cfg.Repo.Path,
				"scanned_candidates":   len(defaultManifestScanCandidates),
				"matched":              0,
				"skipped_sensitive":    skippedSensitive,
				"copy_sources_enabled": copySources,
			})
		}
		out.Warn("No common config paths detected in %s or %s", homeDir, configHome)
		return nil
	}

	data, err := renderSuggestedManifest(entries)
	if err != nil {
		return err
	}

	destination := resolveSuggestionPath(cfg.Repo.Path, outputPath)
	if !flagDryRun {
		if _, statErr := os.Stat(destination); statErr == nil && !flagForce {
			return fmt.Errorf("suggested manifest already exists: %s (use --force to overwrite)", destination)
		} else if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
			return fmt.Errorf("checking output file: %w", statErr)
		}

		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return fmt.Errorf("creating output directory: %w", err)
		}
		if err := os.WriteFile(destination, data, 0o644); err != nil {
			return fmt.Errorf("writing suggested manifest: %w", err)
		}
	}

	copyResults := make([]sourceCopyResult, 0)
	if copySources {
		results, copyErr := copySuggestedSources(cfg.Repo.Path, candidates, flagForce, flagDryRun)
		copyResults = results
		if copyErr != nil {
			if out.IsJSON() {
				return out.JSON(map[string]any{
					"status":               "copy_error",
					"dry_run":              flagDryRun,
					"repo_path":            cfg.Repo.Path,
					"output_path":          destination,
					"scanned_candidates":   len(defaultManifestScanCandidates),
					"matched":              len(entries),
					"skipped_sensitive":    skippedSensitive,
					"copy_sources_enabled": copySources,
					"copied_sources":       copyResults,
					"files":                entries,
					"error":                copyErr.Error(),
				})
			}
			return copyErr
		}
	}

	if out.IsJSON() {
		return out.JSON(map[string]any{
			"status":               "ok",
			"dry_run":              flagDryRun,
			"repo_path":            cfg.Repo.Path,
			"output_path":          destination,
			"scanned_candidates":   len(defaultManifestScanCandidates),
			"matched":              len(entries),
			"skipped_sensitive":    skippedSensitive,
			"copy_sources_enabled": copySources,
			"copied_sources":       copyResults,
			"files":                entries,
		})
	}

	if flagDryRun {
		out.Info("Dry run: suggested manifest would be written to %s", destination)
	} else {
		out.Success("Suggested manifest written to %s", destination)
	}
	out.Info("Detected %d candidate path(s)", len(entries))
	if skippedSensitive > 0 {
		out.Warn("Skipped %d sensitive candidate path(s)", skippedSensitive)
	}
	if copySources {
		copied, overwritten, skipped, copyErrors := summarizeCopyResults(copyResults)
		if flagDryRun {
			out.Info("Source copy dry run: %d would copy, %d would overwrite, %d skipped", copied, overwritten, skipped)
		} else {
			out.Info("Source copy summary: %d copied, %d overwritten, %d skipped, %d errors", copied, overwritten, skipped, copyErrors)
		}
	} else {
		out.Info("Source copy disabled by --no-copy-sources.")
	}
	out.Info("Review %s and merge entries into manifest.yaml.", destination)

	return nil
}

func requestScanAuthorization(in io.Reader, out io.Writer, homeDir, configHome, repoPath string, copySources bool) (bool, error) {
	action := "generate a suggested manifest"
	if copySources {
		action = fmt.Sprintf("generate a suggested manifest and copy detected sources into %s", repoPath)
	}
	question := fmt.Sprintf("dotctl will scan common config paths under %s and %s to %s. Continue? [y/N]: ", homeDir, configHome, action)
	return promptYesNo(in, out, question)
}

func promptYesNo(in io.Reader, out io.Writer, question string) (bool, error) {
	if _, err := fmt.Fprint(out, question); err != nil {
		return false, err
	}

	reader := bufio.NewReader(in)
	answer, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, fmt.Errorf("reading confirmation: %w", err)
	}

	switch strings.ToLower(strings.TrimSpace(answer)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

func discoverManifestCandidates(homeDir, configHome string, candidateRelPaths []string) ([]manifestScanCandidate, int, error) {
	found := make([]manifestScanCandidate, 0, len(candidateRelPaths))
	seenTargets := make(map[string]bool)
	skippedSensitive := 0

	for _, rel := range candidateRelPaths {
		trimmed := strings.TrimSpace(rel)
		if trimmed == "" {
			continue
		}

		abs := candidateAbsPath(homeDir, configHome, trimmed)
		_, err := os.Stat(abs)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, 0, fmt.Errorf("checking candidate %s: %w", abs, err)
		}

		relForSensitivity := strings.TrimPrefix(filepath.ToSlash(trimmed), "./")
		if isSensitiveTrackedPath(relForSensitivity) {
			skippedSensitive++
			continue
		}

		target := candidateTarget(trimmed, homeDir, configHome)
		if seenTargets[target] {
			continue
		}
		seenTargets[target] = true

		found = append(found, manifestScanCandidate{
			Relative: filepath.ToSlash(trimmed),
			Target:   target,
			absPath:  abs,
		})
	}

	sort.Slice(found, func(i, j int) bool {
		return found[i].Target < found[j].Target
	})

	return found, skippedSensitive, nil
}

func candidateAbsPath(homeDir, configHome, rel string) string {
	rel = filepath.Clean(rel)
	if strings.HasPrefix(filepath.ToSlash(rel), ".config/") {
		configRel := strings.TrimPrefix(filepath.ToSlash(rel), ".config/")
		return filepath.Join(configHome, filepath.FromSlash(configRel))
	}
	return filepath.Join(homeDir, rel)
}

func candidateTarget(rel, homeDir, configHome string) string {
	normalized := filepath.ToSlash(filepath.Clean(rel))
	if strings.HasPrefix(normalized, ".config/") {
		configRel := strings.TrimPrefix(normalized, ".config/")
		if pathsEqual(configHome, filepath.Join(homeDir, ".config")) {
			return path.Join("~/.config", configRel)
		}
		return filepath.ToSlash(filepath.Join(configHome, filepath.FromSlash(configRel)))
	}
	return path.Join("~", normalized)
}

func buildSuggestedManifestFiles(candidates []manifestScanCandidate) []suggestedManifestFile {
	files := make([]suggestedManifestFile, 0, len(candidates))
	seenSources := make(map[string]bool)
	seenTargets := make(map[string]bool)

	for _, candidate := range candidates {
		source := suggestedSourcePath(candidate.Relative)
		if source == "" || seenSources[source] || seenTargets[candidate.Target] {
			continue
		}
		seenSources[source] = true
		seenTargets[candidate.Target] = true

		files = append(files, suggestedManifestFile{
			Source: source,
			Target: candidate.Target,
			Mode:   "symlink",
		})
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Target < files[j].Target
	})

	return files
}

func suggestedSourcePath(relative string) string {
	normalized := filepath.ToSlash(filepath.Clean(relative))
	if strings.HasPrefix(normalized, ".config/") {
		return path.Join("configs", strings.TrimPrefix(normalized, ".config/"))
	}

	base := path.Base(normalized)
	category := homeSourceCategoryByBase[base]
	if category == "" {
		category = "home"
	}
	return path.Join("configs", category, base)
}

func renderSuggestedManifest(files []suggestedManifestFile) ([]byte, error) {
	m := suggestedManifest{
		Version: 1,
		Files:   files,
	}

	body, err := yaml.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("encoding suggested manifest: %w", err)
	}

	if _, err := manifest.Parse(body); err != nil {
		return nil, fmt.Errorf("validating suggested manifest: %w", err)
	}

	header := "# generated by dotctl manifest suggest\n# review and merge entries into manifest.yaml\n"
	return append([]byte(header), body...), nil
}

func copySuggestedSources(repoPath string, candidates []manifestScanCandidate, force, dryRun bool) ([]sourceCopyResult, error) {
	// Keep one candidate per source path in manifest (dedupe by first occurrence).
	bySource := make(map[string]manifestScanCandidate)
	for _, candidate := range candidates {
		source := suggestedSourcePath(candidate.Relative)
		if source == "" {
			continue
		}
		if _, exists := bySource[source]; !exists {
			bySource[source] = candidate
		}
	}

	sources := make([]string, 0, len(bySource))
	for source := range bySource {
		sources = append(sources, source)
	}
	sort.Strings(sources)

	results := make([]sourceCopyResult, 0, len(sources))
	errorCount := 0

	for _, source := range sources {
		candidate := bySource[source]
		repoTarget := filepath.Join(repoPath, filepath.FromSlash(source))
		result := sourceCopyResult{
			Source:    source,
			LocalPath: candidate.absPath,
			RepoPath:  repoTarget,
		}

		exists, existsErr := pathExists(repoTarget)
		if existsErr != nil {
			result.Status = "error"
			result.Error = existsErr.Error()
			errorCount++
			results = append(results, result)
			continue
		}

		if exists && !force {
			result.Status = "skipped_exists"
			results = append(results, result)
			continue
		}

		if dryRun {
			if exists {
				result.Status = "would_overwrite"
			} else {
				result.Status = "would_copy"
			}
			results = append(results, result)
			continue
		}

		if exists {
			if err := os.RemoveAll(repoTarget); err != nil {
				result.Status = "error"
				result.Error = fmt.Sprintf("removing existing path %s: %v", repoTarget, err)
				errorCount++
				results = append(results, result)
				continue
			}
		}

		if err := copyPathRecursive(candidate.absPath, repoTarget); err != nil {
			result.Status = "error"
			result.Error = fmt.Sprintf("copying %s to %s: %v", candidate.absPath, repoTarget, err)
			errorCount++
			results = append(results, result)
			continue
		}

		if exists {
			result.Status = "overwritten"
		} else {
			result.Status = "copied"
		}
		results = append(results, result)
	}

	if errorCount > 0 {
		return results, fmt.Errorf("copying detected source files failed with %d error(s)", errorCount)
	}

	return results, nil
}

func summarizeCopyResults(results []sourceCopyResult) (copied, overwritten, skipped, errors int) {
	for _, result := range results {
		switch result.Status {
		case "copied", "would_copy":
			copied++
		case "overwritten", "would_overwrite":
			overwritten++
		case "skipped_exists":
			skipped++
		case "error":
			errors++
		}
	}
	return copied, overwritten, skipped, errors
}

func pathExists(path string) (bool, error) {
	_, err := os.Lstat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, fmt.Errorf("checking %s: %w", path, err)
}

func copyPathRecursive(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}

	switch {
	case info.Mode()&os.ModeSymlink != 0:
		return copySymlink(src, dst)
	case info.IsDir():
		return copyDir(src, dst)
	default:
		return copyFileWithPerm(src, dst, info.Mode().Perm())
	}
}

func copySymlink(src, dst string) error {
	linkTarget, err := os.Readlink(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.Symlink(linkTarget, dst)
}

func copyDir(src, dst string) error {
	rootInfo, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dst, rootInfo.Mode().Perm()); err != nil {
		return err
	}

	return filepath.WalkDir(src, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(src, current)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}

		target := filepath.Join(dst, rel)
		info, err := os.Lstat(current)
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return copySymlink(current, target)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}

		return copyFileWithPerm(current, target, info.Mode().Perm())
	})
}

func copyFileWithPerm(src, dst string, perm os.FileMode) (err error) {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := in.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

func resolveSuggestionPath(repoPath, outputPath string) string {
	trimmed := strings.TrimSpace(outputPath)
	if trimmed == "" {
		return filepath.Join(repoPath, "manifest.suggested.yaml")
	}
	if filepath.IsAbs(trimmed) {
		return trimmed
	}
	return filepath.Join(repoPath, trimmed)
}

func xdgConfigHome(homeDir string) string {
	if fromEnv := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); fromEnv != "" {
		return fromEnv
	}
	return filepath.Join(homeDir, ".config")
}

func pathsEqual(left, right string) bool {
	leftAbs, err := filepath.Abs(left)
	if err != nil {
		return false
	}
	rightAbs, err := filepath.Abs(right)
	if err != nil {
		return false
	}
	leftClean := filepath.Clean(leftAbs)
	rightClean := filepath.Clean(rightAbs)

	if resolvedLeft, err := filepath.EvalSymlinks(leftClean); err == nil {
		leftClean = filepath.Clean(resolvedLeft)
	}
	if resolvedRight, err := filepath.EvalSymlinks(rightClean); err == nil {
		rightClean = filepath.Clean(resolvedRight)
	}

	return leftClean == rightClean
}
