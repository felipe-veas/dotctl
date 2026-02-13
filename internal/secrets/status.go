package secrets

import "fmt"

// GetStatus scans the repo and returns the secrets health status.
func GetStatus(repoRoot string, identityPath string) (*Status, error) {
	status := &Status{}

	// Check identity.
	if identityPath == "" {
		identityPath = DefaultIdentityPath()
	}
	if IdentityExists(identityPath) {
		id, err := FindIdentity(identityPath)
		if err == nil {
			status.Identity = id
		}
	}

	// Check recipient file.
	if RecipientExists(repoRoot) {
		status.RecipientFile = DefaultRecipientFile
	}

	// Find encrypted files.
	encFiles, err := findEncryptedFiles(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("scanning encrypted files: %w", err)
	}
	for _, f := range encFiles {
		status.EncryptedFiles = append(status.EncryptedFiles, FileStatus{
			Path:      f,
			Encrypted: true,
		})
	}

	// Find unprotected sensitive files.
	sensFiles, err := findSensitiveFiles(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("scanning sensitive files: %w", err)
	}
	for _, f := range sensFiles {
		status.UnprotectedFiles = append(status.UnprotectedFiles, FileStatus{
			Path:      f,
			Encrypted: false,
		})
	}

	return status, nil
}
