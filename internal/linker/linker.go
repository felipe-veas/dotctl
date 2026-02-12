package linker

import (
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
		return Result{
			Action: action,
			Status: "error",
			Error:  fmt.Errorf("source %q not found in repo", action.Source),
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

		if action.Backup {
			backupPath, backupErr := backup.Create(action.Target)
			if backupErr != nil {
				return Result{Action: action, Status: "error", Error: fmt.Errorf("backup: %w", backupErr)}
			}

			if err := os.RemoveAll(action.Target); err != nil {
				return Result{Action: action, Status: "error", Error: fmt.Errorf("removing old target: %w", err)}
			}

			return createSymlink(action, sourcePath, targetDir, backupPath)
		}

		// No backup, just overwrite
		if err := os.RemoveAll(action.Target); err != nil {
			return Result{Action: action, Status: "error", Error: fmt.Errorf("removing old target: %w", err)}
		}
	}

	if dryRun {
		return Result{Action: action, Status: "would_create"}
	}

	return createSymlink(action, sourcePath, targetDir, "")
}

func createSymlink(action manifest.Action, sourcePath, targetDir, backupPath string) Result {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return Result{Action: action, Status: "error", Error: fmt.Errorf("creating target dir: %w", err)}
	}

	if err := os.Symlink(sourcePath, action.Target); err != nil {
		return Result{Action: action, Status: "error", Error: fmt.Errorf("creating symlink: %w", err)}
	}

	status := "created"
	if backupPath != "" {
		status = "backed_up"
	}

	return Result{Action: action, Status: status, BackupPath: backupPath}
}

func applyCopy(action manifest.Action, sourcePath, targetDir string, dryRun bool) Result {
	// Check if target already exists
	if _, err := os.Lstat(action.Target); err == nil {
		if dryRun {
			return Result{Action: action, Status: "would_backup_and_copy"}
		}

		if action.Backup {
			backupPath, backupErr := backup.Create(action.Target)
			if backupErr != nil {
				return Result{Action: action, Status: "error", Error: fmt.Errorf("backup: %w", backupErr)}
			}

			if err := os.RemoveAll(action.Target); err != nil {
				return Result{Action: action, Status: "error", Error: fmt.Errorf("removing old target: %w", err)}
			}

			return doCopy(action, sourcePath, targetDir, backupPath)
		}

		if err := os.RemoveAll(action.Target); err != nil {
			return Result{Action: action, Status: "error", Error: fmt.Errorf("removing old target: %w", err)}
		}
	}

	if dryRun {
		return Result{Action: action, Status: "would_copy"}
	}

	return doCopy(action, sourcePath, targetDir, "")
}

func doCopy(action manifest.Action, sourcePath, targetDir, backupPath string) Result {
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return Result{Action: action, Status: "error", Error: fmt.Errorf("creating target dir: %w", err)}
	}

	srcInfo, err := os.Stat(sourcePath)
	if err != nil {
		return Result{Action: action, Status: "error", Error: err}
	}

	if srcInfo.IsDir() {
		if err := copyDir(sourcePath, action.Target); err != nil {
			return Result{Action: action, Status: "error", Error: err}
		}
	} else {
		if err := copyFile(sourcePath, action.Target, srcInfo.Mode()); err != nil {
			return Result{Action: action, Status: "error", Error: err}
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
		return err
	}
	defer func() {
		if closeErr := in.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := out.Close(); err == nil && closeErr != nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(out, in)
	return err
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(src, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)

		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
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
