package backup

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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

var restoreSuffixPattern = regexp.MustCompile(`~\d+$`)

// Snapshot describes one backup snapshot on disk.
type Snapshot struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"created_at"`
	Entries   int       `json:"entries"`
}

// Entry describes one restorable item inside a snapshot.
type Entry struct {
	Snapshot   string `json:"snapshot"`
	BackupPath string `json:"backup_path"`
	Target     string `json:"target"`
	Kind       string `json:"kind"`
}

// RestoreResult describes the outcome of restoring one entry.
type RestoreResult struct {
	Snapshot   string `json:"snapshot"`
	BackupPath string `json:"backup_path"`
	Target     string `json:"target"`
	Kind       string `json:"kind"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
}

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

	snapshot := currentSnapshotName()
	backupBase := filepath.Join(platform.BackupDir(), snapshot)

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
		if err := appendSnapshotMetadata(snapshot, snapshotMetadataEntry{
			Target:     metadataTargetPath(targetPath),
			BackupPath: backupPath,
			Kind:       "symlink",
			Mode:       info.Mode().String(),
		}); err != nil {
			return "", fmt.Errorf("recording backup metadata: %w", err)
		}
		return backupPath, nil
	}

	if info.IsDir() {
		if err := copyDir(targetPath, backupPath); err != nil {
			return "", fmt.Errorf("backing up dir %q: %w", targetPath, err)
		}
		if err := appendSnapshotMetadata(snapshot, snapshotMetadataEntry{
			Target:     metadataTargetPath(targetPath),
			BackupPath: backupPath,
			Kind:       "dir",
			Mode:       info.Mode().String(),
		}); err != nil {
			return "", fmt.Errorf("recording backup metadata: %w", err)
		}
		return backupPath, nil
	}

	if err := copyFile(targetPath, backupPath, info.Mode()); err != nil {
		return "", fmt.Errorf("backing up file %q: %w", targetPath, err)
	}
	if err := appendSnapshotMetadata(snapshot, snapshotMetadataEntry{
		Target:     metadataTargetPath(targetPath),
		BackupPath: backupPath,
		Kind:       "file",
		Mode:       info.Mode().String(),
	}); err != nil {
		return "", fmt.Errorf("recording backup metadata: %w", err)
	}

	return backupPath, nil
}

// List returns available snapshots newest first.
func List() ([]Snapshot, error) {
	base := platform.BackupDir()
	entries, err := os.ReadDir(base)
	if err != nil {
		if os.IsNotExist(err) {
			return []Snapshot{}, nil
		}
		return nil, fmt.Errorf("reading backup dir %q: %w", base, err)
	}

	snapshots := make([]Snapshot, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if err := validateSnapshotName(name); err != nil {
			continue
		}

		path := filepath.Join(base, name)
		targetsRoot := filepath.Join(path, "targets")
		if info, statErr := os.Stat(targetsRoot); statErr != nil || !info.IsDir() {
			continue
		}

		inspected, inspectErr := SnapshotEntries(name)
		if inspectErr != nil {
			return nil, inspectErr
		}

		info, err := entry.Info()
		if err != nil {
			return nil, fmt.Errorf("reading snapshot info %q: %w", name, err)
		}

		snapshots = append(snapshots, Snapshot{
			Name:      name,
			Path:      path,
			CreatedAt: info.ModTime(),
			Entries:   len(inspected),
		})
	}

	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].Name == snapshots[j].Name {
			return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
		}
		return snapshots[i].Name > snapshots[j].Name
	})

	return snapshots, nil
}

// SnapshotEntries returns the restoreable entries for a snapshot.
func SnapshotEntries(snapshot string) ([]Entry, error) {
	if err := validateSnapshotName(snapshot); err != nil {
		return nil, err
	}

	snapshotDir := filepath.Join(platform.BackupDir(), snapshot)
	entries, err := snapshotEntriesFromMetadata(snapshot, snapshotDir)
	if err == nil {
		return entries, nil
	}
	if !errors.Is(err, errSnapshotMetadataNotFound) {
		return nil, err
	}

	targetsRoot := filepath.Join(snapshotDir, "targets")
	if _, err := os.Stat(targetsRoot); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("snapshot %q not found", snapshot)
		}
		return nil, fmt.Errorf("stat snapshot targets %q: %w", targetsRoot, err)
	}

	legacyEntries := make([]Entry, 0)
	err = filepath.WalkDir(targetsRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == targetsRoot {
			return nil
		}

		info, err := os.Lstat(path)
		if err != nil {
			return fmt.Errorf("stat backup entry %q: %w", path, err)
		}

		if info.IsDir() {
			children, err := os.ReadDir(path)
			if err != nil {
				return fmt.Errorf("reading backup dir %q: %w", path, err)
			}
			if len(children) == 0 {
				legacyEntries = append(legacyEntries, Entry{
					Snapshot:   snapshot,
					BackupPath: path,
					Target:     targetFromBackupPath(targetsRoot, path),
					Kind:       "dir",
				})
			}
			return nil
		}

		kind := "file"
		if info.Mode()&os.ModeSymlink != 0 {
			kind = "symlink"
		}

		legacyEntries = append(legacyEntries, Entry{
			Snapshot:   snapshot,
			BackupPath: path,
			Target:     targetFromBackupPath(targetsRoot, path),
			Kind:       kind,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return legacyEntries, nil
}

// RestoreSnapshot restores a snapshot into the current user's home directory.
func RestoreSnapshot(snapshot string, dryRun bool) ([]RestoreResult, error) {
	entries, err := SnapshotEntries(snapshot)
	if err != nil {
		return nil, err
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("detecting home directory: %w", err)
	}
	homeAbs, err := filepath.Abs(filepath.Clean(homeDir))
	if err != nil {
		return nil, fmt.Errorf("resolving home directory: %w", err)
	}

	results := make([]RestoreResult, 0, len(entries))
	var errs []error
	for _, entry := range entries {
		result := RestoreResult{
			Snapshot:   entry.Snapshot,
			BackupPath: entry.BackupPath,
			Target:     entry.Target,
			Kind:       entry.Kind,
		}

		if err := validateRestoreTarget(homeAbs, entry.Target); err != nil {
			result.Status = "rejected"
			result.Error = err.Error()
			errs = append(errs, err)
			results = append(results, result)
			continue
		}

		if dryRun {
			result.Status = "planned"
			results = append(results, result)
			continue
		}

		if err := RestorePath(entry.BackupPath, entry.Target); err != nil {
			result.Status = "error"
			result.Error = err.Error()
			errs = append(errs, fmt.Errorf("restoring %s: %w", entry.Target, err))
			results = append(results, result)
			continue
		}

		result.Status = "restored"
		results = append(results, result)
	}

	return results, errors.Join(errs...)
}

// RestorePath replaces target with the contents of backupPath.
func RestorePath(backupPath, target string) error {
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

	if info.IsDir() {
		if err := copyDir(backupPath, target); err != nil {
			return fmt.Errorf("restoring directory: %w", err)
		}
		return nil
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

	if err := copyFile(backupPath, target, info.Mode()); err != nil {
		return fmt.Errorf("restoring file: %w", err)
	}
	return nil
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

func validateSnapshotName(name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("snapshot name is required")
	}
	if trimmed != name {
		return fmt.Errorf("snapshot name must not contain leading or trailing whitespace")
	}
	if filepath.IsAbs(trimmed) {
		return fmt.Errorf("snapshot name must be relative")
	}
	if vol := filepath.VolumeName(trimmed); vol != "" {
		return fmt.Errorf("snapshot name must not include a volume")
	}
	if trimmed == "." || trimmed == ".." {
		return fmt.Errorf("snapshot name is invalid")
	}
	if strings.Contains(trimmed, string(filepath.Separator)) {
		return fmt.Errorf("snapshot name must not contain path separators")
	}
	if filepath.Clean(trimmed) != trimmed {
		return fmt.Errorf("snapshot name must not contain path traversal")
	}
	return nil
}

func targetFromBackupPath(targetsRoot, backupPath string) string {
	rel, err := filepath.Rel(targetsRoot, backupPath)
	if err != nil {
		return string(filepath.Separator)
	}
	parts := strings.Split(rel, string(filepath.Separator))
	if len(parts) > 0 {
		last := len(parts) - 1
		parts[last] = restoreSuffixPattern.ReplaceAllString(parts[last], "")
	}
	cleaned := filepath.Join(parts...)
	if cleaned == "." || cleaned == "" {
		return string(filepath.Separator)
	}
	return filepath.Join(string(filepath.Separator), cleaned)
}

func validateRestoreTarget(homeAbs, target string) error {
	targetAbs, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return fmt.Errorf("resolving restore target: %w", err)
	}

	rel, err := filepath.Rel(homeAbs, targetAbs)
	if err != nil {
		return fmt.Errorf("checking restore target containment: %w", err)
	}
	if rel == "." {
		return fmt.Errorf("restore target must not be the home directory itself")
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return fmt.Errorf("restore target %s must stay under home directory %s", targetAbs, homeAbs)
	}
	return nil
}
