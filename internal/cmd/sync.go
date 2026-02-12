package cmd

import (
	"fmt"
	"time"

	"github.com/felipe-veas/dotctl/internal/backup"
	"github.com/felipe-veas/dotctl/internal/config"
	"github.com/felipe-veas/dotctl/internal/gitops"
	"github.com/felipe-veas/dotctl/internal/linker"
	"github.com/felipe-veas/dotctl/internal/lock"
	"github.com/felipe-veas/dotctl/internal/logging"
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

func runSync(cmd *cobra.Command, args []string) (err error) {
	out := output.New(flagJSON)

	cfg, cfgPath, err := resolveConfig()
	if err != nil {
		return err
	}
	logging.Info("sync start", "repo_path", cfg.Repo.Path, "profile", cfg.Profile, "dry_run", flagDryRun)

	syncLock, err := lock.Acquire(lock.DefaultSyncLockPath())
	if err != nil {
		return err
	}
	verbosef("sync lock acquired: %s", syncLock.Path())
	logging.Debug("sync lock acquired", "path", syncLock.Path())
	defer func() {
		if releaseErr := syncLock.Release(); releaseErr != nil {
			logging.Error("failed to release sync lock", "path", syncLock.Path(), "error", releaseErr)
			if err == nil {
				err = fmt.Errorf("releasing sync lock: %w", releaseErr)
			}
			return
		}
		logging.Debug("sync lock released", "path", syncLock.Path())
	}()

	pullOutput := ""
	if !flagDryRun {
		pullOutput, err = gitops.PullRebase(cfg.Repo.Path)
		if err != nil {
			return err
		}
		logging.Info("sync pull complete", "output", pullOutput)
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

	if decryptTool, decryptCount, decryptErr := detectDecryptToolForActions(state.Actions); decryptCount > 0 {
		if decryptErr != nil {
			return decryptErr
		}
		logging.Info("decrypt mode enabled", "entries", decryptCount, "tool", decryptTool)
		if !out.IsJSON() {
			out.Info("Decrypt enabled for %d file(s) using %s.", decryptCount, decryptTool)
		}
	}

	for _, s := range state.Skipped {
		out.Info("Skipped: %s (%s)", s.Source, s.SkipReason)
	}

	preHooks := manifest.ResolveHooks(state.Manifest.Hooks.PreSync, state.Context)
	postHooks := manifest.ResolveHooks(state.Manifest.Hooks.PostSync, state.Context)

	preHookResults, err := runHooks(out, "pre_sync", preHooks, cfg.Repo.Path, flagDryRun)
	if err != nil {
		if out.IsJSON() {
			_ = out.JSON(syncResult(nil, state.Skipped, flagDryRun, pullOutput, nil, preHookResults, nil, nil, nil))
		}
		return err
	}

	if len(state.Actions) == 0 {
		out.Info("No actions to apply for profile %q on %s.", cfg.Profile, state.Context.OS)

		postHookResults, err := runHooks(out, "post_sync", postHooks, cfg.Repo.Path, flagDryRun)
		if err != nil {
			if out.IsJSON() {
				_ = out.JSON(syncResult(nil, state.Skipped, flagDryRun, pullOutput, nil, preHookResults, postHookResults, nil, nil))
			}
			return err
		}

		var pushResult *gitops.PushResult
		var backupRotation *backup.RotationResult
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
			backupRotation = rotateBackups(out, cfg)
		}

		if out.IsJSON() {
			return out.JSON(syncResult(nil, state.Skipped, flagDryRun, pullOutput, pushResult, preHookResults, postHookResults, nil, backupRotation))
		}
		return nil
	}

	if flagDryRun {
		out.Header("Dry run (no changes will be made):")
	} else {
		out.Header(fmt.Sprintf("Applying manifest (profile: %s, os: %s)...", cfg.Profile, state.Context.OS))
	}

	results := linker.Apply(state.Actions, cfg.Repo.Path, flagDryRun)
	rollbackResults := make([]linker.RollbackResult, 0)
	rollbackIfNeeded := func(cause error) error {
		if flagDryRun {
			return cause
		}

		rollbackResults = linker.Rollback(results)
		if len(rollbackResults) == 0 {
			return cause
		}

		summary := linker.SummarizeRollback(rollbackResults)
		logging.Warn("sync rollback attempted", "restored", summary.Restored, "removed", summary.Removed, "errors", summary.Errors)
		if !out.IsJSON() {
			if summary.Errors == 0 {
				out.Warn("Sync failed, rollback complete (%d restored, %d removed).", summary.Restored, summary.Removed)
			} else {
				out.Warn("Sync failed, rollback finished with %d error(s).", summary.Errors)
			}
		}

		if summary.Errors > 0 {
			return fmt.Errorf("%w (rollback had %d errors)", cause, summary.Errors)
		}
		return cause
	}

	for _, r := range results {
		switch r.Status {
		case "created":
			out.Success("%s → %s (symlink created)", r.Action.Source, r.Action.Target)
		case "copied":
			if r.Decrypted {
				out.Success("%s → %s (decrypted and copied)", r.Action.Source, r.Action.Target)
			} else {
				out.Success("%s → %s (copied)", r.Action.Source, r.Action.Target)
			}
		case "already_linked":
			out.Success("%s → %s (already linked)", r.Action.Source, r.Action.Target)
		case "backed_up":
			if r.Decrypted {
				out.Success("%s → %s (backed up to %s, decrypted and copied)", r.Action.Source, r.Action.Target, r.BackupPath)
			} else {
				out.Success("%s → %s (backed up to %s)", r.Action.Source, r.Action.Target, r.BackupPath)
			}
		case "would_create":
			out.Info("  Would create symlink: %s → %s", r.Action.Source, r.Action.Target)
		case "would_copy":
			if r.Action.Decrypt {
				out.Info("  Would decrypt and copy: %s → %s", r.Action.Source, r.Action.Target)
			} else {
				out.Info("  Would copy: %s → %s", r.Action.Source, r.Action.Target)
			}
		case "would_backup_and_link":
			out.Info("  Would backup and link: %s → %s", r.Action.Source, r.Action.Target)
		case "would_backup_and_copy":
			if r.Action.Decrypt {
				out.Info("  Would backup, decrypt and copy: %s → %s", r.Action.Source, r.Action.Target)
			} else {
				out.Info("  Would backup and copy: %s → %s", r.Action.Source, r.Action.Target)
			}
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
		err = rollbackIfNeeded(fmt.Errorf("%d errors during sync", summary.Errors))
		if out.IsJSON() {
			_ = out.JSON(syncResult(results, state.Skipped, flagDryRun, pullOutput, nil, preHookResults, nil, rollbackResults, nil))
		}
		return err
	}

	postHookResults, err := runHooks(out, "post_sync", postHooks, cfg.Repo.Path, flagDryRun)
	if err != nil {
		err = rollbackIfNeeded(err)
		if out.IsJSON() {
			_ = out.JSON(syncResult(results, state.Skipped, flagDryRun, pullOutput, nil, preHookResults, postHookResults, rollbackResults, nil))
		}
		return err
	}

	var pushResult *gitops.PushResult
	var backupRotation *backup.RotationResult
	if !flagDryRun {
		res, pushErr := gitops.Push(cfg.Repo.Path, "", cfg.Profile, time.Now())
		if pushErr != nil {
			err = rollbackIfNeeded(pushErr)
			if out.IsJSON() {
				_ = out.JSON(syncResult(results, state.Skipped, flagDryRun, pullOutput, nil, preHookResults, postHookResults, rollbackResults, nil))
			}
			return err
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
			err = rollbackIfNeeded(err)
			if out.IsJSON() {
				_ = out.JSON(syncResult(results, state.Skipped, flagDryRun, pullOutput, pushResult, preHookResults, postHookResults, rollbackResults, nil))
			}
			return err
		}
		backupRotation = rotateBackups(out, cfg)
	}

	if out.IsJSON() {
		return out.JSON(syncResult(results, state.Skipped, flagDryRun, pullOutput, pushResult, preHookResults, postHookResults, rollbackResults, backupRotation))
	}

	logging.Info("sync complete", "profile", cfg.Profile, "dry_run", flagDryRun)
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
	DryRun        bool                `json:"dry_run"`
	PullOutput    string              `json:"pull_output,omitempty"`
	Applied       []actionResultJSON  `json:"applied"`
	Skipped       []skippedJSON       `json:"skipped"`
	PreSyncHooks  []hookResultJSON    `json:"pre_sync_hooks,omitempty"`
	PostSyncHooks []hookResultJSON    `json:"post_sync_hooks,omitempty"`
	Rollback      []rollbackJSON      `json:"rollback,omitempty"`
	BackupRotate  *backupRotationJSON `json:"backup_rotation,omitempty"`
	Summary       summaryJSON         `json:"summary"`
	Push          *gitops.PushResult  `json:"push,omitempty"`
}

type actionResultJSON struct {
	Source     string `json:"source"`
	Target     string `json:"target"`
	Mode       string `json:"mode"`
	Decrypt    bool   `json:"decrypt,omitempty"`
	Status     string `json:"status"`
	BackupPath string `json:"backup_path,omitempty"`
	Decrypted  bool   `json:"decrypted,omitempty"`
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

type rollbackJSON struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type backupRotationJSON struct {
	Kept    int `json:"kept"`
	Removed int `json:"removed"`
}

func syncResult(
	results []linker.Result,
	skipped []manifest.Action,
	dryRun bool,
	pullOutput string,
	push *gitops.PushResult,
	preHooks []hookResultJSON,
	postHooks []hookResultJSON,
	rollback []linker.RollbackResult,
	backupRotation *backup.RotationResult,
) syncResultJSON {
	var applied []actionResultJSON
	for _, r := range results {
		ar := actionResultJSON{
			Source:     r.Action.Source,
			Target:     r.Action.Target,
			Mode:       r.Action.Mode,
			Decrypt:    r.Action.Decrypt,
			Status:     r.Status,
			BackupPath: r.BackupPath,
			Decrypted:  r.Decrypted,
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

	var rollbackList []rollbackJSON
	for _, r := range rollback {
		item := rollbackJSON{
			Source: r.Action.Source,
			Target: r.Action.Target,
			Status: r.Status,
		}
		if r.Error != nil {
			item.Error = r.Error.Error()
		}
		rollbackList = append(rollbackList, item)
	}

	summary := linker.Summarize(results)
	var rotationJSON *backupRotationJSON
	if backupRotation != nil {
		rotationJSON = &backupRotationJSON{
			Kept:    backupRotation.Kept,
			Removed: backupRotation.Removed,
		}
	}

	return syncResultJSON{
		DryRun:        dryRun,
		PullOutput:    pullOutput,
		Applied:       applied,
		Skipped:       skippedList,
		PreSyncHooks:  preHooks,
		PostSyncHooks: postHooks,
		Rollback:      rollbackList,
		BackupRotate:  rotationJSON,
		Summary: summaryJSON{
			Created:   summary.Created + summary.Copied,
			AlreadyOK: summary.AlreadyOK,
			BackedUp:  summary.BackedUp,
			Errors:    summary.Errors,
		},
		Push: push,
	}
}

func rotateBackups(out *output.Printer, cfg *config.Config) *backup.RotationResult {
	result, err := backup.Rotate(cfg.Backup.Keep)
	if err != nil {
		logging.Warn("backup rotation failed", "error", err, "keep", cfg.Backup.Keep)
		if !out.IsJSON() {
			out.Warn("Backup rotation failed: %v", err)
		}
		return nil
	}

	if !out.IsJSON() && result.Removed > 0 {
		out.Info("Backup rotation: removed %d old snapshot(s), keeping %d.", result.Removed, result.Kept)
	}
	return &result
}
