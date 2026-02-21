package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Start the server (UI + scheduler)",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Starting server... (web UI + scheduler TODO in this refactor scaffold)")
			return nil
		},
	}
}
