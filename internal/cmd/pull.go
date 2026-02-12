package cmd

import (
	"github.com/felipe-veas/dotctl/internal/gitops"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/spf13/cobra"
)

func newPullCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pull",
		Short: "Run git pull --rebase",
		Args:  cobra.NoArgs,
		RunE:  runPull,
	}
}

func runPull(cmd *cobra.Command, args []string) error {
	out := output.New(flagJSON)

	cfg, _, err := resolveConfig()
	if err != nil {
		return err
	}

	if flagDryRun {
		if out.IsJSON() {
			return out.JSON(map[string]any{
				"dry_run": true,
				"command": "git pull --rebase",
			})
		}
		out.Info("Would run: git pull --rebase")
		return nil
	}

	pullOutput, err := gitops.PullRebase(cfg.Repo.Path)
	if err != nil {
		return err
	}

	if out.IsJSON() {
		return out.JSON(map[string]any{
			"status": "ok",
			"output": pullOutput,
		})
	}

	if pullOutput == "" {
		out.Success("Pull complete")
	} else {
		out.Success("Pull complete: %s", pullOutput)
	}

	return nil
}
