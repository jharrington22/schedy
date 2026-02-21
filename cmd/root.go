package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	CommitSHA = "none"
	BuildDate = "unknown"
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "resysched",
		Short: "Reservation scheduler + web UI that books Resy reservations during an optimal window",
	}

	root.AddCommand(newVersionCmd())
	root.AddCommand(newKeysCmd())
	root.AddCommand(newServerCmd())
	root.AddCommand(newUserCmd())
	root.AddCommand(newJobCmd())

	return root
}

func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
