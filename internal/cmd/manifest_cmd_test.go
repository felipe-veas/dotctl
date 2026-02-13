package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felipe-veas/dotctl/internal/manifest"
)

func TestPromptYesNo(t *testing.T) {
	var out bytes.Buffer
	yes, err := promptYesNo(strings.NewReader("y\n"), &out, "continue? ")
	if err != nil {
		t.Fatalf("promptYesNo: %v", err)
	}
	if !yes {
		t.Fatal("expected yes=true")
	}
	if !strings.Contains(out.String(), "continue?") {
		t.Fatalf("prompt output = %q, expected question", out.String())
	}
}

func TestPromptYesNoDefaultNo(t *testing.T) {
	no, err := promptYesNo(strings.NewReader("\n"), &bytes.Buffer{}, "continue? ")
	if err != nil {
		t.Fatalf("promptYesNo: %v", err)
	}
	if no {
		t.Fatal("expected yes=false")
	}
}

func TestDiscoverManifestCandidates(t *testing.T) {
	home := t.TempDir()
	configHome := filepath.Join(home, ".config")

	if err := os.WriteFile(filepath.Join(home, ".zshrc"), []byte("# zsh\n"), 0o644); err != nil {
		t.Fatalf("write .zshrc: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(configHome, "nvim"), 0o755); err != nil {
		t.Fatalf("mkdir nvim: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configHome, "starship.toml"), []byte("format = \"$all\"\n"), 0o644); err != nil {
		t.Fatalf("write starship.toml: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".env"), []byte("TOKEN=x\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	candidates, skippedSensitive, err := discoverManifestCandidates(home, configHome, []string{
		".zshrc",
		".config/nvim",
		".config/starship.toml",
		".env",
		".missing",
	})
	if err != nil {
		t.Fatalf("discoverManifestCandidates: %v", err)
	}

	if skippedSensitive != 1 {
		t.Fatalf("skippedSensitive = %d, want 1", skippedSensitive)
	}
	if len(candidates) != 3 {
		t.Fatalf("candidate count = %d, want 3", len(candidates))
	}

	wantTargets := map[string]bool{
		"~/.zshrc":                true,
		"~/.config/nvim":          true,
		"~/.config/starship.toml": true,
	}
	for _, candidate := range candidates {
		if !wantTargets[candidate.Target] {
			t.Fatalf("unexpected target %q", candidate.Target)
		}
	}
}

func TestBuildSuggestedManifestFiles(t *testing.T) {
	files := buildSuggestedManifestFiles([]manifestScanCandidate{
		{Relative: ".zshrc", Target: "~/.zshrc"},
		{Relative: ".gitconfig", Target: "~/.gitconfig"},
		{Relative: ".config/nvim", Target: "~/.config/nvim"},
	})

	if len(files) != 3 {
		t.Fatalf("file count = %d, want 3", len(files))
	}

	got := map[string]string{}
	for _, f := range files {
		got[f.Target] = f.Source
		if f.Mode != "symlink" {
			t.Fatalf("mode = %q, want symlink", f.Mode)
		}
	}

	if got["~/.zshrc"] != "configs/zsh/.zshrc" {
		t.Fatalf("source for ~/.zshrc = %q, want configs/zsh/.zshrc", got["~/.zshrc"])
	}
	if got["~/.gitconfig"] != "configs/git/.gitconfig" {
		t.Fatalf("source for ~/.gitconfig = %q, want configs/git/.gitconfig", got["~/.gitconfig"])
	}
	if got["~/.config/nvim"] != "configs/nvim" {
		t.Fatalf("source for ~/.config/nvim = %q, want configs/nvim", got["~/.config/nvim"])
	}
}

func TestRenderSuggestedManifest(t *testing.T) {
	data, err := renderSuggestedManifest([]suggestedManifestFile{
		{Source: "configs/zsh/.zshrc", Target: "~/.zshrc", Mode: "symlink"},
		{Source: "configs/nvim", Target: "~/.config/nvim", Mode: "symlink"},
	})
	if err != nil {
		t.Fatalf("renderSuggestedManifest: %v", err)
	}

	if !strings.Contains(string(data), "version: 1") {
		t.Fatalf("manifest content missing version: %s", string(data))
	}

	if _, err := manifest.Parse(data); err != nil {
		t.Fatalf("rendered manifest should parse: %v\n%s", err, string(data))
	}
}

func TestCandidateTargetCustomConfigHome(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	configHome := filepath.Join(t.TempDir(), "cfg")

	target := candidateTarget(".config/nvim", home, configHome)
	want := filepath.ToSlash(filepath.Join(configHome, "nvim"))
	if target != want {
		t.Fatalf("target = %q, want %q", target, want)
	}
}

func TestResolveSuggestionPath(t *testing.T) {
	repoPath := filepath.Join(t.TempDir(), "repo")
	rel := resolveSuggestionPath(repoPath, "manifest.scan.yaml")
	if rel != filepath.Join(repoPath, "manifest.scan.yaml") {
		t.Fatalf("relative output path = %q", rel)
	}

	abs := resolveSuggestionPath(repoPath, "/tmp/manifest.scan.yaml")
	if abs != "/tmp/manifest.scan.yaml" {
		t.Fatalf("absolute output path = %q", abs)
	}
}
