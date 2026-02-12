package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/felipe-veas/dotctl/internal/logging"
	"github.com/felipe-veas/dotctl/internal/manifest"
	"github.com/felipe-veas/dotctl/internal/output"
)

type hookResultJSON struct {
	Phase       string `json:"phase"`
	Command     string `json:"command"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status"`
	Output      string `json:"output,omitempty"`
	Error       string `json:"error,omitempty"`
}

func runHooks(out *output.Printer, phase string, hooks []manifest.Hook, repoPath string, dryRun bool) ([]hookResultJSON, error) {
	if len(hooks) == 0 {
		return nil, nil
	}

	results := make([]hookResultJSON, 0, len(hooks))
	for _, hook := range hooks {
		result := hookResultJSON{
			Phase:       phase,
			Command:     hook.Command,
			Description: hook.Description,
		}
		logging.Debug("hook start", "phase", phase, "command", hook.Command, "dry_run", dryRun)

		if dryRun {
			result.Status = "would_run"
			if !out.IsJSON() {
				out.Info("Would run %s hook: %s", phase, hook.Command)
			}
			results = append(results, result)
			continue
		}

		if !out.IsJSON() {
			out.Info("→ %s", hook.Command)
		}

		cmd := exec.Command("/bin/sh", "-c", hook.Command)
		cmd.Dir = repoPath
		cmd.Env = append(os.Environ(),
			"DOTCTL_HOOK_PHASE="+phase,
			"DOTCTL_HOOK_REPO="+repoPath,
		)

		combined, err := cmd.CombinedOutput()
		trimmed := strings.TrimSpace(string(combined))
		result.Output = trimmed
		if err != nil {
			result.Status = "error"
			result.Error = err.Error()
			results = append(results, result)
			logging.Error("hook failed", "phase", phase, "command", hook.Command, "error", err, "output", trimmed)
			return results, fmt.Errorf("%s hook failed (%s): %w", phase, hook.Command, err)
		}

		result.Status = "ok"
		results = append(results, result)
		logging.Info("hook complete", "phase", phase, "command", hook.Command)
	}

	return results, nil
}
