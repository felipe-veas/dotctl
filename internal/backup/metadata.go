package backup

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/felipe-veas/dotctl/internal/platform"
)

const snapshotMetadataVersion = 1

var errSnapshotMetadataNotFound = errors.New("snapshot metadata not found")

var metadataMu sync.Mutex

type snapshotMetadata struct {
	Version   int                     `json:"version"`
	CreatedAt time.Time               `json:"created_at"`
	Entries   []snapshotMetadataEntry `json:"entries"`
}

type snapshotMetadataEntry struct {
	Target     string `json:"target"`
	BackupPath string `json:"backup_path"`
	Kind       string `json:"kind"`
	Mode       string `json:"mode"`
}

func appendSnapshotMetadata(snapshot string, entry snapshotMetadataEntry) error {
	metadataMu.Lock()
	defer metadataMu.Unlock()

	metadata, err := loadSnapshotMetadata(snapshot)
	if err != nil {
		if !errors.Is(err, errSnapshotMetadataNotFound) {
			return err
		}
		metadata = &snapshotMetadata{Version: snapshotMetadataVersion, CreatedAt: time.Now().UTC()}
	}

	if metadata.Version == 0 {
		metadata.Version = snapshotMetadataVersion
	}
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = time.Now().UTC()
	}

	metadata.Entries = append(metadata.Entries, entry)
	return writeSnapshotMetadata(snapshot, metadata)
}

func loadSnapshotMetadata(snapshot string) (*snapshotMetadata, error) {
	path := metadataFilePath(snapshot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errSnapshotMetadataNotFound
		}
		return nil, fmt.Errorf("reading snapshot metadata %q: %w", path, err)
	}

	var metadata snapshotMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("decoding snapshot metadata %q: %w", path, err)
	}
	if metadata.Version != snapshotMetadataVersion {
		return nil, fmt.Errorf("unsupported snapshot metadata version %d in %q", metadata.Version, path)
	}
	return &metadata, nil
}

func writeSnapshotMetadata(snapshot string, metadata *snapshotMetadata) error {
	snapshotDir := snapshotDirPath(snapshot)
	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		return fmt.Errorf("creating snapshot metadata dir: %w", err)
	}

	path := metadataFilePath(snapshot)
	tmp, err := os.CreateTemp(snapshotDir, "metadata-*.json")
	if err != nil {
		return fmt.Errorf("creating temp metadata file: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding snapshot metadata: %w", err)
	}
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing temp metadata file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp metadata file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("committing snapshot metadata: %w", err)
	}
	return nil
}

func snapshotEntriesFromMetadata(snapshot, snapshotDir string) ([]Entry, error) {
	metadata, err := loadSnapshotMetadata(snapshot)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, 0, len(metadata.Entries))
	for i, metaEntry := range metadata.Entries {
		backupPath, err := resolveMetadataBackupPath(snapshotDir, metaEntry.BackupPath)
		if err != nil {
			return nil, fmt.Errorf("validating metadata entry %d: %w", i, err)
		}

		info, err := os.Lstat(backupPath)
		if err != nil {
			return nil, fmt.Errorf("stat metadata backup path %q: %w", backupPath, err)
		}

		kind, err := entryKindFromInfo(info)
		if err != nil {
			return nil, fmt.Errorf("classifying metadata backup path %q: %w", backupPath, err)
		}
		if metaEntry.Kind != kind {
			return nil, fmt.Errorf("metadata entry kind mismatch for %q: got %q want %q", backupPath, metaEntry.Kind, kind)
		}
		if metaEntry.Target == "" {
			return nil, fmt.Errorf("metadata entry target is required for %q", backupPath)
		}

		entries = append(entries, Entry{
			Snapshot:   snapshot,
			BackupPath: backupPath,
			Target:     metaEntry.Target,
			Kind:       kind,
		})
	}

	return entries, nil
}

func metadataFilePath(snapshot string) string {
	return filepath.Join(snapshotDirPath(snapshot), "metadata.json")
}

func snapshotDirPath(snapshot string) string {
	return filepath.Join(platform.BackupDir(), snapshot)
}

func resolveMetadataBackupPath(snapshotDir, raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("backup path is required")
	}

	candidate := raw
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(snapshotDir, candidate)
	}

	absCandidate, err := filepath.Abs(filepath.Clean(candidate))
	if err != nil {
		return "", fmt.Errorf("resolving backup path %q: %w", raw, err)
	}
	snapshotAbs, err := filepath.Abs(filepath.Clean(snapshotDir))
	if err != nil {
		return "", fmt.Errorf("resolving snapshot dir: %w", err)
	}

	rel, err := filepath.Rel(snapshotAbs, absCandidate)
	if err != nil {
		return "", fmt.Errorf("checking backup path containment: %w", err)
	}
	if rel == "." {
		return "", fmt.Errorf("backup path %q must not point at the snapshot dir", absCandidate)
	}
	if rel == ".." || filepath.IsAbs(rel) || rel == "" || rel == ".."+string(filepath.Separator) || (len(rel) > 3 && rel[:3] == ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("backup path %q must stay within snapshot dir %q", absCandidate, snapshotAbs)
	}

	return absCandidate, nil
}

func entryKindFromInfo(info os.FileInfo) (string, error) {
	if info.IsDir() {
		return "dir", nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "symlink", nil
	}
	if info.Mode().IsRegular() {
		return "file", nil
	}
	return "", fmt.Errorf("unsupported backup entry type %q", info.Mode().String())
}

func metadataTargetPath(targetPath string) string {
	cleaned := filepath.Clean(targetPath)
	if abs, err := filepath.Abs(cleaned); err == nil {
		return abs
	}
	return cleaned
}
