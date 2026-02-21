package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/example/resy-scheduler/internal/application/usecases"
	"github.com/example/resy-scheduler/internal/domain/reservation"
	"github.com/example/resy-scheduler/internal/infrastructure/config"
	"github.com/example/resy-scheduler/internal/infrastructure/opentable"
	"github.com/example/resy-scheduler/internal/infrastructure/resy"
)

func NewPingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ping [opentable|resy]",
		Short: "Ping a provider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.FromEnv()
			var p reservation.BookingProvider
			switch args[0] {
			case "opentable":
				p = opentable.New(cfg)
			case "resy":
				p = resy.New(cfg)
			default:
				return fmt.Errorf("unknown provider: %s", args[0])
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			uc := usecases.PingProvider{Provider: p}
			if err := uc.Execute(ctx); err != nil {
				return err
			}
			fmt.Printf("%s: ok\n", p.Name())
			return nil
		},
	}
}
