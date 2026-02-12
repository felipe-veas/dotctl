package cmd

import (
	"github.com/spf13/cobra"
)

// Global flags.
var (
	flagProfile string
	flagJSON    bool
	flagDryRun  bool
	flagVerbose bool
	flagForce   bool
	flagConfig  string
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "dotctl",
		Short:         "Sync dotfiles across machines (macOS + Linux)",
		Long:          "dotctl syncs dotfiles/configs between devices using a private GitHub repo as source of truth.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVar(&flagProfile, "profile", "", "active profile name")
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
		newPullCmd(),
		newPushCmd(),
		newOpenCmd(),
		newDoctorCmd(),
	)

	return root
}
