package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
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

	cmd := &cobra.Command{
		Use:   "suggest",
		Short: "Scan common config paths and generate a suggested manifest",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runManifestSuggest(outputPath)
		},
	}

	cmd.Flags().StringVar(&outputPath, "output", "", "output path for suggested manifest (default: <repo>/manifest.suggested.yaml)")
	return cmd
}

func runManifestSuggest(outputPath string) error {
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
		confirmed, err := requestScanAuthorization(os.Stdin, os.Stdout, homeDir, configHome)
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
				"status":             "no_suggestions",
				"repo_path":          cfg.Repo.Path,
				"scanned_candidates": len(defaultManifestScanCandidates),
				"matched":            0,
				"skipped_sensitive":  skippedSensitive,
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

	if out.IsJSON() {
		return out.JSON(map[string]any{
			"status":             "ok",
			"dry_run":            flagDryRun,
			"repo_path":          cfg.Repo.Path,
			"output_path":        destination,
			"scanned_candidates": len(defaultManifestScanCandidates),
			"matched":            len(entries),
			"skipped_sensitive":  skippedSensitive,
			"files":              entries,
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
	out.Info("Review %s and merge selected entries into manifest.yaml.", destination)

	return nil
}

func requestScanAuthorization(in io.Reader, out io.Writer, homeDir, configHome string) (bool, error) {
	question := fmt.Sprintf(
		"dotctl will scan common config paths under %s and %s to generate a suggested manifest. Continue? [y/N]: ",
		homeDir,
		configHome,
	)
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
