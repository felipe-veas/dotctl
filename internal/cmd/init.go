package cmd

import (
	"errors"
	"fmt"

	"github.com/felipe-veas/dotctl/internal/config"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/felipe-veas/dotctl/internal/platform"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var repoURL string
	var repoPath string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize dotctl: configure repo and profile",
		Long:  "Sets up dotctl on this machine by saving the repo URL and profile to the local config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output.New(flagJSON)

			cfgPath := flagConfig
			if cfgPath == "" {
				cfgPath = config.DefaultPath()
			}

			// Validate required flags
			if repoURL == "" {
				return fmt.Errorf("--repo is required")
			}
			profile := flagProfile
			if profile == "" {
				return fmt.Errorf("--profile is required")
			}

			// Warn if config already exists
			if config.Exists(cfgPath) && !flagForce {
				existing, err := config.Load(cfgPath)
				if err == nil {
					out.Warn("Config already exists (profile: %s, repo: %s)", existing.Profile, existing.Repo.URL)
					out.Warn("Use --force to overwrite")
					return fmt.Errorf("config already exists at %s", cfgPath)
				}
			}

			// Build config
			if repoPath == "" {
				repoPath = platform.RepoDir()
			}

			cfg := &config.Config{
				Repo: config.RepoConfig{
					URL:  repoURL,
					Path: repoPath,
				},
				Profile: profile,
			}

			if err := config.Save(cfgPath, cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			if out.IsJSON() {
				return out.JSON(map[string]string{
					"config_path": cfgPath,
					"repo":        repoURL,
					"profile":     profile,
					"repo_path":   repoPath,
					"status":      "initialized",
				})
			}

			out.Success("Profile set to: %s", profile)
			out.Success("Repo: %s", repoURL)
			out.Success("Repo path: %s", repoPath)
			out.Success("Config saved to %s", cfgPath)
			out.Info("")
			out.Info("Next: run 'dotctl sync' to apply your dotfiles.")

			return nil
		},
	}

	cmd.Flags().StringVar(&repoURL, "repo", "", "GitHub repo URL (HTTPS or SSH)")
	cmd.Flags().StringVar(&repoPath, "path", "", "local path to clone repo (default: ~/.config/dotctl/repo)")

	return cmd
}

// resolveConfig loads the config or returns a helpful error.
func resolveConfig() (*config.Config, string, error) {
	cfgPath := flagConfig
	if cfgPath == "" {
		cfgPath = config.DefaultPath()
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		if errors.Is(err, config.ErrNotFound) {
			return nil, cfgPath, fmt.Errorf("dotctl not initialized — run: dotctl init --repo <url> --profile <name>")
		}
		return nil, cfgPath, err
	}

	// Flag overrides config
	if flagProfile != "" {
		cfg.Profile = flagProfile
	}

	return cfg, cfgPath, nil
}
