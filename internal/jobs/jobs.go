package jobs

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/example/resy-scheduler/internal/db"
)

type Job struct {
	ID               int64
	UserID           int64
	Name             string
	VenueID          string
	PartySize        int
	ReservationDate  time.Time
	PreferredTimes   []string
	ReservationTypes string
	Timezone         string

	WindowStartAt time.Time
	WindowEndAt   time.Time
	IntervalSec   int

	Status        string
	LastAttemptAt *time.Time
	BookedAt      *time.Time
	LastError     *string

	CreatedAt time.Time
	UpdatedAt time.Time
}

func parseTimes(s string) []string {
	parts := strings.Split(s, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		// normalize to HH:MM:SS (resy-cli README uses seconds, e.g. 18:15:00)
		if len(p) == 5 {
			p = p + ":00"
		}
		out = append(out, p)
	}
	return out
}

func joinTimes(times []string) string {
	var cleaned []string
	for _, t := range times {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if len(t) == 5 {
			t += ":00"
		}
		cleaned = append(cleaned, t)
	}
	return strings.Join(cleaned, ",")
}

type Repo struct{ db *db.DB }

func NewRepo(d *db.DB) *Repo { return &Repo{db: d} }

func (r *Repo) Create(ctx context.Context, j Job) (int64, error) {
	var id int64
	err := r.db.QueryRow(ctx, `
INSERT INTO jobs(user_id,name,venue_id,party_size,reservation_date,preferred_times,reservation_types,timezone,window_start_at,window_end_at,interval_seconds,status)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,'active')
RETURNING id`,
		j.UserID, j.Name, j.VenueID, j.PartySize, j.ReservationDate, joinTimes(j.PreferredTimes), j.ReservationTypes, j.Timezone, j.WindowStartAt, j.WindowEndAt, j.IntervalSec,
	).Scan(&id)
	return id, db.WrapNotFound(err)
}

func (r *Repo) ListByUser(ctx context.Context, userID int64) ([]Job, error) {
	rows, err := r.db.Query(ctx, `
SELECT id,user_id,name,venue_id,party_size,reservation_date,preferred_times,reservation_types,timezone,window_start_at,window_end_at,interval_seconds,status,last_attempt_at,booked_at,last_error,created_at,updated_at
FROM jobs
WHERE user_id=$1
ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Job
	for rows.Next() {
		var j Job
		var preferredTimes string
		var lastAttempt, bookedAt *time.Time
		var lastErr *string
		if err := rows.Scan(
			&j.ID, &j.UserID, &j.Name, &j.VenueID, &j.PartySize, &j.ReservationDate, &preferredTimes, &j.ReservationTypes, &j.Timezone,
			&j.WindowStartAt, &j.WindowEndAt, &j.IntervalSec, &j.Status, &lastAttempt, &bookedAt, &lastErr, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, err
		}
		j.PreferredTimes = parseTimes(preferredTimes)
		j.LastAttemptAt = lastAttempt
		j.BookedAt = bookedAt
		j.LastError = lastErr
		out = append(out, j)
	}
	return out, rows.Err()
}

func (r *Repo) GetByIDForUser(ctx context.Context, id, userID int64) (Job, error) {
	var j Job
	var preferredTimes string
	var lastAttempt, bookedAt *time.Time
	var lastErr *string
	err := r.db.QueryRow(ctx, `
SELECT id,user_id,name,venue_id,party_size,reservation_date,preferred_times,reservation_types,timezone,window_start_at,window_end_at,interval_seconds,status,last_attempt_at,booked_at,last_error,created_at,updated_at
FROM jobs
WHERE id=$1 AND user_id=$2`, id, userID).
		Scan(&j.ID, &j.UserID, &j.Name, &j.VenueID, &j.PartySize, &j.ReservationDate, &preferredTimes, &j.ReservationTypes, &j.Timezone,
			&j.WindowStartAt, &j.WindowEndAt, &j.IntervalSec, &j.Status, &lastAttempt, &bookedAt, &lastErr, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		return Job{}, db.WrapNotFound(err)
	}
	j.PreferredTimes = parseTimes(preferredTimes)
	j.LastAttemptAt = lastAttempt
	j.BookedAt = bookedAt
	j.LastError = lastErr
	return j, nil
}

func (r *Repo) SetStatus(ctx context.Context, jobID int64, status string, lastErr *string) error {
	return r.db.Exec(ctx, `UPDATE jobs SET status=$2, last_error=$3 WHERE id=$1`, jobID, status, lastErr)
}

func (r *Repo) MarkAttempt(ctx context.Context, jobID int64, attemptedTime string, success bool, output string, lastErr *string) error {
	if err := r.db.Exec(ctx, `INSERT INTO job_attempts(job_id, success, attempted_time, output) VALUES ($1,$2,$3,$4)`,
		jobID, success, attemptedTime, output); err != nil {
		return err
	}
	if success {
		return r.db.Exec(ctx, `UPDATE jobs SET last_attempt_at=now(), booked_at=now(), status='booked', last_error=NULL WHERE id=$1`, jobID)
	}
	return r.db.Exec(ctx, `UPDATE jobs SET last_attempt_at=now(), last_error=$2 WHERE id=$1`, jobID, lastErr)
}

func (r *Repo) DueJobs(ctx context.Context, limit int) ([]Job, error) {
	rows, err := r.db.Query(ctx, `
SELECT id,user_id,name,venue_id,party_size,reservation_date,preferred_times,reservation_types,timezone,window_start_at,window_end_at,interval_seconds,status,last_attempt_at,booked_at,last_error,created_at,updated_at
FROM jobs
WHERE status='active'
  AND now() >= window_start_at
  AND now() <= window_end_at
ORDER BY window_start_at ASC
LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Job
	for rows.Next() {
		var j Job
		var preferredTimes string
		var lastAttempt, bookedAt *time.Time
		var lastErr *string
		if err := rows.Scan(
			&j.ID, &j.UserID, &j.Name, &j.VenueID, &j.PartySize, &j.ReservationDate, &preferredTimes, &j.ReservationTypes, &j.Timezone,
			&j.WindowStartAt, &j.WindowEndAt, &j.IntervalSec, &j.Status, &lastAttempt, &bookedAt, &lastErr, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, err
		}
		j.PreferredTimes = parseTimes(preferredTimes)
		j.LastAttemptAt = lastAttempt
		j.BookedAt = bookedAt
		j.LastError = lastErr
		out = append(out, j)
	}
	return out, rows.Err()
}

func (j Job) NextAttemptAt(now time.Time) time.Time {
	if j.LastAttemptAt == nil {
		return j.WindowStartAt
	}
	return j.LastAttemptAt.Add(time.Duration(j.IntervalSec) * time.Second)
}

func (j Job) Validate() error {
	if j.Name == "" {
		return fmt.Errorf("name required")
	}
	if j.VenueID == "" {
		return fmt.Errorf("venue_id required")
	}
	if j.PartySize < 1 {
		return fmt.Errorf("party_size required")
	}
	if j.ReservationDate.IsZero() {
		return fmt.Errorf("reservation_date required")
	}
	if len(j.PreferredTimes) == 0 {
		return fmt.Errorf("preferred_times required")
	}
	if j.WindowEndAt.Before(j.WindowStartAt) || j.WindowEndAt.Equal(j.WindowStartAt) {
		return fmt.Errorf("window_end_at must be after window_start_at")
	}
	if j.IntervalSec < 1 {
		return fmt.Errorf("interval_seconds must be >= 1")
	}
	return nil
}
