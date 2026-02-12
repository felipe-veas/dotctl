package config

import (
	"errors"
	"os"
	"path/filepath"
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
		Profile:  "macstudio",
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
	if loaded.Profile != cfg.Profile {
		t.Errorf("Profile = %q, want %q", loaded.Profile, cfg.Profile)
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

func TestSaveCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "config.yaml")

	cfg := &Config{
		Repo:    RepoConfig{URL: "github.com/test/repo"},
		Profile: "test",
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
		Repo:    RepoConfig{URL: "github.com/user/dots"},
		Profile: "test",
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
