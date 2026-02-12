package cmd

import (
	"strings"
	"testing"

	"github.com/felipe-veas/dotctl/internal/manifest"
	"github.com/felipe-veas/dotctl/internal/output"
)

func TestRunHooksDryRun(t *testing.T) {
	hooks := []manifest.Hook{
		{Command: "echo first"},
		{Command: "echo second"},
	}

	results, err := runHooks(output.New(true), "bootstrap", hooks, t.TempDir(), true)
	if err != nil {
		t.Fatalf("runHooks dry-run returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("result count = %d, want 2", len(results))
	}
	for i, r := range results {
		if r.Status != "would_run" {
			t.Fatalf("results[%d].Status = %q, want would_run", i, r.Status)
		}
	}
}

func TestRunHooksExecutesCommand(t *testing.T) {
	hooks := []manifest.Hook{
		{Command: "printf 'hello-hook'"},
	}

	results, err := runHooks(output.New(true), "post_sync", hooks, t.TempDir(), false)
	if err != nil {
		t.Fatalf("runHooks returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("result count = %d, want 1", len(results))
	}
	if results[0].Status != "ok" {
		t.Fatalf("status = %q, want ok", results[0].Status)
	}
	if results[0].Output != "hello-hook" {
		t.Fatalf("output = %q, want hello-hook", results[0].Output)
	}
}

func TestRunHooksStopsOnError(t *testing.T) {
	hooks := []manifest.Hook{
		{Command: "printf 'before-fail'"},
		{Command: "exit 7"},
		{Command: "printf 'should-not-run'"},
	}

	results, err := runHooks(output.New(true), "bootstrap", hooks, t.TempDir(), false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "bootstrap hook failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("result count = %d, want 2 (stop on first error)", len(results))
	}
	if results[1].Status != "error" {
		t.Fatalf("error hook status = %q, want error", results[1].Status)
	}
}
