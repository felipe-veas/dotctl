package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func setupRepo(t *testing.T) (repoRoot, idPath string, id *Identity) {
	t.Helper()
	repoRoot = t.TempDir()
	idDir := t.TempDir()
	idPath = filepath.Join(idDir, "age-identity.txt")

	var err error
	id, err = Init(repoRoot, InitOptions{IdentityPath: idPath})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	return
}

func TestEncryptAndDecrypt(t *testing.T) {
	repoRoot, idPath, _ := setupRepo(t)

	// Create a plaintext file.
	envContent := []byte("API_KEY=sk-test-12345\nDB_PASS=hunter2\n")
	envPath := filepath.Join(repoRoot, "configs", "env")
	if err := os.MkdirAll(envPath, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	plainFile := filepath.Join(envPath, ".env")
	if err := os.WriteFile(plainFile, envContent, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Encrypt.
	encRelPath, err := Encrypt(repoRoot, "configs/env/.env", EncryptOptions{})
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	if encRelPath != "configs/env/.env.enc" {
		t.Errorf("encrypted path = %q, want %q", encRelPath, "configs/env/.env.enc")
	}

	// Original should be deleted.
	if _, err := os.Stat(plainFile); !os.IsNotExist(err) {
		t.Error("original file should be deleted after encrypt")
	}

	// Decrypt.
	plaintext, decRelPath, err := Decrypt(repoRoot, "configs/env/.env.enc", DecryptOptions{
		IdentityPath: idPath,
	})
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if decRelPath != "configs/env/.env" {
		t.Errorf("decrypted path = %q, want %q", decRelPath, "configs/env/.env")
	}
	if string(plaintext) != string(envContent) {
		t.Errorf("decrypted content = %q, want %q", plaintext, envContent)
	}

	// Verify decrypted file permissions.
	info, err := os.Stat(filepath.Join(repoRoot, "configs/env/.env"))
	if err != nil {
		t.Fatalf("stat decrypted file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("decrypted file perms = %o, want 600", perm)
	}
}

func TestEncryptKeep(t *testing.T) {
	repoRoot, _, _ := setupRepo(t)

	plainFile := filepath.Join(repoRoot, ".env")
	if err := os.WriteFile(plainFile, []byte("SECRET=x"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := Encrypt(repoRoot, ".env", EncryptOptions{Keep: true}); err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	// Original should still exist.
	if _, err := os.Stat(plainFile); err != nil {
		t.Error("original file should exist with --keep")
	}
	// Encrypted should exist too.
	if _, err := os.Stat(filepath.Join(repoRoot, ".env.enc")); err != nil {
		t.Error("encrypted file should exist")
	}
}

func TestEncryptAlreadyEncrypted(t *testing.T) {
	repoRoot, _, _ := setupRepo(t)

	encFile := filepath.Join(repoRoot, "config.enc.yaml")
	if err := os.WriteFile(encFile, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := Encrypt(repoRoot, "config.enc.yaml", EncryptOptions{})
	if err == nil {
		t.Error("expected error when encrypting already-encrypted file")
	}
}

func TestDecryptStdout(t *testing.T) {
	repoRoot, idPath, _ := setupRepo(t)

	content := []byte("TOKEN=abc123\n")
	plainFile := filepath.Join(repoRoot, ".env")
	if err := os.WriteFile(plainFile, content, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	if _, err := Encrypt(repoRoot, ".env", EncryptOptions{}); err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	plaintext, path, err := Decrypt(repoRoot, ".env.enc", DecryptOptions{
		IdentityPath: idPath,
		Stdout:       true,
	})
	if err != nil {
		t.Fatalf("Decrypt --stdout: %v", err)
	}

	if path != "" {
		t.Errorf("path should be empty for stdout mode, got %q", path)
	}
	if string(plaintext) != string(content) {
		t.Errorf("plaintext = %q, want %q", plaintext, content)
	}

	// Encrypted file should still exist (stdout mode doesn't delete).
	encFile := filepath.Join(repoRoot, ".env.enc")
	if _, err := os.Stat(encFile); err != nil {
		t.Error("encrypted file should still exist in stdout mode")
	}
}
