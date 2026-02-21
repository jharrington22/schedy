package cli

import (
	"github.com/spf13/cobra"
)

func NewRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "resysched",
		Short: "Reservation scheduler",
	}
	cmd.AddCommand(NewServerCmd())
	cmd.AddCommand(NewUserCmd())
	return cmd
}
