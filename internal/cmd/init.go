package cmd

import (
	"errors"
	"fmt"
	"runtime"

	"github.com/felipe-veas/dotctl/internal/auth"
	"github.com/felipe-veas/dotctl/internal/config"
	"github.com/felipe-veas/dotctl/internal/gitops"
	"github.com/felipe-veas/dotctl/internal/logging"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var repoURL string
	var repoPath string
	var repoName string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize dotctl: configure repo and profile",
		Long:  "Sets up dotctl on this machine by validating auth, cloning the repo, and saving config.",
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output.New(flagJSON)

			cfgPath := flagConfig
			if cfgPath == "" {
				cfgPath = config.DefaultPath()
			}

			if repoURL == "" {
				return fmt.Errorf("--repo is required")
			}
			repoName = config.NormalizeRepoName(repoName)
			if repoName == "" {
				repoName = config.DefaultRepoName
			}

			cfg := &config.Config{}
			if config.Exists(cfgPath) {
				existing, err := config.Load(cfgPath)
				if err != nil {
					return err
				}
				cfg = existing

				if cfg.Profile == "" && flagProfile == "" {
					return fmt.Errorf("--profile is required for first-time initialization")
				}
				if flagProfile != "" {
					cfg.Profile = flagProfile
				}
			} else {
				profile := flagProfile
				if profile == "" {
					return fmt.Errorf("--profile is required")
				}
				cfg.Profile = profile
			}

			if repoPath == "" {
				repoPath = config.DefaultRepoPath(repoName)
			}

			existsByName := false
			for _, repo := range cfg.Repos {
				if repo.Name == repoName {
					existsByName = true
					break
				}
			}
			if existsByName && !flagForce {
				return fmt.Errorf("repo %q already configured; use --force to update it", repoName)
			}

			authMethod := "ssh"
			authUser := ""
			if !gitops.IsSSHURL(repoURL) {
				authMethod = "gh"
				user, err := auth.EnsureGHAuthenticated()
				if err != nil {
					return err
				}
				authUser = user
				if !out.IsJSON() {
					if authUser != "" {
						out.Success("gh authenticated as %s", authUser)
					} else {
						out.Success("gh authenticated")
					}
				}
			}

			repoAlreadyCloned := gitops.IsRepo(repoPath)
			if err := gitops.Clone(repoURL, repoPath); err != nil {
				return err
			}
			if !out.IsJSON() {
				if repoAlreadyCloned {
					out.Info("Repo already cloned at %s", repoPath)
				} else {
					out.Success("Cloned to %s", repoPath)
				}
			}

			_, err := cfg.UpsertRepo(config.RepoConfig{
				Name: repoName,
				URL:  repoURL,
				Path: repoPath,
			})
			if err != nil {
				return err
			}
			if err := cfg.SetActiveRepo(repoName); err != nil {
				return err
			}

			if err := config.Save(cfgPath, cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			if out.IsJSON() {
				return out.JSON(map[string]string{
					"config_path": cfgPath,
					"repo":        repoURL,
					"repo_name":   repoName,
					"profile":     cfg.Profile,
					"repo_path":   repoPath,
					"auth_method": authMethod,
					"auth_user":   authUser,
					"status":      "initialized",
				})
			}

			out.Success("Profile: %s", cfg.Profile)
			out.Success("Repo: %s (%s)", repoName, repoURL)
			out.Info("Configured repos: %d", len(cfg.Repos))
			out.Success("Config saved to %s", cfgPath)
			out.Info("")
			out.Info("Next: run 'dotctl sync' to apply your dotfiles.")

			return nil
		},
	}

	cmd.Flags().StringVar(&repoURL, "repo", "", "GitHub repo URL (HTTPS or SSH)")
	cmd.Flags().StringVar(&repoPath, "path", "", "local path to clone repo (default: ~/.config/dotctl/repo)")
	cmd.Flags().StringVar(&repoName, "name", config.DefaultRepoName, "repo name for multi-repo configs")

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
	if flagRepoName != "" {
		if err := cfg.SetActiveRepo(flagRepoName); err != nil {
			return nil, cfgPath, err
		}
	}

	activeRepo, err := cfg.Active()
	if err != nil {
		return nil, cfgPath, err
	}
	cfg.Repo = activeRepo

	verbosef("config: path=%s repo_name=%s repo=%s profile=%s", cfgPath, cfg.Repo.Name, cfg.Repo.Path, cfg.Profile)
	logging.Debug(
		"resolved config",
		"path", cfgPath,
		"repo_name", cfg.Repo.Name,
		"repo_url", cfg.Repo.URL,
		"repo_path", cfg.Repo.Path,
		"profile", cfg.Profile,
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
	)

	return cfg, cfgPath, nil
}
