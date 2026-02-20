package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/example/resy-scheduler/internal/auth"
	"github.com/example/resy-scheduler/internal/config"
	"github.com/example/resy-scheduler/internal/db"
	"github.com/example/resy-scheduler/internal/migrate"
	"github.com/spf13/cobra"
)

func newUserCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}
	cmd.AddCommand(newUserAddCmd())
	return cmd
}

func newUserAddCmd() *cobra.Command {
	var username, password string

	c := &cobra.Command{
		Use:   "add",
		Short: "Add a local user (username/password)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.FromEnv()
			if err != nil {
				return err
			}

			ctx := context.Background()
			d, err := db.Open(ctx, cfg.DatabaseURL)
			if err != nil {
				return err
			}
			defer d.Close()

			if err := migrate.Up(ctx, d); err != nil {
				return err
			}

			store := auth.NewStore(d, cfg.CookieHashKey, cfg.CookieBlockKey)
			if err := store.CreateUser(ctx, username, password); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "created user %q\n", username)
			return nil
		},
	}

	c.Flags().StringVar(&username, "username", "", "username")
	c.Flags().StringVar(&password, "password", "", "password")
	_ = c.MarkFlagRequired("username")
	_ = c.MarkFlagRequired("password")
	return c
}
