package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/felipe-veas/dotctl/internal/config"
	"github.com/felipe-veas/dotctl/internal/manifest"
	"github.com/felipe-veas/dotctl/internal/profile"
	"github.com/felipe-veas/dotctl/pkg/types"
)

type manifestState struct {
	Context  profile.Context
	Manifest *manifest.Manifest
	Actions  []manifest.Action
	Skipped  []manifest.Action
}

func resolveManifestState(cfg *config.Config) (manifestState, error) {
	ctx := profile.Resolve(cfg.Profile)

	manifestPath := filepath.Join(cfg.Repo.Path, "manifest.yaml")
	m, err := manifest.Load(manifestPath)
	if err != nil {
		return manifestState{}, fmt.Errorf("loading manifest: %w", err)
	}

	actions, skipped, err := manifest.Resolve(m, ctx, cfg.Repo.Path)
	if err != nil {
		return manifestState{}, fmt.Errorf("resolving manifest: %w", err)
	}

	return manifestState{
		Context:  ctx,
		Manifest: m,
		Actions:  actions,
		Skipped:  skipped,
	}, nil
}

func symlinkStatus(actions []manifest.Action, repoRoot string) types.SymlinkStatus {
	status := types.SymlinkStatus{
		Total: len(actions),
	}

	for _, action := range actions {
		detail := types.SymlinkDetail{
			Source: action.Source,
			Target: action.Target,
		}

		sourcePath := filepath.Join(repoRoot, action.Source)
		if _, err := os.Stat(sourcePath); err != nil {
			status.Broken++
			detail.Status = "broken"
			detail.Error = "source missing in repo"
			status.Details = append(status.Details, detail)
			continue
		}

		switch action.Mode {
		case "copy":
			if _, err := os.Stat(action.Target); err != nil {
				status.Broken++
				detail.Status = "broken"
				if errors.Is(err, os.ErrNotExist) {
					detail.Error = "target missing"
				} else {
					detail.Error = err.Error()
				}
				status.Details = append(status.Details, detail)
				continue
			}
			status.OK++
			detail.Status = "ok"
			status.Details = append(status.Details, detail)
		default:
			info, err := os.Lstat(action.Target)
			if err != nil {
				status.Broken++
				detail.Status = "broken"
				if errors.Is(err, os.ErrNotExist) {
					detail.Error = "target missing"
				} else {
					detail.Error = err.Error()
				}
				status.Details = append(status.Details, detail)
				continue
			}
			if info.Mode()&os.ModeSymlink == 0 {
				status.Drift++
				detail.Status = "drift"
				detail.Error = "target is not a symlink"
				status.Details = append(status.Details, detail)
				continue
			}

			linkDest, err := os.Readlink(action.Target)
			if err != nil {
				status.Broken++
				detail.Status = "broken"
				detail.Error = err.Error()
				status.Details = append(status.Details, detail)
				continue
			}
			if linkDest != sourcePath {
				status.Drift++
				detail.Status = "drift"
				detail.Error = fmt.Sprintf("points to %s", linkDest)
				status.Details = append(status.Details, detail)
				continue
			}

			status.OK++
			detail.Status = "ok"
			status.Details = append(status.Details, detail)
		}
	}

	return status
}
