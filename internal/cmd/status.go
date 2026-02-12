package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/felipe-veas/dotctl/internal/auth"
	"github.com/felipe-veas/dotctl/internal/gitops"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/felipe-veas/dotctl/pkg/types"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current dotctl status",
		Args:  cobra.NoArgs,
		RunE:  runStatus,
	}
}

func runStatus(cmd *cobra.Command, args []string) error {
	out := output.New(flagJSON)

	cfg, _, err := resolveConfig()
	if err != nil {
		return err
	}

	status := types.StatusResponse{
		Profile: cfg.Profile,
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		Repo: types.RepoStatus{
			Name:   cfg.Repo.Name,
			URL:    cfg.Repo.URL,
			Status: "not cloned",
		},
		Symlinks: types.SymlinkStatus{},
		Auth: types.AuthStatus{
			Method: "unknown",
			OK:     false,
		},
		Warnings: []string{},
		Errors:   []string{},
	}

	if cfg.LastSync != nil {
		status.Repo.LastSync = cfg.LastSync.Format("2006-01-02 15:04:05")
	}

	if gitops.IsRepo(cfg.Repo.Path) {
		inspect, inspectErr := gitops.Inspect(cfg.Repo.Path)
		if inspectErr != nil {
			status.Errors = append(status.Errors, inspectErr.Error())
			status.Repo.Status = "error"
		} else {
			status.Repo.Branch = inspect.Branch
			status.Repo.LastCommit = inspect.LastCommit
			if inspect.Dirty {
				status.Repo.Status = "dirty"
			} else {
				status.Repo.Status = "clean"
			}
		}

		sensitiveFiles, sensitiveErr := trackedSensitiveFiles(cfg.Repo.Path)
		if sensitiveErr != nil {
			status.Errors = append(status.Errors, fmt.Sprintf("security check failed: %v", sensitiveErr))
		} else if len(sensitiveFiles) > 0 {
			status.Warnings = append(status.Warnings, sensitiveTrackedFilesWarning(sensitiveFiles))
		}
	} else if info, statErr := os.Stat(cfg.Repo.Path); statErr == nil && info.IsDir() {
		status.Repo.Status = "not_git_repo"
		status.Errors = append(status.Errors, fmt.Sprintf("%s exists but is not a git repo", cfg.Repo.Path))
	}

	if gitops.IsSSHURL(cfg.Repo.URL) {
		status.Auth = types.AuthStatus{Method: "ssh", OK: true}
	} else {
		authStatus, authErr := auth.Check()
		if authErr != nil {
			status.Auth = types.AuthStatus{Method: "gh", OK: false}
			status.Errors = append(status.Errors, authErr.Error())
		} else {
			status.Auth = types.AuthStatus{
				Method: authStatus.Method,
				User:   authStatus.User,
				OK:     true,
			}
		}
	}

	state, stateErr := resolveManifestState(cfg)
	if stateErr != nil {
		status.Errors = append(status.Errors, stateErr.Error())
	} else {
		status.Symlinks = symlinkStatus(state.Actions, cfg.Repo.Path)
	}

	if out.IsJSON() {
		return out.JSON(status)
	}

	out.Field("Profile", status.Profile)
	out.Field("OS", status.OS+"/"+status.Arch)
	if status.Repo.Name != "" {
		out.Field("Repo name", status.Repo.Name)
	}
	out.Field("Repo", status.Repo.URL+" ("+status.Repo.Status+")")
	if status.Repo.Branch != "" {
		out.Field("Branch", status.Repo.Branch)
	}
	if status.Repo.LastCommit != "" {
		out.Field("Commit", status.Repo.LastCommit)
	}
	if status.Repo.LastSync != "" {
		out.Field("Last sync", status.Repo.LastSync)
	} else {
		out.Field("Last sync", "never")
	}

	authLine := status.Auth.Method
	if status.Auth.User != "" {
		authLine = authLine + " (" + status.Auth.User + ")"
	}
	if status.Auth.OK {
		authLine += " ok"
	} else {
		authLine += " not configured"
	}
	out.Field("Auth", authLine)

	if status.Symlinks.Total > 0 {
		out.Field("Symlinks", output.StatusLine(status.Symlinks.Total, status.Symlinks.OK, status.Symlinks.Broken, status.Symlinks.Drift))
	}

	for _, item := range status.Errors {
		out.Warn("%s", item)
	}
	for _, item := range status.Warnings {
		out.Warn("%s", item)
	}

	return nil
}
