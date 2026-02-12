package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felipe-veas/dotctl/internal/manifest"
)

func TestDiffSymlinkDrift(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "repo", "a.txt")
	target := filepath.Join(dir, "home", "a.txt")

	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	// Drift: target is regular file instead of symlink.
	if err := os.WriteFile(target, []byte("y"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	action := manifest.Action{
		Source: "a.txt",
		Target: target,
		Mode:   "symlink",
	}
	entry := diffAction(action, source, false)
	if entry.Status != "drift" {
		t.Fatalf("status = %q, want drift", entry.Status)
	}
}

func TestDiffCopyChanged(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "repo", "a.txt")
	target := filepath.Join(dir, "home", "a.txt")

	if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
		t.Fatalf("mkdir source dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir target dir: %v", err)
	}
	if err := os.WriteFile(source, []byte("repo-version"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(target, []byte("local-version"), 0o644); err != nil {
		t.Fatalf("write target: %v", err)
	}

	action := manifest.Action{
		Source: "a.txt",
		Target: target,
		Mode:   "copy",
	}
	entry := diffAction(action, source, false)
	if entry.Status != "changed" {
		t.Fatalf("status = %q, want changed", entry.Status)
	}
}
