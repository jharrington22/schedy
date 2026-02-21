package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/example/resy-scheduler/internal/application/usecases"
	"github.com/example/resy-scheduler/internal/infrastructure/config"
	"github.com/example/resy-scheduler/internal/infrastructure/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User management",
	}
	cmd.AddCommand(newUserAddCmd())
	return cmd
}

func newUserAddCmd() *cobra.Command {
	var username, password string
	c := &cobra.Command{
		Use:   "add",
		Short: "Create a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.FromEnv()
			if err != nil { return err }
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
			if err != nil { return err }
			defer pool.Close()
			if err := postgres.Migrate(ctx, pool); err != nil { return err }

			users := postgres.NewUserRepo(pool)
			u, err := usecases.NewUser(username, password)
			if err != nil { return err }
			if err := users.Create(ctx, u); err != nil { return err }
			_ = users.EnsureCredentialsRow(ctx, u.ID)
			fmt.Println("created user:", u.Username)
			return nil
		},
	}
	c.Flags().StringVar(&username, "username", "", "username")
	c.Flags().StringVar(&password, "password", "", "password")
	_ = c.MarkFlagRequired("username")
	_ = c.MarkFlagRequired("password")
	return c
}
