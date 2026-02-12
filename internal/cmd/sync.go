package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/felipe-veas/dotctl/internal/linker"
	"github.com/felipe-veas/dotctl/internal/manifest"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/felipe-veas/dotctl/internal/profile"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Pull, apply manifest, push changes",
		Long:  "Syncs dotfiles: applies the manifest to create symlinks/copies. (Git pull/push will be added in M2.)",
		Args:  cobra.NoArgs,
		RunE:  runSync,
	}
}

func runSync(cmd *cobra.Command, args []string) error {
	out := output.New(flagJSON)

	cfg, _, err := resolveConfig()
	if err != nil {
		return err
	}

	// Resolve profile context
	ctx := profile.Resolve(cfg.Profile)

	// Load manifest
	manifestPath := filepath.Join(cfg.Repo.Path, "manifest.yaml")
	m, err := manifest.Load(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	// Resolve actions
	actions, skipped, err := manifest.Resolve(m, ctx, cfg.Repo.Path)
	if err != nil {
		return fmt.Errorf("resolving manifest: %w", err)
	}

	// Report skipped entries
	for _, s := range skipped {
		out.Info("Skipped: %s (%s)", s.Source, s.SkipReason)
	}

	if len(actions) == 0 {
		out.Info("No actions to apply for profile %q on %s.", cfg.Profile, ctx.OS)
		if out.IsJSON() {
			return out.JSON(syncResult(nil, skipped, flagDryRun))
		}
		return nil
	}

	if flagDryRun {
		out.Header("Dry run (no changes will be made):")
	} else {
		out.Header(fmt.Sprintf("Applying manifest (profile: %s, os: %s)...", cfg.Profile, ctx.OS))
	}

	// Apply actions
	results := linker.Apply(actions, cfg.Repo.Path, flagDryRun)

	// Print results
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

	// Summary
	summary := linker.Summarize(results)
	if !flagDryRun {
		out.Info("")
		out.Info("Summary: %d created, %d already ok, %d backed up, %d errors",
			summary.Created+summary.Copied, summary.AlreadyOK, summary.BackedUp, summary.Errors)
	}

	if out.IsJSON() {
		return out.JSON(syncResult(results, skipped, flagDryRun))
	}

	if summary.Errors > 0 {
		return fmt.Errorf("%d errors during sync", summary.Errors)
	}

	return nil
}

type syncResultJSON struct {
	DryRun  bool              `json:"dry_run"`
	Applied []actionResultJSON `json:"applied"`
	Skipped []skippedJSON      `json:"skipped"`
	Summary summaryJSON        `json:"summary"`
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

func syncResult(results []linker.Result, skipped []manifest.Action, dryRun bool) syncResultJSON {
	var applied []actionResultJSON
	for _, r := range results {
		ar := actionResultJSON{
			Source: r.Action.Source,
			Target: r.Action.Target,
			Mode:   r.Action.Mode,
			Status: r.Status,
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
		DryRun:  dryRun,
		Applied: applied,
		Skipped: skippedList,
		Summary: summaryJSON{
			Created:   summary.Created + summary.Copied,
			AlreadyOK: summary.AlreadyOK,
			BackedUp:  summary.BackedUp,
			Errors:    summary.Errors,
		},
	}
}
