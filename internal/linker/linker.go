package linker

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/felipe-veas/dotctl/internal/backup"
	"github.com/felipe-veas/dotctl/internal/manifest"
)

// Result represents the outcome of applying a single action.
type Result struct {
	Action     manifest.Action
	Status     string // "created", "already_linked", "backed_up", "copied", "skipped", "error"
	BackupPath string // non-empty if a backup was created
	Error      error
}

// Apply executes a list of actions (creating symlinks or copies).
// repoRoot is the absolute path to the cloned repo.
// If dryRun is true, no filesystem changes are made.
func Apply(actions []manifest.Action, repoRoot string, dryRun bool) []Result {
	var results []Result

	for _, action := range actions {
		sourcePath := filepath.Join(repoRoot, action.Source)
		r := applyOne(action, sourcePath, dryRun)
		results = append(results, r)
	}

	return results
}

func applyOne(action manifest.Action, sourcePath string, dryRun bool) Result {
	// Verify source exists
	if _, err := os.Stat(sourcePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{
				Action: action,
				Status: "error",
				Error:  fmt.Errorf("source %q not found in repo", action.Source),
			}
		}
		return Result{
			Action: action,
			Status: "error",
			Error:  wrapPathError("reading source", sourcePath, err),
		}
	}

	targetDir := filepath.Dir(action.Target)

	switch action.Mode {
	case "symlink":
		return applySymlink(action, sourcePath, targetDir, dryRun)
	case "copy":
		return applyCopy(action, sourcePath, targetDir, dryRun)
	default:
		return Result{Action: action, Status: "error", Error: fmt.Errorf("unknown mode: %s", action.Mode)}
	}
}

func applySymlink(action manifest.Action, sourcePath, targetDir string, dryRun bool) Result {
	// Check if target already exists
	info, err := os.Lstat(action.Target)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Result{
			Action: action,
			Status: "error",
			Error:  wrapPathError("checking existing target", action.Target, err),
		}
	}

	if err == nil {
		// Something exists at target
		if info.Mode()&os.ModeSymlink != 0 {
			// It's a symlink — check if it points to the right place
			dest, readErr := os.Readlink(action.Target)
			if readErr == nil && dest == sourcePath {
				return Result{Action: action, Status: "already_linked"}
			}
		}

		// Exists but is not the correct symlink — need backup
		if dryRun {
			return Result{Action: action, Status: "would_backup_and_link"}
		}

		backupPath := ""
		if action.Backup {
			var backupErr error
			backupPath, backupErr = backup.Create(action.Target)
			if backupErr != nil {
				return Result{Action: action, Status: "error", Error: wrapPathError("creating backup", action.Target, backupErr)}
			}

			if err := os.RemoveAll(action.Target); err != nil {
				return Result{Action: action, Status: "error", BackupPath: backupPath, Error: wrapPathError("removing old target", action.Target, err)}
			}

			return createSymlink(action, sourcePath, targetDir, backupPath)
		}

		// No backup, just overwrite
		if err := os.RemoveAll(action.Target); err != nil {
			return Result{Action: action, Status: "error", Error: wrapPathError("removing old target", action.Target, err)}
		}
	}

	if dryRun {
		return Result{Action: action, Status: "would_create"}
	}

	return createSymlink(action, sourcePath, targetDir, "")
}

func createSymlink(action manifest.Action, sourcePath, targetDir, backupPath string) Result {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return Result{Action: action, Status: "error", BackupPath: backupPath, Error: wrapPathError("creating target directory", targetDir, err)}
	}

	if err := os.Symlink(sourcePath, action.Target); err != nil {
		return Result{Action: action, Status: "error", BackupPath: backupPath, Error: wrapPathError("creating symlink", action.Target, err)}
	}

	status := "created"
	if backupPath != "" {
		status = "backed_up"
	}

	return Result{Action: action, Status: status, BackupPath: backupPath}
}

func applyCopy(action manifest.Action, sourcePath, targetDir string, dryRun bool) Result {
	// Check if target already exists
	if _, err := os.Lstat(action.Target); err != nil && !errors.Is(err, os.ErrNotExist) {
		return Result{
			Action: action,
			Status: "error",
			Error:  wrapPathError("checking existing target", action.Target, err),
		}
	} else if err == nil {
		if dryRun {
			return Result{Action: action, Status: "would_backup_and_copy"}
		}

		backupPath := ""
		if action.Backup {
			var backupErr error
			backupPath, backupErr = backup.Create(action.Target)
			if backupErr != nil {
				return Result{Action: action, Status: "error", Error: wrapPathError("creating backup", action.Target, backupErr)}
			}

			if err := os.RemoveAll(action.Target); err != nil {
				return Result{Action: action, Status: "error", BackupPath: backupPath, Error: wrapPathError("removing old target", action.Target, err)}
			}

			return doCopy(action, sourcePath, targetDir, backupPath)
		}

		if err := os.RemoveAll(action.Target); err != nil {
			return Result{Action: action, Status: "error", Error: wrapPathError("removing old target", action.Target, err)}
		}
	}

	if dryRun {
		return Result{Action: action, Status: "would_copy"}
	}

	return doCopy(action, sourcePath, targetDir, "")
}

func doCopy(action manifest.Action, sourcePath, targetDir, backupPath string) Result {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return Result{Action: action, Status: "error", BackupPath: backupPath, Error: wrapPathError("creating target directory", targetDir, err)}
	}

	srcInfo, err := os.Stat(sourcePath)
	if err != nil {
		return Result{Action: action, Status: "error", BackupPath: backupPath, Error: wrapPathError("reading source", sourcePath, err)}
	}

	if srcInfo.IsDir() {
		if err := copyDir(sourcePath, action.Target); err != nil {
			return Result{Action: action, Status: "error", BackupPath: backupPath, Error: wrapPathError("copying directory", action.Target, err)}
		}
	} else {
		if err := copyFile(sourcePath, action.Target, srcInfo.Mode()); err != nil {
			return Result{Action: action, Status: "error", BackupPath: backupPath, Error: wrapPathError("copying file", action.Target, err)}
		}
	}

	status := "copied"
	if backupPath != "" {
		status = "backed_up"
	}
	return Result{Action: action, Status: status, BackupPath: backupPath}
}

func copyFile(src, dst string, perm fs.FileMode) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return wrapPathError("opening source file", src, err)
	}
	defer func() {
		if closeErr := in.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return wrapPathError("opening destination file", dst, err)
	}
	defer func() {
		if closeErr := out.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(out, in)
	if err != nil {
		return wrapPathError("copying file contents", dst, err)
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return wrapPathError("walking source directory", path, err)
		}

		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return fmt.Errorf("building relative path for %s: %w", path, relErr)
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return wrapPathError("creating directory", target, err)
			}
			return nil
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return wrapPathError("reading source file metadata", path, infoErr)
		}
		return copyFile(path, target, info.Mode())
	})
}

// Summary counts results by status.
type Summary struct {
	Created     int
	AlreadyOK   int
	BackedUp    int
	Copied      int
	Errors      int
	WouldCreate int
	WouldBackup int
}

// Summarize aggregates results into a summary.
func Summarize(results []Result) Summary {
	var s Summary
	for _, r := range results {
		switch r.Status {
		case "created":
			s.Created++
		case "already_linked":
			s.AlreadyOK++
		case "backed_up":
			s.BackedUp++
		case "copied":
			s.Copied++
		case "error":
			s.Errors++
		case "would_create", "would_copy":
			s.WouldCreate++
		case "would_backup_and_link", "would_backup_and_copy":
			s.WouldBackup++
		}
	}
	return s
}

// RollbackResult describes one rollback action.
type RollbackResult struct {
	Action manifest.Action
	Status string // "removed", "restored", "skipped", "error"
	Error  error
}

// RollbackSummary counts rollback outcomes.
type RollbackSummary struct {
	Removed  int
	Restored int
	Skipped  int
	Errors   int
}

// Rollback reverts filesystem changes created by Apply in reverse order.
func Rollback(results []Result) []RollbackResult {
	rolledBack := make([]RollbackResult, 0)

	for i := len(results) - 1; i >= 0; i-- {
		res := results[i]
		if !needsRollback(res) {
			continue
		}
		rolledBack = append(rolledBack, rollbackOne(res))
	}

	return rolledBack
}

// SummarizeRollback aggregates rollback results.
func SummarizeRollback(results []RollbackResult) RollbackSummary {
	var s RollbackSummary
	for _, r := range results {
		switch r.Status {
		case "removed":
			s.Removed++
		case "restored":
			s.Restored++
		case "skipped":
			s.Skipped++
		case "error":
			s.Errors++
		}
	}
	return s
}

func needsRollback(r Result) bool {
	switch r.Status {
	case "created", "copied", "backed_up":
		return true
	case "error":
		return r.BackupPath != ""
	default:
		return false
	}
}

func rollbackOne(result Result) RollbackResult {
	switch result.Status {
	case "created", "copied":
		if err := os.RemoveAll(result.Action.Target); err != nil && !os.IsNotExist(err) {
			return RollbackResult{
				Action: result.Action,
				Status: "error",
				Error:  fmt.Errorf("removing target %q: %w", result.Action.Target, err),
			}
		}
		return RollbackResult{Action: result.Action, Status: "removed"}
	case "backed_up":
		if result.BackupPath == "" {
			return RollbackResult{
				Action: result.Action,
				Status: "error",
				Error:  fmt.Errorf("missing backup path for %q", result.Action.Target),
			}
		}
		if err := restoreFromBackup(result.BackupPath, result.Action.Target); err != nil {
			return RollbackResult{
				Action: result.Action,
				Status: "error",
				Error:  err,
			}
		}
		return RollbackResult{Action: result.Action, Status: "restored"}
	case "error":
		if result.BackupPath == "" {
			return RollbackResult{Action: result.Action, Status: "skipped"}
		}
		if err := restoreFromBackup(result.BackupPath, result.Action.Target); err != nil {
			return RollbackResult{
				Action: result.Action,
				Status: "error",
				Error:  err,
			}
		}
		return RollbackResult{Action: result.Action, Status: "restored"}
	default:
		return RollbackResult{Action: result.Action, Status: "skipped"}
	}
}

func restoreFromBackup(backupPath, target string) error {
	if err := os.RemoveAll(target); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing target before restore: %w", err)
	}

	info, err := os.Lstat(backupPath)
	if err != nil {
		return fmt.Errorf("stat backup %q: %w", backupPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return fmt.Errorf("creating target parent dir: %w", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		dest, readErr := os.Readlink(backupPath)
		if readErr != nil {
			return fmt.Errorf("reading backup symlink: %w", readErr)
		}
		if err := os.Symlink(dest, target); err != nil {
			return fmt.Errorf("restoring symlink: %w", err)
		}
		return nil
	}

	if info.IsDir() {
		if err := copyDir(backupPath, target); err != nil {
			return fmt.Errorf("restoring directory: %w", err)
		}
		return nil
	}

	if err := copyFile(backupPath, target, info.Mode()); err != nil {
		return fmt.Errorf("restoring file: %w", err)
	}
	return nil
}
