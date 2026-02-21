package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/example/resy-scheduler/internal/config"
	"github.com/example/resy-scheduler/internal/db"
	"github.com/example/resy-scheduler/internal/jobs"
	"github.com/example/resy-scheduler/internal/migrate"
	"github.com/spf13/cobra"
)

func newJobCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "Manage reservation jobs (non-UI)",
	}
	cmd.AddCommand(newJobCreateCmd())
	cmd.AddCommand(newJobListCmd())
	return cmd
}

func newJobCreateCmd() *cobra.Command {
	var (
		userID          int64
		name            string
		venueID         string
		partySize       int
		resDate         string
		preferredTimes  string
		resTypes        string
		timezone        string
		daysOut         int
		releaseTime     string
		leadMinutes     int
		windowMinutes   int
		intervalSeconds int
	)

	c := &cobra.Command{
		Use:   "create",
		Short: "Create a job with an 'optimal window' rule",
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

			repo := jobs.NewRepo(d)

			rd, err := time.Parse("2006-01-02", resDate)
			if err != nil {
				return fmt.Errorf("invalid --reservation-date (want YYYY-MM-DD)")
			}

			if timezone == "" {
				timezone = "America/New_York"
			}
			loc, err := time.LoadLocation(timezone)
			if err != nil {
				return fmt.Errorf("invalid --timezone: %w", err)
			}
			if releaseTime == "" {
				releaseTime = "00:00"
			}

			openDate := rd.AddDate(0, 0, -daysOut)
			openAtLocal, err := time.ParseInLocation("2006-01-02 15:04", openDate.Format("2006-01-02")+" "+releaseTime, loc)
			if err != nil {
				return fmt.Errorf("invalid --release-time (want HH:MM): %w", err)
			}

			windowStart := openAtLocal.Add(-time.Duration(leadMinutes) * time.Minute).UTC()
			windowEnd := openAtLocal.Add(time.Duration(windowMinutes) * time.Minute).UTC()

			j := jobs.Job{
				UserID:           userID,
				Name:             name,
				VenueID:          venueID,
				PartySize:        partySize,
				ReservationDate:  rd,
				PreferredTimes:   splitCSV(preferredTimes),
				ReservationTypes: resTypes,
				Timezone:         timezone,
				WindowStartAt:    windowStart,
				WindowEndAt:      windowEnd,
				IntervalSec:      intervalSeconds,
			}
			if err := j.Validate(); err != nil {
				return err
			}

			id, err := repo.Create(ctx, j)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "created job id=%d window_start_utc=%s window_end_utc=%s\n",
				id, windowStart.Format(time.RFC3339), windowEnd.Format(time.RFC3339))
			return nil
		},
	}

	c.Flags().Int64Var(&userID, "user-id", 0, "user id (from DB)")
	c.Flags().StringVar(&name, "name", "", "job name")
	c.Flags().StringVar(&venueID, "venue-id", "", "resy venue id")
	c.Flags().IntVar(&partySize, "party-size", 2, "party size")
	c.Flags().StringVar(&resDate, "reservation-date", "", "reservation date YYYY-MM-DD")
	c.Flags().StringVar(&preferredTimes, "preferred-times", "19:00,19:15,18:45", "comma-separated times (HH:MM or HH:MM:SS)")
	c.Flags().StringVar(&resTypes, "reservation-types", "", "optional reservation types (comma-separated)")
	c.Flags().StringVar(&timezone, "timezone", "America/New_York", "timezone used for window math")
	c.Flags().IntVar(&daysOut, "days-out", 30, "days in advance when slots open")
	c.Flags().StringVar(&releaseTime, "release-time", "00:00", "local release time HH:MM")
	c.Flags().IntVar(&leadMinutes, "lead-minutes", 5, "start attempts N minutes before release time")
	c.Flags().IntVar(&windowMinutes, "window-minutes", 20, "run attempts N minutes after release time")
	c.Flags().IntVar(&intervalSeconds, "interval-seconds", 10, "retry interval seconds")

	_ = c.MarkFlagRequired("user-id")
	_ = c.MarkFlagRequired("name")
	_ = c.MarkFlagRequired("venue-id")
	_ = c.MarkFlagRequired("reservation-date")
	return c
}

func newJobListCmd() *cobra.Command {
	var userID int64
	c := &cobra.Command{
		Use:   "list",
		Short: "List jobs for a user",
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

			repo := jobs.NewRepo(d)
			js, err := repo.ListByUser(ctx, userID)
			if err != nil {
				return err
			}
			for _, j := range js {
				fmt.Fprintf(os.Stdout, "id=%d name=%q status=%s window=%s..%s preferred=%s\n",
					j.ID, j.Name, j.Status, j.WindowStartAt.Format(time.RFC3339), j.WindowEndAt.Format(time.RFC3339), strings.Join(j.PreferredTimes, ","))
			}
			return nil
		},
	}
	c.Flags().Int64Var(&userID, "user-id", 0, "user id")
	_ = c.MarkFlagRequired("user-id")
	return c
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}
