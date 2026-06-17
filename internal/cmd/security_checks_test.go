package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felipe-veas/dotctl/internal/manifest"
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
		{path: "configs/.kube/config", want: true},
		{path: "configs/.aws/credentials", want: true},
		{path: "configs/.gnupg/private-keys-v1.d/key", want: true},
		{path: "configs/.config/gh/hosts.yml", want: true},
		{path: "configs/.config/gcloud/application_default_credentials.json", want: true},
		{path: "configs/zsh/.zshrc", want: false},
		{path: "configs/nvim/init.lua", want: false},
		{path: "README.md", want: false},
		// Encrypted files should not be flagged as sensitive.
		{path: "configs/.aws/credentials.enc", want: false},
		{path: "configs/.kube/config.enc.yaml", want: false},
		{path: "configs/app/config.enc.yaml", want: false},
		{path: ".env.enc", want: false},
		{path: "api.enc.key", want: false},
		{path: "secret.enc", want: false},
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

func TestSensitiveManifestTargetWarnings(t *testing.T) {
	home := t.TempDir()

	tests := []struct {
		name    string
		actions []manifest.Action
		want    bool
	}{
		{name: "ssh config", actions: []manifest.Action{{Source: "configs/ssh/config", Target: filepath.Join(home, ".ssh", "config")}}, want: true},
		{name: "gnupg dir", actions: []manifest.Action{{Source: "configs/gnupg/pubring.kbx", Target: filepath.Join(home, ".gnupg", "pubring.kbx")}}, want: true},
		{name: "kube dir", actions: []manifest.Action{{Source: "configs/kube/config", Target: filepath.Join(home, ".kube", "config")}}, want: true},
		{name: "aws dir", actions: []manifest.Action{{Source: "configs/aws/credentials", Target: filepath.Join(home, ".aws", "credentials")}}, want: true},
		{name: "gh config", actions: []manifest.Action{{Source: "configs/gh/config.yml", Target: filepath.Join(home, ".config", "gh", "config.yml")}}, want: true},
		{name: "gcloud config", actions: []manifest.Action{{Source: "configs/gcloud/config", Target: filepath.Join(home, ".config", "gcloud", "configurations", "config_default")}}, want: true},
		{name: "env file", actions: []manifest.Action{{Source: "configs/app/env", Target: filepath.Join(home, "services", "api", ".env")}}, want: true},
		{name: "normal zshrc", actions: []manifest.Action{{Source: "configs/zshrc", Target: filepath.Join(home, ".zshrc")}}, want: false},
		{name: "normal nvim config", actions: []manifest.Action{{Source: "configs/nvim/init.lua", Target: filepath.Join(home, ".config", "nvim", "init.lua")}}, want: false},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			warnings := sensitiveManifestTargetWarnings(home, tt.actions)
			if tt.want && len(warnings) == 0 {
				t.Fatal("expected warning, got none")
			}
			if !tt.want && len(warnings) != 0 {
				t.Fatalf("expected no warning, got %v", warnings)
			}
		})
	}
}

func TestSensitiveManifestTargetWarningsDeduplicatesAndSorts(t *testing.T) {
	home := t.TempDir()

	actions := []manifest.Action{
		{Source: "z-last", Target: filepath.Join(home, ".aws", "credentials")},
		{Source: "a-first", Target: filepath.Join(home, ".ssh", "config")},
		{Source: "a-first", Target: filepath.Join(home, ".ssh", "config")},
		{Source: "m-middle", Target: filepath.Join(home, ".config", "gcloud", "configurations", "config_default")},
	}

	warnings := sensitiveManifestTargetWarnings(home, actions)
	if len(warnings) != 1 {
		t.Fatalf("warnings = %v, want 1 summary", warnings)
	}

	want := "sensitive manifest targets: a-first -> " + filepath.Join(home, ".ssh", "config") + ", m-middle -> " + filepath.Join(home, ".config", "gcloud", "configurations", "config_default") + ", z-last -> " + filepath.Join(home, ".aws", "credentials")
	if warnings[0] != want {
		t.Fatalf("warning = %q, want %q", warnings[0], want)
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
