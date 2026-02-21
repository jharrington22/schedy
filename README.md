# resysched

A small Go service that provides:

- A **web UI** (login + job management)
- A **scheduler** that runs during an “optimal attempt window”
- Integration with **`resy-cli`** by executing `resy ping` and `resy book ...` (no sorting / scraping)

> Note: You are responsible for complying with Resy’s Terms of Service. This project simply automates your own CLI usage.

## How it works

1. You create a job with:
   - venue id, party size, reservation date
   - preferred times (ordered)
   - an “optimal window” rule (days-out + release time + how long to keep trying)
2. When `now` is inside the computed window, the scheduler:
   - runs `resy ping`
   - tries `resy book ...` for each preferred time until success
   - records attempts and marks the job `booked` or `failed`

`resy-cli`’s README shows `ping` and the internal `book` command usage we invoke.  (Example: `resy book --partySize=2 --reservationDate=... --reservationTimes=18:15:00 --venueId=123 ...`). See upstream docs: https://github.com/lgrees/resy-cli

## Prerequisites

- Go 1.22+
- Podman + podman-compose (recommended for local dev)
- A Resy account + configured `resy-cli` credentials

## Quick start (local dev with podman)

### 1) Generate cookie keys

```bash
make deps   # fetches Go modules and writes go.sum (first time only)
go run . keys
# copy/paste the exports into your shell
```

### 2) Start postgres + app

```bash
cp .env.example .env
podman-compose up --build
```

The UI will be available at http://localhost:8080

### 3) Create your first user

In another terminal (with the same env vars as `.env`):

```bash
make user-add USERNAME=james PASSWORD='a-strong-password'
```

Then sign in at `/login`.

### 4) Configure resy-cli credentials (inside the container)

`resy-cli` stores credentials on disk. For local dev, the compose file mounts `./.resy` into the container at `/data/resy`.

Run:

```bash
podman-compose exec app resy setup
podman-compose exec app resy ping
```

## CLI usage

Run:

```bash
./bin/resysched --help
```

Create a job via CLI (instead of UI):

```bash
./bin/resysched job create   --user-id 1   --name "4 Charles Prime Rib"   --venue-id 123   --party-size 2   --reservation-date 2026-03-20   --preferred-times "19:00,19:15,18:45"   --days-out 30   --release-time "00:00"   --lead-minutes 5   --window-minutes 20   --interval-seconds 10
```

## Environment variables

- `DATABASE_URL` (required)
- `COOKIE_HASH_KEY` (required) base64
- `COOKIE_BLOCK_KEY` (required) base64
- `LISTEN_ADDR` (default `:8080`)
- `RESY_BIN` (default `resy`)
- `SCHED_POLL_SECONDS` (default `2`)

## Kubernetes deployment

See `deploy/k8s/`. It includes:
- Deployment + Service
- A Secret template for DB + cookie keys

You will also need to provide Resy credentials in a persistent volume or secret-mounted directory, because `resy-cli` reads them from disk.

## Safety / rate limiting

This scheduler retries every N seconds within a small window; keep it reasonable.
If you see rate-limit failures, increase `interval_seconds` and/or narrow your time list.
