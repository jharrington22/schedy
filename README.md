# resy-scheduler (DDD-style refactor scaffold) — Go 1.25.5

This tarball refactors the project into a **domain-first (DDD-style)** layout.

## Layout
- `internal/domain/...` — business concepts and rules (no HTTP/DB knowledge)
- `internal/application/...` — use cases and orchestration (calls domain interfaces)
- `internal/infrastructure/...` — external systems (OpenTable/Resy HTTP, DB, config)
- `internal/interfaces/...` — adapters (CLI, Web) translating inputs into use cases
- `cmd/resysched` — binary entrypoint

## What works now
- Provider interface in domain
- Strict time ordering selector: `domain/reservation/ChooseSlotStrict`
- OpenTable provider embedded (no shell-out) behind domain interface
- `resysched ping opentable` (requires `OPENTABLE_TOKEN`)

## What is TODO
- Full web UI + auth + persistence + scheduler loop
- Embedded Resy provider implementation (no shell-out)

## Quick start (podman)
```bash
cp .env.example .env
# set OPENTABLE_TOKEN and BOOKING_* if you want to call Book later
podman-compose up --build
podman-compose exec app sh -lc 'go run ./cmd/resysched ping opentable'
```

## Build locally
```bash
make build
./bin/resysched ping opentable
```
