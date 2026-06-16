package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	now := time.Now().UTC().Truncate(time.Second)
	cfg := &Config{
		Repo: RepoConfig{
			URL:  "github.com/user/dotfiles",
			Path: "/home/user/.config/dotctl/repo",
		},
		LastSync: &now,
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Repo.URL != cfg.Repo.URL {
		t.Errorf("Repo.URL = %q, want %q", loaded.Repo.URL, cfg.Repo.URL)
	}
	if loaded.Repo.Path != cfg.Repo.Path {
		t.Errorf("Repo.Path = %q, want %q", loaded.Repo.Path, cfg.Repo.Path)
	}
	if loaded.LastSync == nil || !loaded.LastSync.Equal(now) {
		t.Errorf("LastSync = %v, want %v", loaded.LastSync, now)
	}
}

func TestLoadDefaultPath(t *testing.T) {
	// DefaultPath should return a non-empty string
	p := DefaultPath()
	if p == "" {
		t.Fatal("DefaultPath returned empty string")
	}
}

func TestLoadNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got: %v", err)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(":\n  :\n    [invalid"), 0o644); err != nil {
		t.Fatalf("write invalid yaml fixture: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadRejectsParentTraversalPath(t *testing.T) {
	_, err := Load("../config.yaml")
	if err == nil {
		t.Fatal("expected error for parent traversal path")
	}
	if !strings.Contains(err.Error(), "parent traversal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSaveCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "config.yaml")

	cfg := &Config{
		Repo: RepoConfig{URL: "github.com/test/repo"},
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save with nested dirs: %v", err)
	}

	if !Exists(path) {
		t.Error("file should exist after Save")
	}
}

func TestLoadDefaultRepoPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Save a config WITHOUT repo.path set
	cfg := &Config{
		Repo: RepoConfig{URL: "github.com/user/dots"},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Repo.Path == "" {
		t.Error("expected Repo.Path to be set to default, got empty")
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if Exists(path) {
		t.Error("Exists should return false for missing file")
	}

	if err := os.WriteFile(path, []byte("test"), 0o644); err != nil {
		t.Fatalf("write config fixture: %v", err)
	}
	if !Exists(path) {
		t.Error("Exists should return true for existing file")
	}
}

func TestLoadSingleRepoConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	body := []byte(`
repo:
  url: github.com/user/dotfiles
  path: /tmp/dotfiles
profile: laptop
`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Repo.URL != "github.com/user/dotfiles" {
		t.Fatalf("Repo.URL = %q, want github.com/user/dotfiles", cfg.Repo.URL)
	}
	if cfg.Repo.Path != "/tmp/dotfiles" {
		t.Fatalf("Repo.Path = %q, want /tmp/dotfiles", cfg.Repo.Path)
	}
}

func TestLoadMigratesSingleDeprecatedRepoEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	body := []byte(`
repos:
  - name: default
    url: github.com/user/dotfiles
    path: /tmp/dotfiles
profile: laptop
`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write deprecated config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Repo.URL != "github.com/user/dotfiles" {
		t.Fatalf("Repo.URL = %q, want github.com/user/dotfiles", cfg.Repo.URL)
	}
}

func TestLoadSelectsActiveDeprecatedRepo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	body := []byte(`
repos:
  - name: default
    url: github.com/user/default
    path: /tmp/default
  - name: work
    url: github.com/user/work
    path: /tmp/work
active_repo: work
profile: laptop
`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write deprecated config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Repo.URL != "github.com/user/work" {
		t.Fatalf("Repo.URL = %q, want github.com/user/work", cfg.Repo.URL)
	}
}

func TestLoadRejectsDeprecatedMultiRepoWithoutActiveSelection(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	body := []byte(`
repos:
  - name: default
    url: github.com/user/default
    path: /tmp/default
  - name: work
    url: github.com/user/work
    path: /tmp/work
profile: laptop
`)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write deprecated config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for deprecated multi-repo config without active selection")
	}
}
