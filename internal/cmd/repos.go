package cmd

import (
	"fmt"

	"github.com/felipe-veas/dotctl/internal/auth"
	"github.com/felipe-veas/dotctl/internal/config"
	"github.com/felipe-veas/dotctl/internal/gitops"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/spf13/cobra"
)

func newReposCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repos",
		Short: "Manage configured repositories",
	}

	cmd.AddCommand(
		newReposListCmd(),
		newReposAddCmd(),
		newReposUseCmd(),
		newReposRemoveCmd(),
	)

	return cmd
}

func newReposListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured repositories",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output.New(flagJSON)

			cfg, _, err := resolveConfig()
			if err != nil {
				return err
			}

			type item struct {
				Name   string `json:"name"`
				URL    string `json:"url"`
				Path   string `json:"path"`
				Active bool   `json:"active"`
			}

			items := make([]item, 0, len(cfg.Repos))
			for _, repo := range cfg.Repos {
				items = append(items, item{
					Name:   repo.Name,
					URL:    repo.URL,
					Path:   repo.Path,
					Active: repo.Name == cfg.ActiveRepo,
				})
			}

			if out.IsJSON() {
				return out.JSON(map[string]any{
					"active_repo": cfg.ActiveRepo,
					"repos":       items,
				})
			}

			out.Field("Active repo", cfg.ActiveRepo)
			for _, repo := range items {
				marker := " "
				if repo.Active {
					marker = "*"
				}
				out.Info("%s %s", marker, repo.Name)
				out.Info("    URL:  %s", repo.URL)
				out.Info("    Path: %s", repo.Path)
			}
			return nil
		},
	}
}

func newReposAddCmd() *cobra.Command {
	var name string
	var url string
	var path string
	var activate bool

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a repository to config",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output.New(flagJSON)

			cfg, cfgPath, err := resolveConfig()
			if err != nil {
				return err
			}

			if url == "" {
				return fmt.Errorf("--url is required")
			}
			name = config.NormalizeRepoName(name)
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			if path == "" {
				path = config.DefaultRepoPath(name)
			}

			for _, repo := range cfg.Repos {
				if repo.Name == name && !flagForce {
					return fmt.Errorf("repo %q already configured; use --force to update", name)
				}
			}

			if !gitops.IsSSHURL(url) {
				if _, err := auth.EnsureGHAuthenticated(); err != nil {
					return err
				}
			}

			alreadyCloned := gitops.IsRepo(path)
			if err := gitops.Clone(url, path); err != nil {
				return err
			}

			if _, err := cfg.UpsertRepo(config.RepoConfig{
				Name: name,
				URL:  url,
				Path: path,
			}); err != nil {
				return err
			}
			if activate {
				if err := cfg.SetActiveRepo(name); err != nil {
					return err
				}
			}

			if err := config.Save(cfgPath, cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			if out.IsJSON() {
				return out.JSON(map[string]any{
					"status":      "ok",
					"name":        name,
					"url":         url,
					"path":        path,
					"active_repo": cfg.ActiveRepo,
					"cloned":      !alreadyCloned,
				})
			}

			if alreadyCloned {
				out.Info("Repo already cloned at %s", path)
			} else {
				out.Success("Cloned to %s", path)
			}
			out.Success("Configured repo %s", name)
			if activate {
				out.Success("Active repo: %s", cfg.ActiveRepo)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "repo name")
	cmd.Flags().StringVar(&url, "url", "", "GitHub repo URL (HTTPS or SSH)")
	cmd.Flags().StringVar(&path, "path", "", "local path to clone repo")
	cmd.Flags().BoolVar(&activate, "activate", true, "set this repo as active")

	return cmd
}

func newReposUseCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set active repo",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output.New(flagJSON)

			cfg, cfgPath, err := resolveConfig()
			if err != nil {
				return err
			}

			if err := cfg.SetActiveRepo(args[0]); err != nil {
				return err
			}
			if err := config.Save(cfgPath, cfg); err != nil {
				return err
			}

			if out.IsJSON() {
				return out.JSON(map[string]any{
					"status":      "ok",
					"active_repo": cfg.ActiveRepo,
				})
			}

			out.Success("Active repo: %s", cfg.ActiveRepo)
			return nil
		},
	}
}

func newReposRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a repo from config (does not delete files)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output.New(flagJSON)

			cfg, cfgPath, err := resolveConfig()
			if err != nil {
				return err
			}

			if err := cfg.RemoveRepo(args[0]); err != nil {
				return err
			}
			if err := config.Save(cfgPath, cfg); err != nil {
				return err
			}

			if out.IsJSON() {
				return out.JSON(map[string]any{
					"status":      "ok",
					"active_repo": cfg.ActiveRepo,
				})
			}

			out.Success("Repo removed")
			out.Success("Active repo: %s", cfg.ActiveRepo)
			return nil
		},
	}
}
