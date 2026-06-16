package cmd

import (
	"testing"

	"github.com/felipe-veas/dotctl/internal/backup"
	"github.com/felipe-veas/dotctl/internal/linker"
	"github.com/felipe-veas/dotctl/internal/manifest"
)

func TestSyncResultIncludesWarnings(t *testing.T) {
	warning := "sensitive manifest targets: configs/ssh/config -> /home/test/.ssh/config"

	result := syncResult(
		[]linker.Result{{
			Action: manifest.Action{Source: "configs/ssh/config", Target: "/home/test/.ssh/config", Mode: "symlink"},
			Status: "created",
		}},
		nil,
		false,
		"pull complete",
		nil,
		nil,
		&backup.RotationResult{Kept: 1, Removed: 0},
		[]string{warning},
	)

	if len(result.Warnings) != 1 || result.Warnings[0] != warning {
		t.Fatalf("warnings = %v, want [%q]", result.Warnings, warning)
	}
	if result.Summary.Created != 1 || result.Summary.Errors != 0 {
		t.Fatalf("summary = %+v, want 1 created and 0 errors", result.Summary)
	}
}
