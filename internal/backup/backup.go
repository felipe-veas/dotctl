package backup

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/felipe-veas/dotctl/internal/platform"
)

// Create backs up a file or directory to the backup directory.
// Returns the path where the backup was stored.
func Create(targetPath string) (string, error) {
	info, err := os.Lstat(targetPath)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", targetPath, err)
	}

	timestamp := time.Now().Format("20060102-150405.000000")
	backupBase := filepath.Join(platform.BackupDir(), timestamp)

	// Use the base name of the target for the backup
	backupPath := filepath.Join(backupBase, filepath.Base(targetPath))

	if err := os.MkdirAll(backupBase, 0o755); err != nil {
		return "", fmt.Errorf("creating backup dir: %w", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		// Backup symlink: read link target and recreate
		linkTarget, readErr := os.Readlink(targetPath)
		if readErr != nil {
			return "", fmt.Errorf("reading symlink %q: %w", targetPath, readErr)
		}
		if err := os.Symlink(linkTarget, backupPath); err != nil {
			return "", fmt.Errorf("creating backup symlink: %w", err)
		}
		return backupPath, nil
	}

	if info.IsDir() {
		if err := copyDir(targetPath, backupPath); err != nil {
			return "", fmt.Errorf("backing up dir %q: %w", targetPath, err)
		}
		return backupPath, nil
	}

	if err := copyFile(targetPath, backupPath, info.Mode()); err != nil {
		return "", fmt.Errorf("backing up file %q: %w", targetPath, err)
	}

	return backupPath, nil
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
