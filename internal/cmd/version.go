package cmd

import (
	"fmt"

	"github.com/felipe-veas/dotctl/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print dotctl version",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("dotctl %s\n", version.Full())
		},
	}
}
