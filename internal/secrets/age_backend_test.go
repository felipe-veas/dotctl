package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateIdentity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "age-identity.txt")

	id, err := GenerateIdentity(path)
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	if id.PrivatePath != path {
		t.Errorf("PrivatePath = %q, want %q", id.PrivatePath, path)
	}
	if id.PublicKey == "" {
		t.Error("PublicKey is empty")
	}
	if id.Recipient == nil {
		t.Error("Recipient is nil")
	}
	if id.identity == nil {
		t.Error("identity is nil")
	}

	// Verify file permissions.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat identity file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("identity file perms = %o, want 600", perm)
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "age-identity.txt")

	id, err := GenerateIdentity(path)
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	original := []byte("DATABASE_URL=postgres://user:pass@localhost/db\nAPI_KEY=sk-test-12345\n")

	ciphertext, err := EncryptBytes(original, id.PublicKey)
	if err != nil {
		t.Fatalf("EncryptBytes: %v", err)
	}

	if string(ciphertext) == string(original) {
		t.Error("ciphertext equals plaintext")
	}

	plaintext, err := DecryptBytes(ciphertext, id)
	if err != nil {
		t.Fatalf("DecryptBytes: %v", err)
	}

	if string(plaintext) != string(original) {
		t.Errorf("decrypted = %q, want %q", plaintext, original)
	}
}

func TestEncryptDecryptFileRoundtrip(t *testing.T) {
	dir := t.TempDir()
	idPath := filepath.Join(dir, "age-identity.txt")

	id, err := GenerateIdentity(idPath)
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	original := []byte("SECRET=hunter2\n")
	plainFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(plainFile, original, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Encrypt file.
	ciphertext, err := EncryptFile(plainFile, id.PublicKey)
	if err != nil {
		t.Fatalf("EncryptFile: %v", err)
	}

	encFile := filepath.Join(dir, ".env.enc")
	if err := os.WriteFile(encFile, ciphertext, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Decrypt file.
	plaintext, err := DecryptFileWithIdentity(encFile, id)
	if err != nil {
		t.Fatalf("DecryptFileWithIdentity: %v", err)
	}

	if string(plaintext) != string(original) {
		t.Errorf("decrypted = %q, want %q", plaintext, original)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	dir := t.TempDir()

	// Generate two different identities.
	id1, err := GenerateIdentity(filepath.Join(dir, "id1.txt"))
	if err != nil {
		t.Fatalf("GenerateIdentity 1: %v", err)
	}
	id2, err := GenerateIdentity(filepath.Join(dir, "id2.txt"))
	if err != nil {
		t.Fatalf("GenerateIdentity 2: %v", err)
	}

	// Encrypt with id1's public key.
	ciphertext, err := EncryptBytes([]byte("secret"), id1.PublicKey)
	if err != nil {
		t.Fatalf("EncryptBytes: %v", err)
	}

	// Decrypt with id2 should fail.
	_, err = DecryptBytes(ciphertext, id2)
	if err == nil {
		t.Error("expected error decrypting with wrong key")
	}
}
