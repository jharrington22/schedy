package scheduler

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/example/resy-scheduler/internal/jobs"
	"github.com/example/resy-scheduler/internal/resy"
)

// Scheduler polls for due jobs and attempts booking via the Resy API.
type Scheduler struct {
	Repo     *jobs.Repo
	Resy     *resy.Client
	Interval time.Duration

	mu sync.Mutex
	wg sync.WaitGroup
}

func (s *Scheduler) Run(ctx context.Context) error {
	t := time.NewTicker(s.Interval)
	defer t.Stop()

	// kick immediately
	s.tick(ctx)

	for {
		select {
		case <-ctx.Done():
			s.wg.Wait()
			return ctx.Err()
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	js, err := s.Repo.DueJobs(ctx, 25)
	if err != nil {
		log.Printf("scheduler: due jobs query failed: %v", err)
		return
	}

	now := time.Now()
	for _, j := range js {
		na := j.NextAttemptAt(now)
		if na.After(now) {
			continue
		}

		j := j
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.runJobAttempt(ctx, j)
		}()
	}
}

func (s *Scheduler) runJobAttempt(ctx context.Context, j jobs.Job) {
	// Ping first (recommended by resy-cli docs / standard troubleshooting)
	if err := s.Resy.Ping(ctx); err != nil {
		msg := fmt.Sprintf("resy ping failed: %v", err)
		_ = s.Repo.MarkAttempt(ctx, j.ID, "ping", false, "", &msg)
		return
	}

	// Book will try preferred times in order and return nil on first success.
	if err := s.Resy.Book(ctx, j.VenueID, j.PartySize, j.ReservationDate, j.PreferredTimes, j.ReservationTypes); err == nil {
		_ = s.Repo.MarkAttempt(ctx, j.ID, "book", true, "booked", nil)
		return
	} else {
		msg := fmt.Sprintf("book failed: %v", err)
		_ = s.Repo.MarkAttempt(ctx, j.ID, "book", false, "", &msg)
	}

	// If we're past the window, mark failed.
	if time.Now().After(j.WindowEndAt) {
		msg := "attempt window ended without success"
		_ = s.Repo.SetStatus(ctx, j.ID, "failed", &msg)
	}
}

