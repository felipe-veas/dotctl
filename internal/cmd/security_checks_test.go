package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsSensitiveTrackedPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{path: ".env", want: true},
		{path: "configs/.env.local", want: true},
		{path: "keys/private.pem", want: true},
		{path: "keys/service.key", want: true},
		{path: ".ssh/id_rsa", want: true},
		{path: "files/.ssh/id_ed25519", want: true},
		{path: "configs/zsh/.zshrc", want: false},
		{path: "README.md", want: false},
	}

	for _, tc := range tests {
		if got := isSensitiveTrackedPath(tc.path); got != tc.want {
			t.Errorf("isSensitiveTrackedPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestSensitiveTrackedFilesWarning(t *testing.T) {
	msg := sensitiveTrackedFilesWarning([]string{".env", "secret.key"})
	if msg == "" {
		t.Fatal("expected non-empty warning")
	}
}

func TestMissingGitignorePatterns(t *testing.T) {
	repo := t.TempDir()
	content := "# comments\n.env\n*.pem\n"
	if err := os.WriteFile(filepath.Join(repo, ".gitignore"), []byte(content), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	missing, err := missingGitignorePatterns(repo, []string{".env", "*.key", "*.pem"})
	if err != nil {
		t.Fatalf("missingGitignorePatterns: %v", err)
	}
	if len(missing) != 1 || missing[0] != "*.key" {
		t.Fatalf("missing = %v, want [*.key]", missing)
	}
}
