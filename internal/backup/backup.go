package backup

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/felipe-veas/dotctl/internal/platform"
)

var (
	sessionMu       sync.Mutex
	sessionSnapshot string
)

// BeginSession sets a shared backup snapshot for subsequent Create calls.
// The returned function restores the previous session snapshot.
func BeginSession() func() {
	sessionMu.Lock()
	prev := sessionSnapshot
	sessionSnapshot = newSnapshotName()
	sessionMu.Unlock()

	return func() {
		sessionMu.Lock()
		sessionSnapshot = prev
		sessionMu.Unlock()
	}
}

// Create backs up a file or directory to the backup directory.
// Returns the path where the backup was stored.
func Create(targetPath string) (string, error) {
	info, err := os.Lstat(targetPath)
	if err != nil {
		return "", fmt.Errorf("stat %q: %w", targetPath, err)
	}

	backupBase := filepath.Join(platform.BackupDir(), currentSnapshotName())

	backupPath, err := buildBackupPath(backupBase, targetPath)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
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

func currentSnapshotName() string {
	sessionMu.Lock()
	defer sessionMu.Unlock()
	if sessionSnapshot != "" {
		return sessionSnapshot
	}
	return newSnapshotName()
}

func newSnapshotName() string {
	return time.Now().Format("20060102-150405.000000")
}

func buildBackupPath(backupBase, targetPath string) (string, error) {
	relative := targetRelativePath(targetPath)
	basePath := filepath.Join(backupBase, "targets", relative)
	return uniqueBackupPath(basePath)
}

func uniqueBackupPath(basePath string) (string, error) {
	candidate := basePath
	for i := 1; ; i++ {
		_, err := os.Lstat(candidate)
		if os.IsNotExist(err) {
			return candidate, nil
		}
		if err != nil {
			return "", fmt.Errorf("checking backup path %q: %w", candidate, err)
		}
		candidate = fmt.Sprintf("%s~%d", basePath, i)
	}
}

func targetRelativePath(targetPath string) string {
	cleaned := filepath.Clean(targetPath)
	if vol := filepath.VolumeName(cleaned); vol != "" {
		cleaned = strings.TrimPrefix(cleaned, vol)
	}
	cleaned = strings.TrimPrefix(cleaned, string(filepath.Separator))

	parts := strings.Split(cleaned, string(filepath.Separator))
	safeParts := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			safeParts = append(safeParts, "__parent__")
		default:
			safeParts = append(safeParts, part)
		}
	}

	if len(safeParts) == 0 {
		return "root"
	}
	return filepath.Join(safeParts...)
}

// RotationResult summarizes backup rotation actions.
type RotationResult struct {
	Kept    int
	Removed int
}

// Rotate removes old backup snapshot directories and keeps only the latest keep snapshots.
func Rotate(keep int) (RotationResult, error) {
	if keep <= 0 {
		keep = 1
	}

	base := platform.BackupDir()
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return RotationResult{}, nil
		}
		return RotationResult{}, fmt.Errorf("reading backup dir %q: %w", base, err)
	}

	snapshots := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		snapshots = append(snapshots, entry.Name())
	}

	sort.Sort(sort.Reverse(sort.StringSlice(snapshots)))
	if len(snapshots) <= keep {
		return RotationResult{Kept: len(snapshots), Removed: 0}, nil
	}

	toRemove := snapshots[keep:]
	removed := 0
	for _, snap := range toRemove {
		if err := os.RemoveAll(filepath.Join(base, snap)); err != nil {
			return RotationResult{Kept: keep, Removed: removed}, fmt.Errorf("removing old backup %q: %w", snap, err)
		}
		removed++
	}

	return RotationResult{
		Kept:    keep,
		Removed: removed,
	}, nil
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
