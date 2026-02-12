package cmd

import (
	"os"
	"runtime"

	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/felipe-veas/dotctl/pkg/types"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current dotctl status",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := output.New(flagJSON)

			cfg, _, err := resolveConfig()
			if err != nil {
				return err
			}

			// Build status response
			repoStatus := "not cloned"
			if info, statErr := os.Stat(cfg.Repo.Path); statErr == nil && info.IsDir() {
				repoStatus = "cloned"
			}

			lastSync := ""
			if cfg.LastSync != nil {
				lastSync = cfg.LastSync.Format("2006-01-02 15:04:05")
			}

			status := types.StatusResponse{
				Profile: cfg.Profile,
				OS:      runtime.GOOS,
				Arch:    runtime.GOARCH,
				Repo: types.RepoStatus{
					URL:      cfg.Repo.URL,
					Status:   repoStatus,
					LastSync: lastSync,
				},
				Symlinks: types.SymlinkStatus{}, // M1 will populate this
				Auth: types.AuthStatus{
					Method: "unknown",
					OK:     false,
				},
				Errors: []string{},
			}

			if out.IsJSON() {
				return out.JSON(status)
			}

			// Human-readable output
			out.Field("Profile", cfg.Profile)
			out.Field("OS", runtime.GOOS+"/"+runtime.GOARCH)
			out.Field("Repo", cfg.Repo.URL+" ("+repoStatus+")")
			if lastSync != "" {
				out.Field("Last sync", lastSync)
			} else {
				out.Field("Last sync", "never")
			}

			return nil
		},
	}
}
