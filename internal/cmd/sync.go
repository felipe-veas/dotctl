package cmd

import (
	"fmt"
	"time"

	"github.com/felipe-veas/dotctl/internal/config"
	"github.com/felipe-veas/dotctl/internal/gitops"
	"github.com/felipe-veas/dotctl/internal/linker"
	"github.com/felipe-veas/dotctl/internal/manifest"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Pull, apply manifest, push changes",
		Long:  "Syncs dotfiles with full flow: git pull --rebase, apply manifest, git push.",
		Args:  cobra.NoArgs,
		RunE:  runSync,
	}
}

func runSync(cmd *cobra.Command, args []string) error {
	out := output.New(flagJSON)

	cfg, cfgPath, err := resolveConfig()
	if err != nil {
		return err
	}

	pullOutput := ""
	if !flagDryRun {
		pullOutput, err = gitops.PullRebase(cfg.Repo.Path)
		if err != nil {
			return err
		}
		if !out.IsJSON() {
			if pullOutput == "" {
				out.Success("Pull complete")
			} else {
				out.Success("Pull complete: %s", pullOutput)
			}
		}
	}

	state, err := resolveManifestState(cfg)
	if err != nil {
		return err
	}

	for _, s := range state.Skipped {
		out.Info("Skipped: %s (%s)", s.Source, s.SkipReason)
	}

	preHooks := manifest.ResolveHooks(state.Manifest.Hooks.PreSync, state.Context)
	postHooks := manifest.ResolveHooks(state.Manifest.Hooks.PostSync, state.Context)

	preHookResults, err := runHooks(out, "pre_sync", preHooks, cfg.Repo.Path, flagDryRun)
	if err != nil {
		if out.IsJSON() {
			_ = out.JSON(syncResult(nil, state.Skipped, flagDryRun, pullOutput, nil, preHookResults, nil))
		}
		return err
	}

	if len(state.Actions) == 0 {
		out.Info("No actions to apply for profile %q on %s.", cfg.Profile, state.Context.OS)

		postHookResults, err := runHooks(out, "post_sync", postHooks, cfg.Repo.Path, flagDryRun)
		if err != nil {
			if out.IsJSON() {
				_ = out.JSON(syncResult(nil, state.Skipped, flagDryRun, pullOutput, nil, preHookResults, postHookResults))
			}
			return err
		}

		var pushResult *gitops.PushResult
		if !flagDryRun {
			res, pushErr := gitops.Push(cfg.Repo.Path, "", cfg.Profile, time.Now())
			if pushErr != nil {
				return pushErr
			}
			pushResult = &res
			if res.NothingToPush {
				out.Info("Nothing to push")
			}
			if err := persistLastSync(cfgPath, cfg); err != nil {
				return err
			}
		}

		if out.IsJSON() {
			return out.JSON(syncResult(nil, state.Skipped, flagDryRun, pullOutput, pushResult, preHookResults, postHookResults))
		}
		return nil
	}

	if flagDryRun {
		out.Header("Dry run (no changes will be made):")
	} else {
		out.Header(fmt.Sprintf("Applying manifest (profile: %s, os: %s)...", cfg.Profile, state.Context.OS))
	}

	results := linker.Apply(state.Actions, cfg.Repo.Path, flagDryRun)

	for _, r := range results {
		switch r.Status {
		case "created":
			out.Success("%s → %s (symlink created)", r.Action.Source, r.Action.Target)
		case "copied":
			out.Success("%s → %s (copied)", r.Action.Source, r.Action.Target)
		case "already_linked":
			out.Success("%s → %s (already linked)", r.Action.Source, r.Action.Target)
		case "backed_up":
			out.Success("%s → %s (backed up to %s)", r.Action.Source, r.Action.Target, r.BackupPath)
		case "would_create":
			out.Info("  Would create symlink: %s → %s", r.Action.Source, r.Action.Target)
		case "would_copy":
			out.Info("  Would copy: %s → %s", r.Action.Source, r.Action.Target)
		case "would_backup_and_link":
			out.Info("  Would backup and link: %s → %s", r.Action.Source, r.Action.Target)
		case "would_backup_and_copy":
			out.Info("  Would backup and copy: %s → %s", r.Action.Source, r.Action.Target)
		case "error":
			out.Error("%s → %s: %v", r.Action.Source, r.Action.Target, r.Error)
		}
	}

	summary := linker.Summarize(results)
	if !flagDryRun {
		out.Info("")
		out.Info("Summary: %d created, %d already ok, %d backed up, %d errors",
			summary.Created+summary.Copied, summary.AlreadyOK, summary.BackedUp, summary.Errors)
	}

	if summary.Errors > 0 {
		if out.IsJSON() {
			return out.JSON(syncResult(results, state.Skipped, flagDryRun, pullOutput, nil, preHookResults, nil))
		}
		return fmt.Errorf("%d errors during sync", summary.Errors)
	}

	postHookResults, err := runHooks(out, "post_sync", postHooks, cfg.Repo.Path, flagDryRun)
	if err != nil {
		if out.IsJSON() {
			_ = out.JSON(syncResult(results, state.Skipped, flagDryRun, pullOutput, nil, preHookResults, postHookResults))
		}
		return err
	}

	var pushResult *gitops.PushResult
	if !flagDryRun {
		res, pushErr := gitops.Push(cfg.Repo.Path, "", cfg.Profile, time.Now())
		if pushErr != nil {
			return pushErr
		}
		pushResult = &res

		if !out.IsJSON() {
			if res.NothingToPush {
				out.Info("Nothing to push")
			} else {
				out.Success("Pushed changes")
			}
		}

		if err := persistLastSync(cfgPath, cfg); err != nil {
			return err
		}
	}

	if out.IsJSON() {
		return out.JSON(syncResult(results, state.Skipped, flagDryRun, pullOutput, pushResult, preHookResults, postHookResults))
	}

	return nil
}

func persistLastSync(cfgPath string, cfg *config.Config) error {
	now := time.Now().UTC()
	cfg.LastSync = &now
	if err := config.Save(cfgPath, cfg); err != nil {
		return fmt.Errorf("saving last sync timestamp: %w", err)
	}
	return nil
}

type syncResultJSON struct {
	DryRun        bool               `json:"dry_run"`
	PullOutput    string             `json:"pull_output,omitempty"`
	Applied       []actionResultJSON `json:"applied"`
	Skipped       []skippedJSON      `json:"skipped"`
	PreSyncHooks  []hookResultJSON   `json:"pre_sync_hooks,omitempty"`
	PostSyncHooks []hookResultJSON   `json:"post_sync_hooks,omitempty"`
	Summary       summaryJSON        `json:"summary"`
	Push          *gitops.PushResult `json:"push,omitempty"`
}

type actionResultJSON struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	Mode       string `json:"mode"`
	Status     string `json:"status"`
	BackupPath string `json:"backup_path,omitempty"`
	Error      string `json:"error,omitempty"`
}

type skippedJSON struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Reason string `json:"reason"`
}

type summaryJSON struct {
	Created   int `json:"created"`
	AlreadyOK int `json:"already_ok"`
	BackedUp  int `json:"backed_up"`
	Errors    int `json:"errors"`
}

func syncResult(
	results []linker.Result,
	skipped []manifest.Action,
	dryRun bool,
	pullOutput string,
	push *gitops.PushResult,
	preHooks []hookResultJSON,
	postHooks []hookResultJSON,
) syncResultJSON {
	var applied []actionResultJSON
	for _, r := range results {
		ar := actionResultJSON{
			Source:     r.Action.Source,
			Target:     r.Action.Target,
			Mode:       r.Action.Mode,
			Status:     r.Status,
			BackupPath: r.BackupPath,
		}
		if r.Error != nil {
			ar.Error = r.Error.Error()
		}
		applied = append(applied, ar)
	}

	var skippedList []skippedJSON
	for _, s := range skipped {
		skippedList = append(skippedList, skippedJSON{
			Source: s.Source,
			Target: s.Target,
			Reason: s.SkipReason,
		})
	}

	summary := linker.Summarize(results)
	return syncResultJSON{
		DryRun:        dryRun,
		PullOutput:    pullOutput,
		Applied:       applied,
		Skipped:       skippedList,
		PreSyncHooks:  preHooks,
		PostSyncHooks: postHooks,
		Summary: summaryJSON{
			Created:   summary.Created + summary.Copied,
			AlreadyOK: summary.AlreadyOK,
			BackedUp:  summary.BackedUp,
			Errors:    summary.Errors,
		},
		Push: push,
	}
}
