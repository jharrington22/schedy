-- Users
CREATE TABLE IF NOT EXISTS users (
  id           BIGSERIAL PRIMARY KEY,
  username     TEXT NOT NULL UNIQUE,
  password_bcrypt TEXT NOT NULL,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Jobs (reservation attempts)
CREATE TABLE IF NOT EXISTS jobs (
  id               BIGSERIAL PRIMARY KEY,
  user_id          BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  name             TEXT NOT NULL,
  venue_id         TEXT NOT NULL,
  party_size       INT NOT NULL CHECK (party_size >= 1 AND party_size <= 20),
  reservation_date DATE NOT NULL,
  preferred_times  TEXT NOT NULL, -- comma-separated HH:MM:SS (resy-cli expects seconds)
  reservation_types TEXT NOT NULL DEFAULT '',

  timezone         TEXT NOT NULL DEFAULT 'America/New_York',

  -- attempt window in local time of timezone
  window_start_at  TIMESTAMPTZ NOT NULL,
  window_end_at    TIMESTAMPTZ NOT NULL,
  interval_seconds INT NOT NULL DEFAULT 10 CHECK (interval_seconds >= 1 AND interval_seconds <= 300),

  status           TEXT NOT NULL DEFAULT 'active', -- active|booked|disabled|failed
  last_attempt_at  TIMESTAMPTZ,
  booked_at        TIMESTAMPTZ,
  last_error       TEXT,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS jobs_status_window_idx ON jobs(status, window_start_at, window_end_at);

-- Attempts
CREATE TABLE IF NOT EXISTS job_attempts (
  id          BIGSERIAL PRIMARY KEY,
  job_id      BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  attempted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  success     BOOLEAN NOT NULL,
  attempted_time TEXT NOT NULL,
  output      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS job_attempts_job_id_idx ON job_attempts(job_id, attempted_at DESC);

-- keep updated_at current
CREATE OR REPLACE FUNCTION set_updated_at() RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_jobs_updated_at ON jobs;
CREATE TRIGGER trg_jobs_updated_at
BEFORE UPDATE ON jobs
FOR EACH ROW EXECUTE FUNCTION set_updated_at();
