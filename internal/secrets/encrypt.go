package secrets

import (
	"fmt"
	"os"
	"path/filepath"
)

// Encrypt encrypts a file using the repo's recipient key.
// Returns the path of the encrypted file (relative to repoRoot).
func Encrypt(repoRoot, filePath string, opts EncryptOptions) (string, error) {
	recipientKey := opts.RecipientKey
	if recipientKey == "" {
		var err error
		recipientKey, err = FindRecipient(repoRoot)
		if err != nil {
			return "", fmt.Errorf("finding recipient: %w", err)
		}
	}

	// Resolve file path relative to repo root.
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(repoRoot, filePath)
	}

	if IsEncryptedName(absPath) {
		return "", fmt.Errorf("file %q is already encrypted", filePath)
	}

	// Encrypt the file contents.
	ciphertext, err := EncryptFile(absPath, recipientKey)
	if err != nil {
		return "", err
	}

	// Determine output path.
	encName := EncryptedName(absPath)
	if err := os.WriteFile(encName, ciphertext, 0o644); err != nil {
		return "", fmt.Errorf("writing encrypted file: %w", err)
	}

	// Remove original unless --keep.
	if !opts.Keep {
		if err := os.Remove(absPath); err != nil {
			return "", fmt.Errorf("removing original file: %w", err)
		}
	}

	// Return path relative to repo root.
	rel, err := filepath.Rel(repoRoot, encName)
	if err != nil {
		return encName, nil
	}
	return rel, nil
}

// Decrypt decrypts a file using the local identity.
// If opts.Stdout is true, returns the plaintext bytes without writing to disk.
// Otherwise, writes the decrypted file and returns its path.
func Decrypt(repoRoot, filePath string, opts DecryptOptions) ([]byte, string, error) {
	idPath := opts.IdentityPath
	if idPath == "" {
		idPath = DefaultIdentityPath()
	}

	id, err := FindIdentity(idPath)
	if err != nil {
		return nil, "", fmt.Errorf("loading identity: %w", err)
	}

	// Resolve file path.
	absPath := filePath
	if !filepath.IsAbs(filePath) {
		absPath = filepath.Join(repoRoot, filePath)
	}

	if !IsEncryptedName(absPath) {
		return nil, "", fmt.Errorf("file %q does not appear to be encrypted", filePath)
	}

	plaintext, err := DecryptFileWithIdentity(absPath, id)
	if err != nil {
		return nil, "", err
	}

	if opts.Stdout {
		return plaintext, "", nil
	}

	// Write decrypted file.
	decName := DecryptedName(absPath)
	if err := os.WriteFile(decName, plaintext, 0o600); err != nil {
		return nil, "", fmt.Errorf("writing decrypted file: %w", err)
	}

	// Remove encrypted file unless --keep.
	if !opts.Keep {
		if err := os.Remove(absPath); err != nil {
			return nil, "", fmt.Errorf("removing encrypted file: %w", err)
		}
	}

	rel, err := filepath.Rel(repoRoot, decName)
	if err != nil {
		return plaintext, decName, nil
	}
	return plaintext, rel, nil
}
