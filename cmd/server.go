package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/example/resy-scheduler/internal/auth"
	"github.com/example/resy-scheduler/internal/config"
	"github.com/example/resy-scheduler/internal/db"
	"github.com/example/resy-scheduler/internal/jobs"
	"github.com/example/resy-scheduler/internal/migrate"
	"github.com/example/resy-scheduler/internal/scheduler"
	"github.com/example/resy-scheduler/internal/web"
	"github.com/spf13/cobra"
)

func newServerCmd() *cobra.Command {
	var migrateUp bool

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Run the web UI + scheduler",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.FromEnv()
			if err != nil {
				return err
			}

			ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
			defer cancel()

			d, err := db.Open(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer d.Close()

			if err := d.Ping(ctx); err != nil {
				return fmt.Errorf("db ping: %w", err)
			}

			if migrateUp {
				if err := migrate.Up(ctx, d); err != nil {
					return err
				}
			}

			authStore := auth.NewStore(d, cfg.CookieHashKey, cfg.CookieBlockKey)
			jobRepo := jobs.NewRepo(d)

			// scheduler
			s := &scheduler.Scheduler{
				Repo:     jobRepo,
				ResyBin:  cfg.ResyBin,
				Interval: cfg.PollInterval,
			}
			go func() { _ = s.Run(ctx) }()
			fmt.Printf("Configured listen addr: %s\n", cfg.ListenAddr)
			// web
			ws := &web.Server{Auth: authStore, Jobs: jobRepo, BaseURL: cfg.BaseURL}
			return web.Start(ctx, cfg.ListenAddr, ws.Routes())
		},
	}

	cmd.Flags().BoolVar(&migrateUp, "migrate", true, "run database migrations on startup")

	cmd.Flags().Lookup("migrate").NoOptDefVal = "true"
	return cmd
}
