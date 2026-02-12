package cmd

import (
	"time"

	"github.com/felipe-veas/dotctl/internal/gitops"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/spf13/cobra"
)

func newPushCmd() *cobra.Command {
	var message string

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Stage, commit and push local repo changes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(message)
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "custom commit message")

	return cmd
}

func runPush(message string) error {
	out := output.New(flagJSON)

	cfg, _, err := resolveConfig()
	if err != nil {
		return err
	}

	if flagDryRun {
		dirty, err := gitops.IsDirty(cfg.Repo.Path)
		if err != nil {
			return err
		}

		if out.IsJSON() {
			return out.JSON(map[string]any{
				"dry_run": true,
				"dirty":   dirty,
				"would":   "git add -A && git commit && git push",
			})
		}

		if dirty {
			out.Info("Would stage, commit and push local changes")
		} else {
			out.Info("Nothing to push")
		}
		return nil
	}

	res, err := gitops.Push(cfg.Repo.Path, message, cfg.Profile, time.Now())
	if err != nil {
		return err
	}

	if out.IsJSON() {
		return out.JSON(res)
	}

	if res.NothingToPush {
		out.Info("Nothing to push")
		return nil
	}

	out.Success("Committed and pushed")
	if res.Message != "" {
		out.Info("Message: %s", res.Message)
	}

	return nil
}
