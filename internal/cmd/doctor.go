package cmd

import (
	"fmt"
	"runtime"

	"github.com/felipe-veas/dotctl/internal/auth"
	"github.com/felipe-veas/dotctl/internal/gitops"
	"github.com/felipe-veas/dotctl/internal/output"
	"github.com/felipe-veas/dotctl/internal/secrets"
	"github.com/felipe-veas/dotctl/pkg/types"
	"github.com/spf13/cobra"
)

type doctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type doctorReport struct {
	Profile  string              `json:"profile"`
	RepoName string              `json:"repo_name,omitempty"`
	OS       string              `json:"os"`
	Arch     string              `json:"arch"`
	RepoPath string              `json:"repo_path"`
	Symlinks types.SymlinkStatus `json:"symlinks"`
	Checks   []doctorCheck       `json:"checks"`
	Warnings []string            `json:"warnings,omitempty"`
	Healthy  bool                `json:"healthy"`
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run health checks for auth, git, manifest and symlinks",
		Args:  cobra.NoArgs,
		RunE:  runDoctor,
	}
}

func runDoctor(cmd *cobra.Command, args []string) error {
	out := output.New(flagJSON)

	cfg, _, err := resolveConfig()
	if err != nil {
		return err
	}

	report := doctorReport{
		Profile:  cfg.Profile,
		RepoName: cfg.Repo.Name,
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		RepoPath: cfg.Repo.Path,
		Checks:   make([]doctorCheck, 0, 8),
		Warnings: []string{},
	}

	addCheck := func(name string, ok bool, detail string) {
		report.Checks = append(report.Checks, doctorCheck{Name: name, OK: ok, Detail: detail})
	}

	if !out.IsJSON() {
		out.Header("System:")
	}
	osDetail := fmt.Sprintf("os: %s/%s", runtime.GOOS, runtime.GOARCH)
	addCheck("os", true, osDetail)
	if !out.IsJSON() {
		out.Success("%s", osDetail)
	}

	gitVersion, err := gitops.GitVersion()
	if err != nil {
		addCheck("git", false, err.Error())
		if !out.IsJSON() {
			out.Error("git: %v", err)
		}
	} else {
		addCheck("git", true, gitVersion)
		if !out.IsJSON() {
			out.Success("%s", gitVersion)
		}
	}

	if gitops.IsSSHURL(cfg.Repo.URL) {
		addCheck("auth", true, "ssh repo URL detected (gh check skipped)")
		if !out.IsJSON() {
			out.Success("auth: ssh repo URL detected")
		}
	} else {
		authStatus, authErr := auth.Check()
		if authErr != nil {
			addCheck("auth", false, authErr.Error())
			if !out.IsJSON() {
				out.Error("auth: %v", authErr)
			}
		} else {
			detail := "gh authenticated"
			if authStatus.User != "" {
				detail = fmt.Sprintf("gh authenticated as %s", authStatus.User)
			}
			addCheck("auth", true, detail)
			if !out.IsJSON() {
				out.Success("%s", detail)
			}
		}
	}

	if !out.IsJSON() {
		out.Header("Repo:")
	}

	if !gitops.IsRepo(cfg.Repo.Path) {
		detail := fmt.Sprintf("repo not found at %s", cfg.Repo.Path)
		addCheck("repo", false, detail)
		if !out.IsJSON() {
			out.Error("%s", detail)
		}
	} else {
		addCheck("repo", true, fmt.Sprintf("repo cloned at %s", cfg.Repo.Path))
		if !out.IsJSON() {
			out.Success("repo cloned at %s", cfg.Repo.Path)
		}

		inspect, inspectErr := gitops.Inspect(cfg.Repo.Path)
		if inspectErr != nil {
			addCheck("repo_status", false, inspectErr.Error())
			if !out.IsJSON() {
				out.Error("repo status: %v", inspectErr)
			}
		} else if inspect.Dirty {
			addCheck("repo_status", false, "repo has uncommitted changes")
			if !out.IsJSON() {
				out.Error("repo dirty")
			}
		} else {
			addCheck("repo_status", true, fmt.Sprintf("repo clean (%s@%s)", inspect.Branch, inspect.LastCommit))
			if !out.IsJSON() {
				out.Success("repo clean (%s@%s)", inspect.Branch, inspect.LastCommit)
			}
		}

		sensitiveFiles, sensitiveErr := trackedSensitiveFiles(cfg.Repo.Path)
		if sensitiveErr != nil {
			addCheck("security", false, fmt.Sprintf("security check failed: %v", sensitiveErr))
			if !out.IsJSON() {
				out.Error("security check failed: %v", sensitiveErr)
			}
		} else if len(sensitiveFiles) > 0 {
			warning := sensitiveTrackedFilesWarning(sensitiveFiles)
			report.Warnings = append(report.Warnings, warning)
			addCheck("security", true, warning)
			if !out.IsJSON() {
				out.Warn("%s", warning)
			}
		} else {
			addCheck("security", true, "no sensitive tracked files detected")
			if !out.IsJSON() {
				out.Success("no sensitive tracked files detected")
			}
		}
	}

	var state manifestState
	state, manifestErr := resolveManifestState(cfg)
	if manifestErr != nil {
		addCheck("manifest", false, manifestErr.Error())
		if !out.IsJSON() {
			out.Error("manifest: %v", manifestErr)
		}
	} else {
		detail := fmt.Sprintf("manifest valid (%d entries, %d for this profile)", len(state.Manifest.Files), len(state.Actions))
		addCheck("manifest", true, detail)
		if !out.IsJSON() {
			out.Success("%s", detail)
		}
	}

	if manifestErr == nil && gitops.IsRepo(cfg.Repo.Path) {
		missing, err := missingGitignorePatterns(cfg.Repo.Path, state.Manifest.Ignore)
		if err != nil {
			addCheck("ignore_patterns", false, fmt.Sprintf("ignore pattern check failed: %v", err))
			if !out.IsJSON() {
				out.Error("ignore pattern check failed: %v", err)
			}
		} else if len(missing) > 0 {
			detail := fmt.Sprintf("missing manifest ignore patterns in .gitignore: %s", joinPreview(missing, 5))
			report.Warnings = append(report.Warnings, detail)
			addCheck("ignore_patterns", true, detail)
			if !out.IsJSON() {
				out.Warn("%s", detail)
			}
		} else {
			addCheck("ignore_patterns", true, "manifest ignore patterns covered by .gitignore")
			if !out.IsJSON() {
				out.Success("manifest ignore patterns covered by .gitignore")
			}
		}
	}

	if manifestErr == nil {
		decryptTool, decryptCount, decryptErr := detectDecryptToolForActions(state.Actions)
		if decryptCount > 0 {
			if decryptErr != nil {
				addCheck("decrypt", false, decryptErr.Error())
				if !out.IsJSON() {
					out.Error("decrypt: %v", decryptErr)
				}
			} else {
				detail := fmt.Sprintf("decrypt ready (%d entries, tool: %s)", decryptCount, decryptTool)
				addCheck("decrypt", true, detail)
				if !out.IsJSON() {
					out.Success("%s", detail)
				}
			}
		}
	}

	// Secrets health check.
	if gitops.IsRepo(cfg.Repo.Path) {
		if !out.IsJSON() {
			out.Header("Secrets:")
		}
		secretsStatus, secretsErr := secrets.GetStatus(cfg.Repo.Path, "")
		if secretsErr != nil {
			addCheck("secrets", false, fmt.Sprintf("secrets check failed: %v", secretsErr))
			if !out.IsJSON() {
				out.Error("secrets check failed: %v", secretsErr)
			}
		} else {
			if secretsStatus.Identity != nil {
				detail := fmt.Sprintf("identity: %s", secretsStatus.Identity.PrivatePath)
				addCheck("secrets_identity", true, detail)
				if !out.IsJSON() {
					out.Success("%s", detail)
				}
			} else {
				addCheck("secrets_identity", true, "identity: not configured")
				if !out.IsJSON() {
					out.Info("  identity: not configured (optional)")
				}
			}

			if len(secretsStatus.EncryptedFiles) > 0 {
				detail := fmt.Sprintf("encrypted files: %d", len(secretsStatus.EncryptedFiles))
				addCheck("secrets_encrypted", true, detail)
				if !out.IsJSON() {
					out.Success("%s", detail)
				}
			}

			if len(secretsStatus.UnprotectedFiles) > 0 {
				names := make([]string, 0, len(secretsStatus.UnprotectedFiles))
				for _, f := range secretsStatus.UnprotectedFiles {
					names = append(names, f.Path)
				}
				detail := fmt.Sprintf("unprotected sensitive files: %s", joinPreview(names, 5))
				report.Warnings = append(report.Warnings, detail)
				addCheck("secrets_unprotected", true, detail)
				if !out.IsJSON() {
					out.Warn("%s", detail)
				}
			}
		}
	}

	if !out.IsJSON() {
		out.Header("Symlinks:")
	}
	if manifestErr != nil {
		report.Symlinks = types.SymlinkStatus{}
		addCheck("symlinks", false, "manifest invalid, symlink checks skipped")
		if !out.IsJSON() {
			out.Error("manifest invalid, symlink checks skipped")
		}
	} else {
		report.Symlinks = symlinkStatus(state.Actions, cfg.Repo.Path)
		if report.Symlinks.Broken == 0 && report.Symlinks.Drift == 0 {
			detail := fmt.Sprintf("%d/%d symlinks ok", report.Symlinks.OK, report.Symlinks.Total)
			addCheck("symlinks", true, detail)
			if !out.IsJSON() {
				out.Success("%s", detail)
			}
		} else {
			detail := fmt.Sprintf("%d ok, %d broken, %d drift", report.Symlinks.OK, report.Symlinks.Broken, report.Symlinks.Drift)
			addCheck("symlinks", false, detail)
			if !out.IsJSON() {
				out.Error("%s", detail)
			}
		}
	}

	healthy := true
	for _, c := range report.Checks {
		if !c.OK {
			healthy = false
			break
		}
	}
	report.Healthy = healthy

	if out.IsJSON() {
		if err := out.JSON(report); err != nil {
			return err
		}
	} else {
		if healthy {
			out.Info("")
			if len(report.Warnings) > 0 {
				out.Info("Overall: HEALTHY (%d warnings)", len(report.Warnings))
			} else {
				out.Info("Overall: HEALTHY")
			}
		} else {
			out.Info("")
			if len(report.Warnings) > 0 {
				out.Info("Overall: UNHEALTHY (%d warnings)", len(report.Warnings))
			} else {
				out.Info("Overall: UNHEALTHY")
			}
		}
	}

	if !healthy {
		return fmt.Errorf("doctor checks failed")
	}
	return nil
}

func joinPreview(items []string, max int) string {
	if len(items) <= max {
		return fmt.Sprintf("%v", items)
	}
	return fmt.Sprintf("%v (+%d more)", items[:max], len(items)-max)
}
