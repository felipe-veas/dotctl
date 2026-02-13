package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/felipe-veas/dotctl/internal/gitops"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/felipe-veas/dotctl/internal/secrets"
	"github.com/spf13/cobra"
)

func newPushCmd() *cobra.Command {
	var message string

	cmd := &cobra.Command{
		Use:   "push",
		Short: "Stage, commit and push local repo changes",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPush(message)
		},
	}

	cmd.Flags().StringVarP(&message, "message", "m", "", "custom commit message")

	return cmd
}

func runPush(message string) error {
	out := output.New(flagJSON)

	cfg, _, err := resolveConfig()
	if err != nil {
		return err
	}
	scope := detectPushScope(cfg.Repo.Path)

	if flagDryRun {
		dirty, err := gitops.IsDirty(cfg.Repo.Path)
		if err != nil {
			return err
		}

		if out.IsJSON() {
			return out.JSON(map[string]any{
				"dry_run": true,
				"dirty":   dirty,
				"would":   "git add -A && git commit && git push",
			})
		}

		if dirty {
			out.Info("Would stage, commit and push local changes")
		} else {
			out.Info("Nothing to push")
			warnPushScopeMismatch(out, cfg.Repo.Path, scope)
		}
		return nil
	}

	// Preflight: check for unencrypted sensitive files.
	if !flagForce {
		if warns := preflightSecretsCheck(cfg.Repo.Path); len(warns) > 0 {
			for _, w := range warns {
				out.Warn("%s", w)
			}
			return fmt.Errorf("unencrypted sensitive files detected (use --force to override, or encrypt with 'dotctl secrets encrypt')")
		}
	}

	res, err := gitops.Push(cfg.Repo.Path, message, cfg.Profile, time.Now())
	if err != nil {
		return err
	}

	if out.IsJSON() {
		return out.JSON(res)
	}

	if res.NothingToPush {
		out.Info("Nothing to push")
		warnPushScopeMismatch(out, cfg.Repo.Path, scope)
		return nil
	}

	out.Success("Committed and pushed")
	if res.Message != "" {
		out.Info("Message: %s", res.Message)
	}

	return nil
}

type pushScope struct {
	CurrentDir              string
	DifferentFromConfigured bool
	CurrentDirIsRepo        bool
	CurrentDirDirty         bool
}

func detectPushScope(configRepoPath string) pushScope {
	info := pushScope{}

	cwd, err := os.Getwd()
	if err != nil {
		return info
	}
	info.CurrentDir = cwd

	if samePath(cwd, configRepoPath) {
		return info
	}
	info.DifferentFromConfigured = true

	if !gitops.IsRepo(cwd) {
		return info
	}
	info.CurrentDirIsRepo = true

	dirty, err := gitops.IsDirty(cwd)
	if err != nil {
		return info
	}
	info.CurrentDirDirty = dirty

	return info
}

func warnPushScopeMismatch(out *output.Printer, configuredRepoPath string, scope pushScope) {
	if scope.DifferentFromConfigured && scope.CurrentDirIsRepo && scope.CurrentDirDirty {
		out.Warn(
			"Current directory has local changes (%s), but dotctl push uses configured repo path (%s)",
			scope.CurrentDir,
			configuredRepoPath,
		)
	}
}

// preflightSecretsCheck scans staged/modified files for unencrypted sensitive files.
// Returns warning messages for each problematic file, or nil if clean.
func preflightSecretsCheck(repoPath string) []string {
	files, err := gitops.TrackedFiles(repoPath)
	if err != nil {
		return nil // non-fatal: don't block push if git scan fails
	}

	var warns []string
	for _, f := range files {
		if secrets.IsSensitiveName(filepath.Base(f)) {
			warns = append(warns, fmt.Sprintf("unencrypted sensitive file tracked: %s (encrypt with 'dotctl secrets encrypt %s')", f, f))
		}
	}
	return warns
}

func samePath(left, right string) bool {
	leftCanonical := canonicalPath(left)
	rightCanonical := canonicalPath(right)
	return leftCanonical != "" && rightCanonical != "" && leftCanonical == rightCanonical
}

func canonicalPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}

	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(resolved)
	}

	return filepath.Clean(abs)
}
