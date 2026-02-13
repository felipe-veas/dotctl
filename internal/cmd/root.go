package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/felipe-veas/dotctl/internal/gitops"
	"github.com/felipe-veas/dotctl/internal/logging"
	"github.com/spf13/cobra"
)

// Global flags.
var (
	flagProfile  string
	flagRepoName string
	flagJSON     bool
	flagDryRun   bool
	flagVerbose  bool
	flagForce    bool
	flagConfig   string
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "dotctl",
		Short:         "Sync dotfiles across machines (macOS + Linux)",
		Long:          "dotctl syncs dotfiles/configs between devices using a private GitHub repo as source of truth.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := logging.Init(flagVerbose); err != nil {
				return fmt.Errorf("initializing logger: %w", err)
			}

			gitops.SetTrace(flagVerbose, os.Stderr)
			logging.Info(
				"command start",
				"command", cmd.CommandPath(),
				"args", args,
				"os", runtime.GOOS,
				"arch", runtime.GOARCH,
				"verbose", flagVerbose,
				"json", flagJSON,
				"dry_run", flagDryRun,
				"force", flagForce,
			)

			if flagVerbose {
				_, _ = fmt.Fprintf(os.Stderr, "[verbose] runtime: %s/%s\n", runtime.GOOS, runtime.GOARCH)
				_, _ = fmt.Fprintf(os.Stderr, "[verbose] log file: %s\n", logging.Path())
			}

			return nil
		},
	}

	root.PersistentFlags().StringVar(&flagProfile, "profile", "", "active profile name")
	root.PersistentFlags().StringVar(&flagRepoName, "repo-name", "", "active repo name (for multi-repo configs)")
	root.PersistentFlags().BoolVar(&flagJSON, "json", false, "output in JSON format")
	root.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "show plan without executing")
	root.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "verbose output")
	root.PersistentFlags().BoolVar(&flagForce, "force", false, "skip confirmations")
	root.PersistentFlags().StringVar(&flagConfig, "config", "", "path to config file")

	root.AddCommand(
		newVersionCmd(),
		newInitCmd(),
		newStatusCmd(),
		newSyncCmd(),
		newDiffCmd(),
		newWatchCmd(),
		newPullCmd(),
		newPushCmd(),
		newManifestCmd(),
		newOpenCmd(),
		newBootstrapCmd(),
		newDoctorCmd(),
		newReposCmd(),
		newSecretsCmd(),
	)

	return root
}
