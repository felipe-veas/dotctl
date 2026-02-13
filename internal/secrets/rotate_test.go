package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRotate(t *testing.T) {
	repoRoot, idPath, id := setupRepo(t)

	// Create and encrypt two files.
	for _, name := range []string{".env", "api.key"} {
		path := filepath.Join(repoRoot, name)
		if err := os.WriteFile(path, []byte("secret-"+name), 0o644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
		if _, err := Encrypt(repoRoot, name, EncryptOptions{}); err != nil {
			t.Fatalf("Encrypt %s: %v", name, err)
		}
	}

	oldPubKey := id.PublicKey

	// Rotate keys.
	result, err := Rotate(repoRoot, RotateOptions{IdentityPath: idPath})
	if err != nil {
		t.Fatalf("Rotate: %v", err)
	}

	// Verify new key is different.
	if result.NewIdentity.PublicKey == oldPubKey {
		t.Error("new key should be different from old key")
	}

	// Verify backup exists.
	if _, err := os.Stat(result.BackupKeyPath); err != nil {
		t.Errorf("backup key file missing: %v", err)
	}

	// Verify re-encrypted files.
	if len(result.ReEncrypted) != 2 {
		t.Errorf("re-encrypted %d files, want 2", len(result.ReEncrypted))
	}

	// Verify we can decrypt with the new key.
	for _, encFile := range result.ReEncrypted {
		plaintext, _, err := Decrypt(repoRoot, encFile, DecryptOptions{
			IdentityPath: idPath,
			Stdout:       true,
			Keep:         true,
		})
		if err != nil {
			t.Errorf("Decrypt %q with new key: %v", encFile, err)
			continue
		}
		if len(plaintext) == 0 {
			t.Errorf("decrypted %q is empty", encFile)
		}
	}

	// Verify recipient file was updated.
	recipientKey, err := FindRecipient(repoRoot)
	if err != nil {
		t.Fatalf("FindRecipient: %v", err)
	}
	if recipientKey != result.NewIdentity.PublicKey {
		t.Errorf("recipient = %q, want %q", recipientKey, result.NewIdentity.PublicKey)
	}
}
