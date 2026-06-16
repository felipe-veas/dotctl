package cmd

import (
	"fmt"
	"time"

	"github.com/felipe-veas/dotctl/internal/backup"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/felipe-veas/dotctl/internal/platform"
	"github.com/spf13/cobra"
)

type backupsListOutput struct {
	BackupDir string            `json:"backup_dir"`
	Snapshots []backup.Snapshot `json:"snapshots"`
}

type backupsRestoreOutput struct {
	Snapshot string                 `json:"snapshot"`
	DryRun   bool                   `json:"dry_run"`
	Targets  []string               `json:"targets,omitempty"`
	Entries  []backup.RestoreResult `json:"entries"`
}

func newBackupsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backups",
		Short: "Inspect and restore local backups",
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(
		newBackupsListCmd(),
		newBackupsRestoreCmd(),
	)

	return cmd
}

func newBackupsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List available backup snapshots",
		Args:  cobra.NoArgs,
		RunE:  runBackupsList,
	}
}

func newBackupsRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <snapshot>",
		Short: "Restore a backup snapshot",
		Args:  cobra.ExactArgs(1),
		RunE:  runBackupsRestore,
	}

	cmd.Flags().StringArray("target", nil, "restore only the specified target (repeatable)")

	return cmd
}

func runBackupsList(cmd *cobra.Command, args []string) error {
	out := output.New(flagJSON)

	snapshots, err := backup.List()
	if err != nil {
		return err
	}

	if out.IsJSON() {
		return out.JSON(backupsListOutput{BackupDir: platform.BackupDir(), Snapshots: snapshots})
	}

	out.Header("Backups")
	out.Field("Backup dir", platform.BackupDir())
	if len(snapshots) == 0 {
		out.Info("No snapshots found.")
		return nil
	}

	for _, snapshot := range snapshots {
		out.Info("%s  (%d entries, %s)", snapshot.Name, snapshot.Entries, snapshot.CreatedAt.Format(time.RFC3339))
	}

	return nil
}

func runBackupsRestore(cmd *cobra.Command, args []string) error {
	out := output.New(flagJSON)
	snapshot := args[0]
	targets, err := cmd.Flags().GetStringArray("target")
	if err != nil {
		return fmt.Errorf("reading target flags: %w", err)
	}

	if !flagDryRun && !flagForce {
		return fmt.Errorf("restore overwrites local targets; rerun with --force after reviewing --dry-run")
	}

	results, err := backup.RestoreSnapshotTargets(snapshot, targets, flagDryRun)
	if out.IsJSON() {
		payload := backupsRestoreOutput{Snapshot: snapshot, DryRun: flagDryRun, Entries: results}
		if len(targets) > 0 {
			payload.Targets = append([]string(nil), targets...)
		}
		if jsonErr := out.JSON(payload); jsonErr != nil {
			return jsonErr
		}
	} else {
		if flagDryRun {
			out.Header("Dry run")
			out.Info("Review the following entries before rerunning with --force.")
		} else {
			out.Header("Restore complete")
		}
		for _, result := range results {
			switch result.Status {
			case "planned":
				out.Info("Would restore %s from %s", result.Target, result.BackupPath)
			case "restored":
				out.Success("Restored %s", result.Target)
			case "rejected":
				out.Warn("Skipped %s: %s", result.Target, result.Error)
			case "error":
				out.Warn("Failed %s: %s", result.Target, result.Error)
			}
		}
	}

	return err
}
