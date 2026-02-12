package cmd

import (
	"fmt"

	"github.com/felipe-veas/dotctl/internal/gitops"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/felipe-veas/dotctl/internal/platform"
	"github.com/spf13/cobra"
)

func newOpenCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "open",
		Short: "Open repository in browser",
		Args:  cobra.NoArgs,
		RunE:  runOpen,
	}
}

func runOpen(cmd *cobra.Command, args []string) error {
	out := output.New(flagJSON)

	cfg, _, err := resolveConfig()
	if err != nil {
		return err
	}

	url := gitops.BrowserURL(cfg.Repo.URL)
	if url == "" {
		return fmt.Errorf("invalid repo URL in config")
	}

	if flagDryRun {
		if out.IsJSON() {
			return out.JSON(map[string]any{
				"dry_run": true,
				"url":     url,
			})
		}
		out.Info("Would open %s", url)
		return nil
	}

	if err := platform.OpenURL(url); err != nil {
		return fmt.Errorf("opening URL: %w", err)
	}

	if out.IsJSON() {
		return out.JSON(map[string]string{
			"status": "opened",
			"url":    url,
		})
	}

	out.Success("Opened %s", url)
	return nil
}
