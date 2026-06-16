package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/felipe-veas/dotctl/internal/config"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/spf13/cobra"
)

var runEditorCommand = func(editor string, args ...string) error {
	cmd := exec.CommandContext(context.Background(), editor, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type editTarget struct {
	kind   string
	target string
}

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit [repo|manifest]",
		Short: "Open the repo or manifest in your editor",
		Args:  cobra.RangeArgs(0, 1),
		RunE:  runEdit,
	}
}

func runEdit(cmd *cobra.Command, args []string) error {
	out := output.New(flagJSON)

	cfg, _, err := resolveConfig()
	if err != nil {
		return err
	}

	target, err := resolveEditTarget(cfg, args)
	if err != nil {
		return err
	}

	editor, err := selectEditor()
	if err != nil {
		return err
	}

	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return fmt.Errorf("no editor configured; set VISUAL or EDITOR")
	}

	invocationArgs := append([]string{}, parts[1:]...)
	invocationArgs = append(invocationArgs, target.target)

	if flagDryRun {
		if out.IsJSON() {
			return out.JSON(map[string]any{
				"dry_run": true,
				"editor":  editor,
				"target":  target.target,
				"kind":    target.kind,
				"args":    parts[1:],
			})
		}
		out.Info("Would run %s", strings.Join(append([]string{parts[0]}, invocationArgs...), " "))
		return nil
	}

	if err := runEditorCommand(parts[0], invocationArgs...); err != nil {
		return fmt.Errorf("opening %s with editor: %w", target.target, err)
	}

	if out.IsJSON() {
		return out.JSON(map[string]string{
			"status": "opened",
			"editor": editor,
			"target": target.target,
			"kind":   target.kind,
		})
	}

	out.Success("Opened %s with %s", target.target, editor)
	return nil
}

func selectEditor() (string, error) {
	for _, name := range []string{"VISUAL", "EDITOR"} {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value, nil
		}
	}

	return "", fmt.Errorf("no editor configured; set VISUAL or EDITOR")
}

func resolveEditTarget(cfg *config.Config, args []string) (editTarget, error) {
	kind := "repo"
	if len(args) == 1 {
		kind = strings.TrimSpace(args[0])
	}

	switch kind {
	case "", "repo":
		info, err := os.Stat(cfg.Repo.Path)
		if err != nil {
			if os.IsNotExist(err) {
				return editTarget{}, fmt.Errorf("repo not found: %s", cfg.Repo.Path)
			}
			return editTarget{}, fmt.Errorf("stat repo path: %w", err)
		}
		if !info.IsDir() {
			return editTarget{}, fmt.Errorf("repo path is not a directory: %s", cfg.Repo.Path)
		}
		return editTarget{kind: "repo", target: cfg.Repo.Path}, nil
	case "manifest":
		target := filepath.Join(cfg.Repo.Path, "manifest.yaml")
		if _, err := os.Stat(target); err != nil {
			if os.IsNotExist(err) {
				return editTarget{}, fmt.Errorf("manifest not found: %s", target)
			}
			return editTarget{}, fmt.Errorf("stat manifest: %w", err)
		}
		return editTarget{kind: "manifest", target: target}, nil
	default:
		return editTarget{}, fmt.Errorf("invalid edit target: %q", kind)
	}
}
