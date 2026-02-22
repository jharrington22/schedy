package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/example/resy-scheduler/internal/application/usecases"
	"github.com/example/resy-scheduler/internal/infrastructure/config"
	"github.com/example/resy-scheduler/internal/infrastructure/crypto"
	"github.com/example/resy-scheduler/internal/infrastructure/postgres"
	"github.com/example/resy-scheduler/internal/interfaces/web"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/spf13/cobra"
)

func NewServerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "server",
		Short: "Start web UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.FromEnv()
			if err != nil { return err }

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()

			pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
			if err != nil { return err }
			defer pool.Close()

			if err := postgres.Migrate(ctx, pool); err != nil { return err }

			aead, err := crypto.New(cfg.CredEncKey)
			if err != nil { return err }

			repo := postgres.NewUserRepo(pool)
			_ = usecases.AuthService{Users: repo} // keep import stable
			auth := usecases.AuthService{Users: repo}
			creds := usecases.CredentialsService{Users: repo, AEAD: aead}

			sessions := web.NewSessionManager(cfg.SessionHashKey, cfg.SessionBlockKey)
			tmpl, err := web.ParseTemplates()
			if err != nil { return err }

			srv := web.New(cfg.HTTPAddr, sessions, auth, creds, tmpl)
			fmt.Printf("HTTP listening on %s\n", cfg.HTTPAddr)
			return srv.ListenAndServe()
		},
	}
}
