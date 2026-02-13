package secrets

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	repoRoot := t.TempDir()
	idDir := t.TempDir()
	idPath := filepath.Join(idDir, "age-identity.txt")

	id, err := Init(repoRoot, InitOptions{IdentityPath: idPath})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	if id.PublicKey == "" {
		t.Error("PublicKey is empty")
	}

	// Verify identity file exists with correct permissions.
	info, err := os.Stat(idPath)
	if err != nil {
		t.Fatalf("identity file missing: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("identity perms = %o, want 600", perm)
	}

	// Verify recipient file exists.
	recipientPath := filepath.Join(repoRoot, DefaultRecipientFile)
	data, err := os.ReadFile(recipientPath)
	if err != nil {
		t.Fatalf("recipient file missing: %v", err)
	}
	if !strings.Contains(string(data), id.PublicKey) {
		t.Error("recipient file does not contain public key")
	}

	// Verify .gitignore updated.
	gitignoreData, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("gitignore missing: %v", err)
	}
	if !strings.Contains(string(gitignoreData), DefaultIdentityFile) {
		t.Error(".gitignore does not contain identity file pattern")
	}
}

func TestInitAlreadyExists(t *testing.T) {
	repoRoot := t.TempDir()
	idDir := t.TempDir()
	idPath := filepath.Join(idDir, "age-identity.txt")

	if _, err := Init(repoRoot, InitOptions{IdentityPath: idPath}); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Second init should fail.
	_, err := Init(repoRoot, InitOptions{IdentityPath: idPath})
	if err == nil {
		t.Error("expected error on duplicate init")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInitForce(t *testing.T) {
	repoRoot := t.TempDir()
	idDir := t.TempDir()
	idPath := filepath.Join(idDir, "age-identity.txt")

	id1, err := Init(repoRoot, InitOptions{IdentityPath: idPath})
	if err != nil {
		t.Fatalf("Init: %v", err)
	}

	// Force overwrite.
	id2, err := Init(repoRoot, InitOptions{IdentityPath: idPath, Force: true})
	if err != nil {
		t.Fatalf("Init --force: %v", err)
	}

	if id1.PublicKey == id2.PublicKey {
		t.Error("force init should generate a new key")
	}
}

func TestInitImport(t *testing.T) {
	repoRoot := t.TempDir()
	srcDir := t.TempDir()
	dstDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "age-identity.txt")
	dstPath := filepath.Join(dstDir, "age-identity.txt")

	// Generate source identity.
	srcID, err := GenerateIdentity(srcPath)
	if err != nil {
		t.Fatalf("GenerateIdentity: %v", err)
	}

	// Import it.
	importedID, err := Init(repoRoot, InitOptions{
		IdentityPath: dstPath,
		ImportPath:   srcPath,
	})
	if err != nil {
		t.Fatalf("Init --import: %v", err)
	}

	if importedID.PublicKey != srcID.PublicKey {
		t.Errorf("imported PublicKey = %q, want %q", importedID.PublicKey, srcID.PublicKey)
	}
	if importedID.PrivatePath != dstPath {
		t.Errorf("imported PrivatePath = %q, want %q", importedID.PrivatePath, dstPath)
	}
}

func TestEnsureGitignoreIdempotent(t *testing.T) {
	repoRoot := t.TempDir()

	// First call adds it.
	if err := ensureGitignore(repoRoot, "test-pattern"); err != nil {
		t.Fatalf("ensureGitignore: %v", err)
	}

	// Second call should not duplicate.
	if err := ensureGitignore(repoRoot, "test-pattern"); err != nil {
		t.Fatalf("ensureGitignore: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	count := strings.Count(string(data), "test-pattern")
	if count != 1 {
		t.Errorf("pattern appears %d times, want 1", count)
	}
}
