package secrets

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Rotate generates a new key and re-encrypts all protected files.
func Rotate(repoRoot string, opts RotateOptions) (*RotateResult, error) {
	idPath := opts.IdentityPath
	if idPath == "" {
		idPath = DefaultIdentityPath()
	}

	// Load old identity.
	oldID, err := FindIdentity(idPath)
	if err != nil {
		return nil, fmt.Errorf("loading old identity: %w", err)
	}

	// Find all encrypted files in repo.
	encFiles, err := findEncryptedFiles(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("scanning encrypted files: %w", err)
	}

	// Decrypt all files with old key (in memory).
	type decryptedFile struct {
		path      string
		plaintext []byte
	}
	var decrypted []decryptedFile
	for _, f := range encFiles {
		absPath := filepath.Join(repoRoot, f)
		plain, err := DecryptFileWithIdentity(absPath, oldID)
		if err != nil {
			return nil, fmt.Errorf("decrypting %q with old key: %w", f, err)
		}
		decrypted = append(decrypted, decryptedFile{path: absPath, plaintext: plain})
	}

	// Backup old key.
	backupPath := idPath + ".bak-" + nowFunc().Format("20060102")
	if err := copyFileBytes(idPath, backupPath); err != nil {
		return nil, fmt.Errorf("backing up old key: %w", err)
	}

	// Generate new identity.
	newID, err := GenerateIdentity(idPath)
	if err != nil {
		return nil, fmt.Errorf("generating new identity: %w", err)
	}

	// Re-encrypt all files with new key.
	var reEncrypted []string
	for _, df := range decrypted {
		ciphertext, err := EncryptBytes(df.plaintext, newID.PublicKey)
		if err != nil {
			return nil, fmt.Errorf("re-encrypting %q: %w", df.path, err)
		}
		if err := os.WriteFile(df.path, ciphertext, 0o644); err != nil {
			return nil, fmt.Errorf("writing re-encrypted %q: %w", df.path, err)
		}
		rel, _ := filepath.Rel(repoRoot, df.path)
		reEncrypted = append(reEncrypted, rel)
	}

	// Update recipient file.
	if err := WriteRecipientFile(repoRoot, newID.PublicKey); err != nil {
		return nil, fmt.Errorf("updating recipient file: %w", err)
	}

	return &RotateResult{
		NewIdentity:   newID,
		BackupKeyPath: backupPath,
		ReEncrypted:   reEncrypted,
	}, nil
}

// findEncryptedFiles walks the repo and returns relative paths of encrypted files.
func findEncryptedFiles(repoRoot string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip .git directory.
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if d.IsDir() {
			return nil
		}
		if IsEncryptedName(d.Name()) {
			rel, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				return relErr
			}
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}

// findSensitiveFiles walks the repo and returns relative paths of unprotected sensitive files.
func findSensitiveFiles(repoRoot string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(repoRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		if IsSensitiveName(d.Name()) {
			rel, relErr := filepath.Rel(repoRoot, path)
			if relErr != nil {
				return relErr
			}
			files = append(files, rel)
		}
		return nil
	})
	return files, err
}

// copyFileBytes copies a file from src to dst preserving content.
func copyFileBytes(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o600)
}

// FindEncryptedFilesInManifest returns encrypted file sources from manifest entries
// that have decrypt: true. This is a helper for --manifest flag operations.
func FindEncryptedFilesInManifest(repoRoot string, manifestSources []string) []string {
	var result []string
	for _, src := range manifestSources {
		if IsEncryptedName(src) {
			absPath := filepath.Join(repoRoot, src)
			if _, err := os.Stat(absPath); err == nil {
				result = append(result, src)
			}
		}
	}
	return result
}

// FindDecryptableSourcesInManifest returns sources with decrypt:true that are not yet encrypted.
func FindDecryptableSourcesInManifest(repoRoot string, manifestSources []string) []string {
	var result []string
	for _, src := range manifestSources {
		if !IsEncryptedName(src) {
			// Check if the encrypted version exists.
			encName := EncryptedName(src)
			absPath := filepath.Join(repoRoot, encName)
			if _, err := os.Stat(absPath); err == nil {
				continue // already encrypted
			}
			// Check if the plaintext exists.
			absPath = filepath.Join(repoRoot, src)
			if _, err := os.Stat(absPath); err == nil {
				result = append(result, src)
			}
		}
	}
	return result
}

// IsGitTracked checks if a file is tracked by git (not in .gitignore).
// This is a simple heuristic: check if the file matches ignore patterns.
func IsGitTracked(repoRoot, relPath string) bool {
	// Check .gitignore for the file pattern.
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		return true // if no .gitignore, assume tracked
	}

	base := filepath.Base(relPath)
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if matched, _ := filepath.Match(line, base); matched {
			return false
		}
		if matched, _ := filepath.Match(line, relPath); matched {
			return false
		}
	}
	return true
}
