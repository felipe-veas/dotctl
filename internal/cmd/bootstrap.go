package cmd

import (
	"github.com/felipe-veas/dotctl/internal/manifest"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/spf13/cobra"
)

type bootstrapResultJSON struct {
	Profile string           `json:"profile"`
	OS      string           `json:"os"`
	DryRun  bool             `json:"dry_run"`
	Hooks   []hookResultJSON `json:"hooks"`
}

func newBootstrapCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap",
		Short: "Run bootstrap hooks from manifest.yaml",
		Args:  cobra.NoArgs,
		RunE:  runBootstrap,
	}
}

func runBootstrap(cmd *cobra.Command, args []string) error {
	out := output.New(flagJSON)

	cfg, _, err := resolveConfig()
	if err != nil {
		return err
	}

	state, err := resolveManifestState(cfg)
	if err != nil {
		return err
	}

	bootstrapHooks := manifest.ResolveHooks(state.Manifest.Hooks.Bootstrap, state.Context)
	results, hookErr := runHooks(out, "bootstrap", bootstrapHooks, cfg.Repo.Path, flagDryRun)
	response := bootstrapResultJSON{
		Profile: cfg.Profile,
		OS:      state.Context.OS,
		DryRun:  flagDryRun,
		Hooks:   results,
	}

	if out.IsJSON() {
		if err := out.JSON(response); err != nil {
			return err
		}
		return hookErr
	}

	if len(bootstrapHooks) == 0 {
		out.Info("No bootstrap hooks configured for profile %q on %s.", cfg.Profile, state.Context.OS)
		return nil
	}

	if flagDryRun {
		out.Info("Dry run complete: %d bootstrap hook(s) would run.", len(bootstrapHooks))
		return nil
	}

	if hookErr != nil {
		return hookErr
	}

	out.Success("Bootstrap complete (%d hook(s)).", len(bootstrapHooks))
	return nil
}
