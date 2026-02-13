package secrets

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindIdentity(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "age-identity.txt")

	// Generate a real identity first.
	generated, err := GenerateIdentity(path)
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	// Find it.
	found, err := FindIdentity(path)
	if err != nil {
		t.Fatalf("FindIdentity: %v", err)
	}

	if found.PublicKey != generated.PublicKey {
		t.Errorf("PublicKey = %q, want %q", found.PublicKey, generated.PublicKey)
	}
	if found.PrivatePath != path {
		t.Errorf("PrivatePath = %q, want %q", found.PrivatePath, path)
	}
}

func TestFindIdentityMissing(t *testing.T) {
	_, err := FindIdentity(filepath.Join(t.TempDir(), "nonexistent.txt"))
	if err == nil {
		t.Error("expected error for missing identity file")
	}
}

func TestFindIdentityInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad-identity.txt")
	if err := os.WriteFile(path, []byte("not a valid identity"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := FindIdentity(path)
	if err == nil {
		t.Error("expected error for invalid identity file")
	}
}

func TestWriteAndFindRecipient(t *testing.T) {
	dir := t.TempDir()
	idPath := filepath.Join(dir, "age-identity.txt")

	id, err := GenerateIdentity(idPath)
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	// Write recipient file.
	if err := WriteRecipientFile(dir, id.PublicKey); err != nil {
		t.Fatalf("WriteRecipientFile: %v", err)
	}

	// Verify file exists.
	recipientPath := filepath.Join(dir, DefaultRecipientFile)
	if _, err := os.Stat(recipientPath); err != nil {
		t.Fatalf("recipient file not found: %v", err)
	}

	// Read it back.
	pubKey, err := FindRecipient(dir)
	if err != nil {
		t.Fatalf("FindRecipient: %v", err)
	}

	if pubKey != id.PublicKey {
		t.Errorf("FindRecipient = %q, want %q", pubKey, id.PublicKey)
	}
}

func TestFindRecipientMissing(t *testing.T) {
	_, err := FindRecipient(t.TempDir())
	if err == nil {
		t.Error("expected error for missing recipient file")
	}
}

func TestIdentityExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "age-identity.txt")

	if IdentityExists(path) {
		t.Error("IdentityExists should return false for nonexistent file")
	}

	if _, err := GenerateIdentity(path); err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	if !IdentityExists(path) {
		t.Error("IdentityExists should return true after generating")
	}
}

func TestRecipientExists(t *testing.T) {
	dir := t.TempDir()

	if RecipientExists(dir) {
		t.Error("RecipientExists should return false initially")
	}

	if err := WriteRecipientFile(dir, "age1ql3z7hjy54pw3hyww5ayyfg7zqgvc7w3j2elw8zmrj2kg5sfn9aqmcac8p"); err != nil {
		t.Fatalf("WriteRecipientFile: %v", err)
	}

	if !RecipientExists(dir) {
		t.Error("RecipientExists should return true after writing")
	}
}
