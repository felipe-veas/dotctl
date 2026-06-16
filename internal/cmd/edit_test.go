package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/felipe-veas/dotctl/internal/config"
	"github.com/spf13/cobra"
)

func TestSelectEditorPrefersVisual(t *testing.T) {
	t.Setenv("VISUAL", "nvim")
	t.Setenv("EDITOR", "vim")

	editor, err := selectEditor()
	if err != nil {
		t.Fatalf("selectEditor: %v", err)
	}
	if editor != "nvim" {
		t.Fatalf("editor = %q, want nvim", editor)
	}
}

func TestSelectEditorMissing(t *testing.T) {
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	_, err := selectEditor()
	if err == nil {
		t.Fatal("expected error when no editor is configured")
	}
	if err.Error() != "no editor configured; set VISUAL or EDITOR" {
		t.Fatalf("error = %v", err)
	}
}

func TestResolveEditTarget(t *testing.T) {
	repo := t.TempDir()
	manifestPath := filepath.Join(repo, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte("version: 1\nfiles: []\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	cfg := &config.Config{Repo: config.RepoConfig{Path: repo}}

	tests := []struct {
		name    string
		args    []string
		want    editTarget
		wantErr string
	}{
		{name: "default repo", args: nil, want: editTarget{kind: "repo", target: repo}},
		{name: "explicit repo", args: []string{"repo"}, want: editTarget{kind: "repo", target: repo}},
		{name: "manifest", args: []string{"manifest"}, want: editTarget{kind: "manifest", target: manifestPath}},
		{name: "invalid", args: []string{"bogus"}, wantErr: `invalid edit target: "bogus"`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := resolveEditTarget(cfg, tc.args)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if err.Error() != tc.wantErr {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolveEditTarget: %v", err)
			}
			if got != tc.want {
				t.Fatalf("target = %#v, want %#v", got, tc.want)
			}
		})
	}
}

func TestResolveEditTargetMissingManifest(t *testing.T) {
	repo := t.TempDir()
	cfg := &config.Config{Repo: config.RepoConfig{Path: repo}}

	_, err := resolveEditTarget(cfg, []string{"manifest"})
	if err == nil {
		t.Fatal("expected error for missing manifest")
	}
	want := filepath.Join(repo, "manifest.yaml")
	if err.Error() != "manifest not found: "+want {
		t.Fatalf("error = %v, want manifest not found", err)
	}
}

func TestResolveEditTargetRejectsNonDirectoryRepo(t *testing.T) {
	repoFile := filepath.Join(t.TempDir(), "repo.txt")
	if err := os.WriteFile(repoFile, []byte("not a dir\n"), 0o644); err != nil {
		t.Fatalf("write repo file: %v", err)
	}

	cfg := &config.Config{Repo: config.RepoConfig{Path: repoFile}}
	_, err := resolveEditTarget(cfg, nil)
	if err == nil {
		t.Fatal("expected error for non-directory repo path")
	}
	if err.Error() != "repo path is not a directory: "+repoFile {
		t.Fatalf("error = %v", err)
	}
}

func TestRunEditDryRunDoesNotInvokeRunner(t *testing.T) {
	defer restoreEditFlags()()
	flagJSON = true
	flagDryRun = true

	repo := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := config.Save(configPath, &config.Config{Repo: config.RepoConfig{URL: "git@example.com:dotfiles.git", Path: repo}}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	flagConfig = configPath
	t.Setenv("VISUAL", "nvim --server /tmp/nvim")

	called := false
	oldRunner := runEditorCommand
	runEditorCommand = func(editor string, args ...string) error {
		called = true
		return nil
	}
	defer func() { runEditorCommand = oldRunner }()

	if err := runEdit(&cobra.Command{}, nil); err != nil {
		t.Fatalf("runEdit dry-run: %v", err)
	}
	if called {
		t.Fatal("expected editor runner not to be called in dry-run")
	}
}

func TestRunEditInvokesRunnerWithParsedEditorArgs(t *testing.T) {
	defer restoreEditFlags()()
	flagJSON = false
	flagDryRun = false

	repo := t.TempDir()
	manifestPath := filepath.Join(repo, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte("version: 1\nfiles: []\n"), 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := config.Save(configPath, &config.Config{Repo: config.RepoConfig{URL: "git@example.com:dotfiles.git", Path: repo}}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	flagConfig = configPath
	t.Setenv("VISUAL", "code --wait")

	var gotEditor string
	var gotArgs []string
	oldRunner := runEditorCommand
	runEditorCommand = func(editor string, args ...string) error {
		gotEditor = editor
		gotArgs = append([]string{}, args...)
		return nil
	}
	defer func() { runEditorCommand = oldRunner }()

	if err := runEdit(&cobra.Command{}, []string{"manifest"}); err != nil {
		t.Fatalf("runEdit: %v", err)
	}

	if gotEditor != "code" {
		t.Fatalf("editor = %q, want code", gotEditor)
	}
	wantArgs := []string{"--wait", manifestPath}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
}

func TestRunEditRejectsMissingEditor(t *testing.T) {
	defer restoreEditFlags()()
	flagJSON = true
	flagDryRun = false

	repo := t.TempDir()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := config.Save(configPath, &config.Config{Repo: config.RepoConfig{URL: "git@example.com:dotfiles.git", Path: repo}}); err != nil {
		t.Fatalf("save config: %v", err)
	}
	flagConfig = configPath
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")

	if err := runEdit(&cobra.Command{}, nil); err == nil {
		t.Fatal("expected error when no editor is configured")
	}
}

func restoreEditFlags() func() {
	oldJSON := flagJSON
	oldDryRun := flagDryRun
	oldForce := flagForce
	oldConfig := flagConfig
	return func() {
		flagJSON = oldJSON
		flagDryRun = oldDryRun
		flagForce = oldForce
		flagConfig = oldConfig
	}
}
