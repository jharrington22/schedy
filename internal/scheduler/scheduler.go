package scheduler

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/example/resy-scheduler/internal/jobs"
)

// Scheduler polls for due jobs and invokes resy-cli to attempt booking.
type Scheduler struct {
	Repo     *jobs.Repo
	ResyBin  string
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
	// Ping first (recommended by resy-cli README)
	if out, err := s.exec(ctx, s.ResyBin, "ping"); err != nil {
		msg := fmt.Sprintf("resy ping failed: %v; out=%s", err, out)
		_ = s.Repo.MarkAttempt(ctx, j.ID, "ping", false, out, &msg)
		return
	}

	for _, t := range j.PreferredTimes {
		args := []string{
			"book",
			fmt.Sprintf("--partySize=%d", j.PartySize),
			fmt.Sprintf("--reservationDate=%s", j.ReservationDate.Format("2006-01-02")),
			fmt.Sprintf("--reservationTimes=%s", t),
			fmt.Sprintf("--venueId=%s", j.VenueID),
			fmt.Sprintf("--reservationTypes=%s", j.ReservationTypes),
		}
		out, err := s.exec(ctx, s.ResyBin, args...)
		if err == nil {
			_ = s.Repo.MarkAttempt(ctx, j.ID, t, true, out, nil)
			return
		}
		msg := fmt.Sprintf("book failed for time=%s: %v", t, err)
		_ = s.Repo.MarkAttempt(ctx, j.ID, t, false, out, &msg)
		// continue to next time
	}

	// If we're past the window, mark failed.
	if time.Now().After(j.WindowEndAt) {
		msg := "attempt window ended without success"
		_ = s.Repo.SetStatus(ctx, j.ID, "failed", &msg)
	}
}

func (s *Scheduler) exec(ctx context.Context, bin string, args ...string) (string, error) {
	// protect against hanging calls
	cctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cctx, bin, args...) //nolint:gosec
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String() + "
" + stderr.String())
	if cctx.Err() == context.DeadlineExceeded {
		return out, fmt.Errorf("timeout")
	}
	if err != nil {
		return out, err
	}
	return out, nil
}
