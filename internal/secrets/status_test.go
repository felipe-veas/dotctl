package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetStatus(t *testing.T) {
	repoRoot, idPath, _ := setupRepo(t)

	// Create an encrypted file.
	plainFile := filepath.Join(repoRoot, ".env")
	if err := os.WriteFile(plainFile, []byte("SECRET=x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := Encrypt(repoRoot, ".env", EncryptOptions{}); err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Create an unprotected sensitive file.
	sensFile := filepath.Join(repoRoot, "api.key")
	if err := os.WriteFile(sensFile, []byte("key-data"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	status, err := GetStatus(repoRoot, idPath)
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	// Check identity.
	if status.Identity == nil {
		t.Error("Identity should be present")
	}

	// Check recipient.
	if status.RecipientFile == "" {
		t.Error("RecipientFile should be present")
	}

	// Check encrypted files.
	if len(status.EncryptedFiles) != 1 {
		t.Errorf("EncryptedFiles = %d, want 1", len(status.EncryptedFiles))
	}

	// Check unprotected files.
	if len(status.UnprotectedFiles) != 1 {
		t.Errorf("UnprotectedFiles = %d, want 1", len(status.UnprotectedFiles))
	}
	if len(status.UnprotectedFiles) > 0 && status.UnprotectedFiles[0].Path != "api.key" {
		t.Errorf("UnprotectedFiles[0].Path = %q, want %q", status.UnprotectedFiles[0].Path, "api.key")
	}
}

func TestGetStatusNoSecrets(t *testing.T) {
	repoRoot := t.TempDir()

	status, err := GetStatus(repoRoot, "")
	if err != nil {
		t.Fatalf("GetStatus: %v", err)
	}

	if status.Identity != nil {
		t.Error("Identity should be nil when not configured")
	}
	if status.RecipientFile != "" {
		t.Error("RecipientFile should be empty when not configured")
	}
	if len(status.EncryptedFiles) != 0 {
		t.Errorf("EncryptedFiles = %d, want 0", len(status.EncryptedFiles))
	}
}
