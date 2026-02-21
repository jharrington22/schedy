package scheduler

import (
	"context"
	"time"

	"github.com/example/resy-scheduler/internal/domain/reservation"
)

// Runner is intentionally small: higher layers (web/cli) can drive it on a ticker.
type Runner struct {
	Provider reservation.BookingProvider
}

func (r Runner) Tick(ctx context.Context, now time.Time, job reservation.ReservationJob) (string, bool, error) {
	if now.Before(job.WindowStart) || !now.Before(job.WindowEnd) {
		return "", false, nil
	}
	req := reservation.ReservationRequest{
		VenueID:         job.VenueID,
		Date:            job.Date,
		PartySize:       job.PartySize,
		PreferredTimes:  job.PreferredTimes,
	}
	// TODO: wire user contact info + persistence + retries/backoff.
	// This runner is kept as a placeholder for the DDD refactor scaffold.
	return "", true, r.Provider.Ping(ctx)
}
