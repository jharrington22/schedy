package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func NewRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resysched",
		Short: "Reservation scheduler (DDD refactor scaffold)",
	}
	cmd.SilenceUsage = true
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)

	cmd.AddCommand(NewServerCmd())
	cmd.AddCommand(NewPingCmd())

	return cmd
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
